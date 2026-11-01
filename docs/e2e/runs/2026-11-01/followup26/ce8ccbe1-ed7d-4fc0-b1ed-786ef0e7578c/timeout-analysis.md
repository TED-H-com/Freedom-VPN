# ce8ccbe1: live verification closure and delayed native delivery

The original IPv6 request failed after **5.003 seconds**, with five SYNs at the member tunnel, five eventual node `fvmb0` receipts and no origin receipt. It had independently passed populated serving admission at **12:48:18.542313158Z**, generation 5. Its kernel counter samples changed **5 → 0**. This is a withdrawal after admission, not evidence that admission never completed.

| Event (UTC) | Evidence and implication |
|---|---|
| 12:48:21.866133 | First member-tunnel SYN, packet 31. |
| 12:48:22.069520–22.069756 | The same type-1 message hash reaches node loopback; the matching receiver-index type-2 response leaves the node and reaches the member with a loopback source. These are structural headers, not proof of successful authentication. |
| 12:48:23.516523162 | Producer sequence 72: failed **live verification**, candidate generation 5. |
| +1,650.453…1,704.071 ms from first SYN | Producer sequence 73: successful baseline write, cause `verification_ISOLATION_POLICY_INVALID`. Sequence 74 writes the same baseline cause again. |
| 12:48:23.604196 | Accepted report: applied generation 5, forwarding=false, `safety_check_failed`, `ISOLATION_POLICY_INVALID` at stage `apply`. The stage name does not turn sequence 72 into an Apply. |
| +5.003 s HTTP elapsed | Original request ends `http_unavailable`; no replacement request is sent. |
| 12:48:28.649856940 | Producer sequence 77: failed **Apply**, still candidate generation 5. Sequence 78 records `apply_rollback`; sequences 79/80 record the same verification baseline cause. The next accepted report additionally carries `apply_candidate`. |
| 12:48:32.309801–32.309802 | All five queued SYN headers arrive at `fvmb0`. Matching the first SYN gives member-egress → node-ingress bounds **[10,443.666, 10,443.671] ms**. This is after the HTTP deadline and after the recorded baseline writes. |
| 12:48:33.654164 | After tracer removal, the central report returns to generation 5, forwarding=true. |
| 12:48:34.358377123–38.950282588 | Independent post-capture recovery gate passes, still generation 5. The failed request stays failed. |

The exact private NFT listing replay identifies the tracer table's unscoped netdev ingress `mangle` as the deciding `ISOLATION_POLICY_INVALID` rejection. The unchanged verifier also rejects the tracer's isolated broad prerouting rule in the regression fixture. The issued generation-5 lease runs from 12:48:13.634556Z to the following day, contains one member, and remains valid across the request. This is not reported lease expiry or an unfinished admission. The baseline's generation-0 marker is local enforcement state; the exported desired generations are 1, 3, 4 and 5, and the accepted applied generation never moves to 0.

The tracer remains present during the failed live read and subsequent failed Apply. Removing it permits recovery without a new generation. This explains why the node closed and did not recover while observed. It **does not establish that closure alone caused the HTTP timeout**: native delivery itself exceeded the request deadline. NetBird reported the peers connected during that interval while the kernel handshake remained 0; the final sample records completed handshakes at 12:48:32. The record keeps those different observations separate.

The re-grant row fails its counted-initial-path prerequisite and sends no re-grant request. It is not a reproduced F14-01 forwarding failure. See [central record](central-record.json), [packet/writer joins](serving-packet-joins.json), [native session intervals](session-intervals.json) and [original request](initial-selection.json). No historical row is reclassified and no product transport fix is selected.
