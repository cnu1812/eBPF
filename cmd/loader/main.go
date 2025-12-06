package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target bpfel -cflags "-D__TARGET_ARCH_arm64" bpf ../../bpf/quic_firewall.c

func main() {
	if err := rlimit.RemoveMemlock(); err != nil { log.Fatal(err) }

	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil { log.Fatal("Loading eBPF objects:", err) }
	defer objs.Close()

	// Attach Uprobe
	exePath := "../../server_bin" 
	ex, err := link.OpenExecutable(exePath)
	if err != nil { log.Fatalf("Could not open binary: %v", err) }

	up, err := ex.Uprobe("main.BanConnection", objs.ProbeBanConnection, nil)
	if err != nil { log.Fatal("Attaching Uprobe:", err) }
	defer up.Close()
	log.Println("✅ Uprobe attached")

	// Attach XDP to Loopback (lo)
	iface, _ := net.InterfaceByName("lo")
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.XdpQuicFilter,
		Interface: iface.Index,
	})
	if err != nil { log.Fatal("Attaching XDP:", err) }
	defer l.Close()
	log.Println("✅ XDP attached")

	log.Println("🔥 Firewall Active. Press Ctrl+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
