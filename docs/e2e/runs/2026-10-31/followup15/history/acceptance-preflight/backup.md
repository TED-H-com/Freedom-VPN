# backup

Archived WAL and isolated physical PITR

Status: pass

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 218
- archive_check_exit: 0
- physical_full_milliseconds: 4648
- physical_full_exit: 0
- repository_info_milliseconds: 47
- repository_info_exit: 0
- backup_label: 20261031-035141F
- backup_type: full
- target_utc: 2026-10-31T03:51:46.096081Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 293
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 6866
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 61
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 12
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 12683
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
