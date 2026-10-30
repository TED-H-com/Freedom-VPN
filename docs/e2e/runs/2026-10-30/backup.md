# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 244
- archive_check_exit: 0
- physical_full_milliseconds: 4722
- physical_full_exit: 0
- repository_info_milliseconds: 48
- repository_info_exit: 0
- backup_label: 20261030-054742F
- backup_type: full
- target_utc: 2026-10-30T05:47:47.441528Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 284
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 7112
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 57
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 12515
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
