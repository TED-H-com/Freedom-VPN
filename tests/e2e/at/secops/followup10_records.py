#!/usr/bin/env python3
"""Fail-closed Follow-up 10 records; preceding evidence stays immutable."""
import datetime as dt
import hashlib
import json
from pathlib import Path

from followup9_records import evaluate_at30, clean_inventory, _ledger_counts
from run import ROOT, PUBLIC, scan, write_json, REQUIREMENTS, EXPECTED


def load(path):
    return json.loads(path.read_text()) if path.exists() else {}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def generation_verdict(row):
    offers = row.get('offers', [])
    observations = row.get('observations', [])
    retained = int(row.get('restored_central', {}).get('node_accepted_desired') or row.get('retained_at_t1', {}).get('node_applied') or row.get('retained_at_t1', {}).get('applied', '0'))
    first = int(offers[0]['generation']) if offers else 0
    return bool(retained and first > retained and any(int(v['applied']) == first for v in observations)
                and row.get('manual_generation_adjustment') is False
                and row.get('stale_generation_log_count') == 0)


def protected_verdict(rows, bulk):
    streams = bulk.get('counts', {}).get('streams', [])
    if len(streams) != 60 or bulk.get('error') is not False or len(rows) < 60:
        return False
    names = [r.get('pre_delete', {}).get('stream') for r in rows]
    if len(set(names)) != len(names) or not {r['stream'] for r in streams}.issubset(names):
        return False
    return all(r.get('pre_delete', {}).get('result') == 'scanned'
               and r['pre_delete'].get('deleted') is True
               and r['pre_delete'].get('scan', {}).get('present') is True
               and r['pre_delete']['scan'].get('mode') == '0:600'
               and r.get('post_delete_scan_ok') is True
               and r.get('post_delete_present') is False for r in rows)


def grace_verdict(row):
    grace, final = row.get('grace_projection', {}), row.get('after_reconcile_projection', {})
    samples = row.get('after_deadline_samples', [])
    if not row.get('passed') or not samples:
        return False
    expected = dt.datetime.fromisoformat(row['expected_grace_until'].replace('Z', '+00:00'))
    actual = dt.datetime.fromisoformat(grace['grace_until'].replace('Z', '+00:00'))
    return (actual == expected and final.get('grace_until') == grace.get('grace_until')
            and final.get('paid_invoice_count') == grace.get('paid_invoice_count') == 1
            and final.get('state') == 'expired'
            and row.get('member_me_grace_status') == row.get('admin_member_grace_status') == 200
            and row.get('member_me_expired_status') == row.get('admin_member_expired_status') == 200
            and all(s.get('http_200') is False and s.get('counter_reads_ok') is True for s in samples))


def measurements(case):
    s1, s2 = case.get('step1_checkpoint_t0', {}), case.get('step2_world_forward_t1', {})
    s3, s4 = case.get('step3_restore_central_to_t0', {}), case.get('step4_reconciliation_observation', {})
    u, proto = s4.get('usage_replay', {}), s3.get('product_restore_protocol', {})
    v = s3.get('secops19_visibility', {})
    hold = v.get('during_restore_hold', {}).get('node_observation_report_usage', {}).get('usage', {})
    zero = v.get('after_complete_recovery_and_replay', {}).get('node_observation_report_usage', {}).get('usage', {})
    return {
        'T0 / backup': f"{s1.get('t0_utc')} / {s1.get('pgbackrest_backup', {}).get('backup_label')}",
        'L0 → L1 sequence': f"{u.get('n0_seq_at_t0')} → {u.get('n1_seq_at_t1')}",
        'L0 → L1 bytes': f"{_ledger_counts(s1.get('ledger_l0', {}))[2]} → {_ledger_counts(s2.get('ledger_l1', {}))[2]}",
        'Primary-overwrite refusal / checklist / PITR seconds': f"{s3.get('primary_overwrite_refusal', {}).get('exit_code')} / {proto.get('restore_info', {}).get('checklist_sections')}/6 / {proto.get('restore_info', {}).get('pitr_seconds')}",
        'Negative restore window (s) / ACK / max seen': f"{s3.get('fail_safe_negative_without_protocol', {}).get('window_seconds')} / {s3.get('fail_safe_negative_without_protocol', {}).get('after_120s_central_ack')} / {s3.get('fail_safe_negative_without_protocol', {}).get('after_120s_central_max_seen')}",
        'Pre-replay database RPO (s)': u.get('pre_replay_db_rpo_seconds'),
        'Post-replay service usage RPO (s)': u.get('post_replay_service_rpo_seconds'),
        'Protocol RTO (s)': proto.get('protocol_rto_seconds'),
        'Total outage RTO (s)': proto.get('total_outage_rto_seconds'),
        'Replay events / hashes / lost / duplicate / gaps': f"{u.get('l2_events_1_to_n1')} / {u.get('canonical_hash_matches_1_to_n1')} / {u.get('lost_t0_t1_acked_events')} / {u.get('double_counted_events')} / {u.get('usage_ledger_gaps')}",
        'Provider reconciliation': s4.get('provider_reconciliation', {}).get('status'),
        'SECOPS-19 hold unacked / bytes → replay unacked / bytes': f"{hold.get('unacked_event_count')} / {hold.get('unacked_payload_bytes')} → {zero.get('unacked_event_count')} / {zero.get('unacked_payload_bytes')}",
        'SECOPS-19 CLI hold unacked / bytes → replay unacked / bytes': f"{v.get('during_restore_hold', {}).get('fvpn_agent_usage_snapshot_cli', {}).get('snapshot', {}).get('unacked_count')} / {v.get('during_restore_hold', {}).get('fvpn_agent_usage_snapshot_cli', {}).get('snapshot', {}).get('unacked_payload_bytes')} → {v.get('after_complete_recovery_and_replay', {}).get('fvpn_agent_usage_snapshot_cli', {}).get('snapshot', {}).get('unacked_count')} / {v.get('after_complete_recovery_and_replay', {}).get('fvpn_agent_usage_snapshot_cli', {}).get('snapshot', {}).get('unacked_payload_bytes')}",
        'Central revoke / fresh post-restore enrollment negatives': f"{s4.get('peer_revocation_reconciliation', {}).get('negative_central_revoke_not_fenced', {}).get('passed')} / {s4.get('peer_revocation_reconciliation', {}).get('negative_post_restore_fresh_enroll_builds', {}).get('passed')}",
        'D-076 fence / regrant': s4.get('peer_revocation_reconciliation', {}).get('status'),
    }


def main():
    run = load(PUBLIC / 'followup10-run.json')
    live_path = PUBLIC / 'followup10-live.json'
    if live_path.exists():
        # Commands record repository-relative paths; no home paths in public evidence.
        live_path.write_text(live_path.read_text().replace(str(ROOT) + '/', ''))
    live = load(live_path)
    case = live.get('AT30_restore_reconcile', {})
    previous = load(PUBLIC / 'history/follow-up-9/restore-reconcile.json').get('case', {})
    status, blockers, evaluation = evaluate_at30(case, http_regrant=True)
    if run.get('live', {}).get('status') != 'passed' or live.get('go_test_failed') is not False:
        status = 'blocked'
        blockers.append('The required live test did not pass; partial evidence cannot close AT30.')
    common = dict(schema_version=1, follow_up=10, source_revision=run.get('source_revision'),
                  application_base='d9e28f8', run_id=live.get('run_id'), driver_run_id=run.get('driver_run_id'),
                  started_at=run.get('started_at'), finished_at=run.get('finished_at'),
                  hardware='shared development host; not candidate hardware',
                  probe_source_sha256=live.get('probe_source_sha256'),
                  record_renderer_sha256=digest(Path(__file__)),
                  commands=[run.get(k, {}).get('command') for k in ('up', 'live', 'down')],
                  image_digests=live.get('image_digests', []),
                  build_digests=[{'name':f.name,'sha256':digest(f)} for f in sorted((ROOT/'.agent/secops/build').glob('*.test'))],
                  policy_revision=live.get('policy_revision'), agent_node_incarnation=live.get('agent_node_incarnation'),
                  previous_run='history/follow-up-9/', projector_owned_rows_written=[])
    written = []
    def write(name, value):
        write_json(PUBLIC / name, value); written.append(name)
    def md(name, value):
        (PUBLIC / name).write_text(value); written.append(name)
    comparison = {'follow_up_9': measurements(previous), 'follow_up_10': measurements(case)}
    write('restore-reconcile.json', dict(common, id='AT30-RESTORE-RECONCILE', status=status, evaluation=evaluation,
          blockers=blockers, case=case, measurement_comparison=comparison))
    write('AT30.json', dict(common, id='AT30', status=status, requirements=['DATA01', 'BILL01', 'MTR03'],
          spec=['§17.3', '§19 AT30'], evaluation=evaluation, blockers=blockers,
          evidence=['restore-reconcile.json', 'D-077.json', 'SECOPS-19.json', 'SECOPS-20.json', 'SECOPS-21.json']))
    s3 = case.get('step3_restore_central_to_t0', {})
    s4 = case.get('step4_reconciliation_observation', {})
    peers = s4.get('peer_revocation_reconciliation', {})
    reg = live.get('D-077', {})
    calls = reg.get('calls', []) + [peers.get('negative_central_revoke_not_fenced', {}).get('admin_authorize_revoked', {})]
    write('D-077.json', dict(common, id='D-077', status='passed' if evaluation.get('regrant_authorize_device_clears_fence') else 'blocked', actual=reg, call_table=calls))
    generation = live.get('SECOPS-21', {})
    if generation.get('offers'):
        first = generation['offers'][0]
        retained = (generation.get('restored_central', {}).get('node_accepted_desired') or generation.get('retained_at_t1', {}).get('node_applied') or generation.get('retained_at_t1', {}).get('applied'))
        generation['first_offer_analysis'] = {
            'generation': first['generation'], 'signed_at': first['signed_at'],
            'retained_generation_before_first_offer': retained,
            'node_applied_before_restart': generation.get('restored_central', {}).get('node_applied'),
            'central_observed_at_t1': generation.get('retained_at_t1', {}).get('applied'),
            'below_retained_generation': int(first['generation']) < int(retained or 0),
            'first_post_restore_observation': next(iter(generation.get('observations', [])), None),
            'interpretation': 'A lower first offer is stale by agentstore.AcceptDesired. The agent silently ignores ErrStaleGeneration, so rejection is inferred from the generation comparison, not claimed as a logged event.'}
    prior_generation = load(PUBLIC / 'followup10-attempt1-live.json').get('SECOPS-21', {})
    prior_stale = bool(prior_generation.get('offers') and
        int(prior_generation['offers'][0]['generation']) < int(prior_generation.get('retained_at_t1', {}).get('applied', '0')))
    if prior_stale:
        generation['prior_attempt_first_offer'] = prior_generation['offers'][0]
        generation['prior_attempt_retained_at_t1'] = prior_generation.get('retained_at_t1')
        generation['prior_attempt_evidence'] = 'followup10-attempt1-live.json'
        generation['repeat_pass_cannot_erase_startup_race'] = True
    findings = {}
    for number, ok, actual in [
        (17, grace_verdict(live.get('SECOPS-17', {})), live.get('SECOPS-17', {})),
        (18, evaluation.get('t0_t1_device_revocation_fenced_d076'), peers),
        (19, evaluation.get('secops19_visibility_and_zero_convergence'), s3.get('secops19_visibility', {})),
        (20, live.get('SECOPS-20', {}).get('passed') is True, live.get('SECOPS-20', {})),
        (21, generation_verdict(live.get('SECOPS-21', {})) and not prior_stale, live.get('SECOPS-21', {})),
    ]:
        name = f'SECOPS-{number}'
        findings[name] = 'closed' if ok else 'open'
        write(name+'.json', dict(common, id=name, status='passed' if ok else 'blocked', request_status=findings[name], actual=actual))
    ordinary = live.get('secret_scan', [])
    public_scan = [{'stream':p.name, 'bytes':p.stat().st_size, 'findings':scan(p.read_bytes())} for p in PUBLIC.iterdir() if p.is_file() and p.suffix in ('.json', '.md')]
    driver_scans = [run.get(k, {}).get('scan', {}) for k in ('up','live','down')]
    protected = live.get('protected_native_streams', [])
    bulk = live.get('bulk_native_streams', {})
    ordinary_ok = bool(ordinary) and all(r.get('read_ok') is True and r.get('known_secret_matches') == 0 and r.get('device_flow_key_lines') == 0 for r in ordinary)
    capture_ok = bool(public_scan) and all(not r['findings'] for r in public_scan) and all(r and not r.get('findings') for r in driver_scans)
    lifecycle_ok = protected_verdict(protected, bulk)
    scan_ok = ordinary_ok and capture_ok and lifecycle_ok
    findings['SECOPS-16'] = 'closed' if scan_ok else 'open'
    scan_record = dict(common, id='SECOPS-16', status='passed' if scan_ok else 'blocked', request_status=findings['SECOPS-16'],
                       criterion='D-066: zero secrets or device-flow values in ordinary retained logs/captures/readable log files; root-owned 0600 transient login credentials are a separate class with final scan and deletion proof',
                       ordinary_streams=ordinary, known_secret_canaries=live.get('known_secret_canaries'),
                       public_capture_scan=public_scan, driver_capture_scan=driver_scans,
                       protected_streams=protected, bulk_final_attribution=bulk,
                       ordinary_passed=ordinary_ok, captures_passed=capture_ok, protected_lifecycle_passed=lifecycle_ok,
                       limitation='Covers this measured stack, retained log files and evidence captures. Configuration/key stores are intended credential stores, not ordinary logs. No claim about every possible logging path; no matched values published.')
    write('SECOPS-16.json', scan_record); write('secret-scan.json', scan_record)
    # Refresh remaining blockers without promoting historical full AT cases.
    remaining = {
        'AT28': ['Full resource-family ownership matrix is beyond this scoped live restore/scan run.'],
        'AT29': ['Established TCP/UDP transport cases await devstack follow-up 15; disposable-guest clock rollback remains an operator run.'],
        'AT31': ['Forced-relay mixed transport awaits devstack follow-up 15; candidate hardware remains unmeasured. The 60-peer run here proves stream attribution only.'],
        'AT32': ['Physical three-origin, physical arm64 and member-platform runs remain operator work; no production capacity or SLA claim.'],
    }
    if not scan_ok:
        remaining['AT28'].append('SECOPS-16 remains open; see the current scan record.')
    if findings.get('SECOPS-17') != 'closed':
        remaining['AT29'].append('SECOPS-17 finite grace remains open; see the current provider record.')
    current_evidence = {'AT28':['SECOPS-16.json'], 'AT29':['SECOPS-17.json'],
                        'AT31':['SECOPS-16.json'], 'AT32':['scope-review.json']}
    for at, gaps in remaining.items():
        write(at+'.json', dict(common, id=at, status='blocked', requirements=REQUIREMENTS[at],
              expected=EXPECTED[at], blockers=gaps, full_case_rerun=False,
              actual={'current_scoped_evidence':current_evidence[at], 'historical_full_record':'history/follow-up-9/'+at+'.json'}))
        md(at+'.md', '# '+at+' — Follow-up 10 scope review\n\nFull status: **blocked**; full case not rerun.\n\n' + ' '.join(gaps) +
           '\n\nCurrent scoped evidence: ' + ', '.join('['+name+']('+name+')' for name in current_evidence[at]) +
           '. Prior case and measurements remain in [Follow-up 9 history](history/follow-up-9/'+at+'.json).\n')
    cleanup_ok = clean_inventory(run.get('inventory_after', {})) and run.get('state_cleanup', {}).get('removed') is True
    write('cleanup.json', dict(common, status='passed' if cleanup_ok else 'blocked', inventory=run.get('inventory_after', {}), state_cleanup=run.get('state_cleanup', {})))
    cases = {k: 'blocked' for k in ['AT28','AT29','AT30','AT31','AT32']};cases['AT30']=status
    write('followup10-verdict.json', dict(common, cases=cases, at30_evaluation=evaluation, requests=findings, cleanup='passed' if cleanup_ok else 'blocked'))
    write('live.json', dict(common, status=status, evidence='followup10-live.json', cases=cases))
    table = '| Measurement | Follow-up 9 (#242) | Follow-up 10 |\n|---|---|---|\n'
    for key, value in comparison['follow_up_10'].items():
        table += f"| {key} | {comparison['follow_up_9'][key]} | {value} |\n"
    call_table = '| Request | HTTP / error | Operation / desired generation | Fence | Flag |\n|---|---|---|---|---|\n'
    for call in calls:
        op, device = call.get('operation', {}), call.get('device', {})
        call_table += f"| {call.get('request')} | {call.get('status')} {call.get('error','')} | {op.get('operation_id','—')} / {op.get('desired_generation','—')} | {call.get('fence_count')} | {device.get('authorization_unconfirmed',call.get('authorization_unconfirmed','—'))} |\n"
    summary = f"# Follow-up 10 — restore and security/operations evidence\n\nAT30: **{status}**. Run `{live.get('run_id')}`; application base `d9e28f8` includes #240, #241, #244 and #245. {run.get('started_at')} → {run.get('finished_at')}. Shared development host, not candidate hardware.\n\n" + table
    summary += '\nD-077 uses an admin session, CSRF, Origin, a fresh UUID key and no body. A still-authorized fenced device returns its existing `member.device_authorize` operation with the original desired generation. Completion is read from `GET /admin/devices` after worker sync.\n\n' + call_table
    summary += '\n' + '; '.join(f'{k}: **{v}**' for k,v in sorted(findings.items())) + '.\n\n'
    e20 = live.get('SECOPS-20', {})
    summary += f"SECOPS-20: `ip_nat_present={e20.get('ip_nat_present')}`, `member_forwarding_enabled={e20.get('product_observation', {}).get('member_forwarding_enabled')}`, `loopback_scoped={len(e20.get('loopback_scoped', []))}` entries. #240 deletes the lab NAT before agent startup and explains this serving transition. #241 is present; the empty loopback-scoped list does not exercise its exception. Follow-up 9’s before capture is unchanged in history.\n\n"
    summary += 'SECOPS-21: compare all central offers and applied observations in [SECOPS-21.json](SECOPS-21.json). The agent silently ignores `ErrStaleGeneration`; zero matching log lines alone is insufficient. No harness generation adjustment is made.\n\n'
    if generation.get('first_offer_analysis'):
        g = generation['first_offer_analysis']
        summary += f"First post-restore offer: `{g['generation']}` at `{g['signed_at']}`; retained node generation before central services resumed: `{g['retained_generation_before_first_offer']}` (node applied `{g['node_applied_before_restart']}`, central T1 observation `{g['central_observed_at_t1']}`). Below retained: `{g['below_retained_generation']}`. When below retained, the remaining issue is startup ordering: compilation ran before fresh retained-generation observation. #245 can only use the applied generation already stored centrally.\n\n"
    summary += 'AT29 established TCP/UDP and AT31 forced relay remain deferred to devstack follow-up 15. The bulk native run is only SECOPS-16 stream attribution. Physical arm64, candidate hardware and guest clock rollback remain operator work. Portal member/admin API state is measured for SECOPS-17; browser rendering is not exercised.\n\n'
    summary += 'Structured measurements: [restore-reconcile.json](restore-reconcile.json), [D-077.json](D-077.json), [secret-scan.json](secret-scan.json), [SECOPS-17.json](SECOPS-17.json).\n'
    if blockers: summary += '\nBlockers: ' + ' '.join(blockers) + '\n'
    md('summary.md',summary);md('restore-reconcile.md',summary);md('AT30.md',summary)
    md('README.md', f"# Wave 4 security and operations acceptance\n\n[Follow-up 10](summary.md) measures the local lab on application base `d9e28f8` (#240/#241/#244/#245), with live harness `{run.get('source_revision')}`. Shared development host; no candidate-hardware or production SLA claim.\n\n"
       + '| Case | Full status | Scope |\n|---|---|---|\n'
       + f'| [AT30](AT30.json) | **{status}** | Physical T0 restore, retained-node replay, provider/device reconciliation, D-077 HTTP re-grant and measured RPO/RTO. |\n'
       + '| [AT28](AT28.json) | blocked | D-066 scan measured; full ownership matrix beyond this run. |\n'
       + '| [AT29](AT29.json) | blocked | Finite past_due grace measured; established transports await devstack follow-up 15, guest clock rollback remains an operator run. |\n'
       + '| [AT31](AT31.json) | blocked | The bulk cohort proves final stream attribution; forced relay awaits devstack follow-up 15 and candidate hardware remains unmeasured. |\n'
       + '| [AT32](AT32.json) | blocked | Physical origins, arm64 and member platforms remain operator work. |\n\n'
       + '[SECOPS-16](SECOPS-16.json) applies D-066: ordinary logs, captures and readable log files must be clean; root-owned 0600 pre-delete login credentials are a separate class with final scan and deletion proof. Protected key-name counts alone are not leaks under this criterion.\n\n'
       + '[SECOPS-20](SECOPS-20.json) records the fresh serving gate and NAT read-back. [SECOPS-21](SECOPS-21.json) records the first post-restore generation, node-local retained state, observations and any remaining startup ordering issue. Prior failed observations are retained.\n\n'
       + 'Use [RUN.md](RUN.md), [requests.md](requests.md), [validation.json](validation.json) and [scope-review.json](scope-review.json). The current target requires AT30, SECOPS-16/17/20 and cleanup; SECOPS-21 remains explicitly open when its first-offer proof fails. [Follow-up 9 history](history/follow-up-9/archive-index.json) is byte-identical; [diagnostic attempt](followup10-attempts.json) records the finalized-run preparation.\n')

    md('secret-scan.md',f"# SECOPS-16 — Follow-up 10\n\nStatus: **{scan_record['status']}**. {len(ordinary)} ordinary streams, {len(protected)} final protected-stream records (60 bulk), {len(public_scan)} public records and three driver captures. Ordinary: {ordinary_ok}; captures: {capture_ok}; protected lifecycle: {lifecycle_ok}.\n\n{scan_record['criterion']}. [Per-stream counts and lifecycle evidence](secret-scan.json). No values or matched lines are published.\n")
    notes = '# Requests to orchestrator — Follow-up 10\n\nDated 2026-10-31 UTC (2026-10-30 brief). Source includes #240/#241/#244/#245; owned harness and evidence paths only.\n\n| Item | Status | Evidence / action |\n|---|---|---|\n'
    reasons = {'SECOPS-16':'D-066 ordinary scan and final per-stream protected lifecycle, including the 60-peer bulk export.',
               'SECOPS-17':'Fake provider past_due preserves the paid invoice; finite 90 s grace, API state, expiry and repeated truth measured.',
               'SECOPS-18':'D-076 fence, log/counter/audit, withdrawn peer and two negatives re-observed.',
               'SECOPS-19':'Nonzero hold outbox then zero after replay in observation and CLI.',
               'SECOPS-20':'#240 lab NAT removal and #241 D-079 present; serving read-back and before history identify the observed cause.',
               'SECOPS-21':'#245 is present. Require first offer acceptance; if compilation precedes the first fresh retained-generation observation, the restored central applied value can still be stale. Review the timestamped first_offer_analysis and order restoration accordingly.'}
    if generation.get('first_offer_analysis'):
        first = generation['first_offer_analysis']
        observed = first.get('first_post_restore_observation') or {}
        reasons['SECOPS-21'] += f" Measured first offer {first['generation']} at {first['signed_at']}; retained node {first['retained_generation_before_first_offer']}; first central receipt of applied {observed.get('applied')} at {observed.get('received_at')}."
    for name in sorted(findings):notes+=f'| {name} | **{findings[name]}** | {reasons[name]} [{name}.json]({name}.json) |\n'
    notes += f'| SECOPS-05 / AT30 | **{status}** | [Restore measurement table](restore-reconcile.md). |\n| D-077 | **{"closed" if reg.get("passed") else "open"}** | [HTTP call table](D-077.json); member session 401, revoked device 409 STATE_CONFLICT, same-key replay, existing operation. |\n| AT29 established transports / AT31 forced relay | **Deferred** | Await devstack follow-up 15; no approximation or rerun. |\n'
    notes += '\nPrior request history is preserved byte-for-byte in [follow-up 9](history/follow-up-9/requests.md). No additional product/lab changes requested unless the recorded open findings require them.\n'
    md('requests.md',notes)
    md('RUN.md','''# Follow-up 10 run instructions

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
make -f mk/at-secops.mk at-secops-followup10
make go-lint
go vet ./...
gofmt -l .
git diff --check
```

Go 1.27.3 and Docker are required. The foreground driver refuses an occupied
`fvpn-w4-secops-*` namespace, brings up the existing lab with port base 30000,
90-second TEST ONLY grace and disk scratch, runs the live test and always tears
down its objects. Raw logs remain in ignored `.agent/secops/` with mode 0600.

The test performs the same T0/T1 physical restore and retained-archive replay
protocol as Follow-up 9, uses the D-077 HTTP admin re-authorization route,
records SECOPS-20 whole-ruleset read-back and SECOPS-21 generation ordering,
then measures finite past_due grace and 60-peer final protected-stream attribution.
No direct AuthorizeDevice call, generation repair or manual NAT removal remains.
Follow-up 9's direct service method is historical, documented in
`history/follow-up-9/RUN.md`; use this target for current re-measurement.

After cleanup, `followup10_records.py` renders records and scope pins. A missing
or failed live check stays blocked. AT29 established transports and AT31 relay
are deferred to devstack follow-up 15. The scan cohort does not claim capacity.
''')
    deps=load(PUBLIC/'dependencies.json')
    for name,state in findings.items():deps.setdefault('requests',{})[name]={'status':state,'detail':reasons[name],'evidence':name+'.json'}
    deps['requests']['SECOPS-05'].update(status='closed' if status == 'passed' else 'open',
        detail='AT30 re-measured in Follow-up 10 with the D-077 HTTP re-grant route.', evidence='restore-reconcile.json')
    deps['requests']['D-077']={'status':'closed' if reg.get('passed') else 'open',
        'detail':'Admin HTTP re-grant, same-key replay, member-session 401 and revoked-device 409 measured.',
        'evidence':'D-077.json'}
    write('dependencies.json',deps)
    scope=load(PUBLIC/'scope-review.json')
    names=sorted(set(written+['followup10-live.json','followup10-run.json','followup10-attempts.json','followup10-attempt1-live.json','followup10-attempt1-run.json','followup10-attempt1-verdict.json','validation.json']))
    scope['follow_up_10']={'reviewed_at':run.get('finished_at'),'source_revision':run.get('source_revision'),
        'scope':'Own Follow-up 10 records only; historical pins remain tied to their historical records. No broader AT or hardware promotion.',
        'records':[{'path':'docs/e2e/at/secops/'+name,'sha256':digest(PUBLIC/name)} for name in names if (PUBLIC/name).is_file()],
        'requests':findings}
    write_json(PUBLIC/'scope-review.json',scope)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
