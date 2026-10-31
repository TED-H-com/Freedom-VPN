# Follow-up 10 run instructions

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
make -f mk/at-secops.mk at-secops-followup10
make go-lint
go vet ./...
gofmt -l .
git diff --check
```

Go 1.27.3 and Docker are required. The foreground driver refuses an occupied
`fvpn-w4-secops-*` namespace, brings up the existing lab with port base 30000,
90-second TEST ONLY grace and disk scratch, runs the live test and always tears
down its objects. Raw logs remain in ignored `.agent/secops/` with mode 0600.

The test performs the same T0/T1 physical restore and retained-archive replay
protocol as Follow-up 9, uses the D-077 HTTP admin re-authorization route,
records SECOPS-20 whole-ruleset read-back and SECOPS-21 generation ordering,
then measures finite past_due grace and 60-peer final protected-stream attribution.
No direct AuthorizeDevice call, generation repair or manual NAT removal remains.
Follow-up 9's direct service method is historical, documented in
`history/follow-up-9/RUN.md`; use this target for current re-measurement.

After cleanup, `followup10_records.py` renders records and scope pins. A missing
or failed live check stays blocked. AT29 established transports and AT31 relay
are deferred to devstack follow-up 15. The scan cohort does not claim capacity.
