# gateway-privacy

Gateway diagnostics exclude transient authentication query values

Status: pass

Run: 14231a83-130d-40fb-a9b4-0d3851520b67

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §17.2, §19 AT28, SECOPS-16

- upstream_fault_status: 502
- sentinel_value_matches: 0
- query_key_matches: 0
- connection_budget_errors: 0
- safe_502_diagnostic: true
- configured_budget: 2 workers x 4096 connections; 60 native peers plus service and login traffic
