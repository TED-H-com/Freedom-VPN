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
from pitr_compatibility import review as review_pitr

ROOT = Path(__file__).resolve().parents[4]
HERE = Path(__file__).resolve().parent
PUBLIC = ROOT / 'docs/e2e/at/secops'
PRIVATE = ROOT / '.agent/secops'
PREFIX = 'w4-secops'
PORT_BASE = '30000'

# Reuse owner-maintained probes. This neither copies nor forks tests/e2e's lab.
SUITES = [
    ('rate-classes', ['AT28', 'AT31'], './internal/httpapi', ['TestClassifyOperationIDsMatchLoadedSpec', 'TestAnonymousAddressBurstUsesIPv6Prefix', 'TestReviewR9_LoginLimiterAddressIdentityBoundsAndRoutes']),
    ('publication-empty', ['AT29', 'AT31'], './internal/nodes/httpwire', ['TestAT08_PublicationWorkerEmptyBindingEvidence']),
    ('login-d060', ['AT28', 'AT31'], './internal/httpapi', ['TestAT31_D060_LoginAddressBurstAndMetric']),
    ('login-subject', ['AT28', 'AT31'], './internal/auth', ['TestSEC05_SubjectLoginLimitAndRequestedRole']),
    ('publication-worker', ['AT29', 'AT31'], './internal/nodes/httpwire', ['TestSECOPS11_PublicationWorkerRequiresArmedMemberDataPath']),
    ('publication-evidence', ['AT29', 'AT31'], './internal/nodes', ['TestSECOPS11_RecordedBaselineDoesNotProveServing']),
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
    'AT31': ['D-050 grants and exported native-peer hooks supply actual admission counts; inspect native_scale in live.json for the observed 20×3 result.', 'API and synthetic ledger diagnostics are not capacity certification. The full sampling/policy/multi-flow/relay profile needs counted serving flows. Candidate hardware remains needs operator run.'],
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
            proc = subprocess.run(args, cwd=ROOT, env=env, stdout=output, stderr=subprocess.STDOUT, timeout=timeout, check=False,
                                  # The lab copies public CoreDNS files for a non-root
                                  # container. Keep private logs 0600, but do not force
                                  # its public render directories to mode 0700.
                                  umask=0o022 if name.endswith('e2e-up') else 0o077)
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


def write_followup3_records(common, live):
    login = live.get('concurrent_sign_in', {})
    thresholds = [live.get(k, {}).get('passed', False) for k in ('login_subject_limit', 'login_41st_request')]
    login_pass = login.get('authenticated') == 20 and login.get('wrong_member') == 0 and all(thresholds)
    write_json(PUBLIC / 'limiter.json', dict(common, id='SECOPS-06', spec=['§19.1', '§19 AT28', '§19 AT31'],
               status='passed' if login_pass else 'blocked', expected='20/20 distinct logins; subject completion 11 rejects; address request 41 rejects with Retry-After 1',
               actual={k:live.get(k) for k in ('login_settings','concurrent_sign_in','login_subject_limit','login_41st_request')},
               supersedes='followup1-login.json; defaults changed by D-060 / PR #106', commands=[common['stack'].get('live_probe',{}).get('command')]))
    if not live.get('publication_negative'):
        live = dict(live, publication_negative={'status': 'blocked', 'passed': False, 'reason': 'Publication request did not reach a worker operation; inspect request and immutable observation.', 'request': live.get('publication_request'), 'owner': 'w3-devstack-e2e: resolver and ready-baseline configuration'})
    write_json(PUBLIC / 'publication.json', dict(common, id='SECOPS-11', spec=['§19 AT29', '§19 AT31'],
               node_incarnation=live.get('agent_node_incarnation'),
               status='blocked', negative_status='passed' if live.get('publication_negative',{}).get('passed') else 'blocked', positive_status='blocked',
               expected='Unarmed publication fails data_path_probe with member_forwarding_unconfirmed; positive needs real armed observation and counted traffic',
               actual={k:live.get(k) for k in ('publication_request','publication_operation','publication_negative','publication_positive','serving_flow_positive','publication_observation_before','publication_observation_after','watchdog_state','flow_probe','counted_flow')},
               commands=[common['stack'].get('live_probe',{}).get('command')]))


def write_followup4_records(common, live):
    rates = live.get('rate_classes', {})
    classes = ['heartbeat', 'lease', 'usage', 'enrollment', 'webhook', 'anonymous_shared']
    rate_pass = all(rates.get(k, {}).get('passed') for k in classes)
    body_pass = bool(rates.get('usage_body_limits')) and all(x['passed'] for x in rates['usage_body_limits'])
    write_json(PUBLIC / 'limiter.json', dict(common, id='SECOPS-06', follow_up=4,
        status='passed' if rate_pass and body_pass else 'blocked',
        login_status='resolved' if live.get('concurrent_sign_in',{}).get('authenticated') == 20 else 'blocked',
        expected='heartbeat/lease request 6; usage/anonymous request 11; 429 RATE_LIMITED, Retry-After 1, matching class metric; valid 8 MiB usage accepted, >8 MiB refused 413',
        actual=dict(rate_classes=rates, **{k:live.get(k) for k in ('login_settings','concurrent_sign_in','login_subject_limit','login_41st_request')}),
        supersedes='followup3-limiter.json: login remains resolved; per-class inference superseded by #145 PascalCase classification fix; IPv6 /64 #141',
        commands=[common['stack'].get('live_probe',{}).get('command')]))
    negative = live.get('publication_without_capacity',{})
    positive = live.get('publication_empty_bindings',{})
    # FU4 asks for the live operation step refusal. A retrying operation may
    # still be running at the snapshot and recover after the capacity command;
    # do not relabel that snapshot as a terminal failed operation.
    op = negative.get('publication_operation', {})
    neg_pass = negative.get('publication_request',{}).get('status') == 202 and op.get('status') != 'succeeded' and any(
        step.get('name') == 'data_path_probe' and step.get('status') == 'failed' and
        step.get('message') == 'member_forwarding_unconfirmed' and step.get('error_code') == 'NODE_NOT_SERVICEABLE'
        for step in op.get('steps', []))
    pos_pass = positive.get('publication_request',{}).get('status') == 202 and positive.get('publication_operation',{}).get('status') == 'succeeded' and positive.get('signed_device_bindings') == 0 and positive.get('signed_publication_intent') == 'published'
    write_json(PUBLIC / 'publication.json', dict(common, id='SECOPS-11', follow_up=4,
        node_incarnation=live.get('agent_node_incarnation'),
        status='passed' if neg_pass and pos_pass else 'blocked',
        negative_status='passed' if neg_pass else 'blocked', positive_status='passed' if pos_pass else 'blocked',
        expected='Unarmed real agent fails data_path_probe with member_forwarding_unconfirmed inside an operation; issued published generation with zero bindings succeeds after #148',
        actual=dict(without_capacity=negative, empty_bindings=positive, with_device=live.get('publication_with_device')),
        counted_flow=live.get('counted_flow'), publication_is_not_traffic_proof=True,
        negative_scope='Operation step refusal at the unarmed snapshot; does not assert a terminal operation failure. Later capacity input may let its retry succeed.',
        commands=[common['stack'].get('live_probe',{}).get('command')]))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--components-only', action='store_true', help='Supporting tests only; never an acceptance pass.')
    parser.add_argument('--follow-up-3', action='store_true', help='Remeasure D-060 and SECOPS-11; preserve and review historical PITR.')
    parser.add_argument('--follow-up-4', action='store_true', help='Corrected rate classes, live publication halves and native scale; preserve historical PITR.')
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
    if args.follow_up_3:
        compatibility = review_pitr(ROOT, started)
        write_json(PUBLIC / 'pitr-compatibility.json', compatibility)
        if compatibility['status'] != 'passed':
            raise SystemExit('PITR dependency/assertion changed; inspect compatibility record before deciding on a new drill')
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
                stale_paths = [PUBLIC / 'live.json']
                if not (args.follow_up_3 or args.follow_up_4):
                    stale_paths += [PUBLIC / 'restore.json', PUBLIC / 'pitr.json', PUBLIC / 'pitr-source.json', PRIVATE / 'pitr-manifest.json', PRIVATE / 'retained.json']
                for stale in stale_paths:
                    stale.unlink(missing_ok=True)
                try:
                    up, raw = command('e2e-up', ['make', '-f', 'mk/e2e.mk', 'e2e-up'], env, 1200)
                    stack['up'] = up
                    if up['exit_code']:
                        stack['reason'] = 'Assigned dev stack failed bring-up; private diagnostics retained.'
                    else:
                        live_command = ['go', 'test', '-json', '-tags', 'e2e,integration', '-p', '1', '-parallel', '2', '-count=1', '-timeout', '55m' if args.follow_up_4 else '15m', '-run', '^TestAT28_AT31_Followup4$' if args.follow_up_4 else '^TestAT28_AT31_LiveStack$', './tests/e2e/at/secops']
                        probe, output = command('live-probe', live_command, dict(env, FVPN_SECOPS_LIVE='1', FVPN_SECOPS_LOGIN_BURST=login_burst, FVPN_SECOPS_FOLLOWUP3='1' if args.follow_up_3 else '0'), 3360 if args.follow_up_4 else 960)
                        stack['live_probe'] = dict(probe, **summarize_go(output, ['TestAT28_AT31_Followup4' if args.follow_up_4 else 'TestAT28_AT31_LiveStack'], probe['exit_code']))
                        if args.follow_up_3 or args.follow_up_4:
                            stack['isolated_restore'] = {'status': 'historical record preserved', 'compatibility': 'historical; no new compatibility assertion in Follow-up 4' if args.follow_up_4 else 'pitr-compatibility.json', 'new_restore_executed': False}
                        elif (PRIVATE / 'pitr-manifest.json').is_file() and (PRIVATE / 'retained.json').is_file():
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
                  probe_source_sha256=live.get('probe_source_sha256', source_digest), build_digests=builds+deployed_builds, image_digests=live.get('image_digests', []),
                  policy_revision=live.get('policy_revision'), node_incarnation=live.get('node_incarnation'), stack=stack)
    if args.follow_up_3:
        write_followup3_records(common, live)
    if args.follow_up_4:
        write_followup4_records(common, live)
    write_json(PUBLIC / 'components.json', {'schema_version': 1, 'run_started_at': started, 'run_finished_at': utc(), 'suites': records})
    write_json(PUBLIC / 'secret-scan.json', {'schema_version': 1, 'scope': 'Captured component subprocess streams plus all stack and dynamically created native-peer container stdout/stderr streams, produced node/member log files, and all audit_events rows. Known live secret canaries plus generic markers; not a claim about unexercised code paths.',
               'streams': [r['scan'] for r in records], 'stack_streams': live.get('secret_scan', []), 'canaries': live.get('known_secret_canaries'),
               'audit_rows': live.get('audit_rows_scanned'), 'full_stack_scan_status': 'passed' if live.get('secret_scan') and all(x['read_ok'] and not x['known_secret_matches'] for x in live['secret_scan']) else 'blocked'})
    for at, requirements in REQUIREMENTS.items():
        if at == 'AT30' and (args.follow_up_3 or args.follow_up_4):
            # Keep its actual run timestamp, images, backup IDs and timings.
            # Current component results are in components.json; no old backup
            # can be attributed to the new stack or its incarnation.
            continue
        related = [r for r in records if at in r['acceptance_cases']]
        component_status = 'failed' if any(r['status'] == 'failed' for r in related) else ('passed' if related and all(r['status'] == 'passed' for r in related) else 'blocked')
        status = 'passed' if at == 'AT28' and component_status == 'passed' and stack.get('live_probe', {}).get('status') == 'passed' and live.get('security_matrix_complete') else 'blocked'
        blockers = list(GAPS[at])
        if component_status == 'failed':
            blockers.insert(0, 'Supporting test failure: '+', '.join(r['id'] for r in related if r['status'] == 'failed')+'; see components.json and requests.md. A failed owner test is not a skipped precondition.')
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
            data['serving'] = {k:live.get(k) for k in ('node_service_precondition','publication_request','publication_operation','publication_negative','publication_positive','serving_flow_positive','publication_observation_before','publication_observation_after','forward_baseline_generation_zero','member_accept_rules','watchdog_heartbeat_present','counted_flow','flow_probe','serving_checks','node_after_publication','watchdog_state','configured_updater_recovery_file_present','short_node_lease_seconds','publication_without_capacity','publication_empty_bindings','publication_with_device')}
        if at == 'AT30':
            data['recovery'] = live.get('recovery', {})
            if (PUBLIC / 'pitr.json').exists() and stack.get('isolated_restore',{}).get('exit_code') == 0:
                data['physical_restore_subset'] = json.loads((PUBLIC / 'pitr.json').read_text())
                data['evidence'].extend(['pitr.json', 'pitr-source.json'])
        if at == 'AT31':
            data['capacity'] = dict(profile='load-profile.json', full_profile_executed=False, candidate_hardware=False, measured_subset=live.get('capacity_subset'), grants=live.get('grant_projection'), requested_devices=60, authorized_devices=live.get('authorized_devices', 0), concurrent_sign_in=live.get('concurrent_sign_in'), synthetic_ledger_ack=live.get('usage_final_ack'), login_41st_request=live.get('login_41st_request'))
            data['capacity']['login_subject_limit'] = live.get('login_subject_limit')
            data['capacity']['login_settings'] = live.get('login_settings')
            data['capacity']['rate_classes'] = live.get('rate_classes')
            data['capacity']['native_scale'] = live.get('native_scale')
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
