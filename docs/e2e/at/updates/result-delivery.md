# Result delivery

Status: blocked

Run: 815b716a-b8da-40a6-aa9d-6446eab8da07

Source: d3009ea28352acb43e0dae5c95acd62f197ef619

Read-only snapshot of the final make-target run journal, copied before teardown; whitelist excludes node subjects, paths, secrets and health challenge material

Preflight failure has health=null, is rejected with HTTP 422, and stays first in the pending product outbox. Later local recovery and rollback results remain pending behind it.

## failed generation 2

```json
{
  "state": "failed",
  "stage": "download",
  "error_code": "ARTIFACT_INVALID",
  "health": null,
  "requested_release_id": "01a18aa7-1e6b-77f6-b51e-29d216ea49fa",
  "current_release_id": "01a18aa7-171a-767a-94c4-efb28b612ed9",
  "manifest_sha256": "d23f0c5c06067280a8b69372dd73cecf921f8a02621f0d1fbef92021d3174c43",
  "job_id": "01a18aa9-4148-7c3a-aee6-91565b843a8d",
  "generation": "2",
  "started_at": "2026-10-30T02:35:18Z",
  "finished_at": "2026-10-30T02:35:18Z",
  "outbox_status": "pending",
  "outbox_error": "STORE_UNAVAILABLE"
}
```

## succeeded generation 3

```json
{
  "state": "succeeded",
  "stage": "report",
  "error_code": null,
  "health": {
    "dns_policy": "passed",
    "identity_and_state": "passed",
    "internet_only": "passed",
    "processes": "passed",
    "tunnel_probe": "passed"
  },
  "requested_release_id": "01a18aa7-1e6b-77f6-b51e-29d216ea49fa",
  "current_release_id": "01a18aa7-1e6b-77f6-b51e-29d216ea49fa",
  "manifest_sha256": "d23f0c5c06067280a8b69372dd73cecf921f8a02621f0d1fbef92021d3174c43",
  "job_id": "01a18aaa-c919-72e4-a83b-7f44dbce2590",
  "generation": "3",
  "started_at": "2026-10-30T02:37:03Z",
  "finished_at": "2026-10-30T02:37:06Z",
  "outbox_status": "pending",
  "outbox_error": null
}
```

## rolled_back generation 4

```json
{
  "state": "rolled_back",
  "stage": "report",
  "error_code": "HEALTH_CHECK_FAILED",
  "health": {
    "dns_policy": "passed",
    "identity_and_state": "passed",
    "internet_only": "passed",
    "processes": "passed",
    "tunnel_probe": "failed"
  },
  "requested_release_id": "01a18aa7-0f90-70c5-93f4-ebc185e8011e",
  "current_release_id": "01a18aa7-1e6b-77f6-b51e-29d216ea49fa",
  "manifest_sha256": "264abcddb2e5889e748ad7de0b6cf61f1820dd61e50c29ea8be1512cc2b57e9e",
  "job_id": "01a18aac-65f2-7283-9b6b-e61b4daef22a",
  "generation": "4",
  "started_at": "2026-10-30T02:38:43Z",
  "finished_at": "2026-10-30T02:38:45Z",
  "outbox_status": "pending",
  "outbox_error": null
}
```
