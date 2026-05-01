# Scanner Egress Policy Example

Scanner Runner validates URL-job browser requests in process, but hosted and self-hosted deployments should also block unsafe egress at the container or host network boundary.

Apply an equivalent policy for scanner job pods in public environments:

- Deny loopback: `127.0.0.0/8`, `::1/128`
- Deny RFC1918 private ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Deny link-local and metadata ranges: `169.254.0.0/16`, `fe80::/10`
- Deny carrier-grade NAT and benchmark ranges: `100.64.0.0/10`, `198.18.0.0/15`
- Deny multicast/reserved ranges: `224.0.0.0/4`, `240.0.0.0/4`, `ff00::/8`
- Allow only DNS and outbound HTTP(S) needed for scanner targets.

Local development is different: the `local` overlay intentionally allows private and loopback targets so developers can scan local apps.

Example host firewall sketch for a dedicated scanner egress interface:

```bash
nft add table inet stageflow_egress
nft add chain inet stageflow_egress output '{ type filter hook output priority 0; policy accept; }'
nft add rule inet stageflow_egress output ip daddr { 10.0.0.0/8, 100.64.0.0/10, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 198.18.0.0/15, 224.0.0.0/4, 240.0.0.0/4 } reject
nft add rule inet stageflow_egress output ip6 daddr { ::1/128, fe80::/10, fc00::/7, ff00::/8 } reject
```

Treat this as a template, not a drop-in production firewall. The exact hook, interface match, and ordering depend on the host network layout.
