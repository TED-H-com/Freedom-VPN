# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 22c14cd9-39fc-444b-a7c5-c04cf3ffbff6

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 314
- archive_check_exit: 0
- physical_full_milliseconds: 6540
- physical_full_exit: 0
- repository_info_milliseconds: 46
- repository_info_exit: 0
- backup_label: 20261030-074255F
- backup_type: full
- target_utc: 2026-10-30T07:43:02.317647Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 718
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 12
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4214
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 55
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 11936
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
