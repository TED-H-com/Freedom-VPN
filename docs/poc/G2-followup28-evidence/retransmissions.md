# Complete retransmission ledger

TEST ONLY. All 11 retransmissions are member → responder data. Sequence/ACK values are absolute TCP header values. No TCP read failed. Every observation, including original emissions and return ACK/reply paths, remains in the complete analysis.

| Retry segment | Application cycle | Original segment | Seq / ACK / bytes | Since original / previous (ms) | Node ingress bracket | Result | Read elapsed (ms) |
|---|---|---|---|---|---|---|---|
| 123 | 40 | 122 | 3104351025 / 427154695 / 4 | 210.037466 / 210.037466 | 1032/install | no node egress / responder ingress | 827.017314 |
| 124 | 40 | 122 | 3104351025 / 427154695 / 4 | 417.157518 / 207.120052 | 1033/install | no node egress / responder ingress | 827.017314 |
| 125 | 40 | 122 | 3104351025 / 427154695 / 4 | 826.029367 / 408.871849 | outside all brackets | all four request stages | 827.017314 |
| 225 | 73 | 224 | 3104351157 / 427154827 / 4 | 202.391347 / 202.391347 | outside all brackets | all four request stages | 203.515010 |
| 256 | 83 | 255 | 3104351197 / 427154867 / 4 | 202.866777 / 202.866777 | 1064/install | no node egress / responder ingress | 411.696382 |
| 257 | 83 | 255 | 3104351197 / 427154867 / 4 | 410.837185 / 207.970408 | outside all brackets | all four request stages | 411.696382 |
| 285 | 92 | 284 | 3104351233 / 427154903 / 4 | 208.004199 / 208.004199 | outside all brackets | all four request stages | 208.984293 |
| 310 | 100 | 309 | 3104351265 / 427154935 / 5 | 207.349568 / 207.349568 | outside all brackets | all four request stages | 208.073823 |
| 335 | 108 | 334 | 3104351305 / 427154975 / 5 | 209.090862 / 209.090862 | 1090/install | no node egress / responder ingress | 417.123217 |
| 336 | 108 | 334 | 3104351305 / 427154975 / 5 | 416.166779 / 207.075917 | outside all brackets | all four request stages | 417.123217 |
| 361 | 116 | 360 | 3104351345 / 427155015 / 5 | 208.456796 / 208.456796 | outside all brackets | all four request stages | 209.156781 |

There are seven affected reads. Eleven emissions are absent after node ingress: seven original data segments and four retransmissions. All eleven overlap an install/release command bracket. Successful retries traverse all four request stages, followed by replies reaching the origin. No SYN, SYN-ACK, reply, FIN or pure-ACK retransmission was observed. Command-boundary overlap plus a missing hop supports a guard-loss explanation, but does not expose the exact kernel commit or rule verdict.

Generation 1031 before/after: socket RTO 201 ms, pending bytes 0 → 4. After generation 1032: one retransmission, RTO still 201 ms. After generation 1033: two retransmissions, RTO 402 ms and backoff 1. Before generation 1035: bytes acknowledged and echo received, RTO back to 201 ms, three retransmissions total. Conntrack ID/state remain unchanged across these reads. The first retry is consistent with a [Linux v7.0 tail-loss probe](https://raw.githubusercontent.com/torvalds/linux/v7.0/net/ipv4/tcp_output.c); no kernel timer trace identifies its exact timer type. The later RTO/backoff change is directly present in ss output and is consistent with the [upstream retransmission timer](https://raw.githubusercontent.com/torvalds/linux/v7.0/net/ipv4/tcp_timer.c). These upstream sources describe a candidate mechanism; they are not an attestation of the distribution kernel build.

All flags, per-hop timestamps and install/release/verify classifications are in [the complete packet analysis](tcp-analysis.json.gz); [the derived summary](tcp-summary.json) includes every RTO sample and all missing-stage segments. See [the outcome and qualifications](overview.md).
