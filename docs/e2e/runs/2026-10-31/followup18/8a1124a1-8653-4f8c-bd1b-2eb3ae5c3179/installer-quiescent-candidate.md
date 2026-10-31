# installer-quiescent-candidate

Isolated candidate health after genuine agent shutdown barrier

Status: pass

Run: 8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179

Source: a0089b8386c749100d9bd19b8002dd4d666d065c

Spec: UPD05, AT26, §13.5

- scope: TEST ONLY candidate-health isolation: explicitly stop watchdog/agent and require the agent's genuine checkpoint. This does not prove the serving-node updater handshake; its separate failed row remains authoritative.
- stage: {"Exit":0,"Status":"staged","Code":"","Incarnation":"","NodeID":"","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"staged; updater run completed; inspect fvpn-updater status for the apply result","Before":{},"Hashes":{},"Services":null}
- checkpoint_before: {"closed_meter_epoch_id":"01a19195-7e5b-7e0e-bc47-af1b43d6871c","last_durable_event_seq":"4","central_acked_through":"3","new_meter_epoch_id":""}

- apply_process: {"time":"2026-10-31T10:51:02.810297923Z","level":"INFO","msg":"applying installer pending release","release_id":"01a19192-b8f2-7424-864b-9b781677ba18"}

- apply_process_success: true
- durable_result: {
  "generation": "1",
  "job_id": "362ed381-4f80-57b8-8d63-3a2c4efedc2a",
  "source": "installer",
  "requested_release_id": "01a19192-b8f2-7424-864b-9b781677ba18",
  "stage": "report",
  "state": "succeeded",
  "journal_stage": "locally_succeeded",
  "current_release_id": "01a19192-b8f2-7424-864b-9b781677ba18",
  "last_good_release_id": "01a19192-b8f2-7424-864b-9b781677ba18",
  "recovery_required": false,
  "last_result_state": "succeeded",
  "last_rejection": null,
  "health": {
    "dns_policy": "passed",
    "identity_and_state": "passed",
    "internet_only": "passed",
    "processes": "passed",
    "tunnel_probe": "passed"
  },
  "candidate_health_json": {
    "dns_policy": "passed",
    "identity_and_state": "passed",
    "internet_only": "passed",
    "processes": "passed",
    "tunnel_probe": "passed"
  }
}

- genuine_meter_checkpoint_present: true
- checkpoint_after: {"closed_meter_epoch_id":"01a19195-9aa0-7268-a18d-ef1cb4f6e10a","last_durable_event_seq":"4","central_acked_through":"4","new_meter_epoch_id":""}

- journal_snapshot: {"bytes":"81920","sha256":"d78f02ceea0bf781c0bb4e7ab9b83e3af40c006cb25df0b0ff1635ea66efff08"}
- journal_copy_complete: true
- current_before: releases/01a19192-ad63-7d61-b846-929be061b789

- current_after: releases/01a19192-b8f2-7424-864b-9b781677ba18

- agent_updater_journal: Oct 31 10:48:00 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:00 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:00.681Z level=INFO msg="meter epoch opened" epoch_id=01a19192-d1a7-7b57-9f8a-f58739848b80 reason=baseline boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:00 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:00.691Z level=INFO msg="usage checkpoint applied"
Oct 31 10:48:00 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:00.693Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:48:00 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:00.744Z level=INFO msg="meter epoch closed" epoch_id=01a19192-d1a7-7b57-9f8a-f58739848b80 reason=final_sample event_seq=0
Oct 31 10:48:00 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:00.773Z level=INFO msg="usage status available"
Oct 31 10:48:10 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:10.713Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:48:10 cf712a23aac8 fvpn-agent[1388]: time=2026-10-31T10:48:10.730Z level=INFO msg="bootstrap stored"
Oct 31 10:48:11 cf712a23aac8 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:48:11 cf712a23aac8 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:48:11 cf712a23aac8 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:11 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:11 cf712a23aac8 fvpn-agent[1666]: time=2026-10-31T10:48:11.856Z level=INFO msg="meter epoch opened" epoch_id=01a19192-fd49-71b8-a2da-8b8a86966248 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:11 cf712a23aac8 fvpn-agent[1666]: time=2026-10-31T10:48:11.866Z level=INFO msg="usage checkpoint applied"
Oct 31 10:48:11 cf712a23aac8 fvpn-agent[1666]: time=2026-10-31T10:48:11.868Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:48:11 cf712a23aac8 fvpn-agent[1666]: time=2026-10-31T10:48:11.930Z level=INFO msg="meter epoch closed" epoch_id=01a19192-fd49-71b8-a2da-8b8a86966248 reason=final_sample event_seq=0
Oct 31 10:48:11 cf712a23aac8 fvpn-agent[1666]: time=2026-10-31T10:48:11.964Z level=INFO msg="usage status available"
Oct 31 10:48:12 cf712a23aac8 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:48:12 cf712a23aac8 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:48:12 cf712a23aac8 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:13 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:13 cf712a23aac8 fvpn-agent[2037]: time=2026-10-31T10:48:13.491Z level=INFO msg="meter epoch opened" epoch_id=01a19193-03af-7df9-b16e-29c01eaf3835 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:13 cf712a23aac8 fvpn-agent[2037]: time=2026-10-31T10:48:13.498Z level=INFO msg="usage checkpoint applied"
Oct 31 10:48:13 cf712a23aac8 fvpn-agent[2037]: time=2026-10-31T10:48:13.500Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:48:13 cf712a23aac8 fvpn-agent[2037]: time=2026-10-31T10:48:13.554Z level=INFO msg="meter epoch closed" epoch_id=01a19193-03af-7df9-b16e-29c01eaf3835 reason=final_sample event_seq=0
Oct 31 10:48:13 cf712a23aac8 fvpn-agent[2037]: time=2026-10-31T10:48:13.582Z level=INFO msg="usage status available"
Oct 31 10:48:14 cf712a23aac8 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:48:14 cf712a23aac8 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:48:14 cf712a23aac8 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:14 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:14 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:14.175Z level=INFO msg="meter epoch opened" epoch_id=01a19193-065a-770f-bce0-1d2236c5b693 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:14 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:14.181Z level=INFO msg="usage checkpoint applied"
Oct 31 10:48:14 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:14.401Z level=INFO msg="meter epoch closed" epoch_id=01a19193-065a-770f-bce0-1d2236c5b693 reason=final_sample event_seq=0
Oct 31 10:48:14 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:14.438Z level=INFO msg="usage status available"
Oct 31 10:48:19 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:19.248Z level=INFO msg="meter epoch opened" epoch_id=01a19193-1a29-7878-ad4d-479e1527544e reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:24 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:24.271Z level=INFO msg="meter epoch closed" epoch_id=01a19193-1a29-7878-ad4d-479e1527544e reason=final_sample event_seq=0
Oct 31 10:48:24 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:24.759Z level=INFO msg="meter epoch opened" epoch_id=01a19193-2f59-7c5b-97aa-a01c3dba4f09 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:29 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:29.251Z level=INFO msg="meter epoch closed" epoch_id=01a19193-2f59-7c5b-97aa-a01c3dba4f09 reason=final_sample event_seq=0
Oct 31 10:48:29 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:29.562Z level=INFO msg="meter epoch opened" epoch_id=01a19193-4278-7776-b1b7-9503ee01ade2 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:34 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:34.254Z level=INFO msg="meter epoch closed" epoch_id=01a19193-4278-7776-b1b7-9503ee01ade2 reason=final_sample event_seq=0
Oct 31 10:48:34 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:34.371Z level=INFO msg="meter epoch opened" epoch_id=01a19193-5541-779e-8fcb-3d1ad0e609cd reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:45 cf712a23aac8 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:48:45 cf712a23aac8 fvpn-agent[2266]: time=2026-10-31T10:48:45.404Z level=INFO msg="meter epoch closed" epoch_id=01a19193-5541-779e-8fcb-3d1ad0e609cd reason=final_sample event_seq=0
Oct 31 10:48:45 cf712a23aac8 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:48:45 cf712a23aac8 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:45 cf712a23aac8 systemd[1]: fvpn-agent.service: Consumed 1.100s CPU time, 29.6M memory peak, 0B memory swap peak.
Oct 31 10:48:47 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:48:47 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:48:47.332Z level=INFO msg="meter epoch opened" epoch_id=01a19193-87df-76c0-81b1-99883cb6f370 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:48:47 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:48:47.339Z level=INFO msg="usage checkpoint applied"
Oct 31 10:48:47 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:48:47.367Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=1
Oct 31 10:48:47 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:48:47.482Z level=INFO msg="usage status available"
Oct 31 10:48:52 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:48:52.427Z level=INFO msg="meter epoch closed" epoch_id=01a19193-87df-76c0-81b1-99883cb6f370 reason=final_sample event_seq=0
Oct 31 10:48:52 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:48:52.542Z level=INFO msg="meter epoch opened" epoch_id=01a19193-9c3b-7dd8-a090-1177bb9ff3c0 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:50:15 cf712a23aac8 systemd[1]: Starting fvpn-updater.service - Freedom VPN node updater...
Oct 31 10:50:15 cf712a23aac8 systemd[1]: fvpn-updater.service: Deactivated successfully.
Oct 31 10:50:15 cf712a23aac8 systemd[1]: Finished fvpn-updater.service - Freedom VPN node updater.
Oct 31 10:50:54 cf712a23aac8 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:50:54 cf712a23aac8 fvpn-agent[3803]: time=2026-10-31T10:50:54.511Z level=INFO msg="meter epoch closed" epoch_id=01a19193-9c3b-7dd8-a090-1177bb9ff3c0 reason=final_sample event_seq=3
Oct 31 10:50:54 cf712a23aac8 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:50:54 cf712a23aac8 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:50:54 cf712a23aac8 systemd[1]: fvpn-agent.service: Consumed 3.772s CPU time, 33.0M memory peak, 0B memory swap peak.
Oct 31 10:50:55 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:50:55 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:50:55.966Z level=INFO msg="meter epoch opened" epoch_id=01a19195-7e5b-7e0e-bc47-af1b43d6871c reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:50:55 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:50:55.972Z level=WARN msg="usage batch unavailable" status=503
Oct 31 10:50:55 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:50:55.972Z level=WARN msg="usage upload retry" backoff=1s err="usageupload: ledger unavailable"
Oct 31 10:50:55 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:50:55.973Z level=INFO msg="usage checkpoint applied"
Oct 31 10:50:56 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:50:56.126Z level=INFO msg="usage status available"
Oct 31 10:50:57 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:50:57.003Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=3
Oct 31 10:51:02 cf712a23aac8 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:51:02 cf712a23aac8 fvpn-agent[7564]: time=2026-10-31T10:51:02.221Z level=INFO msg="meter epoch closed" epoch_id=01a19195-7e5b-7e0e-bc47-af1b43d6871c reason=final_sample event_seq=0
Oct 31 10:51:02 cf712a23aac8 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:51:02 cf712a23aac8 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:51:02 cf712a23aac8 systemd[1]: fvpn-updater.service - Freedom VPN node updater was skipped because of an unmet condition check (ConditionPathExists=!/var/tmp/e2e-rel/hold-fvpn-updater.service).
Oct 31 10:51:03 cf712a23aac8 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:51:03 cf712a23aac8 fvpn-agent[8132]: time=2026-10-31T10:51:03.195Z level=ERROR msg="updater recover" err="forwarding stays blocked"
Oct 31 10:51:03 cf712a23aac8 fvpn-agent[8132]: time=2026-10-31T10:51:03.203Z level=INFO msg="meter epoch opened" epoch_id=01a19195-9aa0-7268-a18d-ef1cb4f6e10a reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:51:03 cf712a23aac8 fvpn-agent[8132]: time=2026-10-31T10:51:03.212Z level=INFO msg="usage checkpoint applied"
Oct 31 10:51:03 cf712a23aac8 fvpn-agent[8132]: time=2026-10-31T10:51:03.223Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=4
Oct 31 10:51:03 cf712a23aac8 fvpn-agent[8132]: time=2026-10-31T10:51:03.312Z level=INFO msg="meter epoch closed" epoch_id=01a19195-9aa0-7268-a18d-ef1cb4f6e10a reason=final_sample event_seq=0
Oct 31 10:51:03 cf712a23aac8 fvpn-agent[8132]: time=2026-10-31T10:51:03.346Z level=INFO msg="usage status available"

- owner: updater / agent-core; durable result decides success, independent of apply exit
