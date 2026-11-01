# container-stop-forced

TEST ONLY PID1 forwarding and 15 second agent stop budget

Status: pass

Run: 2cd90524-b768-4abc-8314-b9cded71a9ad

Source: 4d7163379f80ab0630a78353c7ab10de24387c32

- automatic_resume: true
- stop_started_at: "2026-11-01T13:09:52.340518453Z"
- stop_duration_ms: 15790
- container: {"Running":false,"ExitCode":0,"OOMKilled":false,"StartedAt":"2026-11-01T13:09:51.712195131Z","FinishedAt":"2026-11-01T13:10:08.102316686Z"}
- agent_wait: {"id":"aab2add7-b5a0-476d-8042-99fb83b2482b","pid":886,"start_ticks":"12212958","started_at":"2026-11-01T13:09:52.063538182Z","wait_observed_at":"2026-11-01T13:10:07.695304759Z","wait_status":9,"exit_code":null,"signal":9,"core_dump":false}
- supervisor_sigkill: {"pid":886,"signal":9,"kill_accepted":true,"waited_ms":15200,"requested_at":"2026-11-01T13:10:07.693457648Z","term_requested_at":"2026-11-01T13:09:52.479939162Z"}
