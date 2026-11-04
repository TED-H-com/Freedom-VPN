# revoke

Operator revoke

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §5, §19 AT13

- fresh_admin_auth: true
- node_write: {"started_at":"2026-11-04T06:01:01.102102606Z","finished_at":"2026-11-04T06:01:01.114472043Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"POST","path":"/api/v1/admin/nodes/01a1a511-4190-721c-b8cf-b5151ee3e45e/revoke","if_match":"\"230\"","idempotency_key":"7cb9a032-ee93-4685-b662-0ac22c063885","body_sha256":"ab72742979623ecd9d0c3dcaecefed41ffb223e610fdffb030d9ff49d01b1ea9","started_at":"2026-11-04T06:01:01.102117243Z","finished_at":"2026-11-04T06:01:01.114464751Z","status":202,"error_code":"","transport_error":false}]}
- revoke: 202
- lifecycle: revoked
- connections: existing-flow teardown was not separately probed
