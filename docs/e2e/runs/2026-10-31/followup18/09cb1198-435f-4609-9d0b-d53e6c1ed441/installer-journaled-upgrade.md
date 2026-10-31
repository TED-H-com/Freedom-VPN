# installer-journaled-upgrade

Genuine final sample and staged installer candidate health

Status: fail

Run: 09cb1198-435f-4609-9d0b-d53e6c1ed441

Source: 84b6a342970ffc651aafda81d485451529902233

Spec: UPD05, AT26, §13.5

- stage: {"Exit":0,"Status":"staged","Code":"","Incarnation":"","NodeID":"","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"staged; updater run completed; inspect fvpn-updater status for the apply result","Before":{},"Hashes":{},"Services":null}
- checkpoint_before: absent

- apply_process: {"time":"2026-10-31T10:27:58.300890096Z","level":"INFO","msg":"applying installer pending release","release_id":"01a1917d-d8bf-7212-a6a7-79e0a7c4601e"}

- apply_process_success: true
- durable_result: {
  "generation": "1",
  "job_id": "3153bd88-7127-5cb1-a02a-7664735d39a6",
  "source": "installer",
  "requested_release_id": "01a1917d-d8bf-7212-a6a7-79e0a7c4601e",
  "stage": "final_sample",
  "state": "failed",
  "journal_stage": "rolled_back",
  "current_release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
  "last_good_release_id": "",
  "seeded_baseline": {
    "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686",
    "tree_hash": "bea3d3100cd1fe61e52155f7905bd73d9ee0b52701ab2a9aea80e91212cab198",
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "seeded_at": "2026-10-31T10:25:12.626966402Z"
  },
  "recovery_required": false,
  "last_result_state": "failed",
  "last_rejection": null,
  "health": null,
  "candidate_health_json": null
}

- genuine_meter_checkpoint_present: false
- checkpoint_after: cat: /var/lib/vpn-node/state/meter.checkpoint: No such file or directory

- journal_snapshot: {"bytes":"81920","sha256":"7dd7a58f2b168d23fcf8799e4573181f82f37b5894a568fb233fa928b937918f"}
- journal_copy_complete: true
- current_before: releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8

- current_after: releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8

- agent_updater_journal: Oct 31 10:25:12 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.126Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-efbb-78df-b1b9-67bc2dd60c34 reason=baseline boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.135Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.136Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.163Z level=INFO msg="bootstrap stored"
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.204Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-efbb-78df-b1b9-67bc2dd60c34 reason=final_sample event_seq=0
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.232Z level=INFO msg="usage status available"
Oct 31 10:25:12 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:12 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:12 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:13 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.341Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-f476-744f-b05b-7f8d3dc41e7b reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.350Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.351Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.417Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-f476-744f-b05b-7f8d3dc41e7b reason=final_sample event_seq=0
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.452Z level=INFO msg="usage status available"
Oct 31 10:25:13 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:13 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:13 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:14 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.056Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-fb2b-724d-9fb8-9e1d9f578333 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.064Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.066Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.122Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-fb2b-724d-9fb8-9e1d9f578333 reason=final_sample event_seq=0
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.158Z level=INFO msg="usage status available"
Oct 31 10:25:15 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:15 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:15 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:15 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:15.835Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-fe38-76d6-82aa-7a35e86b2b26 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:15.843Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:16 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:16.058Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-fe38-76d6-82aa-7a35e86b2b26 reason=final_sample event_seq=0
Oct 31 10:25:16 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:16.091Z level=INFO msg="usage status available"
Oct 31 10:25:20 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:20.962Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-123d-7966-9b08-e41ffa842006 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:25 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:25.993Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-123d-7966-9b08-e41ffa842006 reason=final_sample event_seq=0
Oct 31 10:25:26 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:26.135Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-2675-7f2a-a15c-4142d3d24682 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:30 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:30.917Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-2675-7f2a-a15c-4142d3d24682 reason=final_sample event_seq=0
Oct 31 10:25:31 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:31.029Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-3993-7268-a783-2cb02862cd76 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:41 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:41 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:41.326Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-3993-7268-a783-2cb02862cd76 reason=final_sample event_seq=0
Oct 31 10:25:41 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:41 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:43 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.493Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-6a41-711e-951f-20611c2e502e reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.500Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.524Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=1
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.646Z level=INFO msg="usage status available"
Oct 31 10:25:48 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:48.595Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-6a41-711e-951f-20611c2e502e reason=final_sample event_seq=0
Oct 31 10:25:48 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:48.760Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-7ea7-75b6-9e76-1127f6b5762b reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:27:05 9cd9049d833e systemd[1]: Starting fvpn-updater.service - Freedom VPN node updater...
Oct 31 10:27:05 9cd9049d833e systemd[1]: fvpn-updater.service: Deactivated successfully.
Oct 31 10:27:05 9cd9049d833e systemd[1]: Finished fvpn-updater.service - Freedom VPN node updater.
Oct 31 10:27:50 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:27:50 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:27:50.499Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-7ea7-75b6-9e76-1127f6b5762b reason=final_sample event_seq=3
Oct 31 10:27:50 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:27:50 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:27:50 9cd9049d833e systemd[1]: fvpn-agent.service: Consumed 3.785s CPU time, 32.1M memory peak, 0B memory swap peak.
Oct 31 10:27:51 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.814Z level=INFO msg="meter epoch opened" epoch_id=01a19180-5f83-7315-bc3e-9d2785649a80 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.819Z level=WARN msg="usage batch unavailable" status=503
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.820Z level=WARN msg="usage upload retry" backoff=1s err="usageupload: ledger unavailable"
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.821Z level=INFO msg="usage checkpoint applied"
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.950Z level=INFO msg="usage status available"
Oct 31 10:27:52 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:52.852Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=3
Oct 31 10:27:58 9cd9049d833e systemd[1]: fvpn-updater.service - Freedom VPN node updater was skipped because of an unmet condition check (ConditionPathExists=!/var/tmp/e2e-rel/hold-fvpn-updater.service).

- owner: updater / agent-core; durable result decides success, independent of apply exit
