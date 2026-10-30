# Follow-up 7 first diagnostic run

Run `008a155b-a94f-4c36-8cbb-a574258c678b`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T17:47:49Z`.
Prefix `w4-updates`, port base 42000. Make exit 2.

AT24 and AT26 passed. AT27 was blocked by a harness HTTP-status assertion:
the genuine prior-result replay returned HTTP 202 and exactly its original ACK,
with no newer-job, generation or release change. The harness incorrectly
required 200. `contracts/openapi.yaml` /agent/update-results specifies 202.
The assertion was corrected to 202 and the complete suite rerun on a fresh
stack. No product behavior or stored result was changed.

AT23/AT25 remain the retained follow-up 6 blocked records for F5-02.
