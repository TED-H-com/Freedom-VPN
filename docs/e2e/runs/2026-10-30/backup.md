# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 76f0725c-2264-4021-a553-60f6c921e32b

Source: 1b2f358880d38ae643865edc2bb6f19756748456

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 219
- archive_check_exit: 0
- physical_full_milliseconds: 3187
- physical_full_exit: 0
- repository_info_milliseconds: 47
- repository_info_exit: 0
- backup_label: 20261030-172709F
- backup_type: full
- target_utc: 2026-10-30T17:27:12.453629Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 292
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 12
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 4398
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 59
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 8242
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
