# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 273496b3-9897-43b0-b289-a571d1d14561

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 221
- archive_check_exit: 0
- physical_full_milliseconds: 2982
- physical_full_exit: 0
- repository_info_milliseconds: 45
- repository_info_exit: 0
- backup_label: 20261030-101335F
- backup_type: full
- target_utc: 2026-10-30T10:13:38.121656Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 280
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 3533
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 56
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 7163
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
