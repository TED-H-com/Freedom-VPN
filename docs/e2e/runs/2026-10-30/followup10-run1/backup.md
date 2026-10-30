# backup

Archived WAL and isolated physical PITR

Status: pass

Run: cfda038f-dc18-4371-8047-a4d34954066f

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 228
- archive_check_exit: 0
- physical_full_milliseconds: 3772
- physical_full_exit: 0
- repository_info_milliseconds: 46
- repository_info_exit: 0
- backup_label: 20261030-094901F
- backup_type: full
- target_utc: 2026-10-30T09:49:05.395308Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 320
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4449
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 56
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8912
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
