# threat-feed-central

Approved feed bootstrap, wrong pin and resolver publication

Status: pass

Run: 76f0725c-2264-4021-a553-60f6c921e32b

Source: 1b2f358880d38ae643865edc2bb6f19756748456

Spec: §12.3, §12.5, §19 AT22

- prepared_content_sha256: c7561f2cb987ed5790e8f2a17d180f98b50284430460a17bb6f6b0355472a174
- create_e2e-feed: 201
- activate_good_feed: 202
- bootstrap: true
- last_good_content: c7561f2cb987ed5790e8f2a17d180f98b50284430460a17bb6f6b0355472a174
- normalized_rule_count: 10
- content_kind: approved_snapshot
- bootstrap_audit_count: 1
- metrics_after_bootstrap: fvpn_dns_feed_last_good_age_seconds{feed_code="e2e-feed"} 0.504724847; fvpn_dns_feed_refresh_total{feed_code="e2e-feed",result="success"} 1
- create_e2e-feed-profile: 201
- activate_profile: 202
- resolver_document_exists: true
- policy_generation: 1
- resolver_document_sha256: 96efd876f78265df3d8949f797c49f28a52d50df16a0376677c0ded42d70c29a
- document_embeds_normalized_rules_and_hash_matches: true
- create_e2e-feed-bad-pin: 201
- activate_bad_pin: 202
- wrong_pin_rejected: true
- wrong_pin_last_good_content: null
- wrong_pin_last_error_code: content_mismatch
- wrong_pin_consecutive_failures: 1
- refresh_failed_audit_count: 1
- previous_revision_last_good_and_publication_retained: true
- metrics_after_bad_pin: fvpn_dns_feed_last_good_age_seconds{feed_code="e2e-feed"} +Inf; fvpn_dns_feed_refresh_total{feed_code="e2e-feed",result="content_mismatch"} 1; fvpn_dns_feed_refresh_total{feed_code="e2e-feed",result="success"} 1
- scope: Last-good is per revision: the rejected new revision stays null; the prior approved content and resolver generation remain intact. No on-node claim.
- activate_restore_good_feed: 202
- approved_feed_restored_for_node: true
