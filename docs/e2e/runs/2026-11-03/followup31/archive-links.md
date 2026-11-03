# Follow-up 31 archive navigation verification

Verified at 2026-11-03T09:33:11.159936Z. Repository source: `63a30c412106f9f5bc6c1738cf23cd212634ce28`.
Members source read with `git show`: `1019e9591c68a3d8f9323dd51259d94d1e5e5bf8` (`w4-at-members`).
No members file was edited or copied into the worktree.

Added `docs/e2e/at/poc/G1-app-checklist.md` as a relative symlink containing
`../../../poc/G1-app-checklist.md`. It opens the real
[`docs/poc/G1-app-checklist.md`](../../../../../docs/poc/G1-app-checklist.md),
rather than a second checklist or an intermediate navigation page.

The exact `../../../poc/G1-app-checklist.md` link in every file below
normalizes to `docs/e2e/at/poc/G1-app-checklist.md`; resolving that symlink
with `Path.resolve(strict=True)` reaches `docs/poc/G1-app-checklist.md`.
Reading through it returns identical bytes. The real checklist SHA-256 is
`02e23afe8ff2c9e662cc31ce127da0ad75398fa73a202dee85b17abba893c98d`.
All 14 matching archives passed, including follow-up 16 from the
unmerged members branch and follow-ups 10 through 13.

| Archived source file | Link resolution | Source blob SHA-256 |
|---|---|---|
| `docs/e2e/at/members/history/follow-up-10/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-11/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-12/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-13/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-14/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-15/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-16/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-3/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-4/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-5/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-6/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-7/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-8/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |
| `docs/e2e/at/members/history/follow-up-9/OPERATORS.md` | `docs/e2e/at/poc/G1-app-checklist.md` → `docs/poc/G1-app-checklist.md` (PASS) | `f39903cf007b34b02569cdfa3c3f35a5aa2bbd70766d928b3c425a824fe8b782` |

Reproduce from the repository root (read-only apart from the added symlink):

```python
from pathlib import Path
import posixpath
import subprocess

ref = "1019e9591c68a3d8f9323dd51259d94d1e5e5bf8"
files = subprocess.check_output(
    ["git", "ls-tree", "-r", "--name-only", ref, "docs/e2e/at/members/history"],
    text=True,
).splitlines()
link = "../../../poc/G1-app-checklist.md"
target = Path("docs/poc/G1-app-checklist.md").resolve(strict=True)
for name in files:
    if not name.endswith("/OPERATORS.md"):
        continue
    blob = subprocess.check_output(["git", "show", f"{ref}:{name}"])
    if link.encode() not in blob:
        continue
    path = Path(posixpath.normpath(posixpath.join(posixpath.dirname(name), link)))
    assert path.resolve(strict=True) == target
    assert path.read_bytes() == target.read_bytes()
    print(name, "->", path, "-> docs/poc/G1-app-checklist.md: PASS")
```
