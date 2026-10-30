# control-metrics

Control loopback metrics listener

Status: pass

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
