# control-metrics

Control loopback metrics listener

Status: pass

Run: 76f0725c-2264-4021-a553-60f6c921e32b

Source: 1b2f358880d38ae643865edc2bb6f19756748456

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
