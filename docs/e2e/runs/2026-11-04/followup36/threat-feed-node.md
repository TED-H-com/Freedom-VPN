# threat-feed-node

Issued resolver document is applied and blocks a synthetic name

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

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
- desired_resolver_reference: {"content_sha256":"90e2df078b8cdcf9b95586ecf3caddd815d23b9c3e97079e6f4a46b5dc418148","document_id":"01a1a6fb-30e0-7ecd-a423-bf7a8e589e04","generation":"2","profile_revision_id":"01a1a6fb-2774-73c2-8363-757c1bf6c4f2","size_bytes":1174}
- dns_policy_evidence: {"applied_at":"2026-11-04T14:34:40.410795208Z","content_sha256":"90e2df078b8cdcf9b95586ecf3caddd815d23b9c3e97079e6f4a46b5dc418148","document_id":"01a1a6fb-30e0-7ecd-a423-bf7a8e589e04","error_code":null,"feeds":[{"content_sha256":"c7561f2cb987ed5790e8f2a17d180f98b50284430460a17bb6f6b0355472a174","feed_code":"e2e-feed","rule_count":10}],"generation":"2","outcome":"applied"}
- admin_node: 200
- provisioning_state: ready
- resolver_document_response: HTTP/1.1 200 OK
- mtls_body_matches_signed_digest_and_size: true
- resolver_readback: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"063ab3b25eeabfc9f9f7d4ac571f4063202686575ecc166f606d83c21c845b22","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- timed_section: threat-feed-node
