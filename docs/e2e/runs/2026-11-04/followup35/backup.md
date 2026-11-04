# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 266
- archive_check_exit: 0
- physical_full_milliseconds: 3602
- physical_full_exit: 0
- repository_info_milliseconds: 65
- repository_info_exit: 0
- backup_label: 20261104-094034F
- backup_type: full
- target_utc: 2026-11-04T09:40:38.273556Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 349
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 34
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4075
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 79
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 32
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8528
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
- timed_section: backup
