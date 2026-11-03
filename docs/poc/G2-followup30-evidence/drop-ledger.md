# Exact guard-drop ledger

All seven missing probe TCP egresses have unique NFLOG matches. All are member → responder, rule `inet fvpn fv_deadline_guard iifname "fv-fh" drop`. The kernel timestamps lie inside their generation’s install-command bracket; the atomic commit is not assigned to a command endpoint. The window is **unqualified** for host quietness, and B1 stays FAIL.

| Log / segment | Cycle | Type | Generation | Kernel time since install start (ms) | Installed-by after kernel time (ms) | Read elapsed (ms) |
|---|---:|---|---:|---:|---:|---:|
| 1 / 29 | 9 | data | 1002 | 17.286228 | 14.485036 | 211.293988 |
| 2 / 81 | 26 | data | 1018 | 17.536186 | 0.035940 | 416.739885 |
| 3 / 82 | 26 | retry data | 1019 | 7.002816 | 8.023485 | 416.739885 |
| 4 / 152 | 49 | data | 1041 | 6.394455 | 3.630502 | 203.733191 |
| 15 / 282 | 92 | data | 1069 | 8.455400 | 2.770352 | 210.755759 |
| 16 / 285 | 92 | ACK | 1070 | 2.795397 | 13.062457 | 210.755759 |
| 18 / 339 | 110 | data | 1093 | 19.039096 | 1.688438 | 206.931190 |

[The complete summary](window-summary.json) retains absolute kernel timestamps and all four command endpoints, sequence/ACK/IP identities, every retry and CPU breach. [The guard analysis](guard-analysis.json) retains unique node matches and zero contradictory egresses. The [full TCP analysis](tcp-analysis.json.gz) retains every endpoint and hop observation, including all seven retransmissions.
