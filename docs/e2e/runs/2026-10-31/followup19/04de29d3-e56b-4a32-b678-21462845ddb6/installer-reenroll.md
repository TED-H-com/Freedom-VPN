# installer-reenroll

Controlled reinstall, retained identity and retired certificate refusal

Status: pass

Run: 04de29d3-e56b-4a32-b678-21462845ddb6

Source: fb5b8fcee72bd71b3bd0a4536af9049df4055096

Spec: ENR-06, AT07, §6.3

- revoke: 202
- reenroll: {"Exit":0,"Status":"reenrolled","Code":"baseline","Incarnation":"01a19251-5dd5-7c03-9a45-dedda9abdab4","NodeID":"01a1924d-ad8a-75f0-9fab-9ac72ddcc32f","RetiredMode":"700","RetiredHashes":{"key_sha256":"089481b80da5d1a59010dc3a87074576a6b4cf6337fbf075f560818bf4c6d518","cert_sha256":"0416b71e0b602e3840c6af2bce5587b83fe79d0b5ad46d009b297034ab90df8c","node_json_sha256":"1ad2927c061be6a7df7ba221187f17c5973a9aff6bcff2556d4ee7b48a19e00e"},"BoundStateBefore":{"agent_db_sha256":"cd0561ff6f032ae31b37205d8fc378c7bfa852157b5b506c0331ffc8e415771c","agent_db_shm_sha256":"absent","agent_db_wal_sha256":"absent","desired_state_json_sha256":"da52e64c6f47ae81388ca8fd9c022629efe98631a147b25136b4b192a75156b0"},"RetiredBoundState":{"agent_db_sha256":"cd0561ff6f032ae31b37205d8fc378c7bfa852157b5b506c0331ffc8e415771c","agent_db_shm_sha256":"absent","agent_db_wal_sha256":"absent","desired_state_json_sha256":"da52e64c6f47ae81388ca8fd9c022629efe98631a147b25136b4b192a75156b0"},"Message":"identity retired under identity.retired; usage store kept","Before":{"key_sha256":"089481b80da5d1a59010dc3a87074576a6b4cf6337fbf075f560818bf4c6d518","cert_sha256":"0416b71e0b602e3840c6af2bce5587b83fe79d0b5ad46d009b297034ab90df8c","node_json_sha256":"1ad2927c061be6a7df7ba221187f17c5973a9aff6bcff2556d4ee7b48a19e00e","usage_db_sha256":"e2f7a49e6ee16fdb87f6ed11cc4b9cbeef1e6435617e3e508145ffb9a2478019","usage_wal_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1924d-93c5-7e6e-879f-528a14cc7a18"},"Hashes":{"key_sha256":"4c1715f90cfb2ad6efec9ba0702388a0dc97ccb318fb44e363c277f3379f60b0","cert_sha256":"1233cd88bc5495bf93a9de3d0198ac4f6880cb085ebf16c733bbc4e582b615e0","node_json_sha256":"9b9b08a17b43cae9427901d239fc59be9af9c2e672d11e2126ff94a7e4a97d02","usage_db_sha256":"27df2f33ab53e71e32091774d6539b19d0b4c23a2c59f8ab4964b1db8f6dca43","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1924d-93c5-7e6e-879f-528a14cc7a18"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- retired_certificate_http: {"direct":"403","proxy":"403"}
- canonical_origin_events_retained: true
- post_reenroll_seed_check: drifted: unit fvpn-watchdog.service inactive, unit fvpn-agent.service inactive, unit fvpn-updater.timer inactive
{
  "status": "drifted",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a1924d-93c5-7e6e-879f-528a14cc7a18.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a1924d-93c5-7e6e-879f-528a14cc7a18",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T14:16:13.937476413Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a1924d-93c5-7e6e-879f-528a14cc7a18\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18",
  "manifest_sha256": "0f32fde0fd482890b50ebf9888470e789b312ff89391d763d4c1060f721b371f",
  "release_source": "updater_journal",
  "install_record": {
    "release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18",
    "manifest_sha256": "0f32fde0fd482890b50ebf9888470e789b312ff89391d763d4c1060f721b371f",
    "release_sequence": "10",
    "product_version": "1.1.0",
    "arch": "amd64",
    "installed_at": "2026-10-31T14:12:08Z",
    "netbird_package_installed": false
  },
  "updater": {
    "current_release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18",
    "state": "failed",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18",
      "manifest_sha256": "0f32fde0fd482890b50ebf9888470e789b312ff89391d763d4c1060f721b371f"
    }
  },
  "units": {
    "fvpn-agent.service": "inactive",
    "fvpn-netbird.service": "active",
    "fvpn-node-netbird-up.service": "active",
    "fvpn-updater-recover.service": "active",
    "fvpn-updater.service": "inactive",
    "fvpn-updater.timer": "inactive",
    "fvpn-watchdog.service": "inactive"
  },
  "directories": [
    {
      "path": "/etc/vpn-node",
      "mode": "0755",
      "expected": "0755",
      "ok": true
    },
    {
      "path": "/var/lib/vpn-node",
      "mode": "0700",
      "expected": "0700",
      "ok": true
    },
    {
      "path": "/var/lib/vpn-node/identity",
      "mode": "0700",
      "expected": "0700",
      "ok": true
    },
    {
      "path": "/var/lib/vpn-node/state",
      "mode": "0700",
      "expected": "0700",
      "ok": true
    },
    {
      "path": "/var/lib/vpn-node/update-journal",
      "mode": "0700",
      "expected": "0700",
      "ok": true
    },
    {
      "path": "/var/lib/vpn-node/update-staging",
      "mode": "0700",
      "expected": "0700",
      "ok": true
    },
    {
      "path": "/var/lib/vpn-node/install",
      "mode": "0700",
      "expected": "0700",
      "ok": true
    },
    {
      "path": "/opt/vpn-node",
      "mode": "0755",
      "expected": "0755",
      "ok": true
    },
    {
      "path": "/opt/vpn-node/releases",
      "mode": "0755",
      "expected": "0755",
      "ok": true
    },
    {
      "path": "/run/vpn-node",
      "mode": "0755",
      "expected": "0755",
      "ok": true
    }
  ],
  "lock": {
    "held": false,
    "pid": 9076,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 3971129
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a19251-5dd5-7c03-9a45-dedda9abdab4",
    "node_id": "01a1924d-ad8a-75f0-9fab-9ac72ddcc32f"
  },
  "usage_store_present": true,
  "netbird": {
    "version": "0.82.0",
    "daemon": "connected"
  },
  "installer_marker": {
    "table": "fvpn_install",
    "present": true,
    "legacy_owner": false
  },
  "drifts": [
    "unit fvpn-watchdog.service inactive",
    "unit fvpn-agent.service inactive",
    "unit fvpn-updater.timer inactive"
  ]
}

- usage_retained: true
- bound_state_retired: true
- restoration: true
