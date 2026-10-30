# backup

Archived WAL and isolated physical PITR

Status: pass

Run: dc2f6353-89ec-434c-bc68-567066d9fe51

Source: b15ff057cd5e91e227477de6a09d977931fa62d8

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 228
- archive_check_exit: 0
- physical_full_milliseconds: 4410
- physical_full_exit: 0
- repository_info_milliseconds: 47
- repository_info_exit: 0
- backup_label: 20261030-041713F
- backup_type: full
- target_utc: 2026-10-30T04:17:18.203076Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 323
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 7080
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 58
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 12191
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
