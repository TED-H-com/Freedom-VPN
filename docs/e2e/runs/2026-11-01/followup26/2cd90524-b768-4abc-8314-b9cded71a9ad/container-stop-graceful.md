# container-stop-graceful

TEST ONLY PID1 forwarding and 15 second agent stop budget

Status: pass

Run: 2cd90524-b768-4abc-8314-b9cded71a9ad

Source: 4d7163379f80ab0630a78353c7ab10de24387c32

- stop_started_at: "2026-11-01T13:09:51.149963355Z"
- stop_duration_ms: 509
- container: {"Running":false,"ExitCode":0,"OOMKilled":false,"StartedAt":"2026-11-01T13:08:49.055007534Z","FinishedAt":"2026-11-01T13:09:51.624887722Z"}
- agent_wait: {"id":"ebcc6a07-eb3b-41b2-987e-120a7abeb31f","pid":1213,"start_ticks":"12207212","started_at":"2026-11-01T13:08:54.607644419Z","wait_observed_at":"2026-11-01T13:09:51.217582032Z","wait_status":0,"exit_code":0,"signal":null,"core_dump":false}
