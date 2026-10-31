# AT10–AT13 isolation evidence

Current follow-up 8 run: **`7779d83c-2f78-4699-ac88-8fcc59e8cf87`**.
**AT10 blocked, AT11 blocked, AT12 pass, AT13 blocked.**
Unattended ISO-F6-01 recovery passes all six assertions; D-106 clean/rejected
wire digests match; the exact D-104 whole-node packet predicate passes.
The separate 100-reload AT10 gate passes under D-120 with zero losses.

[Summary and AT/ISO table](summary.md), [run procedure](RUN.md),
[requests](requests.md), [validation](validation-results.json).
Remaining scope: D-087's assembled provisional recipe, exact producer timing
exports, native IPv6/platform coverage and the continuous full fault matrix.
The measured renewal window's 17 losses remain explicit.

AT JSON/Markdown pairs are assembled cases. `lab-*` and `reload-stream/` retain
separate mechanism results. `receiver-capture.json` is the closed real-agent
wire stream. `tc-evidence-index.json`, `recovery-verdict.json` and
`deadline-verdict.json` provide reproducible checks against retained raw inputs.
The node `any` trace independently covers fvmb0 and fvup0 (D-108).

Corrected final historical records are immutable under
[follow-up 6](history/follow-up-6/summary.md) and
[follow-up 7](history/follow-up-7/summary.md), each with its run ID and SHA-256
manifest. No living-file handover claim pin needed moving (D-098 / D-109).
