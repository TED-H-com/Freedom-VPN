# publisher-retry-proof

Real publisher conflict attempts and durable command keys

Status: pass

Run: c52b760c-c289-4220-a722-6c5a059819b9

Source: fff680c6c2cd539b77cc534ce9d090f8ad9a8ec3

Spec: §14.5, API01

- test_only_stale_etag: false
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:10:36.066990526Z","key":"48eabcc3-e5a7-4530-9637-3b4520f02f8c","method":"PATCH","read_ok":true,"rows":1}
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:10:36.067102567Z","key":"2dd6949c-317b-4c1d-bb01-3367c136a982","method":"PUT","read_ok":true,"rows":1}
- commands: 2
- real_conflicts: 0
