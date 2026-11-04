# publisher-retry-proof

Real publisher conflict attempts and durable command keys

Status: pass

Run: 95fb7eab-6795-48df-b3f2-ed014710d292

Source: 59cae103757567bb32feb6db71cc8bb89dbb5d6c

Spec: §14.5, API01

- test_only_stale_etag: true
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:33:28.340907718Z","key":"ffcf1ca2-e4c3-4a49-91f6-b204e0bfa651","method":"PATCH","read_ok":true,"rows":1}
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:33:28.341004566Z","key":"adb7e726-009a-4446-87f6-b7ed11165d0f","method":"PUT","read_ok":true,"rows":1}
- commands: 2
- real_conflicts: 2
