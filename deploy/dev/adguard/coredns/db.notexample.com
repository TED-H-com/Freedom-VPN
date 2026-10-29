$ORIGIN notexample.com.
$TTL 60

@ IN SOA ns1.notexample.com. hostmaster.notexample.com. (
    2026102901 ; serial
    3600       ; refresh
    1800       ; retry
    604800     ; expire
    60         ; minimum TTL
)

@                       IN NS    ns1.notexample.com.
ns1                     IN A     192.0.2.5
ns1                     IN AAAA  2001:db8::5

@                       IN A     192.0.2.40
@                       IN AAAA  2001:db8::40
@                       IN HTTPS 1 . alpn="h2,h3" ipv4hint="192.0.2.40" ipv6hint="2001:db8::40"
@                       IN SVCB  1 svc.notexample.com. alpn="h2" port=8443

*                       IN A     192.0.2.40
*                       IN AAAA  2001:db8::40
*                       IN HTTPS 1 . alpn="h2,h3" ipv4hint="192.0.2.40" ipv6hint="2001:db8::40"
*                       IN SVCB  1 svc.notexample.com. alpn="h2" port=8443
