# Follow-up 8 diagnostic attempt 1

Run `942624c2-babc-4610-aa43-9560a9e70683`, harness `2259947`, baseline `6848add` (#284), 2026-10-31. The 20 AT JSON/Markdown files are byte-identical copies, protected by SHA256SUMS. Live make exited 2 (Go 1, 2532.090 seconds).

Recorded case statuses: AT01 blocked; AT02 pass; AT03 pass; AT04 fail; AT05 blocked; AT08 pass; AT09 fail; AT20 fail; AT21 blocked; AT22 fail. AT02's nominal pass is not final completion evidence: its restore matcher could accept a successful child while the parent operation was running. This assertion is corrected in the next run.

Harness corrections required: recognize only ATnn.json as case files (historical AT08-late-operation.json is an operation); match the root operation status; bounded fresh-ETag retries after a nonmutating 412 on suspension; select IPv4 before the eth0-only relay fault; bounded native DNS startup readiness before the one-shot RR matrix; use a nonempty positive-control DNS profile. The empty profile produced VALIDATION_FAILED DNS observations at 08:15:05Z and retained last-good blocks. All 28 RR assertions remain one-shot. Selected-exit family evidence is strengthened separately from aggregate underlay counts.

The second-node full-namespace restart failed independently: unchanged generation 23, TC_FOREIGN_FILTER with fvmb0 read_failed and initial_quarantine down_verify_failed, no serving restoration within 90 seconds. The product assertion remains in the next run. Independent DNS cases move before this fault so it cannot contaminate them. See the dated follow-up 8 section in the current requests.md.

Successful subrows include the immediate finite-grant 3/0 to 2/1 transition, genuine grant deadline expiry, persistent TCP/UDP cutoff, all four bulk native transfers, 27 of 28 initial RR checks, cache expiry, private A/AAAA rebinding isolation, and two-resolver fallback/recovery. They do not erase the failed checks above.
