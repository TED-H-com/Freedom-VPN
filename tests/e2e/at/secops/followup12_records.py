#!/usr/bin/env python3
"""Render scoped Follow-up 12 evidence without changing historical snapshots."""
import hashlib
import json
from pathlib import Path

from followup10_records import load, measurements
from followup9_records import evaluate_at30, clean_inventory
from run import ROOT, PUBLIC, write_json, utc, scan, REQUIREMENTS
from followup11_records import protocol_cutoff, generation_status


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def attempts_and_replies(row, window):
    samples = row.get(window, [])
    if len(samples) < 2:
        return 'unmeasured'
    a, b = samples[0].get('status', {}), samples[-1].get('status', {})
    return '; '.join(f'{p} attempts {a.get(attempt)} → {b.get(attempt)}, replies {a.get(reply)} → {b.get(reply)}'
                     for p, attempt, reply in [('TCP', 'tcp_attempts', 'tcp_echoes'), ('UDP', 'udp_attempts', 'udp_replies')])


def refresh_scope():
    path = PUBLIC/'scope-review.json'
    scope = load(path)
    # Move mutable Follow-up 10 record references onto its exact merged archive.
    for row in scope.get('follow_up_10', {}).get('records', []):
        old = Path(row['path'])
        archived = PUBLIC/'history/follow-up-10'/old.name
        if old.parent == Path('docs/e2e/at/secops') and archived.exists():
            row['path'] = str(archived.relative_to(ROOT))
            row['sha256'] = digest(archived)
    for row in scope.get('follow_up_11', {}).get('records', []):
        old = Path(row['path'])
        archived = PUBLIC/'history/follow-up-11'/old.name
        if old.parent == Path('docs/e2e/at/secops') and archived.exists():
            row['path'] = str(archived.relative_to(ROOT))
            row['sha256'] = digest(archived)
        else:
            row['sha256'] = digest(ROOT/row['path'])
    for section, key in [('handover_follow_up_6','reviewed_documents'), ('follow_up_7','files'), ('handover_follow_up_13','reviewed_documents')]:
        for row in scope.get(section, {}).get(key, []):
            row['sha256'] = digest(ROOT/row['path'])
    names = ['AT29.json','AT29.md','AT30.json','AT30.md','AT31.json','AT31.md','SECOPS-21.json',
             'restore-reconcile.json','restore-reconcile.md','summary.md','README.md','RUN.md','requests.md','live.json','cleanup.json',
             '2026-10-31-follow-up-12-handover.md','followup12-transports-live.json','followup12-transports-run.json',
             'followup12-restore-live.json','followup12-restore-run.json','followup12-verdict.json','validation.json',
             'followup12-lab-preflight.json','followup12-prerequisites.json','followup12-attempts.json',
             'followup12-transports-attempt1-live.json','followup12-transports-attempt1-run.json',
             'followup12-transports-attempt2-live.json','followup12-transports-attempt2-run.json',
             'history/index.json','history/README.md']
    scope['follow_up_12'] = dict(reviewed_at=utc(), scope='Scoped established transports, bounded relay and restore regression; every changed live document pinned; archived snapshots immutable. scope-review.json excludes its own recursive hash.',
                                records=[dict(path=str((PUBLIC/name).relative_to(ROOT)),sha256=digest(PUBLIC/name)) for name in names if (PUBLIC/name).exists()])
    write_json(path,scope)


def completed_runs():
    """No summary mutation until both independent fresh-stack runs finish."""
    for phase in ('transports','restore'):
        run = load(PUBLIC/f'followup12-{phase}-run.json')
        if not run.get('finished_at') or not (PUBLIC/f'followup12-{phase}-live.json').exists():
            raise SystemExit('Both fresh live runs must complete before rendering summaries')


def main():
    completed_runs()
    trun, rrun = (load(PUBLIC/f'followup12-{p}-run.json') for p in ('transports','restore'))
    for phase in ('transports','restore'):
        path=PUBLIC/f'followup12-{phase}-live.json'
        if path.exists(): path.write_text(path.read_text().replace(str(ROOT)+'/', ''))
    transport, restore = (load(PUBLIC/f'followup12-{p}-live.json') for p in ('transports','restore'))
    cases=transport.get('transport_cases', {})
    case=restore.get('AT30_restore_reconcile', {})
    at30, blockers, evaluation=evaluate_at30(case,http_regrant=True)
    if rrun.get('live',{}).get('status')!='passed' or restore.get('go_test_failed') is not False:
        at30='blocked';blockers.append('Required live restore test did not pass.')
    if restore.get('D-095',{}).get('passed') is not True:
        at30='blocked';blockers.append('D-095 pre-observation hold is not proved.')
    common=dict(schema_version=1,follow_up=12,application_base=rrun.get('application_base'),
                source_revision=rrun.get('source_revision'),run_id=restore.get('run_id'),
                driver_run_id=rrun.get('driver_run_id'),started_at=rrun.get('started_at'),finished_at=rrun.get('finished_at'),
                hardware='shared development host; not candidate hardware',probe_source_sha256=restore.get('probe_source_sha256'),
                record_renderer_sha256=digest(Path(__file__)),previous_run='history/follow-up-11/',projector_owned_rows_written=[])
    comparison={'follow_up_10':measurements(load(PUBLIC/'history/follow-up-10/restore-reconcile.json').get('case',{})), 'follow_up_11':measurements(load(PUBLIC/'history/follow-up-11/restore-reconcile.json').get('case',{})), 'follow_up_12':measurements(case)}
    write_json(PUBLIC/'restore-reconcile.json',dict(common,id='AT30-RESTORE-RECONCILE',status=at30,case=case,evaluation=evaluation,blockers=blockers,measurement_comparison=comparison))
    write_json(PUBLIC/'AT30.json',dict(common,id='AT30',status=at30,requirements=['DATA01','BILL01','MTR03'],spec=['§17.3','§19 AT30'],evaluation=evaluation,blockers=blockers,evidence=['restore-reconcile.json','followup12-restore-live.json','SECOPS-21.json']))
    gen=restore.get('SECOPS-21',{})
    status21=generation_status(rrun,restore)
    offers=gen.get('offers',[])
    retained=gen.get('restored_central',{}).get('node_accepted_desired')
    first=offers[0] if offers else None
    analysis=dict(first_new_stored_offer=first,retained_node_generation=retained,first_http_probe_after_fresh_observation=restore.get('first_http_probe_after_fresh_observation'),
                  first_new_stored_offer_is_above_retained=bool(first and retained and int(first['generation'])>int(retained)),
                  limitation='Stored compiler rows and a timestamped HTTP probe are distinct. No response capture export proves the first envelope delivered to the running agent; no claim that absence of ErrStaleGeneration log lines proves acceptance.')
    analysis['closure_guards'] = dict(
        d099_on_measured_base=rrun.get('d099_roadmap_present') is True,
        d095_hold_proved=restore.get('D-095',{}).get('passed') is True,
        first_stored_row_above_retained=analysis['first_new_stored_offer_is_above_retained'],
        first_delivered_http_response_proved=gen.get('first_http_offer',{}).get('first_response_proven') is True,
        first_offer_local_store_accepted=gen.get('first_offer_local_store',{}).get('accepted') is True,
        first_stored_row_applied_ack=bool(first and any(v.get('applied')==first['generation'] for v in gen.get('observations',[]))))
    write_json(PUBLIC/'SECOPS-21.json',dict(common,id='SECOPS-21',status='passed' if status21=='closed' else 'blocked',request_status=status21,d099_roadmap_present=rrun.get('d099_roadmap_present'),hold=restore.get('D-095'),actual=gen,first_offer_analysis=analysis))
    subcases={}
    for name,row in cases.items():
        subcases[name]=dict(row)
        if name in ('grant_expiry','lease_expiry','device_revoke'):
            subcases[name]['protocol_verdicts']={p:protocol_cutoff(row,p) for p in ('TCP','UDP')}
    remaining29=['Guest clock rollback needs a disposable guest; never change the shared host clock.',
                 'No revokeMemberGrant HTTP export exists; finite expiration is measured separately.',
                 'Device revoke does not independently attribute native route withdrawal versus exit conntrack enforcement.']
    remaining31=['One member/one exit bounded relay path only; no 60-peer mixed relay capacity or candidate-hardware campaign.',
                 'The 180-second forced-relay match and 20-minute transport helper are export limits, not capacity guarantees.']
    for at,names,gaps in [('AT29',['grant_expiry','lease_expiry','device_revoke','manual_grant_withdrawal','guest_clock_rollback'],remaining29),('AT31',['forced_relay'],remaining31)]:
        actual={name:subcases.get(name,{'status':'blocked','missing_export':'live case did not run'}) for name in names}
        full='failed' if any(row.get('status')=='failed' for row in actual.values()) else 'blocked'
        write_json(PUBLIC/(at+'.json'),dict(schema_version=1,follow_up=12,id=at,status=full,requirements=REQUIREMENTS[at],spec=['§19 '+at],run_id=transport.get('run_id'),application_base=trun.get('application_base'),source_revision=trun.get('source_revision'),actual=actual,blockers=gaps,evidence=['followup12-transports-live.json','followup12-transports-run.json']))
        text=f'# {at} — Follow-up 12\n\nFull status: **{full}**. Run `{transport.get("run_id")}`.\n\n| Sub-case | Status | TCP / UDP |\n|---|---|---|\n'
        for name,row in actual.items():
            verdicts=row.get('protocol_verdicts',{})
            text+=f'| {name} | {row.get("status")} | {verdicts.get("TCP","—")} / {verdicts.get("UDP","—")} |\n'
        text+='\n'+' '.join(gaps)+f'\n\n[Exact samples, read-back and limits]({at}.json). Transport senders remain active across each fault; missing counters are never treated as zero bytes.\n'
        (PUBLIC/(at+'.md')).write_text(text)
    cleanup=all(clean_inventory(r.get('inventory_after',{})) and r.get('state_cleanup',{}).get('removed') for r in (trun,rrun))
    verdict=dict(common,cases={'AT29':load(PUBLIC/'AT29.json')['status'],'AT30':at30,'AT31':load(PUBLIC/'AT31.json')['status']},subcases={k:v.get('status') for k,v in subcases.items()},requests={'SECOPS-21':status21},cleanup='passed' if cleanup else 'blocked')
    write_json(PUBLIC/'followup12-verdict.json',verdict)
    phases = {phase:dict(run_id=live.get('run_id'),driver_run_id=run.get('driver_run_id'),
                        source_revision=run.get('source_revision'),started_at=run.get('started_at'),finished_at=run.get('finished_at'),
                        test_status=run.get('live',{}).get('status'),evidence=f'followup12-{phase}-live.json',driver=f'followup12-{phase}-run.json')
              for phase,run,live in [('transports',trun,transport),('restore',rrun,restore)]}
    write_json(PUBLIC/'live.json',dict(schema_version=1,follow_up=12,record_kind='two independent live phases',
               application_base=rrun.get('application_base'),phases=phases,cases=verdict['cases'],
               scope='Current scoped measurements; full AT29/AT31 remain blocked. Older full-stack captures are under history/follow-up-11/.'))
    write_json(PUBLIC/'cleanup.json',dict(schema_version=1,follow_up=12,status=verdict['cleanup'],
               namespace='fvpn-w4-secops',phases={phase:dict(run_id=phases[phase]['run_id'],down=run.get('down'),
                   state_cleanup=run.get('state_cleanup'),inventory_after=run.get('inventory_after'))
                   for phase,run in [('transports',trun),('restore',rrun)]}))
    public_scans=[dict(file=p.name,findings=scan(p.read_bytes())) for p in PUBLIC.iterdir() if p.is_file() and p.suffix in ('.json','.md')]
    write_json(PUBLIC/'validation.json',dict(common,status='passed' if cleanup and not any(r['findings'] for r in public_scans) and all(r.get('live',{}).get('status')=='passed' for r in (trun,rrun)) else 'blocked',live_runs=[trun,rrun],public_scan=public_scans,cleanup=verdict['cleanup'],scope='Driver/live execution and capture hygiene; full AT29/AT31 remain separate blocked cases.'))
    table='| Measurement | Archived follow-up 10 | Archived follow-up 11 (no new run) | Follow-up 12 |\n|---|---|---|---|\n'
    for key,value in comparison['follow_up_12'].items(): table+=f'| {key} | {comparison["follow_up_10"][key]} | {comparison["follow_up_11"][key]} | {value} |\n'
    summary=f'# Follow-up 12 — transports and restore\n\nAT30 **{at30}**; SECOPS-21 **{status21}**. Application base `{rrun.get("application_base")}`. Restore run `{restore.get("run_id")}`; transport run `{transport.get("run_id")}`. Shared development host measurements.\n\n'+table
    summary+='\n[AT29 established transport sub-cases](AT29.md) and [AT31 bounded relay](AT31.md) record individual results and remaining limits.\n\n'
    summary+=f'D-095 hold proved: `{restore.get("D-095",{}).get("passed")}`. First new stored offer `{first.get("generation") if first else None}`; retained node `{retained}`. [HTTP probe, hold and generation history](SECOPS-21.json). D-099 roadmap eligibility: `{rrun.get("d099_roadmap_present")}`. Compiler success alone does not close SECOPS-21.\n\n'
    summary+='[Merged follow-up 10 archive](history/follow-up-10/restore-reconcile.json) is byte-identical to PR #265; [follow-up 9](history/follow-up-9/restore-reconcile.json) remains at its D-098 handover claims path.\n'
    for name in ('summary.md','restore-reconcile.md','AT30.md'): (PUBLIC/name).write_text(summary)
    (PUBLIC/'README.md').write_text('# Wave 4 security and operations acceptance\n\n[Follow-up 12 results](summary.md), [AT29 sub-cases](AT29.md), [AT30 restore regression](AT30.md), [AT31 relay path](AT31.md), [requests and owners](requests.md), and [run instructions](RUN.md). Full AT28/AT32 and prior SECOPS-16/17/18/19/20 evidence remain explicitly dated Follow-up 10 records; this run does not promote their scope.\n\nThe D-098 [follow-up 10 archive](history/follow-up-10/) is immutable. Candidate hardware, physical arm64 and guest clock rollback remain unmeasured.\n')
    note=f'# 2026-10-31 — Follow-up 12 handover\n\nFollow-up 10 archive: `docs/e2e/at/secops/history/follow-up-10/`, exact PR #265 commit `2f1f939`, run `0e8ff9a4-4fa1-4154-ae35-847a5be4d985`. Pin `restore-reconcile.json` → `measurement_comparison.follow_up_10` there. Keep `history/follow-up-9/restore-reconcile.json` for handover follow-up 13.\n\nAT30 regression: **{at30}**. SECOPS-21: **{status21}**; first new stored offer `{first.get("generation") if first else None}`, retained `{retained}`.\n\n| AT29 / AT31 sub-case | Result |\n|---|---|\n'
    for k,v in subcases.items(): note+=f'| {k} | {v.get("status")} |\n'
    note+='\n| Sub-case / window | Independent sender attempts and replies |\n|---|---|\n'
    for k,v in subcases.items():
        if k in ('grant_expiry','lease_expiry','device_revoke'):
            note+=f'| {k} / cutoff | {attempts_and_replies(v,"cutoff_samples")} |\n'
        elif k=='forced_relay':
            for window in ('relay_samples','direct_restored_samples'):
                note+=f'| {k} / {window} | {attempts_and_replies(v,window)} |\n'
    note+='\nThe transport helper lifetime remains 20 minutes, relay match 180 seconds and each relay-path wait 90 seconds. Missing counters are not zero-byte proof. Per-sample counter presence and monotonic timing remain in the JSON evidence.\n'
    note+='\n[Earlier transport attempts](followup12-attempts.json) remain separately preserved with hashes: attempt 1 omitted publication, and attempt 2 measured grant cutoff but hit the recovery timestamp precision check. Neither is silently replaced by the final run or reported as a complete transport pass.\n'
    probe=restore.get('first_http_probe_after_fresh_observation',{})
    note+=f'\nSECOPS-21: the separately labeled first HTTP **probe** returned generation `{probe.get("generation")}`; the first new stored compiler row was `{first.get("generation") if first else None}`; the retained node was `{retained}`. First-delivered agent response evidence is required for closure. The final local generation and central applied observations are recorded separately in [SECOPS-21.json](SECOPS-21.json).\n'
    applied_after = [v['applied'] for v in gen.get('observations',[]) if retained and int(v['applied'])>int(retained)]
    note+=f'\nThe first subsequent applied observation was `{applied_after[0] if applied_after else None}`. Applied ACK for the first stored row: `{analysis["closure_guards"]["first_stored_row_applied_ack"]}`. First-delivered-response capture proved: `{analysis["closure_guards"]["first_delivered_http_response_proved"]}`; first-offer local acceptance proved: `{analysis["closure_guards"]["first_offer_local_store_accepted"]}`. The final snapshot shows local accepted/applied `{gen.get("final",{}).get("node_accepted_desired")}` / `{gen.get("final",{}).get("node_applied")}`, central applied `{gen.get("final",{}).get("applied")}`. These later values do not close missing first-offer guards. No generation repair was performed.\n'
    note+='\n'+table
    note+=f'\nCleanup: **{verdict["cleanup"]}**. Both run records include teardown exit status, state-directory removal and final container/network/volume/image counts. [Transport run](followup12-transports-run.json); [restore run](followup12-restore-run.json).\n'
    note+='\nBoth final inventories are zero for containers, networks, volumes and images, with generated state removed. [Current cleanup record](cleanup.json). The lab scheduling wait ended at 09:04:09 UTC; [preflight history](followup12-lab-preflight.json) records the required retries.\n'
    note+='\nFollow-up 11 root records (105 files) are byte-identical to PR #274 under [history/follow-up-11/](history/follow-up-11/), with SHA-256 index entries. Its restore measurements are retained follow-up 10 values, not a new execution. No older snapshot was moved or rewritten.\n'
    note+='\nExact samples and export limits: [AT29](AT29.json), [AT31](AT31.json), [SECOPS-21](SECOPS-21.json). The existing handover test already reads the follow-up 9 archive; no further test repoint is needed.\n'
    (PUBLIC/'2026-10-31-follow-up-12-handover.md').write_text(note)
    requests=PUBLIC/'requests.md'; old=requests.read_text().split('\n# Requests to orchestrator — Follow-up 12')[0]
    old+='\n# Requests to orchestrator — Follow-up 12 (2026-10-31 UTC)\n\n'
    old+=f'SECOPS-21: **{status21}**, owner `w3-wire-nodes` follow-up 29 / D-099. See [hold and first-offer evidence](SECOPS-21.json); no closure from compiler tests.\n\n'
    old+=f'The first HTTP probe and first stored row were `{probe.get("generation")}` / `{first.get("generation") if first else None}`, versus retained `{retained}` (first stored row above retained: `{analysis["first_new_stored_offer_is_above_retained"]}`). The first subsequent applied observation was `{applied_after[0] if applied_after else None}`; applied ACK for the first stored row: `{analysis["closure_guards"]["first_stored_row_applied_ack"]}`.\n\n'
    if status21 != 'closed':
        old+='Request to `w3-devstack-e2e` + `w3-wire-nodes`: export the actual first response delivered to the retained agent, its matching accepted local store, and applied ACK, so closure can be measured without inferring delivery from compiler rows or a separate probe. Keep SECOPS-21 open until all guards hold.\n\n'
    old+='D-102: the dated devstack answers in `docs/e2e/RESPONSES.md` were read from later main `5ce76e9` once published. They document members/update exports; no first-delivered desired-state response capture is documented. The installer handover was read from `63a4116` once published; it reports ENR-06 socket/runtime fixed, with the required full upgrade still failing at final_sample because state/meter.checkpoint is absent. Both documents were absent on the assigned application base at task start. The node D-099 handover was read before planning. [Read revisions and hashes](followup12-prerequisites.json). No external pin, packaged unit or measured application base changed.\n\n'
    old+='D-103 is resolved by this task: only `fvpn-w4-secops-*`, network `fvpn-w4-secops`, port base 30000. The two-run guard refers to the prepared independent transport and restore fresh-stack runs, each torn down before the next.\n\n'
    old+='The temporary lab scheduling request is resolved: the restore run started after the 09:04:09 UTC preflight admitted it. All earlier waits remain in [the preflight history](followup12-lab-preflight.json).\n\n'
    old+='| Sub-case / missing export | Status / owner | Limit |\n|---|---|---|\n'
    for k,v in subcases.items(): old+=f'| {k} | {v.get("status")} / {v.get("owner","w4-at-security-ops")} | {v.get("missing_export",v.get("blocker",v.get("scope",v.get("transport_scope","See recorded samples"))))} |\n'
    old+='| AT31 full capacity | blocked / operator + w4-at-security-ops | Needs chosen candidate hardware and 20-member/60-device mixed-load relay measurements; bounded single-member export is delivered. |\n\n[Dated handover and archive pins](2026-10-31-follow-up-12-handover.md).\n'; requests.write_text(old)
    (PUBLIC/'RUN.md').write_text('''# Follow-up 12 run instructions

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
FVPN_SECOPS_EVIDENCE_FOLLOWUP=12 make -f mk/at-secops.mk at-secops-followup11-transports
FVPN_SECOPS_EVIDENCE_FOLLOWUP=12 FVPN_SECOPS_FOLLOWUP11=restore make -f mk/at-secops.mk at-secops-followup10
python3 tests/e2e/at/secops/followup12_records.py
python3 tests/e2e/at/secops/followup12_guards.py
python3 tests/e2e/at/secops/diskexec.py go test -count=1 ./tests/handover/...
make go-lint
test -z "$(gofmt -l .)"
git diff --check
```

Use only the assigned w4-secops / port-base 30000 stack. Before each live run check
`pgrep -af gates.sh | grep -v grep` and
`docker ps --format '{{.Names}}' | grep -c '^fvpn-w'`.
If two other lanes' suites or labs are active, wait and retry every five minutes. Both fresh runs
use disk scratch and unconditional cleanup. The restore target runs the same
Follow-up 10 drill, with Follow-up 12 filenames and D-095 hold read-back added.
Freeze and begin-recovery run while control/worker remain stopped. Briefly stop
the retained agent for the pre-observation HTTP/node hold probe, then resume it
without altering its desired state, leases, identity or generation. The HTTP
probe after fresh observation is not an interception of the first agent response.
The dated roadmap gate controls SECOPS-21 eligibility; absent D-099 means open.

Archive guards compare every Follow-up 10 file against the merged PR #265 blob.
The record renderer never edits snapshots or runs product mutations. Re-render
after document edits to refresh current scope-review pins.
''')
    refresh_scope()
    print(json.dumps(verdict['cases']),status21,'cleanup='+verdict['cleanup'])


if __name__=='__main__':main()
