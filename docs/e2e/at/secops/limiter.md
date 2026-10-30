# SECOPS-06: class buckets passed; usage size boundary blocked

Source `9b17127c888f94191d759617c99fe1225078a092`. Unmodified settings are in [runtime settings](runtime-settings.json).

| Class / probe | First rejection | Retry-After | Metric delta |
|---|---:|---|---:|
| heartbeat | 6 | `1` | 1 |
| lease | 6 | `1` | 1 |
| usage | 11 | `1` | 1 |
| enrollment | 11 | `1` | 1 |
| webhook | 11 | `1` | 1 |
| anonymous_shared | 11 | `1` | 1 |

All refusals above are 429 RATE_LIMITED with the matching class/limiter label. Enrollment and provider webhook share the anonymous address bucket. Observations use schema-invalid `{}` to avoid fabricated telemetry; lease and usage payloads are valid.

| Valid usage body bytes | Expected HTTP | Actual HTTP |
|---:|---:|---:|
| 1048576 | 200 | 200 |
| 1048577 | 200 | 400 |
| 8388608 | 200 | 400 |
| 8388609 | 413 | 413 |

Bodies contain the same valid event plus legal trailing JSON spaces. The optional Idempotency-Key header is absent. The 1 MiB canonicalization boundary conflicts with the configured 8 MiB usage limit (SECOPS-15). Ten small uploads returned one accepted and nine duplicate results; one ledger row remains.

The barrier-started login completed **20/20**, with zero wrong identities. GET / and session responses were 200 for all 20. This resolves SECOPS-14 landing behavior, not browser-render/platform acceptance.

Login completion 11 / Retry-After 4 and initiation 41 / Retry-After 1 remain the attributed [Follow-up 3 measurement](followup3-limiter.json). The combined run stopped during native scale before repeating those late calls. IPv6 /64 remains component verified; the live source uses IPv4 loopback.

[JSON record](limiter.json) includes UTC, commands, digests and response/metric evidence. Follow-up 3 per-class assumptions are superseded by #145; the earlier login conclusion remains resolved.
