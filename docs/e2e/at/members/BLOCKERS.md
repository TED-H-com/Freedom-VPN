# Current blockers — Follow-up 14

**2 PASS / 5 BLOCKED / 3 FAIL**, with **628 passed / 6 blocked / 5 failed subchecks**. One assembled ten-case run; live make exit **2**.

AT02 fails counted re-grant delivery and provisional baseline-transition availability. AT04 fails an incomplete primary-expiry withdrawal read. AT09 fails the receive-only IPv6 limitation and the confounded relay hold. The original rows stay failed. AT05, AT01, AT20, AT21 and AT22 retain their owner-resource blockers; AT03 and AT08 pass.

**Harness qualification limit:** the compiled harness adds the routed hop before the original eth0-only relay hold. The added member interface can bypass that hold; the original relay row fails with `relayed=1 direct=1`. This is a confounded harness result, not proof of a product relay regression. Commit `682d4434` moves the routed rows after the relay check. The correction passes offline gates but is **not live-qualified**; no second assembled run was made.

See the [nine dated handoff groups](requests.md). F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 remain owner-resource rows. Routed-hop and positive IDNA helpers, command opt-in and private parse diagnostics have been delivered; those are no longer absent-helper blockers. Historical F11-01, F12-04 and both F13-01 failed rows are not rewritten by fresh successes or new evidence.
