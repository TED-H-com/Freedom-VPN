# usage-hold

Withhold usage acknowledgements and read the ledger snapshot

Status: pass

Run: 630b9f07-62d2-4db1-9737-9060732db1f1

Source: 3c25ce9daa081874988c9b09bc8c26e66d3c1b46

Spec: §10, ENR-05

- schema_version: 1
- unacked_count: 8
- snapshot_bytes: 3281
- held_status: 503
- held_body: HTTP/1.1 503 Service Unavailable
  Cache-Control: no-store
  Content-Type: application/json
  Date: Sat, 31 Oct 2026 04:02:45 GMT
  Content-Length: 169
{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"usage acknowledgements withheld","r
- other_route: not withheld
- released_status: 415
- released_body: HTTP/1.1 415 Unsupported Media Type
  Cache-Control: no-store
  Content-Length: 161
  Content-Security-Policy: default-src 'none'; frame-ancestors 'none'
  Content-Type: application/json; charset=utf-8
  Date: Sat, 31 Oct 2026 04:02:45 GMT

- knob: network-side marker on the node resolver proxy; control-side freeze cannot be released without a new ledger epoch
