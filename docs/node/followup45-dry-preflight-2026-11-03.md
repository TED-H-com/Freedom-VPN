# Follow-up 45 dry preflight — 2026-11-03

Passed before any reservation, with exact `free` admission for every Docker
command. One `fvpn-w2-agent-core-f45` container used A's frozen lab fixture,
with no scenario and no timing leg. The runner's real Docker admission wrapper
returned parseable probe JSON on stdout and `free` on stderr. Its three clock
measurements all matched: both namespace offset readings were zero and all
five host/probe endpoints were ordered. Namespace identities differed, which
is allowed by the corrected clock condition. Both identities and offset
readings are preserved in the [JSON receipt](followup45-dry-preflight-2026-11-03.json).
Container, image, network and volume inventory was empty after cleanup.

The new runner and results tool derive from repaired 44. Beyond follow-up
names and fresh output locations, the runner changes only its clock predicate;
the results calculations are unchanged. The existing 44 clock probe is reused
unchanged. Both newly built probe binaries match 44's declared SHA-256.

All 41 inherited/new synthetic tests pass from the repository. The 17 tests
for 45 also pass from another directory, using scratch located from the test
file under the repository's `.agent/tmp`. Synthetic cases cover matching same
and different namespaces, each nonzero seconds/nanoseconds offset on either
side, every bracket inequality, mismatch stopping, and the inherited monitor,
filtering, failure, pooling and stderr behavior. A changed-working-directory
scratch test passes. Three tests from 44 and one from 43 still fail from another
directory because their scratch path is relative; they are left unchanged.

Every reused A/B archive SHA-256, every extracted tree blob (18,889 A; 18,939 B),
overlay and fixture hash was checked again. No rebuild or new source hash was
needed. Earlier 39/43/44 runners, tools, tests and records remain unchanged.
The #429 production equality command from 44 was rerun once and returned exit
0 with empty output; its exact command is retained in the declaration pins.
