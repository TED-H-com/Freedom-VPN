# gateway-principals

Gateway preserves independent client rate principals

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §14.2, §19 AT28, §19 AT31

- trusted_proxies: ["172.28.30.21/32"]
- rates: unchanged defaults: browser 10/s burst 30; ingress 20/s burst 60
- distinct_members: true
- client_source_addresses: ["172.28.30.128","172.28.30.129"]
- forged_header_scope: rotating TEST ONLY documentation IPv4 values; source sockets unchanged
- member_a_gateway: {"status_counts":{"200":31,"429":89},"rate_limited_codes":89,"retry_after_present":89,"address_refusals":57,"subject_refusals":32,"elapsed_seconds":0.19675555}
- member_a_forged_gateway: {"status_counts":{"200":1,"429":119},"rate_limited_codes":119,"retry_after_present":119,"address_refusals":117,"subject_refusals":2,"elapsed_seconds":0.111299532}
- member_b_gateway: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.005091875}
- metrics_before: {"address":0,"subject":0}
- metrics_after_gateway: {"address":174,"subject":34}
- member_a_forged_direct: {"status_counts":{"200":2,"429":118},"rate_limited_codes":118,"retry_after_present":118,"address_refusals":117,"subject_refusals":1,"elapsed_seconds":0.014171989}
- metrics_after_direct: {"address":291,"subject":35}
- member_b_forged_direct: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.004157721}
- limiter_state_reads: none; responses and public control metric counters only
