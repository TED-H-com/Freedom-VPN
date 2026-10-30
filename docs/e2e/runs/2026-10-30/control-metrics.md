# control-metrics

Control loopback metrics listener

Status: pass

Run: 5de03570-0436-477d-9847-6f6984432bd8

Source: a7a712aab8a67fcc33514a9ac76cb877a90c8143

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
