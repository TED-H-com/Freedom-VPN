# worker-restart

Worker process crash restarts and claims events

Status: pass

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §17

- readyz: true
- deliveries_after_kill: 2
- claims_after_kill: 0
- within_30s: true
- note: SIGKILL of pid 1; /readyz on 127.0.0.1:9092; a post-kill delivery or a new claim
