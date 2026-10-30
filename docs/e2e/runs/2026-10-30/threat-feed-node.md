# threat-feed-node

Issued resolver document reaches the node resolver

Status: blocked

Run: dc2f6353-89ec-434c-bc68-567066d9fe51

Source: b15ff057cd5e91e227477de6a09d977931fa62d8

Spec: §12.5, §19 AT22, D-061

- credential_owner_mode: 0:600
- dns_config: agent-run only; lab admin 127.0.0.1:8080 and DNS 127.0.0.1:53; AdGuard shares node network namespace
- document_id: 01a18b06-ede8-72cd-ada2-b80cdcae7dd8
- content_sha256: 96efd876f78265df3d8949f797c49f28a52d50df16a0376677c0ded42d70c29a
- generation: 2
- size_bytes: 1174
- desired_resolver_reference_issued: false
- resolver_document_response: {"error":{"code":"NOT_FOUND","message":"Resolver document not found.","request_id":"01a18b0b-09bd-7445-b575-7876088cc360","retryable":false,"details":{}}}  HTTP/1.1 404 Not Found
- admin_node: 200
- dns_policy_evidence: null
- resolver_readback: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":0,"rules_sha256":"4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945","statistics_enabled":false,"synthetic_answer_count":1,"synthetic_blocked":false}
- owner: w3-wire-nodes / internal/desiredstate: compiler emits resolver addresses but no resolver_document reference; D-056 issuance is missing
- tampered_document_case: blocked by the same missing issued reference; no last-good node apply exists to preserve. No signed state or projector row is fabricated.
