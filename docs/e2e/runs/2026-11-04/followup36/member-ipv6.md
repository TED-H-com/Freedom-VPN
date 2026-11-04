# member-ipv6

Member IPv6 path

Status: blocked

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §8.3, ISO03, AT11

- fvnb0_ipv6:
- member_disable_ipv6: true
- dns_resolver_ipv6: empty in the prepared node config
- blocker: native member uses DisableIPv6; no IPv6 underlay or platform route/capture campaign is provisioned. An address or ctv6 rule alone cannot prove AT11.
- owner: w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements
- timed_section: member-ipv6
