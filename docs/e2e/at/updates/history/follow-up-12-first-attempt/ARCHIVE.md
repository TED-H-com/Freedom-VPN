# Follow-up 12 first attempt — owned supplemental harness error

Run `1f2b57de-de94-470b-a88c-b462b676b710`, source `c80701f2`, product main
`45209609`, 2026-10-31. These 17 generated files are preserved byte-identically
before the corrected rerun. Regular AT23–AT27 **5/5**, installer **27/27**,
shared wrapper **exit 0**, route capture **13 POSTs** all passed. Go test exit 0
in 501.781 s; the complete make target exit **2** stays failed.

The new lane-owned `supplemental.py` supplied the raw relative current symlink
as `seed --tree`. Seed requires the resolved release directory, so it returned
`CHECKPOINT_FAILED: seed tree must be the resolved release directory`, exit 1,
before download. The cache was restored byte-identically, config/current/status
stayed unchanged. This is a harness path error, not an artifact-origin rejection.
The generated `artifact-origin.json`'s unconditional `decision` sentence is
incorrect; its `passed=false`, exit, stderr and unchanged status are the deciding
observations. That sentence is retained here to preserve the original bytes.
The corrected harness resolves the pointer and conditions that narrative on
actual success. No product change or origin input is justified.

The runner completed teardown. No pass from this attempt is combined with the
final run; the full sequence is repeated after the owned fix. Verify these
immutable pins with `sha256sum -c SHA256SUMS` from this directory.
