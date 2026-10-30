# Gateway client principals (AT28 / AT31)

Two real member browser sessions issue requests from two distinct owned
container addresses through the same nginx public listener. The first sends a
bounded burst of authenticated `GET /api/v1/member/devices` requests beyond the
unmodified browser and ingress rates. The record must show 429 responses,
`RATE_LIMITED`, Retry-After, and increases in
`fvpn_control_rate_limited_total{class="browser",limiter="address"}` and
`limiter="subject"`. The other member must still receive 200.

A second burst varies client-supplied `X-Forwarded-For`; nginx appends its real
source, so refusals must persist. Direct requests to control from the untrusted
first client, with a forged header, must also remain address-limited; the
second client using the same forged value must still receive 200.
These distinguish connection principals through responses and metric deltas.
No limiter table is queried. Sessions/cookies stay in protected stdin and
memory; only response counts, fixed error codes/messages and bounded timing
are retained. The public listener trusts exactly the prepared gateway /32.

This is separate from nginx's worker connection budget and the agent mTLS
listener. The latter stays direct and ignores forwarded headers. A pass proves
this bounded two-client case, not the entire AT31 capacity workload.
