# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 217
- archive_check_exit: 0
- physical_full_milliseconds: 5331
- physical_full_exit: 0
- repository_info_milliseconds: 45
- repository_info_exit: 0
- backup_label: 20261031-040754F
- backup_type: full
- target_utc: 2026-10-31T04:08:00.083653Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 361
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 6820
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 51
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 12
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 12967
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
