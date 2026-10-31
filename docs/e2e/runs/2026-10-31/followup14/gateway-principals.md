# gateway-principals

Gateway preserves independent client rate principals

Status: pass

Run: 4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §14.2, §19 AT28, §19 AT31

- trusted_proxies: ["172.28.30.21/32"]
- rates: unchanged defaults: browser 10/s burst 30; ingress 20/s burst 60
- distinct_members: true
- client_source_addresses: ["172.28.30.128","172.28.30.129"]
- forged_header_scope: rotating TEST ONLY documentation IPv4 values; source sockets unchanged
- member_a_gateway: {"status_counts":{"200":30,"429":90},"rate_limited_codes":90,"retry_after_present":90,"address_refusals":58,"subject_refusals":32,"elapsed_seconds":0.143080985}
- member_a_forged_gateway: {"status_counts":{"200":2,"429":118},"rate_limited_codes":118,"retry_after_present":118,"address_refusals":117,"subject_refusals":1,"elapsed_seconds":0.086598731}
- member_b_gateway: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.004379542}
- metrics_before: {"address":0,"subject":0}
- metrics_after_gateway: {"address":175,"subject":33}
- member_a_forged_direct: {"status_counts":{"200":2,"429":118},"rate_limited_codes":118,"retry_after_present":118,"address_refusals":117,"subject_refusals":1,"elapsed_seconds":0.011854845}
- metrics_after_direct: {"address":292,"subject":34}
- member_b_forged_direct: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.002752293}
- limiter_state_reads: none; responses and public control metric counters only
