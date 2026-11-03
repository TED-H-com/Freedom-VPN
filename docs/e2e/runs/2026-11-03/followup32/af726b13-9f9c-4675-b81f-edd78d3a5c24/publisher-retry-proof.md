# publisher-retry-proof

Real publisher conflict attempts and durable command keys

Status: blocked

Run: af726b13-9f9c-4675-b81f-edd78d3a5c24

Source: 419c065cbf93b83b0114f84071fee62981d90d99

Spec: §14.5, API01

- test_only_stale_etag: true
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:24:39.895794151Z","key":"9bc52246-5cca-4dbb-b776-76d515b7bf48","method":"PATCH","read_ok":true,"rows":1}
- durable_command: {"accepted_operation_rows":1,"at":"2026-11-03T23:24:39.895889721Z","key":"d4814143-839b-459d-8cf7-f41407269b70","method":"PUT","read_ok":true,"rows":1}
- commands: 2
- real_conflicts: 2
