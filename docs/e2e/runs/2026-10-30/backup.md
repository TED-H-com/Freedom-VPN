# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 3a129919-af3f-4bc9-b54a-8075eae8af45

Source: 79259af1facc1faa798fbb83cc20aa4e74539d1f

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 221
- archive_check_exit: 0
- physical_full_milliseconds: 3557
- physical_full_exit: 0
- repository_info_milliseconds: 37
- repository_info_exit: 0
- backup_label: 20261030-162433F
- backup_type: full
- target_utc: 2026-10-30T16:24:37.101281Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 288
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 12
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 3783
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 56
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 7990
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
