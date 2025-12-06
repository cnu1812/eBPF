package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
)

// --- DASHBOARD BROADCASTER ---
type Dashboard struct {
	clients map[chan string]bool
	mu      sync.Mutex
}

var dashboard = &Dashboard{clients: make(map[chan string]bool)}

func (d *Dashboard) AddClient() chan string {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan string, 10)
	d.clients[ch] = true
	return ch
}

func (d *Dashboard) RemoveClient(ch chan string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.clients, ch)
	close(ch)
}

func (d *Dashboard) Broadcast(typeStr, msg string) {
	payload, _ := json.Marshal(map[string]string{"type": typeStr, "msg": msg})
	d.mu.Lock()
	defer d.mu.Unlock()
	for ch := range d.clients {
		select {
		case ch <- string(payload):
		default: // Drop if slow
		}
	}
}

// --- APP LOGIC ---

var (
	knownCIDs      = make(map[string][]byte)
	knownCIDsMutex sync.Mutex
)

//go:noinline
func BanConnection(cid []byte) {
	dashboard.Broadcast("ban", fmt.Sprintf("XDP BLOCK: %x", cid))
}

func triggerBan() {
	knownCIDsMutex.Lock()
	defer knownCIDsMutex.Unlock()
	for _, cid := range knownCIDs {
		BanConnection(cid)
	}
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	clientChan := dashboard.AddClient()
	defer dashboard.RemoveClient(clientChan)
	notify := r.Context().Done()
	for {
		select {
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-notify:
			return
		}
	}
}

func main() {
	// Start Dashboard
	go func() {
		http.Handle("/", http.FileServer(http.Dir("cmd/server/static")))
		http.HandleFunc("/events", handleEvents)
		fmt.Println("🌐 DASHBOARD: http://localhost:8080")
		http.ListenAndServe(":8080", nil)
	}()

	// Setup Tracer
	tracerFunc := func(ctx context.Context, p logging.Perspective, ci logging.ConnectionID) *logging.ConnectionTracer {
		return &logging.ConnectionTracer{
			StartedConnection: func(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
				knownCIDsMutex.Lock()
				if srcConnID.Len() > 0 {
					knownCIDs[string(srcConnID.Bytes())] = srcConnID.Bytes()
					dashboard.Broadcast("conn", fmt.Sprintf("New Connection: %s", remote))
				}
				knownCIDsMutex.Unlock()
			},
			ReceivedShortHeaderPacket: func(hdr *logging.ShortHeader, size logging.ByteCount, ecn logging.ECN, frames []logging.Frame) {
				if hdr.DestConnectionID.Len() > 0 {
					knownCIDsMutex.Lock()
					sid := string(hdr.DestConnectionID.Bytes())
					if _, exists := knownCIDs[sid]; !exists {
						knownCIDs[sid] = hdr.DestConnectionID.Bytes()
						dashboard.Broadcast("packet", "CID Rotation Detected")
					}
					knownCIDsMutex.Unlock()
				}
			},
		}
	}

	quicConf := &quic.Config{Tracer: tracerFunc}
	listener, err := quic.ListenAddr("0.0.0.0:4242", generateTLSConfig(), quicConf)
	if err != nil { panic(err) }
	
	fmt.Println("🚀 SERVER LISTENING :4242")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil { continue }
		go func(c quic.Connection) {
			for {
				stream, err := c.AcceptStream(context.Background())
				if err != nil { return }
				buf, _ := io.ReadAll(stream)
				if string(buf) == "BAN" {
					triggerBan()
					stream.Write([]byte("BANNED"))
					time.Sleep(50 * time.Millisecond)
					stream.Close()
				} else {
					stream.Write([]byte("PONG"))
					stream.Close()
				}
			}
		}(conn)
	}
}

func generateTLSConfig() *tls.Config {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, _ := tls.X509KeyPair(certPEM, keyPEM)
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
