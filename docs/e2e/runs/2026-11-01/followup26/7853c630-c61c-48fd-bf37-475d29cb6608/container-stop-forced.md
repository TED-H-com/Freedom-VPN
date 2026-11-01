# container-stop-forced

TEST ONLY PID1 forwarding and 15 second agent stop budget

Status: pass

Run: 7853c630-c61c-48fd-bf37-475d29cb6608

Source: 4d7163379f80ab0630a78353c7ab10de24387c32

- automatic_resume: true
- stop_started_at: "2026-11-01T13:04:32.827688291Z"
- stop_duration_ms: 15640
- container: {"Running":false,"ExitCode":0,"OOMKilled":false,"StartedAt":"2026-11-01T13:04:32.184528248Z","FinishedAt":"2026-11-01T13:04:48.441401867Z"}
- agent_wait: {"id":"979a5bf6-ea05-4fab-9ccd-84201a5127c9","pid":880,"start_ticks":"12181004","started_at":"2026-11-01T13:04:32.525792569Z","wait_observed_at":"2026-11-01T13:04:48.039841936Z","wait_status":9,"exit_code":null,"signal":9,"core_dump":false}
- supervisor_sigkill: {"pid":880,"signal":9,"kill_accepted":true,"waited_ms":15080,"requested_at":"2026-11-01T13:04:48.038201097Z","term_requested_at":"2026-11-01T13:04:32.944515138Z"}
