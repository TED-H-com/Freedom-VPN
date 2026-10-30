# threat-feed-tamper

Corrupted resolver response retains node last-good

Status: blocked

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §12.5, §19 AT22

- positive_node_delivery: true
- last_good_resolver_readback: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"b41cd722654313b510451934b2f59fd90f12ac3996dc44312ecfb43a157d3ae8","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- fault_injected: false
- owner: w3-devstack-e2e: node mTLS response corruption hook required; central wrong-pin evidence does not test node tampering or last-good retention
