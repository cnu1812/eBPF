//go:build ignore
#include <linux/types.h>
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#ifdef __TARGET_ARCH_arm64
struct user_pt_regs {
	__u64		regs[31];
	__u64		sp;
	__u64		pc;
	__u64		pstate;
};
#endif
#include <bpf/bpf_tracing.h>

// MAP: 4-byte key
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, __u32); 
    __type(value, __u8);  
} blacklist SEC(".maps");

SEC("uprobe/ban_connection")
int probe_ban_connection(struct pt_regs *ctx) {
    void *data_ptr = (void *)PT_REGS_PARM1(ctx); 
    __u64 data_len = PT_REGS_PARM2(ctx);           

    if (data_len != 4) return 0; 

    __u32 cid = 0;
    bpf_probe_read_user(&cid, sizeof(cid), data_ptr);

    __u8 val = 1;
    bpf_map_update_elem(&blacklist, &cid, &val, BPF_ANY);
    bpf_printk("eBPF Uprobe: BANNED CID %x\n", cid);
    return 0;
}

SEC("xdp")
int xdp_quic_filter(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    struct ethhdr *eth = data;
    if ((void*)(eth + 1) > data_end) return XDP_PASS;

    struct iphdr *ip = (void*)(eth + 1);
    if ((void*)(ip + 1) > data_end) return XDP_PASS;
    if (ip->protocol != IPPROTO_UDP) return XDP_PASS;

    struct udphdr *udp = (void*)((void*)ip + (ip->ihl * 4));
    if ((void*)(udp + 1) > data_end) return XDP_PASS;

    __u8 *quic_data = (void*)(udp + 1);
    if ((void*)(quic_data + 1) > data_end) return XDP_PASS;

    __u8 flags = *quic_data;
    
    // Check Short Header
    if ((flags & 0x80) == 0 && (flags & 0x40) == 0x40) {
        // Assume CID is 4 bytes at offset +1
        __u32 *cid_ptr = (__u32*)(quic_data + 1);
        if ((void*)(cid_ptr + 1) > data_end) return XDP_PASS;

        __u32 cid = *cid_ptr; 

        // Debug output to help verify
        // bpf_printk("Wire CID: %x\n", cid); 

        if (bpf_map_lookup_elem(&blacklist, &cid)) {
            bpf_printk("eBPF XDP: !!! DROP MATCH !!! %x\n", cid);
            return XDP_DROP; 
        }
    }
    return XDP_PASS; 
}
char __license[] SEC("license") = "GPL";
