# control-metrics

Control loopback metrics listener

Status: pass

Run: 3a129919-af3f-4bc9-b54a-8075eae8af45

Source: 79259af1facc1faa798fbb83cc20aa4e74539d1f

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
