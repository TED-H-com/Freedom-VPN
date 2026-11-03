# Published harness commit mapping

Additive provenance note, 2026-11-03. Run `0ffb355a-945b-4198-a551-3794120a686c` measured the first commit below. Publication redacted one home-directory path in its preflight `gocache` JSON line. The final published tree is identical to the final measured-branch tree. Archived evidence retains the original IDs verbatim; no pinned file or historical outcome was rewritten.

| Original ID | Published ID |
|---|---|
| `3a8b61594b447d164bb7c9481545e207f8688ec2` | `5d9e5f043ea14ec032698df4aa8349572157ef03` |
| `cbd59ccba4390b6746a83a159cedb884e1e135f4` | `876c0558a82ee21ff034d14057027dba363160e5` |
| `7852fba62ffdcc6271b2f8fbc72d63331096747d` | `f97da33d732bf85b495f93143915e51af8d21dad` |

Verification: `git cat-file -e <published-id>^{commit}` succeeds for all three published commits. Read-only tree comparison of `3a8b6159` and `5d9e5f04` reports exactly one insertion and one deletion, solely the `gocache` field in `docs/e2e/at/members/F16-host-preflight-admission-stop-1.json`. All other JSON values and files are identical. `git diff --quiet 7852fba6 f97da33d` exits 0. The redacted value is not reproduced here.

Future public `go env GOCACHE` receipts use `${HOME}/…` or `<home>/…`, never an absolute home-directory path. This additive note is outside the original 81-file SHA manifest.
