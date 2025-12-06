package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"time"

	"github.com/quic-go/quic-go"
)

func main() {
	fmt.Println("🚦 REALISTIC TRAFFIC SIMULATION ENGINE")
	fmt.Println("   [Blue] Legitimate Users (Browsing...)")
	fmt.Println("   [Red]  DDoS Botnet (Blocked by XDP)")

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}

	// Initial burst to populate the dashboard
	for i := 0; i < 3; i++ {
		go simulateUser(tlsConf, false)
	}

	for {
		// 1. Legitimate Traffic (Randomized)
		// Real users don't arrive like a metronome.
		if rand.Float32() < 0.6 {
			go simulateUser(tlsConf, false)
		}

		// 2. Attack Vectors (Bursts)
		// Attacks usually come in waves.
		if rand.Float32() < 0.15 {
			// Spawn a small "botnet" of 3 bad requests at once
			for i := 0; i < 3; i++ {
				go simulateUser(tlsConf, true)
			}
		}

		// Slower loop for realism (Human timescale)
		time.Sleep(time.Duration(rand.Intn(800)+400) * time.Millisecond)
	}
}

func simulateUser(tlsConf *tls.Config, isMalicious bool) {
	// Good users stay connected longer (reading the page)
	duration := 15 * time.Second
	if isMalicious {
		duration = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	conn, err := quic.DialAddr(ctx, "localhost:4242", tlsConf, nil)
	if err != nil { return } // XDP blocked handshake
	defer conn.CloseWithError(0, "bye")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil { return }

	if isMalicious {
		// --- ATTACK ---
		stream.Write([]byte("BAN"))
		time.Sleep(100 * time.Millisecond)
		// The attack payload
		stream.Write([]byte("DDoS_PAYLOAD_XXXXXXXX")) 
	} else {
		// --- NORMAL USER ---
		stream.Write([]byte("GET /index.html"))
		// Simulate reading time
		time.Sleep(time.Duration(rand.Intn(5)+3) * time.Second)
	}
}
