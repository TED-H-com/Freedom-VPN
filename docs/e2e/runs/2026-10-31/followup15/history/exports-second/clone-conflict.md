# clone-conflict

Cloned identity conflict and automatic-mapping freeze

Status: pass

Run: 630b9f07-62d2-4db1-9737-9060732db1f1

Source: 3c25ce9daa081874988c9b09bc8c26e66d3c1b46

Spec: §6.2, AT07, D-072

- presentation_subnet: 172.29.30.0/24
- presentation_network: fvpn-e2e-node-clone-net
- mapping_frozen: true
- conflict_count: 1
- last_conflict_at_present: true
- provisioning_state: failed
- member_forwarding_observed: false
- identity_conflict_audits: 1
- audit_read: true
- identity_conflict_audits_after_wait: 1
- revoke: 202
- reenroll_path: entrypoint waiting_token after identity files and agent.db were removed; not an installer wrapper
- reenroll_stage: enrolled
- reconciled_mapping_frozen: false
- reconciled_conflict_count: 0
- reconciled_last_conflict_null: true
- note: second /24 presentation; admin identity_conflict; revoke and entrypoint re-enroll of the same node
