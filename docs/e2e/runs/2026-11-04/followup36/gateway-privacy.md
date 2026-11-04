# gateway-privacy

Gateway diagnostics exclude transient authentication query values

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §17.2, §19 AT28, SECOPS-16

- upstream_fault_status: 502
- sentinel_value_matches: 0
- query_key_matches: 0
- connection_budget_errors: 0
- safe_502_diagnostic: true
- configured_budget: 2 workers x 4096 connections; 60 native peers plus service and login traffic
- timed_section: gateway-privacy
