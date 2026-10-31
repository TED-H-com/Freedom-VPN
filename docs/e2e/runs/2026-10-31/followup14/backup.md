# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 233
- archive_check_exit: 0
- physical_full_milliseconds: 4225
- physical_full_exit: 0
- repository_info_milliseconds: 37
- repository_info_exit: 0
- backup_label: 20261031-004211F
- backup_type: full
- target_utc: 2026-10-31T00:42:16.131601Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 1344
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 12
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 7549
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 56
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 13921
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
