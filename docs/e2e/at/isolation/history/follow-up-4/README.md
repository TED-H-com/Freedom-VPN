# Isolation acceptance evidence (AT10–AT13)

`AT10.json` through `AT13.json` are the assembled dev stack. `lab-AT10.json` through `lab-AT13.json` are the privileged mechanism lab. A lab pass does not change the product status (§19.2).

The stack prefix is `w4-isolation` and the port base is 36000. The lab container prefix is `fvpn-w4-at-isolation`.

```
make -f mk/at-isolation.mk at-isolation-lint
make -f mk/at-isolation.mk at-isolation
```

`operator-run.md` is the manual script for member platforms, physical arm64 and a second NAT. Those steps stay `needs operator run`.
