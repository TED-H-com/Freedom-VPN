# Follow-up 17 run ledger

Baseline `a7497ddf0b08805e4cddfc0e3b5f47399205054e` (current main when this lane began, including #400, #401 and #406). The one assembled run is pending admission. Its measured harness commit, actual preflight `go env GOCACHE` in home-relative form, `vmstat 5 7`, process inventory and wait duration will be recorded here.

Before every `make`, `go test` and `docker` invocation, the authorized private admission command must print `free`; otherwise wait 60 seconds and recheck. A private executable wrapper also covers nested invocations. Priority timing suites run first. Process inventory excludes shell command text, deduplicates suite ancestors, and is checked again after stack startup. Any observed chain-gate overlap will be marked contention-confounded with no retry or cause attribution.

Scratch uses disk under `.agent/tmp` through Python `TemporaryDirectory`; no cache override and no shell cleanup trap is introduced. Docker prefix `fvpn-w4-members-`, `FVPN_E2E_PORT_BASE=24000`. Shared Postgres and other lanes are untouched.

Foreground gates, sequentially: area lint; the ten-case `make -f mk/at-members.mk at-members`; members offline tests; uncached handover tests; root `make go-lint go-with-disk-tmp=`; root vet; tracked gofmt check; diff whitespace; both architecture helper builds; committed-evidence hygiene. The root wrapper override disables its shell cleanup trap while retaining its checks.

[Archive and pins](history/follow-up-16/INDEX.md), [published commit mapping](history/follow-up-16/PUBLISHED-COMMITS.md), [results](summary.md).
