# worker-restart

Worker process crash restarts and claims events

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §17

- readyz: true
- deliveries_after_kill: 2
- claims_after_kill: 0
- within_30s: true
- note: SIGKILL of pid 1; /readyz on 127.0.0.1:9092; a post-kill delivery or a new claim
