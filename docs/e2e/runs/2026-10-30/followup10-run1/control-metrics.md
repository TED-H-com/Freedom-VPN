# control-metrics

Control loopback metrics listener

Status: pass

Run: cfda038f-dc18-4371-8047-a4d34954066f

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
