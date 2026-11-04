# gateway-principals

Gateway preserves independent client rate principals

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §14.2, §19 AT28, §19 AT31

- trusted_proxies: ["172.28.30.21/32"]
- rates: unchanged defaults: browser 10/s burst 30; ingress 20/s burst 60
- distinct_members: true
- client_source_addresses: ["172.28.30.128","172.28.30.129"]
- forged_header_scope: rotating TEST ONLY documentation IPv4 values; source sockets unchanged
- member_a_gateway: {"status_counts":{"200":31,"429":89},"rate_limited_codes":89,"retry_after_present":89,"address_refusals":57,"subject_refusals":32,"elapsed_seconds":0.165776936}
- member_a_forged_gateway: {"status_counts":{"200":2,"429":118},"rate_limited_codes":118,"retry_after_present":118,"address_refusals":116,"subject_refusals":2,"elapsed_seconds":0.093590995}
- member_b_gateway: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.010844623}
- metrics_before: {"address":0,"subject":0}
- metrics_after_gateway: {"address":173,"subject":34}
- member_a_forged_direct: {"status_counts":{"200":3,"429":117},"rate_limited_codes":117,"retry_after_present":117,"address_refusals":114,"subject_refusals":3,"elapsed_seconds":0.019201071}
- metrics_after_direct: {"address":287,"subject":37}
- member_b_forged_direct: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.002710182}
- limiter_state_reads: none; responses and public control metric counters only
