# member-traffic

Member device and exit traffic

Status: blocked

Run: 5852f09a-30e8-45bc-8983-b5f212e33640

Source: 5eb716e1a36ced17b8013646c9282c92746c81da

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: c7809b72-e116-427c-b1f1-8ab95bff5a93
- plan_revision_id: 36fe29c5-1258-4f80-9bda-012918beae23
- grant: 202
- join: embedded IdP user, same subject as the portal session
- device_flow_path: /oauth2/device
- device_flow_status: 200
- device_login: submitted
- member_netbird: netbird_up
- netbird_user_id: [redacted]
- peer_user_id: [redacted]
- same_subject: true
- peers_with_user: 1
- peer_total: 2
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding: authorized device has no desired-state binding
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- traffic: not claimed; no flow was counted
- owner: w3-wire-members: device admission; w3-wire-nodes: exit publication

