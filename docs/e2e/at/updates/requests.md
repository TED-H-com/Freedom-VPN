# Updates requests — follow-up 11, 2026-10-31

Run `daddad24-b6f7-4c32-ba8e-1c7d24f76ea2`, product main
`96ec0630e4834e13a3a14d65553ba8108bb5da55`, measured harness
`3ac13f7e115edf22e2ec9d3a6450733ec1fb7b6a`. The generated-reporting
extension passes without editing the generated configuration. Prior requests
remain [archived with follow-up 10](history/follow-up-10/requests.md).

### Requests to orchestrator

| ID / date | Owner | Deciding observation | Requested change outside this lane |
| --- | --- | --- | --- |
| F11-01 / 2026-10-31 | Devstack | [Installer hand-off](installer-handoff.json), `install_wrapper`: exit **64**, `usage`, explicit loopback-opt-in diagnostic; before/after key, cert, identity, usage and current remain absent. The shared spec uses `https://127.0.0.1:9444`. Repeating the same install through `ProbeInstallerExit` with **TEST ONLY `--allow-loopback`** succeeds, exit **0**. | In `tests/e2e/installer_exports.go`, let `installArgs`/the installer spec supply the explicit TEST ONLY loopback opt-in for its existing local proxy. Do not change production defaults or generated reporting fields. Publish the D-102 answer in `docs/e2e/RESPONSES.md`. This lane supplies the documented input only in its own harness; no shared file was edited. |
| F11-02 / 2026-10-31 | Updater, installer consult | Initial install/seed succeeds using the retained archive. `packaging/installer/install.go` calls `retainArchive` on initial and same-release installs. The [uncached probe](artifact-origin.json), run after deliberately withholding that cache, succeeds and adds a verified current baseline while preserving transaction state. | Correct the **Offline installation** paragraph in `docs/updates/node-updater.md`, which still says initial install retains only the envelope and needs a reachable artifact URL unless already staged. Describe the current installer's retained verified archive and the distinct missing-cache fallback. This is a documentation mismatch, not an updater failure. |

### F10 closure and artifact-origin decision

F10-01's shared layout correction is consumed from devstack 20 / #343 and
its [D-102 answer](../../RESPONSES.md#updates--2026-10-31-follow-up-20-f10-01-d-102).
The two owned executable checks now use the release root. See the current
[AT table](follow-up-11.md) for this lane's independent sequence.

F10-02 closes within the measured installer extension: generated URL/server
CA/client identity/exact origin policy, unchanged config hash, real sealed
receipt and matching local/central ACK, `NodeUpdateSummary.source=installer`,
D-110 recomputation, and [one automatic POST per receipt](installer-delivery-transport.json).
The existing lab enrollment proxy forwards authenticated requests to the real
central; no endpoint/CA/origin reporting adapter remains.

The anticipated artifact-origin rejection **did not occur**. Normal hand-off
uses verified staged archives; the separate uncached probe also exits **0**.
The shared release hostname resolves to an owned **non-private** Docker origin.
`allowed_artifact_origins` grants private/CGNAT exceptions; it is not an exclusive
list of every downloadable host. This matches `node/updater/verify.go` and the
D-046 paragraphs in `docs/updates/node-updater.md`. Release-process documentation
permits immutable separate HTTPS artifact URLs. Neither hosting nor generated
policy was changed, and no artifact-origin installer input is requested on an
unobserved rejection. Private mirror deployment remains outside this result.

F18-01 is retained as an intended negative: stopped producer apply/replay/recover
**1/1/2**, `CHECKPOINT_FAILED`, original sealed failure and blocked forwarding;
it is never reported as a passing upgrade. The old failure row is unchanged.

Request count: **2**, neither blocks the measured acceptance sequence. The
wrapper request has an explicit owned-harness input; the other is documentation.
