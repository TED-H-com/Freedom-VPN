# revoke

Operator revoke

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §5, §19 AT13

- fresh_admin_auth: true
- node_write: {"started_at":"2026-11-04T10:03:07.926391994Z","finished_at":"2026-11-04T10:03:07.940297897Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"POST","path":"/api/v1/admin/nodes/01a1a5ee-fed7-781a-9ba1-00398dc11b55/revoke","if_match":"\"234\"","idempotency_key":"4177f783-61b2-4940-a055-a40118c44086","body_sha256":"ab72742979623ecd9d0c3dcaecefed41ffb223e610fdffb030d9ff49d01b1ea9","started_at":"2026-11-04T10:03:07.926408657Z","finished_at":"2026-11-04T10:03:07.940290651Z","status":202,"error_code":"","transport_error":false}]}
- revoke: 202
- lifecycle: revoked
- connections: existing-flow teardown was not separately probed
- timed_section: revoke
