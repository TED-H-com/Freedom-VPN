# SECOPS-06: passed under D-060

The real `w4-secops` / `30000` run authenticated **20/20** distinct members
from a barrier start, with separate cookie jars and one source address.
Every session returned HTTP 200 with the correct member UUID. Neither limiter
counter advanced during the burst. Settings were the documented defaults:
address burst 40/refill 1 per second, subject 10 completions per minute.

| Probe | First rejection | Error | Exact Retry-After |
|---|---|---|---|
| One identity completing login repeatedly | Completion 11, `/auth/callback` | 429 RATE_LIMITED | `4` |
| One address initiating login without redirects | Request 41, `/auth/login` | 429 RATE_LIMITED | `1` |

The subject sequence took 2.593 seconds, so the remaining token refill wait was
four seconds. Another identity still authenticated. The address sequence reached
its rejection in 0.075926144 seconds. An untrusted forwarding header did not
bypass it; session bootstrap stayed HTTP 200, and a new login initiation succeeded
after the one-second wait. Counters increased once for each matching limiter.
Only natural refill waits were used; no clock or setting changes.

The final lab `GET /` landing response was HTTP 500 for all 20 users. The shared
login walker accepts that index-less landing; authenticated session checks are
separate. This is authentication proof, not portal rendering proof (SECOPS-14).

[Complete record](limiter.json) contains UTC times, build/image digests, settings,
bounded request/response evidence and commands. [Original 0/20 result](followup1-login.json)
is preserved and marked superseded by control follow-up 10 / PR #106.
