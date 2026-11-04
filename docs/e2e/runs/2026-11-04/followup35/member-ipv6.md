# member-ipv6

Member IPv6 path

Status: blocked

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §8.3, ISO03, AT11

- fvnb0_ipv6:
- member_disable_ipv6: true
- dns_resolver_ipv6: empty in the prepared node config
- blocker: native member uses DisableIPv6; no IPv6 underlay or platform route/capture campaign is provisioned. An address or ctv6 rule alone cannot prove AT11.
- owner: w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements
- timed_section: member-ipv6
