# member-ipv6

Member IPv6 path

Status: blocked

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §8, AT20

- fvnb0_ipv6:
- member_disable_ipv6: true
- dns_resolver_ipv6: empty in the prepared node config
- blocker: native member uses DisableIPv6; no IPv6 underlay or platform route/capture campaign is provisioned. An address or ctv6 rule alone cannot prove AT11.
- owner: w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements
