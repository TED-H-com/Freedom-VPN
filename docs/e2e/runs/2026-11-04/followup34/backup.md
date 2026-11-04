# backup

Archived WAL and isolated physical PITR

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 326
- archive_check_exit: 0
- physical_full_milliseconds: 3526
- physical_full_exit: 0
- repository_info_milliseconds: 101
- repository_info_exit: 0
- backup_label: 20261104-053830F
- backup_type: full
- target_utc: 2026-11-04T05:38:34.344466Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 447
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 65
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4049
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 112
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 65
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8720
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
