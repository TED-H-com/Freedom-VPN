# threat-feed-node

Issued resolver document is applied and blocks a synthetic name

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

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
- desired_resolver_reference: {"content_sha256":"90e2df078b8cdcf9b95586ecf3caddd815d23b9c3e97079e6f4a46b5dc418148","document_id":"01a1a511-15bd-7742-ad7f-9f15a6a2ad8f","generation":"2","profile_revision_id":"01a1a511-0cc3-77d8-87c0-78146cfd6844","size_bytes":1174}
- dns_policy_evidence: {"applied_at":"2026-11-04T05:39:28.74720411Z","content_sha256":"90e2df078b8cdcf9b95586ecf3caddd815d23b9c3e97079e6f4a46b5dc418148","document_id":"01a1a511-15bd-7742-ad7f-9f15a6a2ad8f","error_code":null,"feeds":[{"content_sha256":"c7561f2cb987ed5790e8f2a17d180f98b50284430460a17bb6f6b0355472a174","feed_code":"e2e-feed","rule_count":10}],"generation":"2","outcome":"applied"}
- admin_node: 200
- provisioning_state: ready
- resolver_document_response: HTTP/1.1 200 OK
- mtls_body_matches_signed_digest_and_size: true
- resolver_readback: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"063ab3b25eeabfc9f9f7d4ac571f4063202686575ecc166f606d83c21c845b22","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
