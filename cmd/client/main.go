package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"time"

	"github.com/quic-go/quic-go"
)

func main() {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}

	fmt.Println("Dialing QUIC connection...")
	// Dial a raw QUIC connection (bypass HTTP/3 to ensure single connection)
	conn, err := quic.DialAddr(context.Background(), "localhost:4242", tlsConf, nil)
	if err != nil {
		panic(err)
	}
	defer conn.CloseWithError(0, "bye")

	// --- REQUEST 1: Trigger Ban ---
	fmt.Println(">> Opening Stream 1 (Trigger Ban)...")
	stream1, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}
	
	// Send "BAN" command
	stream1.Write([]byte("BAN"))
	stream1.Close() // Close write side to signal EOF

	// Read response
	buf, _ := io.ReadAll(stream1)
	fmt.Printf("Server Response 1: %s\n", string(buf))

	// Wait for Uprobe to sync to Kernel Map
	fmt.Println(">> Waiting for Ban to take effect...")
	time.Sleep(1 * time.Second)

	// --- REQUEST 2: Should Fail ---
	fmt.Println(">> Opening Stream 2 (Should Fail)...")
	
	// If XDP works, this OpenStream (or the Write) will timeout/fail
	// because packets are dropping.
	stream2, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		fmt.Println("SUCCESS: Connection blocked by XDP! (OpenStream failed)", err)
		return
	}

	_, err = stream2.Write([]byte("DATA"))
	if err != nil {
		fmt.Println("SUCCESS: Connection blocked by XDP! (Write failed)", err)
		return
	}

	// Try to read
	stream2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadAll(stream2)
	if err != nil {
		fmt.Println("SUCCESS: Connection blocked by XDP! (Read Timeout)", err)
	} else {
		fmt.Println("FAIL: Stream 2 succeeded (XDP didn't drop it)")
	}
}
