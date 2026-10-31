# member-ipv6

Member IPv6 path

Status: blocked

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §8.3, ISO03, AT11

- fvnb0_ipv6:
- member_disable_ipv6: true
- dns_resolver_ipv6: empty in the prepared node config
- blocker: native member uses DisableIPv6; no IPv6 underlay or platform route/capture campaign is provisioned. An address or ctv6 rule alone cannot prove AT11.
- owner: w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements
