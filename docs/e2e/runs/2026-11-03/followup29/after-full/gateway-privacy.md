# gateway-privacy

Gateway diagnostics exclude transient authentication query values

Status: pass

Run: 91eb0372-e42b-4698-a137-7cce53b263fa

Source: f52fc9381405d4aafd064dc0ef055977de97f418

Spec: §17.2, §19 AT28, SECOPS-16

- upstream_fault_status: 502
- sentinel_value_matches: 0
- query_key_matches: 0
- connection_budget_errors: 0
- safe_502_diagnostic: true
- configured_budget: 2 workers x 4096 connections; 60 native peers plus service and login traffic
