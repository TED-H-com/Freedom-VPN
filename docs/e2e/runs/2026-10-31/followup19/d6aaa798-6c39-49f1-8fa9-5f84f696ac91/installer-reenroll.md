# installer-reenroll

Controlled reinstall, retained identity and retired certificate refusal

Status: pass

Run: d6aaa798-6c39-49f1-8fa9-5f84f696ac91

Source: a9ad1c471720a2df27f7d4aaa3a0ae1a565fa2cd

Spec: ENR-06, AT07, §6.3

- revoke: 202
- reenroll: {"Exit":0,"Status":"reenrolled","Code":"baseline","Incarnation":"01a19249-968e-755c-9e61-7bd24ebd4f80","NodeID":"01a19246-b349-7974-9962-fee669622101","RetiredMode":"700","RetiredHashes":{"key_sha256":"7118ad330d921e3bc30b01927a6168eb9c26668fd1cc7f8220f4e2ca68c5aa3a","cert_sha256":"b7e1187bf0c0ce1f55bf02b11ba79ee861f527df1d71c8af4de470eb780ee436","node_json_sha256":"85d982754f5c1b3d5b59d51dac606b24a2771b981d3cc282e00043ccbea91165"},"BoundStateBefore":{"agent_db_sha256":"6bfe1c3c86f7eb70a024a377f5270dc86f845de751169d16bdf20e42ac92b491","agent_db_shm_sha256":"absent","agent_db_wal_sha256":"absent","desired_state_json_sha256":"1c6898460bc2fde5a1f0783a3d620d27721069e75dfec95c92e1d09234570d1f"},"RetiredBoundState":{"agent_db_sha256":"6bfe1c3c86f7eb70a024a377f5270dc86f845de751169d16bdf20e42ac92b491","agent_db_shm_sha256":"absent","agent_db_wal_sha256":"absent","desired_state_json_sha256":"1c6898460bc2fde5a1f0783a3d620d27721069e75dfec95c92e1d09234570d1f"},"Message":"identity retired under identity.retired; usage store kept","Before":{"key_sha256":"7118ad330d921e3bc30b01927a6168eb9c26668fd1cc7f8220f4e2ca68c5aa3a","cert_sha256":"b7e1187bf0c0ce1f55bf02b11ba79ee861f527df1d71c8af4de470eb780ee436","node_json_sha256":"85d982754f5c1b3d5b59d51dac606b24a2771b981d3cc282e00043ccbea91165","usage_db_sha256":"a0d5495444b7980edcd62528ed5c450711da9e5cc7b54cb2ac31179a6b02449a","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19246-805d-71cc-9dc5-86452d407914"},"Hashes":{"key_sha256":"804cca0cd84eded2ea1365fe0e0133b2abb72bf2699310b33e8638db0c5b4c07","cert_sha256":"4f935c2055a56f4afe92a1a1914d6680b9fdfbb602940516888785d6bc6d6bf3","node_json_sha256":"d6b2105ca24bb24cd7524c4a321d6ca8d3080ca4bec9ee6535c39f026432957e","usage_db_sha256":"b0bf5d796790c77f01f77eca8000929a7a4ab13bd0d967066dc83a608b3b592e","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19246-805d-71cc-9dc5-86452d407914"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- retired_certificate_http: {"direct":"403","proxy":"403"}
- canonical_origin_events_retained: true
- post_reenroll_seed_check: {
  "status": "drifted",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a19246-805d-71cc-9dc5-86452d407914.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a19246-805d-71cc-9dc5-86452d407914",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T14:07:39.254725931Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a19246-805d-71cc-9dc5-86452d407914\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a19246-805d-71cc-9dc5-86452d407914",
  "manifest_sha256": "a37030bb7c9986724efc836e13966fe197ff6dc82d082cf1b98ed8678f821206",
  "release_source": "updater_journal",
  "install_record": {
    "release_id": "01a19246-7581-7226-bc63-2d51bcab5202",
    "manifest_sha256": "c34a245bc536f401e4e0717929f801903a8d41b7a6118d3d450b5d9b6cadca05",
    "release_sequence": "10",
    "product_version": "1.1.0",
    "arch": "amd64",
    "installed_at": "2026-10-31T14:04:35Z",
    "netbird_package_installed": false
  },
  "updater": {
    "current_release_id": "01a19246-805d-71cc-9dc5-86452d407914",
    "state": "succeeded",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a19246-805d-71cc-9dc5-86452d407914",
      "manifest_sha256": "a37030bb7c9986724efc836e13966fe197ff6dc82d082cf1b98ed8678f821206"
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
    "pid": 8802,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 3919658
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a19249-968e-755c-9e61-7bd24ebd4f80",
    "node_id": "01a19246-b349-7974-9962-fee669622101"
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
drifted: unit fvpn-watchdog.service inactive, unit fvpn-agent.service inactive, unit fvpn-updater.timer inactive

- usage_retained: true
- bound_state_retired: true
- restoration: true
