# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 5de03570-0436-477d-9847-6f6984432bd8

Source: a7a712aab8a67fcc33514a9ac76cb877a90c8143

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 223
- archive_check_exit: 0
- physical_full_milliseconds: 3271
- physical_full_exit: 0
- repository_info_milliseconds: 46
- repository_info_exit: 0
- backup_label: 20261030-030715F
- backup_type: full
- target_utc: 2026-10-30T03:07:19.047194Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 299
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 12
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4085
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 55
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8828
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
