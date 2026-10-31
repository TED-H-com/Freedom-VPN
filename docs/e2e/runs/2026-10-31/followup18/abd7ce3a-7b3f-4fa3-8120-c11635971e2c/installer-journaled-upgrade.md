# installer-journaled-upgrade

Genuine final sample and staged installer candidate health

Status: pass

Run: abd7ce3a-7b3f-4fa3-8120-c11635971e2c

Source: 0b9fb5ebbff0d1618679b8eed6925b7aa9cc7daf

Spec: UPD05, AT26, §13.5

- stage: {"Exit":0,"Status":"staged","Code":"","Incarnation":"","NodeID":"","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"staged; updater run completed; inspect fvpn-updater status for the apply result","Before":{},"Hashes":{},"Services":null}
- checkpoint_before: {"closed_meter_epoch_id":"01a19176-b9d6-7167-8f46-7140fd4beeb0","last_durable_event_seq":"3","central_acked_through":"1","new_meter_epoch_id":""}

- apply_process: {"time":"2026-10-31T10:19:54.693815989Z","level":"INFO","msg":"applying installer pending release","release_id":"01a19175-ee15-7a11-8e75-01a53cdd0a93"}

- apply_process_success: true
- durable_result: {
  "generation": "1",
  "job_id": "9f586e27-b82a-5eca-93d0-bcaa511878f8",
  "source": "installer",
  "requested_release_id": "01a19175-ee15-7a11-8e75-01a53cdd0a93",
  "stage": "report",
  "state": "succeeded",
  "journal_stage": "locally_succeeded",
  "current_release_id": "01a19175-ee15-7a11-8e75-01a53cdd0a93",
  "last_good_release_id": "01a19175-ee15-7a11-8e75-01a53cdd0a93",
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
- checkpoint_after: {"closed_meter_epoch_id":"01a19179-190d-7aa0-ac7e-2b8467b365c3","last_durable_event_seq":"3","central_acked_through":"3","new_meter_epoch_id":""}

- journal:
- current_before: releases/01a19175-e341-792a-b34b-af9c102815b4

- current_after: releases/01a19175-ee15-7a11-8e75-01a53cdd0a93

- agent_updater_journal: Oct 31 10:16:33 490b8f357f31 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:33 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:33.203Z level=INFO msg="meter epoch opened" epoch_id=01a19176-04b1-71bc-974d-564ab769c190 reason=baseline boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:16:33 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:33.217Z level=INFO msg="usage checkpoint applied"
Oct 31 10:16:33 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:33.220Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:16:33 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:33.285Z level=INFO msg="meter epoch closed" epoch_id=01a19176-04b1-71bc-974d-564ab769c190 reason=final_sample event_seq=0
Oct 31 10:16:33 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:33.313Z level=INFO msg="usage status available"
Oct 31 10:16:43 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:43.237Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:16:43 490b8f357f31 fvpn-agent[1395]: time=2026-10-31T10:16:43.253Z level=INFO msg="bootstrap stored"
Oct 31 10:16:43 490b8f357f31 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:16:43 490b8f357f31 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:16:43 490b8f357f31 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:44 490b8f357f31 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:44 490b8f357f31 fvpn-agent[1674]: time=2026-10-31T10:16:44.336Z level=INFO msg="meter epoch opened" epoch_id=01a19176-302c-7504-a5bb-8032f31bf671 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:16:44 490b8f357f31 fvpn-agent[1674]: time=2026-10-31T10:16:44.345Z level=INFO msg="usage checkpoint applied"
Oct 31 10:16:44 490b8f357f31 fvpn-agent[1674]: time=2026-10-31T10:16:44.347Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:16:44 490b8f357f31 fvpn-agent[1674]: time=2026-10-31T10:16:44.404Z level=INFO msg="meter epoch closed" epoch_id=01a19176-302c-7504-a5bb-8032f31bf671 reason=final_sample event_seq=0
Oct 31 10:16:44 490b8f357f31 fvpn-agent[1674]: time=2026-10-31T10:16:44.438Z level=INFO msg="usage status available"
Oct 31 10:16:44 490b8f357f31 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:16:44 490b8f357f31 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:16:44 490b8f357f31 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:46 490b8f357f31 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:46 490b8f357f31 fvpn-agent[2046]: time=2026-10-31T10:16:46.239Z level=INFO msg="meter epoch opened" epoch_id=01a19176-379b-76d4-8116-a1c4ecf4f73e reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:16:46 490b8f357f31 fvpn-agent[2046]: time=2026-10-31T10:16:46.247Z level=INFO msg="usage checkpoint applied"
Oct 31 10:16:46 490b8f357f31 fvpn-agent[2046]: time=2026-10-31T10:16:46.249Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:16:46 490b8f357f31 fvpn-agent[2046]: time=2026-10-31T10:16:46.312Z level=INFO msg="meter epoch closed" epoch_id=01a19176-379b-76d4-8116-a1c4ecf4f73e reason=final_sample event_seq=0
Oct 31 10:16:46 490b8f357f31 fvpn-agent[2046]: time=2026-10-31T10:16:46.351Z level=INFO msg="usage status available"
Oct 31 10:16:46 490b8f357f31 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:16:46 490b8f357f31 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:16:46 490b8f357f31 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:46 490b8f357f31 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:16:47 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:47.007Z level=INFO msg="meter epoch opened" epoch_id=01a19176-3a9a-728a-a527-70e207bc1bdd reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:16:47 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:47.016Z level=INFO msg="usage checkpoint applied"
Oct 31 10:16:47 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:47.231Z level=INFO msg="meter epoch closed" epoch_id=01a19176-3a9a-728a-a527-70e207bc1bdd reason=final_sample event_seq=0
Oct 31 10:16:47 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:47.266Z level=INFO msg="usage status available"
Oct 31 10:16:52 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:52.117Z level=INFO msg="meter epoch opened" epoch_id=01a19176-4e94-7940-a752-997cf2b85276 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:16:57 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:57.109Z level=INFO msg="meter epoch closed" epoch_id=01a19176-4e94-7940-a752-997cf2b85276 reason=final_sample event_seq=0
Oct 31 10:16:57 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:16:57.235Z level=INFO msg="meter epoch opened" epoch_id=01a19176-6291-7a14-8a8e-1b4526242098 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:17:02 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:17:02.105Z level=INFO msg="meter epoch closed" epoch_id=01a19176-6291-7a14-8a8e-1b4526242098 reason=final_sample event_seq=0
Oct 31 10:17:02 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:17:02.219Z level=INFO msg="meter epoch opened" epoch_id=01a19176-760a-7154-aa9f-a5cbc566d1da reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:17:12 490b8f357f31 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:17:12 490b8f357f31 fvpn-agent[2277]: time=2026-10-31T10:17:12.485Z level=INFO msg="meter epoch closed" epoch_id=01a19176-760a-7154-aa9f-a5cbc566d1da reason=final_sample event_seq=0
Oct 31 10:17:12 490b8f357f31 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:17:12 490b8f357f31 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:17:12 490b8f357f31 systemd[1]: fvpn-agent.service: Consumed 1.012s CPU time, 28.6M memory peak, 0B memory swap peak.
Oct 31 10:17:14 490b8f357f31 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:17:14 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:14.366Z level=INFO msg="meter epoch opened" epoch_id=01a19176-a57b-7394-a4a7-336846f71718 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:17:14 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:14.372Z level=INFO msg="usage checkpoint applied"
Oct 31 10:17:14 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:14.396Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=1
Oct 31 10:17:14 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:14.530Z level=INFO msg="usage status available"
Oct 31 10:17:19 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:19.458Z level=INFO msg="meter epoch closed" epoch_id=01a19176-a57b-7394-a4a7-336846f71718 reason=final_sample event_seq=2
Oct 31 10:17:19 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:19.575Z level=INFO msg="meter epoch opened" epoch_id=01a19176-b9d6-7167-8f46-7140fd4beeb0 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:17:20 490b8f357f31 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:17:20 490b8f357f31 fvpn-agent[3671]: time=2026-10-31T10:17:20.270Z level=INFO msg="meter epoch closed" epoch_id=01a19176-b9d6-7167-8f46-7140fd4beeb0 reason=final_sample event_seq=3
Oct 31 10:17:20 490b8f357f31 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:17:20 490b8f357f31 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:17:21 490b8f357f31 systemd[1]: fvpn-agent.service: Start request repeated too quickly.
Oct 31 10:17:21 490b8f357f31 systemd[1]: fvpn-agent.service: Failed with result 'start-limit-hit'.
Oct 31 10:17:21 490b8f357f31 systemd[1]: Failed to start fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:19:54 490b8f357f31 systemd[1]: fvpn-updater.service - Freedom VPN node updater was skipped because of an unmet condition check (ConditionPathExists=!/var/tmp/e2e-rel/hold-fvpn-updater.service).
Oct 31 10:19:54 490b8f357f31 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:19:55 490b8f357f31 fvpn-agent[5317]: time=2026-10-31T10:19:55.017Z level=ERROR msg="updater recover" err="forwarding stays blocked"
Oct 31 10:19:55 490b8f357f31 fvpn-agent[5317]: time=2026-10-31T10:19:55.024Z level=INFO msg="meter epoch opened" epoch_id=01a19179-190d-7aa0-ac7e-2b8467b365c3 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:19:55 490b8f357f31 fvpn-agent[5317]: time=2026-10-31T10:19:55.032Z level=INFO msg="usage checkpoint applied"
Oct 31 10:19:55 490b8f357f31 fvpn-agent[5317]: time=2026-10-31T10:19:55.057Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=3
Oct 31 10:19:55 490b8f357f31 fvpn-agent[5317]: time=2026-10-31T10:19:55.137Z level=INFO msg="meter epoch closed" epoch_id=01a19179-190d-7aa0-ac7e-2b8467b365c3 reason=final_sample event_seq=0
Oct 31 10:19:55 490b8f357f31 fvpn-agent[5317]: time=2026-10-31T10:19:55.169Z level=INFO msg="usage status available"

- owner: updater / agent-core; durable result decides success, independent of apply exit
