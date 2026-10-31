# usage-hold

Withhold usage acknowledgements and read the ledger snapshot

Status: pass

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §10, ENR-05

- schema_version: 1
- unacked_count: 8
- snapshot_bytes: 3281
- held_status: 503
- held_body: HTTP/1.1 503 Service Unavailable
  Cache-Control: no-store
  Content-Type: application/json
  Date: Sat, 31 Oct 2026 04:27:17 GMT
  Content-Length: 169
{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"usage acknowledgements withheld","r
- other_route: not withheld
- released_status: 415
- released_body: HTTP/1.1 415 Unsupported Media Type
  Cache-Control: no-store
  Content-Length: 161
  Content-Security-Policy: default-src 'none'; frame-ancestors 'none'
  Content-Type: application/json; charset=utf-8
  Date: Sat, 31 Oct 2026 04:27:17 GMT

- knob: network-side marker on the node resolver proxy; control-side freeze cannot be released without a new ledger epoch
