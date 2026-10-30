# Follow-up 7 functional pass before provenance correction

Run `46c64443-af2d-4295-a3a9-5e6c69bc4828`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T17:57:42Z`.
Prefix `w4-updates`, port base 42000. Make exit 2 solely for retained AT23/AT25.
AT24/AT26/AT27 passed.

The recorded harness file SHA-256 values are accurate. The additional
`differs_from_source_commit` boolean was incorrectly false: Git ran from Go's
test package directory with a repository-relative pathspec. The HTTP replay
assertion had changed from 200 to the contract's 202 since the source commit.
The Git invocation now explicitly uses the repository root. A fresh complete
run provides the final evidence with consistent hashes and difference flag.
No product behavior or test criteria changed in this correction.
