# Handover follow-up 50 — consolidated cut

This cut records **37 merged pull requests: #486–#518 and #520–#523**, through
`fe9bb436654e21f469afe507344bb4fbe25996d2`. The cutoff is the roadmap commit after
the final installer delivery, whose squash is `73d3c4b00571c34e0ad6ca97d5d51b2421ad81d6`.
[Follow-up 49](follow-up-49.md) covered #482–#485 and was itself merged as #487;
it included #483, the preceding handover cut, by the same rule.
[#519](https://github.com/TED-H-com/Freedom-VPN/pull/519) was closed without merging
and is not part of the 37 deliveries.

Each row links to its merged PR and records its title, full merge commit and
operator, member or developer effect. The [source manifest](../../tests/handover/testdata/followup50-main.json)
also retains PR heads, merge times and the following roadmap commits. These are
summaries of merged work, not new execution or measurement evidence. Any recorded
acceptance result below belongs to the cited delivery; this cut does not rerun
its campaigns or claim a production release.

## Guide revisions at this cut

The [operator guide](operator-guide.md) now directs weekly key inspection (§1.2)
to [vault metadata inspection](../pki/inspect.md#metadata-inspection), central
rotation/revocation (§8.1) to [key commands](../pki/keys.md),
[inspection and export](../pki/inspect.md) and the
[rotation runbook](../pki/rotation.md#central-signing-keys), and release trust
updates (§8.2) to the [release process](../updates/release-process.md) and
[central apply procedure](../pki/rotation.md#release-verification-trust).
[Node trust files](../node/install.md#node-trust-files) defines the files and
process actions needed to confirm adoption. Rotation activates the new central
signer immediately; exporting or delivering trust alone does not prove adoption.

Both language versions of the [member guide](member-guide.md) §5.3 now describe
shadow mode as normal member-visible status with null tier, rate and reason,
matching [FairUseStatus](../../contracts/openapi.yaml) and the
[portal documentation](../portal/README.md). An adjustment notice requires
adjusted status; applying can be shown separately.

The [Chinese](../DELIVERY.md) and [English](../DELIVERY.en.md) delivery guides
now describe scoped revisions to living guides and point here for this revision.
The [pre-cut provenance](../../tests/handover/testdata/followup50-snapshots.json)
pins exact copies from the cutoff. The claims test preserves older assertions
against those copies and checks every other byte of the guide bodies, allowing
only the cut lines and the four revised places (shadow mode in both languages).
Historical registers and content-specific references retain their earlier cut.

## Delivery and operating entry points

| PR | Merged title | Merge commit | Effect and boundary |
|---|---|---|---|
| [#487](https://github.com/TED-H-com/Freedom-VPN/pull/487) | Consolidate delivery guides and attribute the admission proof to its packet | `d6bdfca878d57ac14cae403292b318329ceb64d8` | Publishes follow-up 49 and consolidates delivery entry points; the policy-routing proof now attributes its drop to the injected packet rather than a shared counter. |
| [#489](https://github.com/TED-H-com/Freedom-VPN/pull/489) | Introduce Freedom VPN with bilingual project pages and READMEs | `d1f4dca661d555503ab0afc15557ad87d5bcbf24` | Adds English and Traditional Chinese project pages, READMEs and delivery navigation for the existing service and its operating limits. |
| [#490](https://github.com/TED-H-com/Freedom-VPN/pull/490) | Rewrite the bilingual introduction around VPN use and operation | `dd5f098d3662b9af4a0beafd58221693fb221d47` | Rewrites the bilingual introduction around VPN use, membership and node operation, with direct paths into deployment and source documentation. |
| [#491](https://github.com/TED-H-com/Freedom-VPN/pull/491) | docs: align bilingual README and official-app onboarding | `2d2fbe177d37206af2a64c5bdf1163a6db57ec4c` | Aligns both READMEs with official NetBird app onboarding while retaining developer commands, support boundaries and historical evidence. |
| [#498](https://github.com/TED-H-com/Freedom-VPN/pull/498) | docs: align bilingual delivery guides with completed serving-read work | `4d42af277fa8a5c1b588572821004aa59000d4ca` | Delivery entry points describe the completed serving-read integration and its acceptance while keeping archived checkpoint facts and outstanding device limits. |
| [#502](https://github.com/TED-H-com/Freedom-VPN/pull/502) | Clarify current runbook guidance and retained node verification | `4f5d616618ddb8671125a35a3d82777fd99ac7e6` | Runbook entry points distinguish current delivery guidance from archived cuts, and the operator guide explains retained verifier evidence in both languages. |
| [#505](https://github.com/TED-H-com/Freedom-VPN/pull/505) | Clarify deployment origin exceptions and database role configuration | `48e87301df9358e805d56997e8ce845c7abe65f6` | Deployment instructions preserve the required private/CGNAT control-origin exception and separate control, worker and migrator database credentials through existing configuration overrides. |
| [#521](https://github.com/TED-H-com/Freedom-VPN/pull/521) | docs: list the key and trust commands and the out-of-date handover statements in the delivery guides | `ce3d5b546827e7c164d46e23260982fdcbe99ba8` | Both delivery guides list the current key/trust commands and identify four outdated guide statements. This cut resolves those statements in place and replaces the temporary outdated-statement table. |

## Retained verification and diagnostic evidence

| PR | Merged title | Merge commit | Effect and boundary |
|---|---|---|---|
| [#486](https://github.com/TED-H-com/Freedom-VPN/pull/486) | isolation: retain clone lifecycle evidence and the failed follow-up 27 campaign | `a59de0268e17cca1c66d4954c5db077e035ce0b9` | Retains clone lifecycle and terminal-stop evidence plus the failed follow-up 27 campaign, so operators can inspect its missing ingress and source-bracket gaps without changing the original verdict. |
| [#492](https://github.com/TED-H-com/Freedom-VPN/pull/492) | nodes: project retained serving-read evidence with selected observations | `c6ff8d5efc2799b736808071b80458ccb574e5c4` | Central projects retained serving-read evidence from the same selected observation as current checks, scoped to the current incarnation. Migration 00195 requires draining old writers before migration and updated Central startup; mixed writers are unsupported. |
| [#493](https://github.com/TED-H-com/Freedom-VPN/pull/493) | diagnostics: show retained verifier results in portal and evidence exports | `279aca6682b8866522bdcf12566c4b6d5fc03f5e` | The bilingual admin view and evidence exporters show retained verifier results while preserving absent/null distinctions, exact generations and current failures; recovery windows compare exact UTC instants. |
| [#494](https://github.com/TED-H-com/Freedom-VPN/pull/494) | agent: report serving evidence without overwriting check failures | `c96fe032fa32af78c8d798d288056158aba70e85` | Node observations emit the dedicated serving_read field without overwriting an existing QoS failure cause, and queued reports preserve their original payloads across retries. |
| [#495](https://github.com/TED-H-com/Freedom-VPN/pull/495) | isolation: wait for a coherent source launch before clone evidence | `d98719004c5f6d2aaa8f64476d51764452336446` | Clone collection waits within the existing budget for a coherent source launch and preserves the successful snapshot as its initial bracket; timeout leaves the bracket unavailable. |
| [#496](https://github.com/TED-H-com/Freedom-VPN/pull/496) | node: preserve TC failure context and complete serving-read acceptance | `ad35d8bc36115056bcf3e3c83bb30f415a2ca797` | Preserves bounded TC failure context and records D164 acceptance closing F17-01 and retiring the D-163 exception. The merged record reports 27 passing rows and two expected blocks; earlier failures and platform/fault-matrix limits remain. |
| [#497](https://github.com/TED-H-com/Freedom-VPN/pull/497) | e2e: retain production logs and DNS failure evidence | `7b49ddcb2b0b76439c1bae52b73eba2861054d3c` | Recovery collection decodes production slog text as well as watchdog JSON, and DNS failures retain safe query errors plus explicit after-counter availability. |
| [#499](https://github.com/TED-H-com/Freedom-VPN/pull/499) | Preserve Apply and member transport startup failure stages | `6d52fadf7ced435f79cc930ca60fcbdecb1a4c71` | Apply and member transport diagnostics preserve the stage that actually failed and safe context observations before cleanup; safety decisions and deadlines remain unchanged. |
| [#503](https://github.com/TED-H-com/Freedom-VPN/pull/503) | Capture bounded Apply timing in E2E recovery diagnostics | `666e7b1379cf4b48d6b190203f718f886bf8a132` | E2E recovery artifacts retain bounded Apply timing and effective deadline observations for later investigation; the diagnostic change does not establish the historical timeout’s cause. |
| [#509](https://github.com/TED-H-com/Freedom-VPN/pull/509) | isolation: retain probe failure diagnostics and incomplete reports | `96de57eaa114e8a3e4b9a1961216d9874fab05b1` | Isolation probe failures retain bounded socket/send diagnostics, and missing ingress yields explicit incomplete reports with no unsupported kernel or deadline attribution. |

## Keys, release trust and node installation

| PR | Merged title | Merge commit | Effect and boundary |
|---|---|---|---|
| [#506](https://github.com/TED-H-com/Freedom-VPN/pull/506) | release: validate keygen trust before writing output files | `2c9c6bc41500f917c0ca220c38bb3afed6feab1f` | Release key generation validates requested public-trust windows before writing either output; invalid input preserves existing files, while later filesystem failures can still leave partial output. |
| [#507](https://github.com/TED-H-com/Freedom-VPN/pull/507) | pki: add explicit one-shot FileProvider initialization | `81d3bfda84ec0dc5fac0abe8c1c338b6b6703e4a` | Adds explicit one-shot staging vault initialization with purpose keys, an independently verified release public root and a matching CA; existing destinations and retries cannot overwrite a vault. |
| [#510](https://github.com/TED-H-com/Freedom-VPN/pull/510) | pki: inspect an existing FileProvider vault and export its public node trust | `199c64f5fa61b92bb8735d57e5f898ff364eab7d` | Adds read-only vault inspection and public trust export, with explicit metadata versus key-use checks, bounded locking and destination refusal rules. |
| [#511](https://github.com/TED-H-com/Freedom-VPN/pull/511) | release: add offline trust-update and trust-verify for release root rotation | `6c98ab807129c640b5d00a3b571c431156092d59` | Adds offline release trust-update and trust-verify commands for one signed root operation at a time; offline success does not apply an update to Central or establish node adoption. |
| [#513](https://github.com/TED-H-com/Freedom-VPN/pull/513) | pki: add fvpn-vault-keys for conditional rotation and revocation of central signing keys | `4770476afb7f118d0be3e3f48bebd64690b7d9c3` | Adds conditional central key rotation and revocation using inspected ledger revision and expected signer/purpose, with explicit refusal, uncertain-durability and lost-output handling. |
| [#515](https://github.com/TED-H-com/Freedom-VPN/pull/515) | release: keygen refuses key IDs that cannot sign | `49c2c8dfa788d42f1aaac5a0c262fe6b9e74648e` | Release key generation rejects IDs that cannot sign before generating a key or touching outputs; legacy opaque IDs remain valid retirement and revocation targets. |
| [#517](https://github.com/TED-H-com/Freedom-VPN/pull/517) | pki: add fvpn-vault-keys apply-release-update for conditional release trust updates | `dba3aed4bdc1d5c4ec7770d403214bd508e5a80b` | Adds conditional central application of signed release-trust updates with inspected ledger revision, guarded last-root removal and explicit outcomes; possible repeats require inspection when their signer is revoked or expired. |
| [#522](https://github.com/TED-H-com/Freedom-VPN/pull/522) | node: document trust files and rollout actions; test desired-trust checks and installer copy | `a623e36e67490abbaf6b9ea71aa3264440f4245c` | Documents the three node trust files, their readers and rollout actions, and adds desired-trust verification/copy checks; the fallback it recorded is superseded by the next installer change below. |
| [#523](https://github.com/TED-H-com/Freedom-VPN/pull/523) | installer: default desired-state trust to the trust bundle and check it before install | `73d3c4b00571c34e0ad6ca97d5d51b2421ad81d6` | Installer desired-state trust now defaults to the central trust bundle and is checked before host identification or changes. Earlier release-only copies need explicit replacement and a fresh updater invocation; upgrade and already-current reruns do not refresh them. |

## Member status and fair use

| PR | Merged title | Merge commit | Effect and boundary |
|---|---|---|---|
| [#512](https://github.com/TED-H-com/Freedom-VPN/pull/512) | portal: member service screen follows the entitlement state, with live expired and suspended checks | `f3bfb4a5c5ae5f1938a85d971dfe14caf8e3b4c9` | Member service and billing guidance follow the actual entitlement state, server next actions and available dates. The PR records live expired/suspended portal checks, while members acceptance must consume that evidence separately. |
| [#514](https://github.com/TED-H-com/Freedom-VPN/pull/514) | members: record the delivered F4-04 portal evidence in the request register | `a7e8a6fbcef730a71d5f2f39bf01a582460aa837` | The members request register records the delivered portal evidence and its limits; it leaves AT04 and F4-04 closure to members acceptance. |
| [#516](https://github.com/TED-H-com/Freedom-VPN/pull/516) | portal: fair-use notice follows the server's reason, date and applying state; billing labels translated | `1f8911ba28dbc6960699dc78ecaa21b16e3ac743` | Member fair-use notices use the server’s reason, reset date and applying state, while billing labels are translated; an administrator adjustment is no longer described as high usage. |

## Developer fixtures and maintenance

| PR | Merged title | Merge commit | Effect and boundary |
|---|---|---|---|
| [#488](https://github.com/TED-H-com/Freedom-VPN/pull/488) | Use the bound control address in the binary integration test | `55dec4b894092dd109b8c1cc80b9c04ced664bff` | The binary integration harness connects to the service’s reported bound address and retains safe early-exit diagnostics instead of relying on ports released before the build. |
| [#500](https://github.com/TED-H-com/Freedom-VPN/pull/500) | Use a consistent monotonic clock in Apply diagnostic tests | `51e30722b495d0984a75709839d487b64d84a9e3` | Apply diagnostic tests use a consistent real monotonic sample, avoiding future model timestamps on short-uptime CI runners without changing production timing. |
| [#501](https://github.com/TED-H-com/Freedom-VPN/pull/501) | Make rendered CoreDNS fixtures readable under restrictive umasks | `1f1cf675058b189bcfec3cdb604f4c109de2411b` | Rendered public CoreDNS fixtures receive readable modes under restrictive umasks while secret modes, parent directories and the caller’s umask remain unchanged. |
| [#504](https://github.com/TED-H-com/Freedom-VPN/pull/504) | Fix offline recovery observer timing fixture | `f758354fbc6dcfe4f2d1e75419f4b8863add1eec` | The offline observer fixture accepts the collector’s exact timing-read command and exercises read failure without turning partial data into available evidence. |
| [#508](https://github.com/TED-H-com/Freedom-VPN/pull/508) | backup: annotate EXIT-trap cleanup for ShellCheck 0.11 | `103bde3fc114a75297e01cac00545f6e66053541` | A function-scoped ShellCheck annotation accounts for EXIT-trap cleanup; executable backup script bytes are unchanged. |
| [#518](https://github.com/TED-H-com/Freedom-VPN/pull/518) | tests: build authorized-device fixtures from the run's clock so they do not expire with the calendar | `6733305b307f732e014c880b1ba188cc9cdbedce` | Authorized-device integration fixtures derive entitlement windows and related dates from the run’s clock so calendar expiry does not invalidate their original assertions. |
| [#520](https://github.com/TED-H-com/Freedom-VPN/pull/520) | tests: date the policies HTTP and DNS publication fixtures from the run's clock before they expire | `03fccc4284588c4c9ff3a3d1328bc3a690ebf6d8` | Policy HTTP override and DNS publication fixtures derive their live-clock windows from the run, preserving their assertions and leaving fixed-clock unit helpers intact. |

## Acceptance boundaries carried forward

The [D164 acceptance record](../e2e/at/isolation/d164-acceptance.md), delivered
with #496, records the fresh lab and assembled result; the earlier
[failed isolation campaign](../e2e/at/isolation/followup27-2026-11-05.md) remains
failed. The reporting and diagnostic changes do not independently authorize
forwarding or change lease decisions. Portal evidence delivered in #512 and
recorded by #514 does not itself close the members acceptance request.

Real official-app devices, the remaining IPv6 and fault-transition coverage,
physical arm64 and candidate hardware, independent-clock validation and
production configuration retain the limits stated in the [delivery guide](../DELIVERY.en.md#verification-scope-and-known-limits)
and the linked acceptance records. This consolidation adds no new device,
network, throughput, deployment or CI result.
