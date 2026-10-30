#!/usr/bin/env python3
"""Bounded, fail-closed Wave 4 evidence driver; no alternate dev stack.

Raw subprocess output stays private. Public evidence contains command argv,
exit codes, test names/outcomes and scan categories, never raw response bodies.
"""
import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[4]
HERE = Path(__file__).resolve().parent
PUBLIC = ROOT / 'docs/e2e/at/secops'
PRIVATE = ROOT / '.agent/secops'
PREFIX = 'w4-secops'
PORT_BASE = '30000'

# Reuse owner-maintained probes. This neither copies nor forks tests/e2e's lab.
SUITES = [
    ('boundary', ['AT28'], './tests/e2e/at/secops', ['TestAT28_AllContractRoleBoundaries', 'TestAT28_TransportSecretCanaries', 'TestAT28_RealMTLSTwoNodesExpiredRevokedAndForgedIDs']),
    ('sessions', ['AT28'], './internal/auth', ['TestAT28_OIDCVerificationAndSessions', 'TestAT28_RoleCookiesCoexistAndLogoutIndependently', 'TestAT28_BrowserSelectorRequiresBothCookies', 'TestID01_ScopedIdentitySessionRotationAndExpiry', 'TestAT01_AdminIdentityAndEverySessionAudited']),
    ('http-security', ['AT28'], './internal/httpapi', ['TestAT28_RolesAndFallback', 'TestAT28_CertificateIdentityAndForgedHeaders', 'TestAT28_CSRFSchemaSizeAndPreconditions', 'TestAT28_ErrorAndLogRedaction', 'TestAT28_BootstrapTransport', 'TestAT28_MemberOperationHTTPIsolation', 'TestAT28_IngressCannotEvictOrExhaustPrincipalBuckets', 'TestAT28_BrowserRateAndConcurrencyRetryAfter', 'TestAPI01_IdempotencyReplayConflictConcurrentExpiry', 'TestAPI01_ETag']),
    ('pki-enrollment-audit', ['AT28', 'AT32'], './internal/enrollment', ['TestAT28_AuthenticationRenewalAndPermanentRevocation', 'TestAT28_BootstrapOnceOverAuthenticatedPathAndNoSecrets', 'TestD029_EnrollmentAuditAndOperationsRoundTrip', 'TestAT06_IdempotentCreationLostResponseAndRace', 'TestAT06_ExpiryRevocationAndTransactionalRollback', 'TestAT05_UniformDeliveryMetadataAndPlatforms']),
    ('node-http', ['AT28', 'AT29'], './internal/nodes/httpwire', ['TestAT06_AT28_NodeCompositionUsesMiddlewareAndPKI', 'TestAT28_HTTPBootstrapExactlyOnce', 'TestENR02_HTTPCertificateRenewalAndIdentityChecks', 'TestAT29_HTTPSignedLeaseWindowAndReplay', 'TestAT29_HTTPKeyedLeaseHoldsShareLocksAndRollsBack']),
    ('member-ownership', ['AT28', 'AT30'], './internal/members/httpadapter', ['TestAT02_DeviceLimitRevokeOwnershipReadbackHTTP', 'TestAT04_SuspensionGrantGraceAndHTTPPreconditions', 'TestAPI01_ConcurrentHTTPReplayAndRealSessionRateLimit', 'TestBILL01_WorkerReconciliationRepairsMissedWebhook', 'TestMEM03_NetBirdLostCreateIdentityDriftAndPersistentBindings']),
    ('watchdog', ['AT29'], './tests/e2e/at/secops', ['TestAT29_IndependentWatchdogFailureMatrix', 'TestAT29_EarliestFiniteServiceDeadline']),
    ('lease-service', ['AT29'], './internal/leases', ['TestAT29_LeaseMathReplayAndEntitlementImmutability', 'TestLeases_SuspendDuringSignRebuildsAndExcludesSuspendedMember', 'TestLeases_UnpublishedDisabledAndRevokedNode']),
    ('elapsed-time', ['AT29'], './node/internal/timeanchor', ['TestAT29_WallJumpDoesNotExtendDeadline', 'TestRebootLosesAnchor', 'TestReplayDoesNotRestartWindow']),
    ('agent-deadlines', ['AT29'], './node/agent', ['TestStep_LeaseGatesAnd304', 'TestStep_WallJumpDoesNotExtendLease', 'TestStep_RebootRequiresNewerLease']),
    ('kernel-render', ['AT29'], './node/enforce', ['TestRender_TimeoutCeilAndExpiry', 'TestDeadline_Earliest']),
    ('restore-ledger', ['AT30'], './internal/usage', ['TestAT30_RestoreSnapshotRecoveryReplayAndReplayUnavailable', 'TestAT30_HTTPRecoveryFreezeEpochRotationReplayAndAuditPagination', 'TestAT15_DedupeOutOfOrderAckAndConflict', 'Test17_3_FreezeFencesInFlightIngest']),
    ('restore-cli', ['AT30'], './internal/usage/cmd/fvpn-usage-admin', ['TestCLI_FreezeRecoveryCompleteAdjustAndAudit']),
    ('node-archive', ['AT30'], './node/internal/usagestore', ['TestAT30_LedgerEpochRestore', 'TestPurgeArchiveKeepsUnacked', 'TestIdentityAndLowerACK']),
    ('node-replay', ['AT30'], './node/internal/usageupload', ['TestEpochMismatchFetchesCheckpoint', 'TestRestoreSchedulesAndClears']),
    ('payment-reconciliation', ['AT30'], './internal/billing', ['TestBILL01_ReconciliationRepairsDroppedWebhookAndGraceExpires', 'TestAT04_CancelRefundRenewSuspendAndExactlyOneOutbox']),
    ('peer-reconciliation', ['AT30'], './internal/netbird', ['TestAT08_LostCreateResponsesDoNotDuplicate', 'TestAT08_DriftDuplicatesAndUnconfirmedWrites', 'TestMEM03_LostAuthorizationWrite']),
    ('scope-schema', ['AT32'], './tests/e2e/at/secops', ['TestAT32_UnifiedNodeContractSurface']),
    ('scope-lifecycle', ['AT32'], './internal/nodes', ['TestAT32_CentralEnrollConfigureObserveLifecycle']),
]

REQUIREMENTS = {
    'AT28': ['SEC01', 'API01', 'ID01'],
    'AT29': ['ENT02', 'ISO02', 'SEC01'],
    'AT30': ['DATA01', 'BILL01', 'MTR03'],
    'AT31': ['NFR01', 'NODE02', 'FUP04'],
    'AT32': ['NODE01', 'NODE04', 'SCOPE01'],
}
EXPECTED = {
    'AT28': 'Every resource is role/owner scoped; mTLS binds node/incarnation; invalid credentials, CSRF, oversize bodies and replay refuse safely; all stack logs and audit streams are secret-free.',
    'AT29': 'Established TCP/UDP and new flows stop at the earliest finite deadline despite control loss, agent/watchdog failure and clock rollback; only a fresh verified policy restores forwarding.',
    'AT30': 'An isolated DB/WAL restore replays previously ACKed node archives, reconciles provider and peers without duplicates or entitlement extension, and measures complete-service RPO/RTO.',
    'AT31': '20 members/60 authorized devices under the recorded mixed load; correct authorization and ledger; read p95 <500 ms, policy <=60 s, usage <=120 s, CPU/memory/IO and relay throughput recorded.',
    'AT32': 'One source-independent Node model and update path; notes never authorize; public statements match measured behavior; no partner approval or maintenance product workflow.',
}
GAPS = {
    'AT28': [],
    'AT29': ['A counted serving flow is required before outage/deadline assertions; see the live publication, rule, admission and flow observations.', 'Grace/grant/short-lease flow cutoffs and rollback of wall time remain unmeasured. A disposable VM or injected time source is required for clock rollback; never change the shared host clock.'],
    'AT30': ['Provider/NetBird/revocation-journal reconciliation and actual node SQLite archive replay are outside the isolated physical subset. Complete-service RPO/RTO remain null.'],
    'AT31': ['D-050 grants and native admission replace projected SQL rows. The shared lab has one native member daemon; a documented 20-user/3-peer-per-user hook is needed for 60 devices.', 'API and synthetic ledger diagnostics are not capacity certification. The full sampling/policy/multi-flow/relay profile needs counted serving flows. Candidate hardware remains needs operator run.'],
    'AT32': ['DOC01–DOC13 were resolved on main and re-reviewed. Physical three-origin/platform comparison and complete Wave 4 runtime evidence still need operator runs.'],
}


def utc():
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec='seconds').replace('+00:00', 'Z')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def scan(raw):
    """Return locations/categories only, never a matched secret or a hash of it."""
    patterns = {
        'private_key': rb'-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----',
        'authorization_value': rb'(?i)(?:authorization["\s:=]+|bearer\s+)[A-Za-z0-9_./+=-]{20,}',
        'enrollment_token': rb'fvpn_enr_[A-Za-z0-9_-]{20,}',
        'credential_url': rb'(?i)postgres(?:ql)?://[^\s/:]+:[^\s@]+@',
    }
    return [{'category': kind, 'line': raw.count(b'\n', 0, m.start()) + 1}
            for kind, pattern in patterns.items() for m in re.finditer(pattern, raw)]


def summarize_go(raw, required, returncode):
    outcomes, elapsed, packages = {}, {}, {}
    for line in raw.splitlines():
        try:
            e = json.loads(line)
        except (ValueError, TypeError):
            continue
        if e.get('Action') not in ('pass', 'fail', 'skip'):
            continue
        if e.get('Test'):
            outcomes[e['Test']] = e['Action']
            elapsed[e['Test']] = e.get('Elapsed', 0)
        elif e.get('Package'):
            packages[e['Package']] = e['Action']
    missing = [name for name in required if name not in outcomes]
    skipped = [name for name, result in outcomes.items() if result == 'skip']
    failed = [name for name, result in outcomes.items() if result == 'fail']
    if returncode or failed:
        status = 'failed'
    elif missing or skipped or not packages or any(v != 'pass' for v in packages.values()):
        status = 'blocked'
    else:
        status = 'passed'
    return dict(status=status, required_tests=required, missing=missing, skipped=skipped,
                tests=[dict(name=n, result=outcomes[n], elapsed_seconds=elapsed[n]) for n in sorted(outcomes)],
                package_results=packages)


def command(name, args, env, timeout=600):
    path = PRIVATE / (name + '.log')
    started, clock = utc(), time.monotonic()
    with path.open('wb') as output:
        path.chmod(0o600)
        try:
            proc = subprocess.run(args, cwd=ROOT, env=env, stdout=output, stderr=subprocess.STDOUT, timeout=timeout, check=False)
            code = proc.returncode
        except (subprocess.TimeoutExpired, OSError):
            code = 124
    raw = path.read_bytes()
    record = dict(command=args, started_at=started, finished_at=utc(), elapsed_seconds=round(time.monotonic()-clock, 3), exit_code=code,
                  scan={'stream': name, 'bytes': len(raw), 'findings': scan(raw)})
    return record, raw


def inventory(env):
    # Query only this task's namespace; never exec or delete another stack.
    queries = {
        'containers': ['docker', 'ps', '-a', '--filter', 'name=^fvpn-w4-secops-', '--format', '{{.Names}}'],
        'networks': ['docker', 'network', 'ls', '--filter', 'name=^fvpn-w4-secops', '--format', '{{.Name}}'],
        'volumes': ['docker', 'volume', 'ls', '--filter', 'name=^fvpn-w4-secops-', '--format', '{{.Name}}'],
        'images': ['docker', 'images', '--filter', 'reference=fvpn-w4-secops-*', '--format', '{{.Repository}}:{{.Tag}}'],
    }
    results = {}
    for name, argv in queries.items():
        record, raw = command('inventory-'+name, argv, env, 30)
        # Docker name filters can match substrings: enforce exact prefix again.
        names = [line for line in raw.decode(errors='replace').splitlines()
                 if line == 'fvpn-w4-secops' or line.startswith('fvpn-w4-secops-')]
        results[name] = {'exit_code': record['exit_code'], 'count': len(names)}
    return results


def write_json(path, data):
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--components-only', action='store_true', help='Supporting tests only; never an acceptance pass.')
    args = parser.parse_args()
    if os.getenv('FVPN_E2E_PREFIX', PREFIX) != PREFIX or os.getenv('FVPN_E2E_PORT_BASE', PORT_BASE) != PORT_BASE:
        parser.error('this task is assigned prefix w4-secops and port base 30000; no substitute is allowed')
    PRIVATE.mkdir(parents=True, exist_ok=True)
    PUBLIC.mkdir(parents=True, exist_ok=True)
    (ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
    os.umask(0o077)
    revision = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    history = subprocess.check_output(['git', 'log', '--format=%s'], cwd=ROOT, text=True)
    login_burst = '1' if 'w2-control-core follow-up 10' in history else '0'
    source_digest = hashlib.sha256(b''.join(p.read_bytes() for p in sorted(HERE.glob('*.go')))).hexdigest()
    records, builds = [], []
    started = utc()
    with tempfile.TemporaryDirectory(prefix='secops.', dir=ROOT / '.agent/tmp') as scratch:
        env = dict(os.environ, GOTOOLCHAIN='local', CGO_ENABLED='0', TMPDIR=scratch, GOTMPDIR=scratch,
                   E2E_RESET='1', FVPN_E2E_PREFIX=PREFIX, FVPN_E2E_PORT_BASE=PORT_BASE, FVPN_E2E_STATE=str(ROOT / '.agent/e2e-state-w4-secops'))
        stack = {'status': 'blocked', 'reason': 'Full-stack probes not run in components-only mode.'}
        if not args.components_only:
            before = inventory(env)
            stack['inventory_before'] = before
            if any(v['exit_code'] or v['count'] for v in before.values()):
                stack['reason'] = 'Assigned namespace is occupied or Docker inventory unavailable; no bring-up or teardown authorized by this runner.'
            else:
                # Never attach a previous run's backup or evidence to this run.
                for stale in (PUBLIC / 'live.json', PUBLIC / 'restore.json', PUBLIC / 'pitr.json', PUBLIC / 'pitr-source.json', PRIVATE / 'pitr-manifest.json', PRIVATE / 'retained.json'):
                    stale.unlink(missing_ok=True)
                try:
                    up, raw = command('e2e-up', ['make', '-f', 'mk/e2e.mk', 'e2e-up'], env, 1200)
                    stack['up'] = up
                    if up['exit_code']:
                        stack['reason'] = 'Assigned dev stack failed bring-up; private diagnostics retained.'
                    else:
                        live_command = ['go', 'test', '-json', '-tags', 'e2e,integration', '-p', '1', '-parallel', '2', '-count=1', '-timeout', '15m', '-run', '^TestAT28_AT31_LiveStack$', './tests/e2e/at/secops']
                        probe, output = command('live-probe', live_command, dict(env, FVPN_SECOPS_LIVE='1', FVPN_SECOPS_LOGIN_BURST=login_burst), 960)
                        stack['live_probe'] = dict(probe, **summarize_go(output, ['TestAT28_AT31_LiveStack'], probe['exit_code']))
                        if (PRIVATE / 'pitr-manifest.json').is_file() and (PRIVATE / 'retained.json').is_file():
                            restore, _ = command('isolated-restore', ['python3', 'tests/e2e/at/secops/pitr.py'], env)
                            stack['isolated_restore'] = restore
                        else:
                            stack['isolated_restore'] = {'status': 'blocked', 'reason': 'Current live run produced no backup/retained window; no stale restore reused.'}
                        stack['status'] = 'running-stack-observed'
                        stack['reason'] = 'Live probes executed on the assigned prefix; case-specific preconditions and measurements follow.'

                finally:
                    down, _ = command('e2e-down', ['make', '-f', 'mk/e2e.mk', 'e2e-down'], env, 180)
                    stack['down'] = down
                    stack['inventory_after'] = inventory(env)
        for arch in ('amd64', 'arm64'):
            binary = Path(scratch) / ('secops-'+arch+'.test')
            build, _ = command('build-'+arch, ['go', 'test', '-c', '-tags', 'e2e,integration', '-o', str(binary), './tests/e2e/at/secops'], dict(env, GOOS='linux', GOARCH=arch), 600)
            build['command'][build['command'].index('-o')+1] = '<disk-scratch>/secops-'+arch+'.test'
            builds.append({'arch': arch, 'kind': 'acceptance_test_binary', 'sha256': sha(binary) if binary.exists() else None, **build})
        for name, ats, package, required in SUITES:
            print('running '+name, flush=True)
            argv = ['go', 'test', '-json', '-tags', 'e2e,integration', '-p', '1', '-parallel', '2', '-count=1', '-timeout', '8m', '-run', '^('+'|'.join(required)+')$', package]
            cmd, raw = command(name, argv, env)
            record = dict(id=name, acceptance_cases=ats, evidence_level='component', **cmd, **summarize_go(raw, required, cmd['exit_code']))
            if record['scan']['findings']:
                record['status'] = 'failed'
            records.append(record)
            print(name+': '+record['status'], flush=True)
    review = json.loads((PUBLIC / 'scope-review.json').read_text())
    live = json.loads((PUBLIC / 'live.json').read_text()) if stack.get('live_probe') and (PUBLIC / 'live.json').exists() else {}
    deployed_builds = []
    for path in sorted((ROOT / '.agent/e2e-build').glob('fvpn-*')):
        if live and path.is_file():
            deployed_builds.append(dict(name=path.name, arch='amd64', sha256=sha(path), kind='lab_build_binary'))
    common = dict(schema_version=1, run_started_at=started, run_finished_at=utc(), source_revision=revision,
                  probe_source_sha256=source_digest, build_digests=builds+deployed_builds, image_digests=live.get('image_digests', []),
                  policy_revision=live.get('policy_revision'), node_incarnation=live.get('node_incarnation'), stack=stack)
    write_json(PUBLIC / 'components.json', {'schema_version': 1, 'run_started_at': started, 'run_finished_at': utc(), 'suites': records})
    write_json(PUBLIC / 'secret-scan.json', {'schema_version': 1, 'scope': 'Captured component subprocess streams plus all twelve stack container stdout/stderr streams, produced node/member log files, and all audit_events rows. Known live secret canaries plus generic markers; not a claim about unexercised code paths.',
               'streams': [r['scan'] for r in records], 'stack_streams': live.get('secret_scan', []), 'canaries': live.get('known_secret_canaries'),
               'audit_rows': live.get('audit_rows_scanned'), 'full_stack_scan_status': 'passed' if live.get('secret_scan') and all(x['read_ok'] and not x['known_secret_matches'] for x in live['secret_scan']) else 'blocked'})
    for at, requirements in REQUIREMENTS.items():
        related = [r for r in records if at in r['acceptance_cases']]
        component_status = 'failed' if any(r['status'] == 'failed' for r in related) else ('passed' if related and all(r['status'] == 'passed' for r in related) else 'blocked')
        status = 'passed' if at == 'AT28' and component_status == 'passed' and stack.get('live_probe', {}).get('status') == 'passed' and live.get('security_matrix_complete') else 'blocked'
        blockers = list(GAPS[at])
        if not live or (at == 'AT28' and status != 'passed'):
            blockers.insert(0, 'A required live/component probe failed or was not executed; inspect live.json, components.json and private diagnostic logs.')
        if at == 'AT28' and live.get('device_idor_blocker'):
            blockers.append(live['device_idor_blocker'])
        if at in ('AT29', 'AT31'):
            blockers.extend(live[k] for k in ('admission_blocker', 'binding_blocker', 'serving_blocker') if live.get(k))
        if at == 'AT31' and live.get('concurrent_sign_in'):
            login = live['concurrent_sign_in']
            if login['authenticated'] != login['attempts']:
                blockers.append(f"{'Historical Follow-up 1' if login.get('remeasurement_status') else 'Current'} concurrent login diagnostic completed {login['authenticated']}/{login['attempts']} sessions; final login HTTP statuses {login.get('last_login_http_status', {})}. Limits were not tuned.")
        data = dict(common, id=at, spec=['§19.1', '§19 '+at], requirements=requirements, status=status, verdict=status,
                    expected=EXPECTED[at], actual={'component_status': component_status, 'component_suites': [r['id'] for r in related], 'full_stack_executed': bool(live), 'full_acceptance_profile_executed': status == 'passed'},
                    blockers=blockers, commands=[r['command'] for r in related],
                    operator_run={'status': 'needs operator run', 'script': 'operator-runs.md#'+at.lower()},
                    evidence=['live.json', 'components.json', 'secret-scan.json', 'requests.md'])
        if at == 'AT29':
            data['node_incarnation'] = live.get('agent_node_incarnation')
            data['policy_revision'] = live.get('node_after_publication', {}).get('config_generation', {}).get('applied')
            data['serving'] = {k:live.get(k) for k in ('node_service_precondition','publication_request','publication_operation','forward_baseline_generation_zero','member_accept_rules','watchdog_heartbeat_present','counted_flow','flow_probe','serving_checks','node_after_publication','watchdog_state','configured_updater_recovery_file_present','short_node_lease_seconds')}
        if at == 'AT30':
            data['recovery'] = live.get('recovery', {})
            if (PUBLIC / 'pitr.json').exists() and stack.get('isolated_restore',{}).get('exit_code') == 0:
                data['physical_restore_subset'] = json.loads((PUBLIC / 'pitr.json').read_text())
                data['evidence'].extend(['pitr.json', 'pitr-source.json'])
        if at == 'AT31':
            data['capacity'] = dict(profile='load-profile.json', full_profile_executed=False, candidate_hardware=False, measured_subset=live.get('capacity_subset'), grants=live.get('grant_projection'), requested_devices=60, authorized_devices=live.get('authorized_devices', 0), concurrent_sign_in=live.get('concurrent_sign_in'), synthetic_ledger_ack=live.get('usage_final_ack'), login_41st_request=live.get('login_41st_request'))
        if at == 'AT32':
            data['documentation_findings'] = [{'id': f['id'], 'status': f['status']} for f in review['findings']]
            data['evidence'].append('scope-review.json')
        write_json(PUBLIC / (at+'.json'), data)
        lines = [f'# {at}: {status}', '', EXPECTED[at], '', f'Run UTC: {started}. Source: `{revision}`.', '',
                 f'Supporting component result: **{component_status}**. Acceptance: **{status}**.', '',
                 'Build/image digests, policy/incarnation, command argv, UTC times, expected/actual and limitations are in ['+at+'.json]('+at+'.json).', '']
        if blockers:
            lines += ['Blocking conditions:', ''] + ['- '+b for b in blockers]
        lines += ['', 'Evidence: [live probes](live.json), [component tests](components.json), [secret scan](secret-scan.json), [requests](requests.md), [operator script](operator-runs.md#'+at.lower()+').', '']
        (PUBLIC / (at+'.md')).write_text('\n'.join(lines))
    # A component-only check is useful in CI, but full acceptance never turns
    # missing measurements, missing required tests, or skips into success.
    ok = all(r['status'] == 'passed' for r in records) and all(b['exit_code'] == 0 for b in builds)
    print('evidence: docs/e2e/at/secops; consult each AT verdict; full suite exits 2 while any AT is blocked', flush=True)
    return (0 if ok else 1) if args.components_only else 2


if __name__ == '__main__':
    raise SystemExit(main())
