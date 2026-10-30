#!/usr/bin/env python3
"""Render existing Follow-up 4 records; does not run or upgrade any measurement."""
import datetime as dt
import json
import re
from run import PUBLIC, write_json, write_followup4_records, utc

COMMON = ('schema_version','run_started_at','run_finished_at','source_revision','probe_source_sha256','build_digests','image_digests','policy_revision','node_incarnation','stack')


def normalize_times(value):
    if isinstance(value, dict):
        return {k: normalize_times(v) for k,v in value.items()}
    if isinstance(value, list):
        return [normalize_times(v) for v in value]
    if isinstance(value, str) and re.fullmatch(r'\d{4}-\d\d-\d\dT[\d:.]+[+-]\d\d:\d\d', value):
        return dt.datetime.fromisoformat(value).astimezone(dt.timezone.utc).isoformat().replace('+00:00','Z')
    return value


def main():
    base = json.loads((PUBLIC/'AT28.json').read_text())
    common = {k:base.get(k) for k in COMMON}
    live = normalize_times(json.loads((PUBLIC/'live.json').read_text()))
    if live.get('source_revision') != '9b17127c888f94191d759617c99fe1225078a092' or live.get('started_at') != '2026-10-30T06:49:36.469735179Z':
        raise SystemExit('This dated renderer is bound to the recorded Follow-up 4 main run; review a new run separately.')
    if 'device_scale_blocker' in live:
        live['superseded_legacy_scale_note']=live.pop('device_scale_blocker')
        live['device_scale_hook']='Available after #143; actual combined/isolated admission counts remain qualified.'
    if 'publication_positive' in live:
        live['serving_flow_positive']=live.pop('publication_positive')
        live['serving_flow_positive']['scope']='Counted forwarding prerequisite; distinct from the successful publication operations.'
    write_json(PUBLIC/'live.json',live)
    write_followup4_records(common,live)
    rates = live['rate_classes']
    limiter = json.loads((PUBLIC/'limiter.json').read_text())
    limiter['runtime_settings'] = 'runtime-settings.json'
    limiter['login_threshold_history'] = {'record':'followup3-limiter.json','status':'resolved historical measurement; not remeasured after the combined run stopped','reason':'20/20 landing/session burst succeeded in this run; the later threshold calls were not reached.'}
    write_json(PUBLIC/'limiter.json',limiter)
    lines = ['# SECOPS-06: class buckets passed; usage size boundary blocked','',
             'Source `'+live['source_revision']+'`. Unmodified settings are in [runtime settings](runtime-settings.json).', '',
             '| Class / probe | First rejection | Retry-After | Metric delta |', '|---|---:|---|---:|']
    for key in ['heartbeat','lease','usage','enrollment','webhook','anonymous_shared']:
        item=rates[key]
        lines.append(f"| {key} | {item['actual_first_rejection']} | `{item['first_retry_after']}` | {item['metric_delta']} |")
    lines += ['', 'All refusals above are 429 RATE_LIMITED with the matching class/limiter label. Enrollment and provider webhook share the anonymous address bucket. Observations use schema-invalid `{}` to avoid fabricated telemetry; lease and usage payloads are valid.', '',
              '| Valid usage body bytes | Expected HTTP | Actual HTTP |', '|---:|---:|---:|']
    for item in rates['usage_body_limits']:
        lines.append(f"| {item['bytes']} | {item['expected_status']} | {item['actual_status']} |")
    login=live['concurrent_sign_in']
    lines += ['', 'Bodies contain the same valid event plus legal trailing JSON spaces. The optional Idempotency-Key header is absent. The 1 MiB canonicalization boundary conflicts with the configured 8 MiB usage limit (SECOPS-15). Ten small uploads returned one accepted and nine duplicate results; one ledger row remains.', '',
              f"The barrier-started login completed **{login['authenticated']}/{login['attempts']}**, with zero wrong identities. GET / and session responses were 200 for all 20. This resolves SECOPS-14 landing behavior, not browser-render/platform acceptance.", '',
              'Login completion 11 / Retry-After 4 and initiation 41 / Retry-After 1 remain the attributed [Follow-up 3 measurement](followup3-limiter.json). The combined run stopped during native scale before repeating those late calls. IPv6 /64 remains component verified; the live source uses IPv4 loopback.', '',
              '[JSON record](limiter.json) includes UTC, commands, digests and response/metric evidence. Follow-up 3 per-class assumptions are superseded by #145; the earlier login conclusion remains resolved.', '']
    (PUBLIC/'limiter.md').write_text('\n'.join(lines))
    pub=json.loads((PUBLIC/'publication.json').read_text())
    (PUBLIC/'publication.md').write_text('''# SECOPS-11: live gate and zero-binding publication verified

The real ready baseline, with capacity unset, accepted PUT as 202 and recorded
`data_path_probe=failed`, `NODE_NOT_SERVICEABLE`, `member_forwarding_unconfirmed`.
The operation was still running/retrying at that snapshot. This proves the step
refusal, not a terminal failed operation. After the finite capacity input, its
retry could recover; both snapshots remain in the record.

The next publication succeeded with an issued **published** generation containing
**zero device bindings**. All required operation steps succeeded after #148.
Publication with one genuinely admitted device also succeeded. No observation
report or projected row was written by the harness.

The real watchdog reported serving=1, reason=serving, with DNS/lease/QoS/source
binding checks passed. The observation-named lease generation matched applied
generation 9. The controlled native request failed and both per-device admission
counters stayed at zero. This keeps AT29 and full AT31 blocked on the devstack
follow-up 9 counted-flow gate; no forwarding diagnosis beyond these counters.

SECOPS-13's exact owner test passes against its frozen testdata fixture (#130),
including after this live record changed. [JSON record](publication.json),
[live evidence](live.json), [component results](components.json) and
[requests](requests.md) preserve command, UTC, incarnation and digest evidence.
''')
    observed=live['native_scale']['new_authorized_peers']+live['authorized_devices']
    for at in ('AT28','AT29','AT31','AT32'):
        path=PUBLIC/(at+'.json'); rec=json.loads(path.read_text())
        rec['record_rendered_at']=utc()
        if at=='AT28':
            rec['blockers']=['SECOPS-15: valid usage requests above 1 MiB fail 400 despite the 8 MiB setting.',
                             'The combined run ended on a transport/login failure after 51 native admission observations; late node-B revocation and login-threshold calls were not reached. Supporting revoked-certificate tests pass.',
                             'SECOPS-16: native gateway errors contain device-flow query values; deleted transient native CLI logs are also outside the available inventory.']
        if at=='AT29':
            rec['serving']['observed_lease_generation']=live.get('observed_lease_generation')
            rec['serving'].pop('publication_positive',None)
            rec['serving']['serving_flow_positive']=live.get('serving_flow_positive')
            rec['serving']['publication_gate_status']=pub['status']
            rec['blockers']=['Controlled flow produced zero admission-byte delta despite serving=1 and admitted device; owning gate is w3-devstack-e2e follow-up 9.', 'Paid/grace/grant short-deadline matrix and guest wall-clock rollback remain unmeasured; operator script retained.']
        if at=='AT31':
            rec['capacity']['initial_authorized_device_snapshot']=live['authorized_devices']
            rec['capacity']['authorized_devices']=None
            rec['capacity']['main_run_native_peer_authorization_observations']=observed
            rec['capacity']['cohort_60_confirmed_in_main_run']=False
            rec['capacity']['main_run_stop']='User index 16 final read had no HTTP response; subsequent admin login failed. Root cause unclassified in the original capture; zero is not an HTTP status or a proved zero-device count.'
            rec['capacity']['login_threshold_history']='followup3-limiter.json'
            rec['blockers']=['No counted serving flow; full sampling/policy/multi-flow/relay profile not executed. Candidate hardware remains needs operator run.', 'SECOPS-15: usage body limit mismatch.', 'Combined run did not confirm 60 devices. Isolated recheck admitted 50, then native login 51 failed with gateway connection exhaustion: deploy/devstack/nginx.conf worker_connections=256 (SECOPS-10).']
            if (PUBLIC/'native-scale.json').exists():
                scale=json.loads((PUBLIC/'native-scale.json').read_text())
                rec['capacity']['isolated_native_admission_subset']={'status':scale['status'],'record':'native-scale.json','source_revision':scale['source_revision'],'node_incarnation':None,'flow_or_capacity_claim':False}
                rec['evidence']=list(dict.fromkeys(rec['evidence']+['native-scale.json','native-scale-live.json','native-cohort.json','native-diagnostics.json']))
        if at=='AT32':
            rec['blockers']=['DOC01–DOC14 resolved; physical origin/platform/hardware comparison remains needs operator run. No full Wave 4 serving or capacity claim.']
        write_json(path,normalize_times(rec))
        text=['# '+at+': '+rec['status'],'',rec['expected'],'',
              'Source `'+rec['source_revision']+'`; run UTC '+rec['run_started_at']+'.', '',
              'Supporting component suites passed. Full acceptance remains **blocked**.','']
        text += ['- '+x for x in rec['blockers']]
        text += ['', 'Details: ['+at+'.json]('+at+'.json), [live](live.json), [requests](requests.md), [components](components.json), [operator scripts](operator-runs.md#'+at.lower()+').','']
        (PUBLIC/(at+'.md')).write_text('\n'.join(text))
    scan=json.loads((PUBLIC/'secret-scan.json').read_text())
    scan['scope']='All inventoried main-run container stdout/stderr, available produced node/member log files, audit rows and captured command streams; known-secret canaries plus generic command-output markers. Deleted transient native device-flow logs were unavailable and are not claimed as scanned.'
    scan['main_run_commands']=[common['stack'][k]['scan'] for k in ('up','live_probe','down') if common['stack'].get(k,{}).get('scan')]
    scan['transient_log_limit']='JoinNativeMember removes netbird-up.log before returning; see requests.md for scan-before-delete hook request.'
    if (PUBLIC/'native-scale-live.json').exists():
        scale=json.loads((PUBLIC/'native-scale-live.json').read_text())
        scan['isolated_admission_streams']=scale.get('secret_scan',[])
    if (PUBLIC/'native-diagnostics.json').exists():
        diagnostic=json.loads((PUBLIC/'native-diagnostics.json').read_text())
        scan['native_gateway_auth_query_logging']=diagnostic['auth_query_logging']
        canary_streams=scan['stack_streams']+scan.get('isolated_admission_streams',[])
        scan['known_canary_scan_status']='passed' if canary_streams and all(s.get('read_ok') and s.get('known_secret_matches') == 0 for s in canary_streams) else 'blocked'
        scan['full_stack_scan_status']='blocked'
        scan['reason']='Native gateway error diagnostics contain device-flow query values; deleted transient native CLI logs are also outside the available inventory. See SECOPS-16.'
    write_json(PUBLIC/'secret-scan.json',scan)


if __name__=='__main__':
    main()
