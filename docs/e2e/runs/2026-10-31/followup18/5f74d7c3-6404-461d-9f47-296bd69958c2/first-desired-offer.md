# first-desired-offer

SECOPS-21 actual response, local accepted digest and applied receipt

Status: pass

Run: 5f74d7c3-6404-461d-9f47-296bd69958c2

Source: 4e0ff498c88d97e90a78ae97813784a950e4c868

Spec: SECOPS-21, §14.7

- capture_id: "adaf8e56-a3b3-4c34-b8be-c9f443f843d9"
- configuration_command: 202
- retained_generation: 3
- capture: {"capture_id":"adaf8e56-a3b3-4c34-b8be-c9f443f843d9","first":{"http_status":200,"generation":"4","body_sha256":"05bc28b50a0c50e1a8622edc1779bc540df5b926fe0084bca170404596fcae58","bytes":1479,"delivered_at":"2026-10-31T10:54:54.90936473Z","write_complete":true},"local":{"accepted_generation":"4","applied_generation":"4","envelope_sha256":"05bc28b50a0c50e1a8622edc1779bc540df5b926fe0084bca170404596fcae58","sampled_at":"2026-10-31T10:54:54.973507865Z"},"ack":{"report_id":"01a19199-23f3-796e-9fae-69e01f3f36bb","applied_generation":"4","http_status":202,"received_at":"2026-10-31T10:54:54.973195828Z"}}
- scope: live hold/release of retained state; full PostgreSQL restore and greater-than-retained assertion belong to secops
- central_receipt: {"report_id":"01a19199-23f3-796e-9fae-69e01f3f36bb","received_at":"2026-10-31T10:54:54.970363+00:00","applied_generation":"4"}
