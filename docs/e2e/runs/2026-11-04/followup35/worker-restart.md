# worker-restart

Worker process crash restarts and claims events

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §17

- readyz: true
- deliveries_after_kill: 2
- claims_after_kill: 0
- within_30s: true
- note: SIGKILL of pid 1; /readyz on 127.0.0.1:9092; a post-kill delivery or a new claim
- timed_section: worker-restart
