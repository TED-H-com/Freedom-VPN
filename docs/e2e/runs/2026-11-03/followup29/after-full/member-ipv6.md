# member-ipv6

Member IPv6 path

Status: blocked

Run: 91eb0372-e42b-4698-a137-7cce53b263fa

Source: f52fc9381405d4aafd064dc0ef055977de97f418

Spec: §8.3, ISO03, AT11

- fvnb0_ipv6:
- member_disable_ipv6: true
- dns_resolver_ipv6: empty in the prepared node config
- blocker: native member uses DisableIPv6; no IPv6 underlay or platform route/capture campaign is provisioned. An address or ctv6 rule alone cannot prove AT11.
- owner: w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements
