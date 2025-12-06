# eBPF QUIC Firewall

A high-performance "Self-Defending" QUIC server that uses **eBPF XDP** to drop malicious connections at the network driver level, bypassing the OS stack.

## Architecture

1.  **User Space (Uprobes):** The Go server traces its own function calls. When it decides to ban a user (via `/ban`), it extracts the live Connection ID (CID) and updates an eBPF Map.
2.  **Kernel Space (XDP):** An XDP program hooks the network interface. It parses every incoming UDP packet, extracts the QUIC Short Header, checks the CID against the Map, and drops packets instantly if banned.

## Prerequisites

* Linux Kernel 5.10+ (Requires BTF support)
* Go 1.22+
* Clang/LLVM

## How to Run

### 1. Generate BPF Artifacts
```bash
cd cmd/loader
go generate
```

### 2. Build the Server
```bash
# Must disable inlining (-l) and optimization (-N) for Uprobes to work reliably
cd ../..
go build -gcflags "-N -l" -o server_bin ./cmd/server
```

### 3. Run the Loader (Firewall)
This loads the XDP program and attaches the Uprobes.
```bash
sudo -E go run cmd/loader/main.go
```

### 4. Run the Server
```bash
./server_bin
```

### 5. Trigger the Ban (Client)
```bash
go run cmd/client/main.go
```
