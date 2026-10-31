# Worker restart

`worker`, `control`, and `netbird` use `restart: unless-stopped`. Every other
compose service stays `restart: "no"` and is an operator restart. NetBird
management, signal, and relay are the one `netbird` container. `docker stop`
does not bring an `unless-stopped` container back.

`worker-restart` runs after member traffic. It sends `SIGKILL` to pid 1 inside
the worker container. Within 30 seconds the container must answer
`http://127.0.0.1:9092/readyz` and the new process must deliver an outbox row
or claim one the dead process had not claimed. The clock is taken after the
signal returns. The scenario does not insert an outbox row and does not use
`docker inspect` as the readiness signal.

See [RUN.md](../RUN.md).
