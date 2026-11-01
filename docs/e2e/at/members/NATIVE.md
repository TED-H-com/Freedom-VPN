# Native evidence — follow-up 15

Run `e80bb592-27ed-4ab5-a8c3-6433e7278938`, compiled harness `cc6e0298a791ed6c41b268f1634f543446f8c562`, baseline `9dde5eeb60a6cb6b9b10974c3100ef1ea1352698` (current main at admission; #375, #376, #377, #381 and #382 included).

**682d4434 is now live-qualified:** the forced relay row passes with **Relayed=2 / Direct=0**, followed by same-socket relay-to-direct recovery. The routed hop is added only afterward. Follow-up 14's relay FAIL remains recorded and confounded; it is not reclassified.

The routed IPv6 row passes **64,064 bytes in 1 ms**, with one oversized ingress, one type-2/code-0 PTB seen at router and sender, learned PMTU 1280, and **16** router fragment egresses received. The routed IPv4 row passes **64,064 bytes in 1 ms**: **25 DF=0 oversized datagrams → 50 fragments / 25 groups**, all received, no PTB. The known-PMTU IPv6 row separately passes **64,064 bytes in 1 ms**.

Receive-only rows remain limitations: dual-underlay MTU 1280 is **FAIL**, with **12,012 bytes in 12,011 ms**; IPv6 receive-only transfers **64,064 bytes in 1 ms** but remains **BLOCKED**. Neither is upgraded to a supported-PMTU pass. [Original independent rows](F15-D137-native-paths.json), [packet analyses](history/follow-up-15-captures/e80bb592-27ed-4ab5-a8c3-6433e7278938/analysis.json). Historical F8-02 failures remain unattributed; IPv4 overlay payloads do not prove IPv6 overlay support (D-123).

AT01 remains **BLOCKED**. Its original first selection has **five member SYNs, zero node/origin inner packets, zero origin requests**, with generation **5** and **six member accept rules** before and after. Captures are complete and zero-drop (**201 / 200 / 45** member/node/origin records). Cause is **unproven**; neither the node's sampled generation nor this failure locates the missing native hop. Devstack 25's campaign is consumed as evidence, not a repair on main. [Original attempt/readiness](F15-F11-01-first-selection.json), [native analysis](history/follow-up-15-captures/e80bb592-27ed-4ab5-a8c3-6433e7278938/initial-selection-analysis.json).

All **three D-077 re-grant attempts** pass this run. Every attempt retains its original readiness, member/node/origin packet observations, origin receipt and device counters. These are the captures the harness already made; detailed new WireGuard captures are not invented. `cause_status=unproven`; the earlier F14-01 remains failed and assigned to devstack's native investigation. A passing first selection or re-grant closes neither F11-01 nor F14-01. [All attempts](F15-D077-regrant-attempts.json).

| Original action | Command duration | Stored lease evidence |
|---|---:|---|
| AT02 supervisor stop after direct signal | 0.315547960 s | unchanged closed storage |
| AT02 supervisor resume | 0.470609069 s | new producer retains prior-run lease before renewal |
| AT08 disabled-node Docker restart | 10.107214827 s | new producer retains prior-run lease before renewal |
| AT09 selected namespace stop | 5.157674441 s | unchanged closed storage |
| AT09 namespace restart + observation | 90.551346091 s | includes required 90-second capture; prior-run lease retained before renewal |

[Exact original transitions and producer joins](F15-agent-transitions.json). Docker restart/namespace-stop SIGTERM-to-agent-exit intervals are **not observable here**: the retained parent receipt is unavailable. Their command durations are not graceful shutdown measurements or comparisons against the 13 s process budget.
