# control-metrics

Control loopback metrics listener

Status: pass

Run: 22c14cd9-39fc-444b-a7c5-c04cf3ffbff6

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
