# gateway-principals

Gateway preserves independent client rate principals

Status: pass

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §14.2, §19 AT28, §19 AT31

- trusted_proxies: ["172.28.30.21/32"]
- rates: unchanged defaults: browser 10/s burst 30; ingress 20/s burst 60
- distinct_members: true
- client_source_addresses: ["172.28.30.128","172.28.30.129"]
- forged_header_scope: rotating TEST ONLY documentation IPv4 values; source sockets unchanged
- member_a_gateway: {"status_counts":{"200":31,"429":89},"rate_limited_codes":89,"retry_after_present":89,"address_refusals":57,"subject_refusals":32,"elapsed_seconds":0.191520077}
- member_a_forged_gateway: {"status_counts":{"200":2,"429":118},"rate_limited_codes":118,"retry_after_present":118,"address_refusals":117,"subject_refusals":1,"elapsed_seconds":0.115381363}
- member_b_gateway: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.004356524}
- metrics_before: {"address":0,"subject":0}
- metrics_after_gateway: {"address":174,"subject":33}
- member_a_forged_direct: {"status_counts":{"200":1,"429":119},"rate_limited_codes":119,"retry_after_present":119,"address_refusals":117,"subject_refusals":2,"elapsed_seconds":0.01424395}
- metrics_after_direct: {"address":291,"subject":35}
- member_b_forged_direct: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.004004569}
- limiter_state_reads: none; responses and public control metric counters only
