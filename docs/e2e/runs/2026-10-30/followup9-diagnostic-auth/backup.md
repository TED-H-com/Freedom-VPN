# backup

Archived WAL and isolated physical PITR

Status: pass

Run: 376e5f84-535e-4ddd-902b-985852d4691b

Source: 724ab19fd46a974a66d138ac0cddcd7f803bafb7

Spec: §17.2, §17.3, §19 AT30

- enrollment_before: 201
- archive_check_milliseconds: 224
- archive_check_exit: 0
- physical_full_milliseconds: 2895
- physical_full_exit: 0
- repository_info_milliseconds: 47
- repository_info_exit: 0
- backup_label: 20261030-064755F
- backup_type: full
- target_utc: 2026-10-30T06:47:58.193140Z
- enrollment_after: 201
- primary_marker_count: 2
- wal_switch_archive_milliseconds: 268
- wal_switch_archive_exit: 0
- primary_image_milliseconds: 11
- primary_image_exit: 0
- physical_pitr_restore_milliseconds: 3634
- physical_pitr_restore_exit: 0
- restored_readback_milliseconds: 56
- restored_readback_exit: 0
- restored_before_after_recovery_archive: 1|0|f|off
- restore_isolation_milliseconds: 11
- restore_isolation_exit: 0
- network_and_published_ports: none|0
- total_milliseconds: 7172
- scope: Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.
