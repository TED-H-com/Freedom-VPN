# lab-AT13-d147

Mechanism: D-147 RELATED IPv4 fragmentation-needed per source set and overlap

Status: pass

Run: df9dbad7-049d-4eaa-ab98-5490906cc3e9; started UTC: 2026-11-04T14:06:12Z; source dirty: false

Spec: §19 AT13, ISO01, ISO02, MTR01, D-146, D-147

UTC: 2026-11-04T14:06:37Z

Build: 56d814113913ddc6177d47d4a5160a65c93add25

Image digests:

- fvpn-w4-isolation-lab:lab sha256:9b53681fbed522d7b3a619f5991cfc66abb8fb4d9d82847bdb8fa39c73fc98b5

Policy revision: generation 4200

Node incarnation: mechanism-lab

Expected: Genuine router IPv4 ICMP fragmentation-needed (type 3, code 4, MTU 1000) from resolver_v4-only, denied_v4-only, host_v4-only, and denied_v4+host_v4 overlap sources reaches the admitted member when conntrack marks it RELATED, is counted once in d_<device> at the device's assigned priority (1:2f), and matches independent captures; time-exceeded is dropped at ctdenyd, unsolicited fragmentation-needed at ctinv, and member upload to that source is dropped with 0 upload usage.

Actual: Verified D-147 on generation 4200 across all 4 source-set rows (resolver_v4_only(203.0.113.53), denied_v4_only(192.168.34.2), host_v4_only(203.0.113.33), denied_v4_and_host_v4_overlap(192.168.34.1)) with genuine router frag-needed (MTU 1000) counted once in d_<device> at 1:2f matching independent captures, and all 3 negatives dropped per row.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- forward_view: generation=4200 lab_member_accept=27 mss_clamp=6 pmtu_accept=9
- d147_resolver_v4_only: src=203.0.113.53 sets=resolver_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=38p/7289B [tcp=1529B + frag_needed=10p/5760B] (tc 1:2f 38p/7821B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->ctbadns=1p/84B (u=0)
- d147_denied_v4_only: src=192.168.34.2 sets=denied_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=51p/7965B [tcp=2205B + frag_needed=10p/5760B] (tc 1:2f 51p/8679B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->ctdeny=1p/84B (u=0)
- d147_host_v4_only: src=203.0.113.33 sets=host_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=42p/7497B [tcp=1737B + frag_needed=10p/5760B] (tc 1:2f 42p/8085B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->cthost=1p/84B (u=0)
- d147_denied_v4_and_host_v4_overlap: src=192.168.34.1 sets=denied_v4+host_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=60p/8433B [tcp=2673B + frag_needed=10p/5760B] (tc 1:2f 60p/9273B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->ctdeny=1p/84B (u=0)
- scope: mechanism lab; D-147 per-source-set and overlap verification; product status is AT13.json

Checks (full positive controls and observations in JSON):

- D-147 resolver_v4_only (203.0.113.53): pass; src=203.0.113.53 sets=resolver_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=38p/7289B [tcp=1529B + frag_needed=10p/5760B] (tc 1:2f 38p/7821B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->ctbadns=1p/84B (u=0); counted positive=true
- D-147 denied_v4_only (192.168.34.2): pass; src=192.168.34.2 sets=denied_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=51p/7965B [tcp=2205B + frag_needed=10p/5760B] (tc 1:2f 51p/8679B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->ctdeny=1p/84B (u=0); counted positive=true
- D-147 host_v4_only (203.0.113.33): pass; src=203.0.113.33 sets=host_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=42p/7497B [tcp=1737B + frag_needed=10p/5760B] (tc 1:2f 42p/8085B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->cthost=1p/84B (u=0); counted positive=true
- D-147 denied_v4_and_host_v4_overlap (192.168.34.1): pass; src=192.168.34.1 sets=denied_v4+host_v4 pmtu=1000 upload_bytes=64064 u_019a00000000700080000000000000a1=84p/80007B (tc 1:2f 84p/81183B drops=0) d_019a00000000700080000000000000a1=60p/8433B [tcp=2673B + frag_needed=10p/5760B] (tc 1:2f 60p/9273B drops=0); negatives: time_exceeded->ctdenyd=1p/112B (d=0), unsolicited_frag_needed->ctinv=1p/60B (d=0), member_upload->ctdeny=1p/84B (u=0); counted positive=true
