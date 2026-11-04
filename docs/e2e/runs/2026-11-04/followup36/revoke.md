# revoke

Operator revoke

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §5, §19 AT13

- fresh_admin_auth: true
- node_write: {"started_at":"2026-11-04T14:56:36.202014643Z","finished_at":"2026-11-04T14:56:36.214534136Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"POST","path":"/api/v1/admin/nodes/01a1a6fb-5e05-741c-ac02-cd000d4036be/revoke","if_match":"\"340\"","idempotency_key":"c94b673e-e8cc-4b29-b44e-73aa80719f14","body_sha256":"ab72742979623ecd9d0c3dcaecefed41ffb223e610fdffb030d9ff49d01b1ea9","started_at":"2026-11-04T14:56:36.202023825Z","finished_at":"2026-11-04T14:56:36.21452757Z","status":202,"error_code":"","transport_error":false}]}
- revoke: 202
- lifecycle: revoked
- connections: existing-flow teardown was not separately probed
- timed_section: revoke
