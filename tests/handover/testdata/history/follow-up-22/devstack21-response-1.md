## updates — 2026-10-31 (follow-up 21, F11-01, D-102)

The shared `installerSpec` now supplies the explicit TEST ONLY loopback opt-in
for its existing `https://127.0.0.1:9444` authenticated proxy. `installArgs`
requires both that boolean and the exact local origin before adding
`--allow-loopback`. Production defaults and generated reporting fields are
unchanged. [Fresh shared-wrapper run 33b9621a](runs/2026-10-31/followup21/33b9621a-41d7-49fb-bc27-1c3f05e4ab44/installer-loopback-wrapper.md)
passes 3/0/0: actual install exit **0**, status `installed`, generated
`result_url=https://127.0.0.1:9444/api/v1/agent/update-results`. The generated
configuration SHA-256 stays
`14a8975a8a36df9558b9b957e07613476e43de351281eadaae4657e860f06164`.
No-client-certificate POST returns **403**; the same deliberately invalid empty
receipt with the generated identity/server CA reaches central validation (**400**).
This verifies the authenticated endpoint and unchanged configuration, not a
new successful receipt delivery. Updates 11 retains the real automatic-delivery
proof. The [run overview](runs/2026-10-31/followup21/README.md) also preserves the
initial setup failure from sharing a stack with the third-node fixture.

F10-01 needs no further fixture change: updates 11 consumed the regular
release-root correction and measured installer 27/27 and regular AT23–AT27
5/5 without the former reporting adapter. Its F10-02 generated-reporting proof
also stands. Updates F11-02's obsolete offline-install paragraph belongs in
`docs/updates/node-updater.md`, outside this lane: updater/installer should
explain initial/same-release retained verified archives and distinguish the
uncached fallback. No artifact-origin change is warranted by a rejection that
did not occur; the supplied exact-origin/private-address interpretation is
accepted. This answer does not claim private-mirror deployment coverage.
