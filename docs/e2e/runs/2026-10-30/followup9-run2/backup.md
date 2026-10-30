# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 65712989-0b05-4117-9a50-689499cd31be

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 217
- archive_check_exit: 0
- physical_full_milliseconds: 3701
- physical_full_exit: 0
- repository_info_milliseconds: 37
- repository_info_exit: 0
- backup_label: 20261030-072457F
- backup_type: full
- target_utc: 2026-10-30T07:25:01.276001Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 322
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4496
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 55
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 12
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8872
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
