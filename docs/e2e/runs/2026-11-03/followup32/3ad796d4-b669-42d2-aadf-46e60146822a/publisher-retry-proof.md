# publisher-retry-proof

Real publisher conflict attempts and durable command keys

Status: pass

Run: 3ad796d4-b669-42d2-aadf-46e60146822a

Source: 59cae103757567bb32feb6db71cc8bb89dbb5d6c

Spec: §14.5, API01

- test_only_stale_etag: false
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:35:41.182215718Z","key":"0e3d8b25-8ac9-443e-9c43-bb4ecdbea20d","method":"PATCH","read_ok":true,"rows":1}
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:35:41.182304373Z","key":"86824d24-2f79-405b-a76a-792fc7d39234","method":"PUT","read_ok":true,"rows":1}
- commands: 2
- real_conflicts: 0
