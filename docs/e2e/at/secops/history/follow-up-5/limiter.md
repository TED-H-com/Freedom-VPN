# SECOPS-15 body boundary retest: passed

Usage at 8 MiB; other route classes at 1 MiB. D-060 burst checks retain thresholds and metrics.

| Class | Bytes | Expected | Actual | Code |
|---|---:|---:|---:|---|
| usage | 8388607 | 200 | 200 |  |
| usage | 8388608 | 200 | 200 |  |
| usage | 8388609 | 413 | 413 | PAYLOAD_TOO_LARGE |
| browser | 1048575 | 201 | 201 |  |
| browser | 1048576 | 201 | 201 |  |
| browser | 1048577 | 413 | 413 | PAYLOAD_TOO_LARGE |
| agent | 1048575 | 400 | 400 | MALFORMED_REQUEST |
| agent | 1048576 | 400 | 400 | MALFORMED_REQUEST |
| agent | 1048577 | 413 | 413 | PAYLOAD_TOO_LARGE |
| heartbeat | 1048575 | 403 | 403 | INCARNATION_MISMATCH |
| heartbeat | 1048576 | 403 | 403 | INCARNATION_MISMATCH |
| heartbeat | 1048577 | 413 | 413 | PAYLOAD_TOO_LARGE |
| lease | 1048575 | 200 | 200 |  |
| lease | 1048576 | 200 | 200 |  |
| lease | 1048577 | 413 | 413 | PAYLOAD_TOO_LARGE |
| enrollment | 1048575 | 401 | 401 | ENROLLMENT_TOKEN_INVALID |
| enrollment | 1048576 | 401 | 401 | ENROLLMENT_TOKEN_INVALID |
| enrollment | 1048577 | 413 | 413 | PAYLOAD_TOO_LARGE |
| webhook | 1048575 | 401 | 401 | WEBHOOK_SIGNATURE_INVALID |
| webhook | 1048576 | 401 | 401 | WEBHOOK_SIGNATURE_INVALID |
| webhook | 1048577 | 413 | 413 | PAYLOAD_TOO_LARGE |

Other-class cap tests include an actual node-A observation under node-B identity (403), invalid enrollment credentials, unsigned webhook, and whitespace on a bodyless bootstrap route (400). These are size-path tests, not successful forged observations. The usage body is schema-valid with legal trailing JSON whitespace. Duplicate ledger evidence is retained.

[Current JSON](limiter.json); [superseded Follow-up 4 measurement](followup4-limiter.json).
