# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 14231a83-130d-40fb-a9b4-0d3851520b67

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 224
- archive_check_exit: 0
- physical_full_milliseconds: 2915
- physical_full_exit: 0
- repository_info_milliseconds: 47
- repository_info_exit: 0
- backup_label: 20261030-103811F
- backup_type: full
- target_utc: 2026-10-30T10:38:14.137006Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 287
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 3501
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 56
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 7072
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
