# usage-hold

Withhold usage acknowledgements and read the ledger snapshot

Status: pass

Run: 926791b9-36ef-4051-9d44-cd7eafdc8792

Source: 9e493e8f6407dd7f091963fb7e133a11123305c3

Spec: §10, ENR-05

- schema_version: 1
- unacked_count: 0
- snapshot_bytes: 609
- held_status: 503
- held_body: HTTP/1.1 503 Service Unavailable
  Cache-Control: no-store
  Content-Type: application/json
  Date: Sat, 31 Oct 2026 03:43:50 GMT
  Content-Length: 169
{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"usage acknowledgements withheld","r
- other_route: not withheld
- released_status: 415
- released_body: HTTP/1.1 415 Unsupported Media Type
  Cache-Control: no-store
  Content-Length: 161
  Content-Security-Policy: default-src 'none'; frame-ancestors 'none'
  Content-Type: application/json; charset=utf-8
  Date: Sat, 31 Oct 2026 03:43:50 GMT

- knob: network-side marker on the node resolver proxy; control-side freeze cannot be released without a new ledger epoch
