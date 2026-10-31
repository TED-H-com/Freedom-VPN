#!/usr/bin/env python3
"""First-delivery closure guards and restore-only Follow-up 13 evidence."""
import datetime as dt
import hashlib
import json
from pathlib import Path
import re

from followup10_records import load, measurements
from followup9_records import evaluate_at30, clean_inventory
from run import ROOT, PUBLIC, write_json, utc

ARCHIVE = PUBLIC / 'history/follow-up-12'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def decimal(value):
    return (isinstance(value, str) and re.fullmatch(r'(0|[1-9][0-9]{0,19})', value)
            and int(value) <= 18446744073709551615)


def ordered(*values):
    try:
        times = [dt.datetime.fromisoformat(v.replace('Z', '+00:00')) for v in values]
        return all(t.tzinfo is not None for t in times) and all(a <= b for a, b in zip(times, times[1:]))
    except (ValueError, TypeError, AttributeError):
        return False


def closure_guards(run, live):
    row = live.get('first_desired_offer', {})
    capture = row.get('capture', {})
    first, local, ack = (capture.get(k) or {} for k in ('first', 'local', 'ack'))
    receipt = row.get('central_receipt', {})
    retained = row.get('retained_before_restore', {})
    prior, generation = retained.get('node_accepted_desired'), first.get('generation')
    observation = row.get('retained_generation_observation', {})
    body_digest = first.get('body_sha256', '')
    return {
        'live_restore_passed': run.get('live', {}).get('status') == 'passed' and live.get('go_test_failed') is False,
        'd099_present': run.get('d099_roadmap_present') is True,
        'd095_hold_proved': live.get('D-095', {}).get('passed') is True,
        'one_matching_capture': row.get('arm_count') == 1 and bool(row.get('capture_id')) and row.get('capture_id') == capture.get('capture_id'),
        'retained_generation_observed_before_release': bool(decimal(prior) and retained.get('node_applied') == prior
            and row.get('agent_stopped_before_arm') is True and row.get('retained_generation_ordering_proved') is True
            and observation.get('applied_generation') == observation.get('central_applied') == prior
            and observation.get('report_id')
            and ordered(row.get('armed_at'), observation.get('received_at'), row.get('release_started_at'), first.get('delivered_at'))),
        'first_delivered_http_200_complete': first.get('http_status') == 200 and first.get('write_complete') is True
            and isinstance(first.get('bytes'), int) and first['bytes'] > 0 and not capture.get('error'),
        'first_generation_above_retained': bool(decimal(prior) and decimal(generation) and int(generation) > int(prior)),
        'first_offer_local_digest_and_applied_match': bool(generation and re.fullmatch('[0-9a-f]{64}', body_digest)
            and local.get('accepted_generation') == local.get('applied_generation') == generation
            and local.get('envelope_sha256') == body_digest
            and ordered(first.get('delivered_at'), local.get('sampled_at'))),
        'first_offer_applied_ack_with_central_receipt': bool(generation and ack.get('applied_generation') == receipt.get('applied_generation') == generation
            and ack.get('http_status') in (200, 202) and ack.get('report_id')
            and ack.get('report_id') == receipt.get('report_id') and receipt.get('node_id')
            and ordered(first.get('delivered_at'), receipt.get('received_at'), ack.get('received_at'))),
        'no_generation_repair': live.get('SECOPS-21', {}).get('manual_generation_adjustment') is False,
    }


def refresh_scope():
    scope = load(PUBLIC / 'scope-review.json')
    def refresh(value):
        if isinstance(value, dict):
            if 'path' in value and 'sha256' in value:
                path = Path(value['path'])
                archived = ARCHIVE / path.name
                if path.parent == Path('docs/e2e/at/secops') and archived.exists():
                    path = archived.relative_to(ROOT)
                    value['path'] = str(path)
                if (ROOT / path).is_file():
                    value['sha256'] = digest(ROOT / path)
            for child in value.values():
                refresh(child)
        elif isinstance(value, list):
            for child in value:
                refresh(child)
    # Old claims follow their immutable records; current pins below are separate.
    scope.pop('follow_up_13', None)
    refresh(scope)
    names = ['AT29.json', 'AT29.md', 'AT30.json', 'AT30.md', 'AT31.json', 'AT31.md', 'SECOPS-21.json',
             'restore-reconcile.json', 'restore-reconcile.md', 'summary.md', 'README.md', 'RUN.md', 'requests.md',
             'live.json', 'cleanup.json', 'validation.json', '2026-10-31-follow-up-13-handover.md',
             'followup13-restore-live.json', 'followup13-restore-run.json', 'followup13-verdict.json',
             'followup13-lab-preflight.json', 'history/index.json', 'history/README.md', 'history/follow-up-12/archive-index.json']
    scope['follow_up_13'] = dict(reviewed_at=utc(), scope='Restore-only first-delivery proof; AT29/AT31 retained blocked; immutable Follow-up 12 archive.',
        records=[dict(path=str((PUBLIC / n).relative_to(ROOT)), sha256=digest(PUBLIC / n)) for n in names])
    write_json(PUBLIC / 'scope-review.json', scope)


def main():
    run = load(PUBLIC / 'followup13-restore-run.json')
    path = PUBLIC / 'followup13-restore-live.json'
    if not run.get('finished_at') or not path.exists():
        raise SystemExit('Complete the one restore run before rendering current records')
    path.write_text(path.read_text().replace(str(ROOT) + '/', ''))
    live = load(path)
    case = live.get('AT30_restore_reconcile', {})
    status, blockers, evaluation = evaluate_at30(case, http_regrant=True)
    guards = closure_guards(run, live)
    for key in ('live_restore_passed', 'd095_hold_proved'):
        if not guards[key]:
            status = 'blocked'
            blockers.append(key + ' is not proved')
    closed = all(guards.values())
    common = dict(schema_version=1, follow_up=13, application_base=run.get('application_base'),
                  source_revision=run.get('source_revision'), run_id=live.get('run_id'), driver_run_id=run.get('driver_run_id'),
                  started_at=run.get('started_at'), finished_at=run.get('finished_at'),
                  hardware='shared development host; not candidate hardware', record_renderer_sha256=digest(Path(__file__)),
                  previous_run='history/follow-up-12/', projector_owned_rows_written=[])
    current = measurements(case)
    baseline = measurements(load(ARCHIVE / 'restore-reconcile.json').get('case', {}))
    # Baseline display is the orchestrator's rounded development-host comparison;
    # preserve exact previous values alongside it, without calling this an SLA.
    deltas = {}
    for key, previous in [('Pre-replay database RPO (s)', 106.45), ('Total outage RTO (s)', 74.53)]:
        value = current[key]
        deltas[key] = dict(follow_up_12_baseline_seconds=previous, follow_up_13_seconds=value,
                          delta_seconds=value - previous if isinstance(value, (int, float)) else None)
    write_json(PUBLIC / 'restore-reconcile.json', dict(common, id='AT30-RESTORE-RECONCILE', status=status, case=case,
        evaluation=evaluation, blockers=blockers, measurement_comparison={'follow_up_12': baseline, 'follow_up_13': current},
        development_host_deltas=deltas, production_sla_claimed=False))
    write_json(PUBLIC / 'AT30.json', dict(common, id='AT30', status=status, requirements=['DATA01','BILL01','MTR03'],
        spec=['§17.3','§19 AT30'], evaluation=evaluation, blockers=blockers, development_host_deltas=deltas,
        evidence=['restore-reconcile.json','followup13-restore-live.json','SECOPS-21.json']))
    gen = live.get('SECOPS-21', {})
    offers = gen.get('offers', [])
    write_json(PUBLIC / 'SECOPS-21.json', dict(common, id='SECOPS-21', status='passed' if closed else 'blocked',
        request_status='closed' if closed else 'open', closure_guards=guards, hold=live.get('D-095'),
        actual=gen, first_delivered_offer=live.get('first_desired_offer'),
        first_http_probe_after_fresh_observation=live.get('first_http_probe_after_fresh_observation'),
        first_new_stored_compiler_row=offers[0] if offers else None,
        harness_validation_only='../../runs/2026-10-31/followup18/5f74d7c3-6404-461d-9f47-296bd69958c2/first-desired-offer.md'))
    for at in ('AT29','AT31'):
        old = load(ARCHIVE / (at + '.json'))
        old.update(follow_up=13, status='blocked', record_kind='carried forward; no transport rerun',
                   current_restore_run_id=live.get('run_id'),
                   evidence=['history/follow-up-12/'+at+'.json', 'history/follow-up-12/followup12-transports-live.json',
                             'history/follow-up-12/followup12-transports-run.json'])
        write_json(PUBLIC / (at + '.json'), old)
        text = (ARCHIVE / (at + '.md')).read_text().replace('— Follow-up 12', '— Follow-up 13 (retained Follow-up 12 measurements)')
        text += '\nNo transport rerun in Follow-up 13. Full acceptance remains **blocked** on the same resources and exports. '
        text += f'[Original measured row](history/follow-up-12/{at}.json); [resource requests](requests.md).\n'
        (PUBLIC / (at + '.md')).write_text(text)
    cleanup = clean_inventory(run.get('inventory_after', {})) and run.get('state_cleanup', {}).get('removed') is True
    verdict = dict(common, cases={'AT29':'blocked', 'AT30':status, 'AT31':'blocked'},
                   requests={'SECOPS-21':'closed' if closed else 'open'}, cleanup='passed' if cleanup else 'blocked')
    write_json(PUBLIC / 'followup13-verdict.json', verdict)
    write_json(PUBLIC / 'live.json', dict(common, record_kind='one restore regression; earlier transport measurements archived',
        cases=verdict['cases'], evidence=['followup13-restore-live.json','followup13-restore-run.json']))
    write_json(PUBLIC / 'cleanup.json', dict(common, status=verdict['cleanup'], inventory_after=run.get('inventory_after'),
        down=run.get('down'), state_cleanup=run.get('state_cleanup')))
    write_json(PUBLIC / 'validation.json', dict(common, live_status=run.get('live',{}).get('status'),
        closure_guards=guards, cleanup=verdict['cleanup'], gates='See .agent/REPORT.md for actual offline gate output and exit codes.'))
    table = '| Measurement | Follow-up 12 | Follow-up 13 |\n|---|---|---|\n'
    table += ''.join(f'| {key} | {baseline.get(key)} | {value} |\n' for key, value in current.items())
    row = live.get('first_desired_offer', {})
    capture = row.get('capture', {})
    first, local, ack = (capture.get(k, {}) for k in ('first','local','ack'))
    prior = row.get('retained_before_restore', {}).get('node_accepted_desired')
    probe = live.get('first_http_probe_after_fresh_observation', {})
    first_row = offers[0].get('generation') if offers else None
    intro = f'AT30 **{status}**; SECOPS-21 **{verdict["requests"]["SECOPS-21"]}**. Restore run `{live.get("run_id")}` on application base `{run.get("application_base")}`. Shared development host measurements; no production SLA.\n\n'
    proof = f'First delivered offer: HTTP `{first.get("http_status")}`, generation `{first.get("generation")}` versus retained `{prior}`, delivered `{first.get("delivered_at")}`; complete write `{first.get("write_complete")}`. '
    proof += f'Body SHA-256 `{first.get("body_sha256")}`; local accepted/applied `{local.get("accepted_generation")}` / `{local.get("applied_generation")}`, digest `{local.get("envelope_sha256")}`. '
    proof += f'Applied ACK HTTP `{ack.get("http_status")}`, generation `{ack.get("applied_generation")}`, report `{ack.get("report_id")}`, joined to the [central receipt and ordering evidence](SECOPS-21.json). Capture armed once; no retry or generation repair.\n\n'
    proof += f'The first separate HTTP **probe** returned `{probe.get("generation")}` at `{probe.get("utc")}`. The first new stored **compiler row** was `{first_row}`. Neither substitutes for the captured delivery. '
    proof += f'D-095 exact HTTP 503 / `pending fresh observation after restore` drift and unchanged generation, row count and envelope fingerprint: `{live.get("D-095",{}).get("passed")}`.\n\n'
    delta_text = '\n'.join(f'{key}: `{d["follow_up_13_seconds"]}` s versus `{d["follow_up_12_baseline_seconds"]}` s; delta `{d["delta_seconds"]}` s.' for key,d in deltas.items()) + '\n\n'
    blocked = 'Full AT29 remains **blocked**: independently clocked disposable guest and manual grant-withdrawal export. Full AT31 remains **blocked**: candidate hardware and 20-member/60-device mixed relay campaign. Follow-up 12 transport sub-cases and bounded relay passes are retained without promotion or rerun. [AT29](AT29.json), [AT31](AT31.json), [unchanged resource references](history/follow-up-12/requests.md).\n\n'
    archive = '[Follow-up 12 archive](history/follow-up-12/archive-index.json): transport `cf81f6f8-4ccf-43d2-aae2-10121f6b169d`, restore `dd845a7f-2660-431c-a8e6-c513108cc07d`, byte-identical to PR #301 (`8b5e809b`) with SHA-256 pins. The follow-up 9 and 10 handover archives are unchanged.\n\n'
    harness = 'Devstack 18 [proof run](../../runs/2026-10-31/followup18/5f74d7c3-6404-461d-9f47-296bd69958c2/first-desired-offer.md) (2/0/0; first 4 > retained 3; HTTP 202 central receipt) validates the export only. This lane uses the separate restore run above for closure. [Export contract](../../RESPONSES.md).\n\n'
    tail = f'Cleanup **{verdict["cleanup"]}**: [container/network/volume/image inventory and disk state removal](cleanup.json). [Run commands](RUN.md).\n'
    (PUBLIC / 'restore-reconcile.md').write_text('# AT30 restore — Follow-up 13\n\n' + intro + delta_text + table + '\n' + proof)
    (PUBLIC / 'AT30.md').write_text((PUBLIC / 'restore-reconcile.md').read_text())
    for name in ('summary.md','README.md'):
        (PUBLIC / name).write_text('# Follow-up 13 — first post-restore delivery\n\n' + intro + proof + delta_text + blocked + archive + tail)
    (PUBLIC / '2026-10-31-follow-up-13-handover.md').write_text('# 2026-10-31 — Follow-up 13 handover\n\n' + intro + proof + delta_text + table + '\n' + blocked + archive + harness + tail)
    requests = PUBLIC / 'requests.md'
    old = '# Requests to orchestrator — Follow-up 13 (2026-10-31 UTC)\n\n' + intro + proof
    if not closed:
        old += 'SECOPS-21 remains open: failed closure guards `' + '`, `'.join(k for k,v in guards.items() if not v) + '`. Preserve the first capture for review; do not re-arm. Owner: w3-wire-nodes / w3-devstack-e2e.\n\n'
    old += blocked + harness + archive
    old += '| Remaining item | Owner / resource | Required action |\n|---|---|---|\n'
    old += '| AT29 guest clock rollback | operator + w3-devstack-e2e | Supply an independently clocked disposable guest; never change the shared host clock. |\n'
    old += '| AT29 manual grant withdrawal | w3-wire-members / contracts | Export an explicit revokeMemberGrant operation; finite expiry is already measured separately. |\n'
    old += '| AT31 full mixed relay capacity | operator + acceptance lane | Supply candidate hardware and a 20-member/60-device mixed-load window. |\n\n'
    old += 'D-102: only this lane writes this request register. Devstack answers in [RESPONSES.md](../../RESPONSES.md). No export-contract discrepancy was observed in this run.\n'
    requests.write_text(old)
    (PUBLIC / 'RUN.md').write_text('''# Follow-up 13 run instructions

```sh
pgrep -af 'gates.sh|mk/at-'
make -f mk/at-secops.mk at-secops-followup13
python3 tests/e2e/at/secops/followup13_records.py
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
python3 tests/e2e/at/secops/diskexec.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/secops/diskexec.py make go-lint go-with-disk-tmp=
python3 tests/e2e/at/secops/followup13_guards.py
git diff --check
```

Wait while two other integration suites are active; record the preflight before
starting. Use only fvpn-w4-secops objects, network fvpn-w4-secops and port base
30000. Foreground runner, disk TMPDIR/GOTMPDIR under .agent/tmp, no shell traps.
The runner cleans its objects and generated bind state even on test failure.

This target runs AT30 only, using its own freshly enrolled primary. It skips
transport, grace and bulk-peer campaigns. Stop the retained agent, arm the shared
first-offer export exactly once before the clean restore, freeze/begin-recovery
while central services are stopped, and prove D-095 through direct-control HTTP.
Resume observations with desired GETs held, wait for the retained applied
generation at central, then release and read FirstDesiredOffer. The separate
HTTP probe runs after capture so it cannot induce compilation before delivery.
A first 304/error/stale generation or missing local/receipt match stays open;
never re-arm to obtain a passing replacement. No generation or SQLite writes.

Record rendering never changes history. Follow-up 12 is SHA-256 pinned and
compared with PR #301; older handover archives remain in place. Re-render after
current document edits to refresh scope pins. Devstack owns RESPONSES.md; this
lane records export discrepancies in requests.md and the orchestrator report.
''')
    refresh_scope()
    print(json.dumps(verdict['cases']), verdict['requests'], 'cleanup='+verdict['cleanup'])


if __name__ == '__main__':
    main()
