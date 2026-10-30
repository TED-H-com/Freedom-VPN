"""Revalidate historical AT30 evidence without claiming another restore run."""
import hashlib
import json
import subprocess


def review(root, checked_at):
    path = root / 'docs/e2e/at/secops/pitr.json'
    raw = path.read_bytes()
    record = json.loads(raw)
    boundary = record['physical_boundary']
    dependencies = ['deploy/backup/restore-pitr.sh', 'deploy/backup/postgres/Dockerfile',
                    'deploy/backup/post-restore-checklist.sql', 'db/migrations']
    argv = ['git', 'diff', '--name-only', record['source_revision']+'..HEAD', '--'] + dependencies
    changed = subprocess.check_output(argv, cwd=root, text=True).splitlines()
    migrations = sorted((root / 'db/migrations').glob('*.sql'))
    assertions = {
        'historical_pass': record['status'] == 'passed',
        'target_ack_and_ledger': boundary['ack'] == '7' and boundary['ledger_rows'] == 7,
        'pre_target_grant_operation_audit': boundary['pre_target_grant_present'] and boundary['pre_target_operation_present'] and boundary['pre_target_audit_rows_unchanged'] == 2,
        'post_target_writes_absent': boundary['post_target_events'] == 0 and all(boundary[k] for k in ('post_target_grant_absent', 'post_target_operation_absent', 'post_target_audit_absent')),
        'deduplicated_replay': record['final_ack'] == '10' and record['ledger_rows'] == 10 and record['final_member_total_bytes'] == '3000' and record['replayed_accepted'] == 3 and record['replayed_duplicate'] == 7 and record['second_replay_duplicate'] == 10,
        'entitlements_preserved': record['entitlements_unchanged'],
        'isolated_and_cleaned': record['network'] == 'none' and record['published_ports'] == [] and record['restore_objects_remaining'] == 0,
        'full_service_limits_explicit': record['full_service_rpo_seconds'] is None and record['full_service_rto_seconds'] is None and not record['external_reconciliation'],
        'commands_and_migrations_unchanged': not changed,
        'current_migration_count_matches': len(migrations) == boundary['applied_migrations'],
    }
    return dict(schema_version=1, checked_at=checked_at, status='passed' if all(assertions.values()) else 'blocked',
                kind='historical evidence and command compatibility review; not a new physical restore',
                reviewed_revision=subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
                historical_run_utc=record['utc'], historical_source_revision=record['source_revision'],
                pitr_sha256=hashlib.sha256(raw).hexdigest(), changed_dependencies=changed,
                migration_files=len(migrations), assertions=assertions, commands=[argv],
                attribution='Backups follow-up 4 / PR #111 removed literal migration IDs from the handover example; the restore command, Dockerfile, SQL checklist and migrations used by this run are unchanged.',
                expected='Historical PITR assertions valid; dependent commands and migrations unchanged',
                actual='Review assertions recorded individually; original RTO/RPO and build/image provenance retained in pitr.json')
