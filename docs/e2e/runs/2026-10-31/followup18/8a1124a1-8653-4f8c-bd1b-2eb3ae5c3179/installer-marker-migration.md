# installer-marker-migration

D-116 legacy marker migrated by same-release install

Status: pass

Run: 8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179

Source: a0089b8386c749100d9bd19b8002dd4d666d065c

Spec: AT07, ENR-05, D-116

- before_ruleset: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 2}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 334, "packets": 8, "bytes": 1519}}, {"counter": {"family": "inet", "name": "nd_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 335, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 336, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 337, "packets": 9, "bytes": 1153}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 425, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 426, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 427, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 428, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 429, "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 430, "comment": "fvpn_class=01a19193-2b52-7583-bcc1-3a52bfcd3219:7e87", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 431, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 432, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 433, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 434, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 435, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 436, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}]}

- legacy_ruleset: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 334, "packets": 8, "bytes": 1519}}, {"counter": {"family": "inet", "name": "nd_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 335, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 336, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 337, "packets": 9, "bytes": 1153}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 425, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 426, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 427, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 428, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "owner", "handle": 437, "comment": "managed-by=fvpn-node-install"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 429, "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 430, "comment": "fvpn_class=01a19193-2b52-7583-bcc1-3a52bfcd3219:7e87", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 431, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 432, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 433, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 434, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 435, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 436, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}]}

- legacy_check: {
  "status": "drifted",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a19192-ad63-7d61-b846-929be061b789.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a19192-ad63-7d61-b846-929be061b789",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T10:48:12.878481166Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a19192-ad63-7d61-b846-929be061b789\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a19192-ad63-7d61-b846-929be061b789",
  "manifest_sha256": "ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c",
  "release_source": "updater_journal",
  "install_record": {
    "release_id": "01a19192-ad63-7d61-b846-929be061b789",
    "manifest_sha256": "ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c",
    "release_sequence": "10",
    "product_version": "1.1.0",
    "arch": "amd64",
    "installed_at": "2026-10-31T10:48:00Z",
    "netbird_package_installed": false
  },
  "updater": {
    "current_release_id": "01a19192-ad63-7d61-b846-929be061b789",
    "state": "",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a19192-ad63-7d61-b846-929be061b789",
      "manifest_sha256": "ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c"
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
    "pid": 2266,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 2723163
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a19192-d10b-7c18-b918-36b9b6fb6a04",
    "node_id": "01a19192-cabd-7a26-8cdc-3049aeda33fa"
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

- migration: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a19192-d10b-7c18-b918-36b9b6fb6a04","NodeID":"01a19192-cabd-7a26-8cdc-3049aeda33fa","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"f6928dc6476e1646f20090580acc9e4256071e794e1275c91a77f9ecb9eb6e7c","cert_sha256":"3e338a0721f4a6c03ae98aa18e761946cfe2d978757c0fba46d14115f09d2c8a","node_json_sha256":"249ee2d4b60c59191244473e4e0669ab005ad0f89a715e4a8732446295700dd8","usage_db_sha256":"9e327e65fe762ba673fad8cd1c01cdf5c6923ed3ac513027ad79902f5360a500","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19192-ad63-7d61-b846-929be061b789"},"Hashes":{"key_sha256":"f6928dc6476e1646f20090580acc9e4256071e794e1275c91a77f9ecb9eb6e7c","cert_sha256":"3e338a0721f4a6c03ae98aa18e761946cfe2d978757c0fba46d14115f09d2c8a","node_json_sha256":"249ee2d4b60c59191244473e4e0669ab005ad0f89a715e4a8732446295700dd8","usage_db_sha256":"9e327e65fe762ba673fad8cd1c01cdf5c6923ed3ac513027ad79902f5360a500","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19192-ad63-7d61-b846-929be061b789"},"Services":{"fvpn-agent.service":"inactive","fvpn-updater.service":"inactive","fvpn-updater.timer":"inactive","fvpn-watchdog.service":"inactive"}}
- after_ruleset: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 334, "packets": 8, "bytes": 1519}}, {"counter": {"family": "inet", "name": "nd_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 335, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 336, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a191934a2e7d3388c967effa3ab59e", "table": "fvpn", "handle": 337, "packets": 9, "bytes": 1153}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 425, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 426, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 427, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 428, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 429, "comment": "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 430, "comment": "fvpn_class=01a19193-2b52-7583-bcc1-3a52bfcd3219:7e87", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 431, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 432, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"counter": "ctbase"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 433, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 434, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 435, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 436, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 4}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}]}

- migrated_check: {
  "status": "drifted",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a19192-ad63-7d61-b846-929be061b789.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a19192-ad63-7d61-b846-929be061b789",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T10:48:46.576567688Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a19192-ad63-7d61-b846-929be061b789\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a19192-ad63-7d61-b846-929be061b789",
  "manifest_sha256": "ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c",
  "release_source": "updater_journal",
  "install_record": {
    "release_id": "01a19192-ad63-7d61-b846-929be061b789",
    "manifest_sha256": "ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c",
    "release_sequence": "10",
    "product_version": "1.1.0",
    "arch": "amd64",
    "installed_at": "2026-10-31T10:48:00Z",
    "netbird_package_installed": false
  },
  "updater": {
    "current_release_id": "01a19192-ad63-7d61-b846-929be061b789",
    "state": "",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a19192-ad63-7d61-b846-929be061b789",
      "manifest_sha256": "ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c"
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
    "pid": 3642,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 2726392
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a19192-d10b-7c18-b918-36b9b6fb6a04",
    "node_id": "01a19192-cabd-7a26-8cdc-3049aeda33fa"
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
