$ORIGIN example.com.
$TTL 60

@ IN SOA ns1.example.com. hostmaster.example.com. (
    2026102901 ; serial
    3600       ; refresh
    1800       ; retry
    604800     ; expire
    60         ; minimum TTL
)

@                       IN NS    ns1.example.com.
ns1                     IN A     192.0.2.2
ns1                     IN AAAA  2001:db8::2

; Apex records (RFC 5737 / RFC 3849 documentation prefixes)
@                       IN A     192.0.2.10
@                       IN AAAA  2001:db8::10
@                       IN HTTPS 1 . alpn="h2,h3" ipv4hint="192.0.2.10" ipv6hint="2001:db8::10"
@                       IN SVCB  1 svc.example.com. alpn="h2" port=8443

svc                     IN A     192.0.2.11
svc                     IN AAAA  2001:db8::11
svc                     IN HTTPS 1 . alpn="h2,h3" ipv4hint="192.0.2.11" ipv6hint="2001:db8::11"
svc                     IN SVCB  1 svc.example.com. alpn="h2" port=8443

; Allowed CNAME chain: cname-chain-ok -> cname-ok-hop -> safe-target.example.com
cname-chain-ok          IN CNAME cname-ok-hop.example.com.
cname-ok-hop            IN CNAME safe-target.example.com.
safe-target             IN A     192.0.2.15
safe-target             IN AAAA  2001:db8::15
safe-target             IN HTTPS 1 . alpn="h2,h3" ipv4hint="192.0.2.15" ipv6hint="2001:db8::15"

; Blocked CNAME chain: cname-chain-blocked -> cname-hop -> cname-blocked-target.example.net
cname-chain-blocked     IN CNAME cname-hop.example.com.
cname-hop               IN CNAME cname-blocked-target.example.net.

; Wildcard covering all single- and multi-label subdomains under example.com
*                       IN A     192.0.2.10
*                       IN AAAA  2001:db8::10
*                       IN HTTPS 1 . alpn="h2,h3" ipv4hint="192.0.2.10" ipv6hint="2001:db8::10"
*                       IN SVCB  1 svc.example.com. alpn="h2" port=8443
