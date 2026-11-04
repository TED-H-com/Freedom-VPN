# worker-restart

Worker process crash restarts and claims events

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §17

- readyz: true
- deliveries_after_kill: 2
- claims_after_kill: 0
- within_30s: true
- note: SIGKILL of pid 1; /readyz on 127.0.0.1:9092; a post-kill delivery or a new claim
