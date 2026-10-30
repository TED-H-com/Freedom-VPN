# control-metrics

Control loopback metrics listener

Status: pass

Run: 376e5f84-535e-4ddd-902b-985852d4691b

Source: 724ab19fd46a974a66d138ac0cddcd7f803bafb7

Spec: §17.1

- config_metrics_address: 127.0.0.1:9091
- compose_override: none; worker alone overrides FVPN_METRICS_ADDRESS=:9092
- response: HTTP/1.1 200 OK
- metric_families: 1
