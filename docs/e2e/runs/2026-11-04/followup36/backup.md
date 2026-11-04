# backup

Archived WAL and isolated physical PITR

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 263
- archive_check_exit: 0
- physical_full_milliseconds: 3276
- physical_full_exit: 0
- repository_info_milliseconds: 68
- repository_info_exit: 0
- backup_label: 20261104-143350F
- backup_type: full
- target_utc: 2026-11-04T14:33:54.182726Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 353
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 33
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 3926
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 77
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 32
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8050
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
- timed_section: backup
