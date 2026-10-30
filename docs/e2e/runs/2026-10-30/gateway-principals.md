# gateway-principals

Gateway preserves independent client rate principals

Status: pass

Run: 76f0725c-2264-4021-a553-60f6c921e32b

Source: 1b2f358880d38ae643865edc2bb6f19756748456

Spec: §14.2, §19 AT28, §19 AT31

- trusted_proxies: ["172.28.30.21/32"]
- rates: unchanged defaults: browser 10/s burst 30; ingress 20/s burst 60
- distinct_members: true
- client_source_addresses: ["172.28.30.128","172.28.30.129"]
- forged_header_scope: rotating TEST ONLY documentation IPv4 values; source sockets unchanged
- member_a_gateway: {"status_counts":{"200":31,"429":89},"rate_limited_codes":89,"retry_after_present":89,"address_refusals":58,"subject_refusals":31,"elapsed_seconds":0.152007621}
- member_a_forged_gateway: {"status_counts":{"200":1,"429":119},"rate_limited_codes":119,"retry_after_present":119,"address_refusals":117,"subject_refusals":2,"elapsed_seconds":0.092473924}
- member_b_gateway: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.004456132}
- metrics_before: {"address":0,"subject":0}
- metrics_after_gateway: {"address":175,"subject":33}
- member_a_forged_direct: {"status_counts":{"200":2,"429":118},"rate_limited_codes":118,"retry_after_present":118,"address_refusals":117,"subject_refusals":1,"elapsed_seconds":0.012659939}
- metrics_after_direct: {"address":292,"subject":34}
- member_b_forged_direct: {"status_counts":{"200":1},"rate_limited_codes":0,"retry_after_present":0,"address_refusals":0,"subject_refusals":0,"elapsed_seconds":0.002898269}
- limiter_state_reads: none; responses and public control metric counters only
