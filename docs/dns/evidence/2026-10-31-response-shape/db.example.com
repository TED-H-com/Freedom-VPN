; TEST ONLY synthetic authority. No forwarding or public DNS.
$ORIGIN example.com.
$TTL 300
@ IN SOA ns.example.com. hostmaster.example.com. (2026103101 3600 1800 604800 300)
@ IN NS ns.example.com.
@ IN A 192.0.2.10
ns IN A 192.0.2.11
blocked IN CNAME target.example.com.
chain IN CNAME hop.example.com.
hop IN CNAME blocked.example.com.
child.wild IN CNAME target.example.com.
target IN A 192.0.2.10
target IN AAAA 2001:db8::10
target IN SVCB 1 service.example.com. alpn="h2" ipv4hint="192.0.2.10" ipv6hint="2001:db8::10"
target IN HTTPS 1 service.example.com. alpn="h2" ipv4hint="192.0.2.10" ipv6hint="2001:db8::10"
target IN TXT "TEST ONLY response-shape control"
target IN MX 10 mail.example.com.
target IN NS ns.example.com.
service IN A 192.0.2.10
mail IN A 192.0.2.12
