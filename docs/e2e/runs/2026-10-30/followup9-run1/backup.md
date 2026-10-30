# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 8fb30ba4-b012-42d9-ba79-4ecb46adff53

Source: 943e2a15aac3a45990abdbcdd25957376fa997e0

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 221
- archive_check_exit: 0
- physical_full_milliseconds: 3638
- physical_full_exit: 0
- repository_info_milliseconds: 47
- repository_info_exit: 0
- backup_label: 20261030-070657F
- backup_type: full
- target_utc: 2026-10-30T07:07:01.183121Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 287
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4039
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 50
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8330
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
