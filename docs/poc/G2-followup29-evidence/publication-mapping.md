# Follow-up 28 publication mapping (additive)

The orchestrator rewrote the unpushed follow-up 28 commits before PR [#411](https://github.com/TED-H-com/Freedom-VPN/pull/411): declaration `ef58364c` unchanged; measured harness `b398b5a0` → `c91eb3d5`; delivery `20ad4553` → `aa25dd8a`.

As executed, `host.py` read the admission script at a fixed location in the orchestrator's private repository. As published it reads `FVPN_HOLD_SH`. No private location is reproduced here. Local `git diff --name-only` and zero-context diffs verified that the harness trees differ only in that one `host.py` line; the delivery trees differ only in that line and `host.py`'s size/hash entry in `retention-pins.json`. The [JSON record](publication-mapping.json) retains the checked paths and diff digest, without the old line.

The manifest changed from `0f09a17da7da53c81ba3094570f0bf2d4dd9e832b27671fd743a4a9fd872a66f` to `dda5a42a27bd1e215aeb7384d71cdb5ef69286b9bfa873172d318138749f8e5d`. Follow-up 28's [overview](../G2-followup28-evidence/overview.md) and [window claim](../G2-followup28-evidence/window/claim.json) retain the executed harness name `b398b5a0`; its published counterpart is `c91eb3d5`.

No follow-up 28 file was edited. Its full-lab gate 2 remains FAIL; F27-01 remains unresolved and B1 remains FAIL.
