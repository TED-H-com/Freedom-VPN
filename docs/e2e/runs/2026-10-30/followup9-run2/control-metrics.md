# control-metrics

Control loopback metrics listener

Status: pass

Run: 65712989-0b05-4117-9a50-689499cd31be

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
