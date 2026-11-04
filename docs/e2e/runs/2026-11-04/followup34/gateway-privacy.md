# gateway-privacy

Gateway diagnostics exclude transient authentication query values

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §17.2, §19 AT28, SECOPS-16

- upstream_fault_status: 502
- sentinel_value_matches: 0
- query_key_matches: 0
- connection_budget_errors: 0
- safe_502_diagnostic: true
- configured_budget: 2 workers x 4096 connections; 60 native peers plus service and login traffic
