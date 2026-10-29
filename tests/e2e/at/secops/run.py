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
PORT_BASE = '23000'

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
    'AT28': ['No running production stack: full per-object family IDOR matrix and complete stack log/audit-stream scan are blocked. Middleware role checks are not object authorization proof.'],
    'AT29': ['Serving-node precondition: signed policy generation >0, active member binding, timed allowset, successful established TCP/UDP traffic. Prior main evidence had generation=0 and zero member accepts; this run cannot re-observe it.', 'No documented stack hook for a short member service/grace/grant deadline and isolated wall-clock fault injection. Never change the shared host clock.'],
    'AT30': ['No isolated DB/WAL restore hook or backup/WAL configuration in the documented dev stack. The existing restore-ledger test seeds two isolated databases; it is NOT a physical backup restore.', 'No full-stack coordinated provider/NetBird/identity-journal restore checkpoint. RPO/RTO remain null. No admin database or live stack is restored.'],
    'AT31': ['Stack unavailable; 20/60 load profile is specified but not executed. No sign-in/heartbeat/sample/policy/relay latency or resource measurements are claimed.', 'Candidate hardware, physical arm64 and real member platforms require operator runs.'],
    'AT32': ['Document mismatches remain outside owned paths. Three-source physical end-to-end comparison and complete Wave 4 results are unavailable on this checkout.'],
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
        parser.error('this task is assigned prefix w4-secops and port base 23000; no substitute is allowed')
    PRIVATE.mkdir(parents=True, exist_ok=True)
    PUBLIC.mkdir(parents=True, exist_ok=True)
    (ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
    os.umask(0o077)
    revision = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    source_digest = hashlib.sha256(b''.join(p.read_bytes() for p in sorted(HERE.glob('*.go')))).hexdigest()
    records, builds = [], []
    started = utc()
    with tempfile.TemporaryDirectory(prefix='secops.', dir=ROOT / '.agent/tmp') as scratch:
        env = dict(os.environ, GOTOOLCHAIN='local', CGO_ENABLED='0', TMPDIR=scratch, GOTMPDIR=scratch,
                   FVPN_E2E_PREFIX=PREFIX, FVPN_E2E_PORT_BASE=PORT_BASE, FVPN_E2E_STATE=str(ROOT / '.agent/e2e-state-w4-secops'))
        stack = {'status': 'blocked', 'reason': 'Full-stack probes not run in components-only mode.'}
        if not args.components_only:
            before = inventory(env)
            stack['inventory_before'] = before
            if any(v['exit_code'] or v['count'] for v in before.values()):
                stack['reason'] = 'Assigned namespace is occupied or Docker inventory unavailable; no bring-up or teardown authorized by this runner.'
            else:
                try:
                    up, raw = command('e2e-up', ['make', '-f', 'mk/e2e.mk', 'e2e-up'], env, 1200)
                    stack['up'] = up
                    if b'port base must be 18000 plus a multiple of 6000' in raw:
                        stack['reason'] = 'needs devstack follow-up 2: parseLayout rejects the assigned port base 23000; accepts only 18000 + 6000*n. No alternate base used.'
                    elif up['exit_code']:
                        stack['reason'] = 'Assigned dev stack failed bring-up; private diagnostics retained.'
                    else:
                        # The stock suite has private lab helpers and writes to unowned
                        # docs/e2e/runs. It is not run here without an export/output hook.
                        stack['reason'] = 'Stack started; reusable secops fixture/evidence hooks are not exposed. See requests.md. No serving state or load pass inferred.'
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
    common = dict(schema_version=1, run_started_at=started, run_finished_at=utc(), source_revision=revision,
                  probe_source_sha256=source_digest, build_digests=builds, image_digests=[],
                  policy_revision=None, node_incarnation=None,
                  unavailable_metadata_reason='No deployed node or image was observed in this run. Test binaries are identified separately.',
                  stack=stack)
    write_json(PUBLIC / 'components.json', {'schema_version': 1, 'run_started_at': started, 'run_finished_at': utc(), 'suites': records})
    write_json(PUBLIC / 'secret-scan.json', {'schema_version': 1, 'scope': 'All captured component subprocess output; in-memory application log/response/audit assertions are named in components.json. No stack streams existed.',
               'streams': [r['scan'] for r in records], 'stack_streams': [], 'full_stack_scan_status': 'blocked'})
    for at, requirements in REQUIREMENTS.items():
        related = [r for r in records if at in r['acceptance_cases']]
        component_status = 'failed' if any(r['status'] == 'failed' for r in related) else ('passed' if related and all(r['status'] == 'passed' for r in related) else 'blocked')
        data = dict(common, id=at, spec=['§19.1', '§19 '+at], requirements=requirements, status='blocked', verdict='not passed',
                    expected=EXPECTED[at], actual={'component_status': component_status, 'component_suites': [r['id'] for r in related], 'full_stack_executed': False},
                    blockers=[stack['reason']]+GAPS[at], commands=[r['command'] for r in related],
                    operator_run={'status': 'needs operator run', 'script': 'operator-runs.md#'+at.lower()},
                    evidence=['components.json', 'secret-scan.json', 'requests.md'])
        if at == 'AT30':
            data['recovery'] = dict(backup_id=None, wal_checkpoint=None, restored_checkpoint=None, rpo_seconds=None, rto_seconds=None, model_only=True)
        if at == 'AT31':
            data['capacity'] = dict(profile='load-profile.json', executed=False, candidate_hardware=False, read_p50_ms=None, read_p95_ms=None, read_p99_ms=None, policy_latency_seconds=None, usage_latency_seconds=None, cpu=None, memory=None, io=None)
        if at == 'AT32':
            data['documentation_findings'] = [f['id'] for f in review['findings']]
            data['evidence'].append('scope-review.json')
        write_json(PUBLIC / (at+'.json'), data)
        lines = [f'# {at}: blocked', '', EXPECTED[at], '', f'Run UTC: {started}. Source: `{revision}`.', '',
                 f'Supporting component result: **{component_status}**. Full-stack acceptance is **not passed**.', '',
                 'Build digests, command argv, UTC times, expected/actual and unavailable metadata are in ['+at+'.json]('+at+'.json).', '',
                 'Blocking conditions:', ''] + ['- '+b for b in data['blockers']]
        lines += ['', 'Evidence: [component tests](components.json), [secret scan](secret-scan.json), [requests](requests.md), [operator script](operator-runs.md#'+at.lower()+').', '']
        (PUBLIC / (at+'.md')).write_text('\n'.join(lines))
    # A component-only check is useful in CI, but full acceptance never turns
    # missing measurements, missing required tests, or skips into success.
    ok = all(r['status'] == 'passed' for r in records) and all(b['exit_code'] == 0 for b in builds)
    print('evidence: docs/e2e/at/secops; AT28–AT32 blocked', flush=True)
    return (0 if ok else 1) if args.components_only else 2


if __name__ == '__main__':
    raise SystemExit(main())
