# control-metrics

Control loopback metrics listener

Status: pass

Run: dc2f6353-89ec-434c-bc68-567066d9fe51

Source: b15ff057cd5e91e227477de6a09d977931fa62d8

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
