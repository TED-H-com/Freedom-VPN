# threat-feed-node

Issued resolver document is applied and blocks a synthetic name

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §12.5, §19 AT22, D-061

- credential_owner_mode: 0:600
- agent_run_dns: {
    "resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json",
    "resolvers": [
      {
        "admin_url": "http://127.0.0.1:8080",
        "dns_addr": "127.0.0.1:53",
        "id": "lab"
      }
    ]
  }
- desired_resolver_reference: {"content_sha256":"96efd876f78265df3d8949f797c49f28a52d50df16a0376677c0ded42d70c29a","document_id":"01a19024-c3a3-719f-bc27-758f240f073f","generation":"2","profile_revision_id":"01a19024-ba6f-7811-8a5e-d893731f8244","size_bytes":1174}
- dns_policy_evidence: {"applied_at":"2026-10-31T04:08:51.160031433Z","content_sha256":"96efd876f78265df3d8949f797c49f28a52d50df16a0376677c0ded42d70c29a","document_id":"01a19024-c3a3-719f-bc27-758f240f073f","error_code":null,"feeds":[{"content_sha256":"c7561f2cb987ed5790e8f2a17d180f98b50284430460a17bb6f6b0355472a174","feed_code":"e2e-feed","rule_count":10}],"generation":"2","outcome":"applied"}
- admin_node: 200
- provisioning_state: ready
- resolver_document_response: HTTP/1.1 200 OK
- mtls_body_matches_signed_digest_and_size: true
- resolver_readback: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"b41cd722654313b510451934b2f59fd90f12ac3996dc44312ecfb43a157d3ae8","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
