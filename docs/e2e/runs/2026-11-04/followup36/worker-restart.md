# worker-restart

Worker process crash restarts and claims events

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §17

- readyz: true
- deliveries_after_kill: 1
- claims_after_kill: 0
- within_30s: true
- note: SIGKILL of pid 1; /readyz on 127.0.0.1:9092; a post-kill delivery or a new claim
- timed_section: worker-restart
