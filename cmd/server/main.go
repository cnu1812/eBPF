package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
)

var (
	knownCIDs      = make(map[string][]byte)
	knownCIDsMutex sync.Mutex
)

//go:noinline
func BanConnection(cid []byte) {
	fmt.Printf("[APP] Banning CID: %x\n", cid)
}

// Helper to ban all known IDs for this connection
func triggerBan() {
	knownCIDsMutex.Lock()
	defer knownCIDsMutex.Unlock()
	
	fmt.Printf("--- BAN TRIGGERED (%d IDs) ---\n", len(knownCIDs))
	for _, cid := range knownCIDs {
		BanConnection(cid)
	}
}

func main() {
	tracerFunc := func(ctx context.Context, p logging.Perspective, ci logging.ConnectionID) *logging.ConnectionTracer {
		return &logging.ConnectionTracer{
			StartedConnection: func(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
				knownCIDsMutex.Lock()
				if srcConnID.Len() > 0 {
					knownCIDs[string(srcConnID.Bytes())] = srcConnID.Bytes()
				}
				knownCIDsMutex.Unlock()
			},
			ReceivedShortHeaderPacket: func(hdr *logging.ShortHeader, size logging.ByteCount, ecn logging.ECN, frames []logging.Frame) {
				if hdr.DestConnectionID.Len() > 0 {
					knownCIDsMutex.Lock()
					sid := string(hdr.DestConnectionID.Bytes())
					if _, exists := knownCIDs[sid]; !exists {
						knownCIDs[sid] = hdr.DestConnectionID.Bytes()
						fmt.Printf("[TRACER] New Live CID: %x\n", hdr.DestConnectionID.Bytes())
					}
					knownCIDsMutex.Unlock()
				}
			},
		}
	}

	quicConf := &quic.Config{Tracer: tracerFunc}
	
	// Create Listener
	listener, err := quic.ListenAddr("0.0.0.0:4242", generateTLSConfig(), quicConf)
	if err != nil { panic(err) }
	
	fmt.Println("Raw QUIC Server listening on :4242")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil { continue }
		
		go func(c quic.Connection) {
			for {
				stream, err := c.AcceptStream(context.Background())
				if err != nil { return }
				
				// Read command
				buf, _ := io.ReadAll(stream)
				cmd := string(buf)
				
				if cmd == "BAN" {
					triggerBan()
					stream.Write([]byte("BANNED"))
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
