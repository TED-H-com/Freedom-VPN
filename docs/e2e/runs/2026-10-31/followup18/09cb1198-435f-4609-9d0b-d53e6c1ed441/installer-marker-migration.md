# installer-marker-migration

D-116 legacy marker migrated by same-release install

Status: pass

Run: 09cb1198-435f-4609-9d0b-d53e6c1ed441

Source: 84b6a342970ffc651aafda81d485451529902233

Spec: AT07, ENR-05, D-116

- before_ruleset: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 2}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 279, "packets": 8, "bytes": 1519}}, {"counter": {"family": "inet", "name": "nd_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 280, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 281, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 282, "packets": 9, "bytes": 1153}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 370, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 371, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 372, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 373, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 374, "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 375, "comment": "fvpn_class=01a1917e-1662-7571-8a9a-60da4e86666d:a753", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 376, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 377, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 378, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 379, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 380, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 381, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}]}

- legacy_ruleset: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 279, "packets": 8, "bytes": 1519}}, {"counter": {"family": "inet", "name": "nd_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 280, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 281, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 282, "packets": 9, "bytes": 1153}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 370, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 371, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 372, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 373, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "owner", "handle": 382, "comment": "managed-by=fvpn-node-install"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 374, "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 375, "comment": "fvpn_class=01a1917e-1662-7571-8a9a-60da4e86666d:a753", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 376, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 377, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 378, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 379, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 380, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 381, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}]}

- legacy_check: {
  "status": "drifted",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a1917d-cdde-777b-9e4c-6caa1495d8d8.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T10:25:14.398201352Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a1917d-cdde-777b-9e4c-6caa1495d8d8\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
  "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686",
  "release_source": "updater_journal",
  "install_record": {
    "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686",
    "release_sequence": "10",
    "product_version": "1.1.0",
    "arch": "amd64",
    "installed_at": "2026-10-31T10:25:11Z",
    "netbird_package_installed": false
  },
  "updater": {
    "current_release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "state": "",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
      "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686"
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
    "pid": 2263,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 2585329
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a1917d-eedf-75ab-a77e-babbc7fb81dc",
    "node_id": "01a1917d-e899-7f0f-a666-35b7f0ddd137"
  },
  "usage_store_present": true,
  "netbird": {
    "version": "0.82.0",
    "daemon": "connected"
  },
  "installer_marker": {
    "table": "fvpn_install",
    "present": false,
    "legacy_owner": true
  },
  "drifts": [
    "installer_marker",
    "unit fvpn-watchdog.service inactive",
    "unit fvpn-agent.service inactive",
    "unit fvpn-updater.timer inactive"
  ]
}
drifted: installer_marker, unit fvpn-watchdog.service inactive, unit fvpn-agent.service inactive, unit fvpn-updater.timer inactive

- migration: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a1917d-eedf-75ab-a77e-babbc7fb81dc","NodeID":"01a1917d-e899-7f0f-a666-35b7f0ddd137","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"669eaca2922a922794cc66f21498dbb09bb98c2c6d5b42ec7e983818df0e74ed","cert_sha256":"70bd5bfc06daf6b2bee486aab5ae422fef4ae68420e45105accb279d453986d4","node_json_sha256":"c0843540cd6c7cf3e79175baadbf83d724e32863cbd528581670f52d3a79f788","usage_db_sha256":"60044aef91e8aef82daec820cf289cec923ac7a3bee944779c8925a02cbf1c0c","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8"},"Hashes":{"key_sha256":"669eaca2922a922794cc66f21498dbb09bb98c2c6d5b42ec7e983818df0e74ed","cert_sha256":"70bd5bfc06daf6b2bee486aab5ae422fef4ae68420e45105accb279d453986d4","node_json_sha256":"c0843540cd6c7cf3e79175baadbf83d724e32863cbd528581670f52d3a79f788","usage_db_sha256":"60044aef91e8aef82daec820cf289cec923ac7a3bee944779c8925a02cbf1c0c","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8"},"Services":{"fvpn-agent.service":"inactive","fvpn-updater.service":"inactive","fvpn-updater.timer":"inactive","fvpn-watchdog.service":"inactive"}}
- after_ruleset: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 279, "packets": 8, "bytes": 1519}}, {"counter": {"family": "inet", "name": "nd_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 280, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 281, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 282, "packets": 9, "bytes": 1153}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 370, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 371, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 372, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 373, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 374, "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 375, "comment": "fvpn_class=01a1917e-1662-7571-8a9a-60da4e86666d:a753", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 376, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 377, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 378, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 379, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 380, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 381, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 4}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}]}

- migrated_check: drifted: unit fvpn-watchdog.service inactive, unit fvpn-agent.service inactive, unit fvpn-updater.timer inactive
{
  "status": "drifted",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a1917d-cdde-777b-9e4c-6caa1495d8d8.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T10:25:42.741157175Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a1917d-cdde-777b-9e4c-6caa1495d8d8\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
  "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686",
  "release_source": "updater_journal",
  "install_record": {
    "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686",
    "release_sequence": "10",
    "product_version": "1.1.0",
    "arch": "amd64",
    "installed_at": "2026-10-31T10:25:11Z",
    "netbird_package_installed": false
  },
  "updater": {
    "current_release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
    "state": "",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8",
      "manifest_sha256": "b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686"
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
    "pid": 3500,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 2587979
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a1917d-eedf-75ab-a77e-babbc7fb81dc",
    "node_id": "01a1917d-e899-7f0f-a666-35b7f0ddd137"
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
