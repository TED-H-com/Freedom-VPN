# Follow-up 22 verification receipts

[Original gate receipts and output tails](F22-gates.json), [post-live admission](F22-gate-admission.json), [archive/link/scope audit](F22-evidence-audit.json).

| Gate | Exit | Seconds |
|---|---:|---:|
| `01c-area-lint` | 0 | 0.907 |
| `02c-e2e-lint` | 0 | 1.611 |
| `03c-offline-members` | 0 | 29.965 |
| `03d-offline-members-e2e` | 0 | 34.551 |
| `04b-assembled` | 2 | 2942.676 |
| `05-handover` | 0 | 4.61 |
| `06-root-lint` | 0 | 3.952 |
| `07a-vet` | 0 | 0.455 |
| `07b-tagged-vet` | 0 | 0.226 |
| `08-gofmt` | 0 | 0.134 |
| `09-diff` | 0 | 0.464 |
| `10-hygiene` | 0 | 27.379 |

Live exit 2 is the unchanged blocked-case result. The first area-wide e2e lint U1000 failure and the owned admission-parser pre-check failure remain in the ledger; both were corrected before the measured retry. Offline admission-parser tests passed 4/4. No case was rerun. Root lint supplies disk temp directories through the foreground runner and disables the root shell cleanup wrapper.

The handover suite includes `TestHandover_MarkdownLinks`; it passed without a handover file edit. Independent link audit resolves 633 paths, including all three run-2 G2 links. All 98 original follow-up 21 archive files, including gzip, match baseline bytes and SHA-256 pins. Earlier archives and dated notes are untouched. Committed-evidence hygiene scanned 270 case records and 633 Markdown documents.

`git diff --check` and whole-module gofmt are clean. No new file exceeds 20 MB without lossless gzip and both byte pins. The measured source remains `8a6dcf34`; evidence publication adds no product or live-predicate change. The current-main mismatch and requested disposition are recorded in [requests](requests.md).
