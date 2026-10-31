# Follow-up 21 scenario ledger (TEST ONLY)

All statuses and source commits below come from the original records. The first
installer attempt aborted during setup after its login record; its process exit
is 1 despite that partial record being a pass. See README and gates.

| Run | Scenario | Status | Source | Requirements |
|---|---|---|---|---|
| 33b9621a | [admin-login](33b9621a-41d7-49fb-bc27-1c3f05e4ab44/admin-login.json) | pass | `5bc41666` | §14.2 |
| 33b9621a | [installer-loopback-wrapper](33b9621a-41d7-49fb-bc27-1c3f05e4ab44/installer-loopback-wrapper.json) | pass | `5bc41666` | ENR02, UPD05, AT07, D-102 |
| 33b9621a | [serving-catalogue](33b9621a-41d7-49fb-bc27-1c3f05e4ab44/serving-catalogue.json) | pass | `5bc41666` | §9, §14.2 |
| 52d682cd | [admin-login](52d682cd-83b6-419c-9fa0-9da6d2ecc027/admin-login.json) | pass | `b27bb361` | §14.2 |
| 99027e5d | [admin-login](99027e5d-ae5e-40d4-b595-7b550f89a52d/admin-login.json) | pass | `b27bb361` | §14.2 |
| 99027e5d | [desired-state](99027e5d-ae5e-40d4-b595-7b550f89a52d/desired-state.json) | pass | `b27bb361` | §5, §8 |
| 99027e5d | [dns-negative](99027e5d-ae5e-40d4-b595-7b550f89a52d/dns-negative.json) | pass | `b27bb361` | §12.5, §12.8, AT20, ISO01, DNS02 |
| 99027e5d | [enroll-bootstrap](99027e5d-ae5e-40d4-b595-7b550f89a52d/enroll-bootstrap.json) | pass | `b27bb361` | §6, §19 AT06 |
| 99027e5d | [fixture-baseline](99027e5d-ae5e-40d4-b595-7b550f89a52d/fixture-baseline.json) | pass | `b27bb361` | §13.2 |
| 99027e5d | [fixture-member-admission](99027e5d-ae5e-40d4-b595-7b550f89a52d/fixture-member-admission.json) | pass | `b27bb361` | §8, AT02, AT12 |
| 99027e5d | [fleet-retained](99027e5d-ae5e-40d4-b595-7b550f89a52d/fleet-retained.json) | pass | `b27bb361` | §6.2, AT24, AT26, D-089 |
| 99027e5d | [native-dns-config](99027e5d-ae5e-40d4-b595-7b550f89a52d/native-dns-config.json) | pass | `b27bb361` | §12.2, AT21 |
| 99027e5d | [native-dns-fallback](99027e5d-ae5e-40d4-b595-7b550f89a52d/native-dns-fallback.json) | pass | `b27bb361` | §12.2, AT21 |
| 99027e5d | [publish](99027e5d-ae5e-40d4-b595-7b550f89a52d/publish.json) | pass | `b27bb361` | §5.2, §14.2 |
| 99027e5d | [second-resolver](99027e5d-ae5e-40d4-b595-7b550f89a52d/second-resolver.json) | pass | `b27bb361` | §12.2, AT21 |
| 99027e5d | [serving-catalogue](99027e5d-ae5e-40d4-b595-7b550f89a52d/serving-catalogue.json) | pass | `b27bb361` | §9, §14.2 |
| cba88bbc | [admin-login](cba88bbc-a7bf-49f8-9dab-3182123d02aa/admin-login.json) | pass | `5bc41666` | §14.2 |
| cba88bbc | [desired-state](cba88bbc-a7bf-49f8-9dab-3182123d02aa/desired-state.json) | pass | `5bc41666` | §5, §8 |
| cba88bbc | [dns-history-integrity](cba88bbc-a7bf-49f8-9dab-3182123d02aa/dns-history-integrity.json) | pass | `5bc41666` | DNS02, AT20, D-102 |
| cba88bbc | [dns-negative](cba88bbc-a7bf-49f8-9dab-3182123d02aa/dns-negative.json) | pass | `5bc41666` | §12.5, §12.8, AT20, ISO01, DNS02 |
| cba88bbc | [enroll-bootstrap](cba88bbc-a7bf-49f8-9dab-3182123d02aa/enroll-bootstrap.json) | pass | `5bc41666` | §6, §19 AT06 |
| cba88bbc | [fixture-baseline](cba88bbc-a7bf-49f8-9dab-3182123d02aa/fixture-baseline.json) | pass | `5bc41666` | §13.2 |
| cba88bbc | [fixture-member-admission](cba88bbc-a7bf-49f8-9dab-3182123d02aa/fixture-member-admission.json) | pass | `5bc41666` | §8, AT02, AT12 |
| cba88bbc | [fleet-retained](cba88bbc-a7bf-49f8-9dab-3182123d02aa/fleet-retained.json) | pass | `5bc41666` | §6.2, AT24, AT26, D-089 |
| cba88bbc | [native-dns-config](cba88bbc-a7bf-49f8-9dab-3182123d02aa/native-dns-config.json) | pass | `5bc41666` | §12.2, AT21 |
| cba88bbc | [native-dns-fallback](cba88bbc-a7bf-49f8-9dab-3182123d02aa/native-dns-fallback.json) | pass | `5bc41666` | §12.2, AT21 |
| cba88bbc | [publish](cba88bbc-a7bf-49f8-9dab-3182123d02aa/publish.json) | pass | `5bc41666` | §5.2, §14.2 |
| cba88bbc | [second-resolver](cba88bbc-a7bf-49f8-9dab-3182123d02aa/second-resolver.json) | pass | `5bc41666` | §12.2, AT21 |
| cba88bbc | [serving-catalogue](cba88bbc-a7bf-49f8-9dab-3182123d02aa/serving-catalogue.json) | pass | `5bc41666` | §9, §14.2 |
