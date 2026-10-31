## Development dependency advisory — 2026-10-31, Follow-up 19

Dependabot alert #22 reports `minimatch < 11.0.1`,
[GHSA-g3hv-75wg-m249 / CVE-2026-118764](https://github.com/advisories/GHSA-g3hv-75wg-m249),
with `scope: development`. The alert remains open: the permitted dependency-only
repair breaks the current accessibility linter. Both `package.json` and
`package-lock.json` were restored byte-for-byte; no tool versions, lint rules,
portal behavior, or live-stack fixtures changed.

### Dependency paths and attempted repair

`npm ls minimatch --all` before the attempts and after restoration:

```text
@freedom-vpn/portal@0.1.0
├─┬ eslint-plugin-jsx-a11y@6.10.2
│ └── minimatch@3.1.5
├─┬ eslint@9.39.5
│ ├─┬ @eslint/config-array@0.21.2
│ │ └── minimatch@3.1.5 deduped
│ ├─┬ @eslint/eslintrc@3.3.7
│ │ └── minimatch@3.1.5 deduped
│ └── minimatch@3.1.5 deduped
└─┬ typescript-eslint@8.25.0
  └─┬ @typescript-eslint/parser@8.25.0
    └─┬ @typescript-eslint/typescript-estree@8.25.0
      └── minimatch@9.0.9
```

1. `npm audit fix` without `--force` exited 1, retained both versions, and still
   reported the advisory (12 moderate entries including dependent packages).
2. Adding `"overrides": { "minimatch": "^11.0.1" }` and running `npm install`
   resolved every path to 11.0.3. `npm audit --audit-level=moderate` reported zero
   vulnerabilities. A clean `npm ci` succeeded and typechecking passed, but
   `npm run lint` exited 2, including after deleting `node_modules` and reinstalling:

   ```text
   ESLint: 9.39.5
   TypeError: (0 , _minimatch.default) is not a function
   Rule: "jsx-a11y/label-has-associated-control"
   ```

   The failing input is the existing label at
   `apps/portal/src/pages/admin/AdminDnsPolicyPage.tsx:494`.
   `eslint-plugin-jsx-a11y/lib/util/mayContainChildComponent.js:38` calls the
   legacy callable export; minimatch 11 exposes named exports. A per-parent
   override that keeps this parent on 3.x cannot satisfy the requirement that
   every installed minimatch be at least 11.0.1. Disabling the rule or patching
   the dependent tool would exceed this dependency-only change.

### Reachability evidence

Minimatch is a glob matcher used here by development tooling only. Both locked
copies have `dev: true`; `npm ls minimatch --omit=dev --all` returns an empty
tree (exit 1). The dependency paths above accept lint configuration, file paths,
and accessibility component patterns, rather than portal request input.

The advisory describes amplification in `firstPhasePreProcess` with
`optimizationLevel >= 2`. The installed 3.1.5 implementation has no such pass.
The 9.0.9 copy defaults to level 1; its direct caller in
`@typescript-eslint/typescript-estree/dist/useProgramFromProjectService.js`
passes no options. That optional project service is not enabled in
`apps/portal/eslint.config.js`. These are evidence about the current callers,
not a claim that all uses of the affected versions are safe.

`npm run build -- --manifest --sourcemap` exited 0 with Vite 6.4.5:

```text
✓ 128 modules transformed.
dist/.vite/manifest.json          0.19 kB
dist/assets/index-BqlzUxff.css    7.76 kB
dist/assets/index-DsVXBaUd.js   538.42 kB
```

The generated manifest has one entry, `index.html`, referencing the JavaScript
and CSS above, with no additional imported chunks. Its JavaScript source map
lists 80 sources; none is minimatch, ESLint, or typescript-eslint. Together with
the empty production dependency tree for minimatch, this shows no path into the
built production portal bundle for this revision. The optional manifest/source maps
were generated for inspection only; build configuration is unchanged.

After restoration, `make -f mk/portal.mk portal-verify` passed the clean install,
typecheck, ESLint, formatting, 9 harness tests, 65 Vitest tests, coverage (85.1%
lines), and production build. `make -f mk/portal.mk portal-e2e` passed all 9
Playwright tests against freshly built Go control and worker binaries. The
isolated test database, processes, and disk scratch were removed afterward.

The repository owner should review the evidence and keep the alert open while
upstream dependents adopt a compatible patched minimatch API. Do not apply the
blanket override solely to clear the advisory. Any eventual dependency repair
must pass a clean install, `portal-verify`, and the live `portal-e2e` suite, plus
the PR's own Portal CI before merge. No alert dismissal was performed.
