# installer-serving-readback

D-116 four-chain verifier and separate hookless marker after traffic restoration

Status: pass

Run: d6aaa798-6c39-49f1-8fa9-5f84f696ac91

Source: a9ad1c471720a2df27f7d4aaa3a0ae1a565fa2cd

Spec: ENR-05, D-116, AT07

- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39182282580183
wall_unix_ns 1793455644766209047
policy_expires_mono_ns 39659041440644
baseline_ok 1
serving 1
generation 6
reason serving

- table_fvpn: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 148, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 149, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 150, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 151, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 152, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 153, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 154, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a192472e6c73ffb0ce065412fb82ee", "table": "fvpn", "handle": 312, "packets": 36, "bytes": 3345}}, {"counter": {"family": "inet", "name": "nd_01a192472e6c73ffb0ce065412fb82ee", "table": "fvpn", "handle": 313, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a192472e6c73ffb0ce065412fb82ee", "table": "fvpn", "handle": 314, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a192472e6c73ffb0ce065412fb82ee", "table": "fvpn", "handle": 315, "packets": 50, "bytes": 3701}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"], "elem": [{"elem": {"val": "100.64.184.43", "timeout": 476, "expires": 473}}]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 155, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 156, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 157, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 1242, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 1243, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 1244, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 1245, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1246, "comment": "fvpn_generation=6 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1247, "comment": "fvpn_class=01a19246-f179-7972-8f38-af09f7104c66:6d59", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1248, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1249, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1250, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": "invalid"}}, {"counter": "ctinv"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1251, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": "invalid"}}, {"counter": "ctinv"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1252, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "!=", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}}, {"counter": "ctunku"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1253, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.64.184.43"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 53}}, {"counter": "nu_01a192472e6c73ffb0ce065412fb82ee"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:6d59"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1254, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.64.184.43"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "dport"}}, "right": 53}}, {"counter": "nu_01a192472e6c73ffb0ce065412fb82ee"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:6d59"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1255, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"counter": "ctbadns"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1256, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@denied_v4"}}, {"counter": "ctdeny"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1257, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@host_v4"}}, {"counter": "cthost"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1258, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.64.184.43"}}, {"counter": "u_01a192472e6c73ffb0ce065412fb82ee"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:6d59"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1259, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "!=", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@allowed_v4"}}, {"counter": "ctunkd"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1261, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "!=", "left": {"ct": {"key": "state"}}, "right": {"set": ["established", "related"]}}}, {"counter": "ctnodn"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1262, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.64.184.43"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "sport"}}, "right": 53}}, {"counter": "nd_01a192472e6c73ffb0ce065412fb82ee"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:6d59"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1263, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.64.184.43"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "sport"}}, "right": 53}}, {"counter": "nd_01a192472e6c73ffb0ce065412fb82ee"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:6d59"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1264, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.64.184.43"}}, {"counter": "d_01a192472e6c73ffb0ce065412fb82ee"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:6d59"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 1265, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 1266, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "postrouting", "handle": 1267, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}}, {"snat": {"family": "ip", "addr": "192.0.3.24"}}]}}]}

- table_fvpn_install: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 4}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}]}

- installer_check: {
  "status": "healthy",
  "baseline": {
    "status": "seeded",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a19246-7581-7226-bc63-2d51bcab5202.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a19246-7581-7226-bc63-2d51bcab5202",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 0,
      "stderr": "{\"time\":\"2026-10-31T14:07:23.999621861Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a19246-7581-7226-bc63-2d51bcab5202\"}\n"
    }
  },
  "os_id": "ubuntu",
  "os_version": "24.04",
  "arch": "amd64",
  "kernel": "Linux 7.0.0-38-generic",
  "release_id": "01a19246-7581-7226-bc63-2d51bcab5202",
  "manifest_sha256": "c34a245bc536f401e4e0717929f801903a8d41b7a6118d3d450b5d9b6cadca05",
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
    "current_release_id": "01a19246-7581-7226-bc63-2d51bcab5202",
    "state": "",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a19246-7581-7226-bc63-2d51bcab5202",
      "manifest_sha256": "c34a245bc536f401e4e0717929f801903a8d41b7a6118d3d450b5d9b6cadca05"
    }
  },
  "units": {
    "fvpn-agent.service": "active",
    "fvpn-netbird.service": "active",
    "fvpn-node-netbird-up.service": "active",
    "fvpn-updater-recover.service": "active",
    "fvpn-updater.service": "inactive",
    "fvpn-updater.timer": "active",
    "fvpn-watchdog.service": "active"
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
    "held": true,
    "pid": 7604,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 3918205
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a19246-cb9e-70d2-9e31-e611a924c6a6",
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
  "drifts": []
}

- agent_journal: Oct 31 14:04:35 5698bce62483 fvpn-agent[1389]: time=2026-10-31T14:04:35.790Z level=INFO msg="usage checkpoint applied"
Oct 31 14:04:35 5698bce62483 fvpn-agent[1389]: time=2026-10-31T14:04:35.791Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 14:04:35 5698bce62483 fvpn-agent[1389]: time=2026-10-31T14:04:35.815Z level=INFO msg="bootstrap stored"
Oct 31 14:04:35 5698bce62483 fvpn-agent[1389]: time=2026-10-31T14:04:35.861Z level=INFO msg="meter epoch closed" epoch_id=01a19246-cc44-78ff-8a56-60e2334eafbe reason=final_sample event_seq=0
Oct 31 14:04:35 5698bce62483 fvpn-agent[1389]: time=2026-10-31T14:04:35.887Z level=INFO msg="usage status available"
Oct 31 14:04:36 5698bce62483 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:04:36 5698bce62483 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:04:36 5698bce62483 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:04:36 5698bce62483 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:04:36 5698bce62483 fvpn-agent[1655]: time=2026-10-31T14:04:36.879Z level=INFO msg="meter epoch opened" epoch_id=01a19246-d08c-798d-b465-f90fd415f295 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:04:36 5698bce62483 fvpn-agent[1655]: time=2026-10-31T14:04:36.886Z level=INFO msg="usage checkpoint applied"
Oct 31 14:04:36 5698bce62483 fvpn-agent[1655]: time=2026-10-31T14:04:36.888Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 14:04:36 5698bce62483 fvpn-agent[1655]: time=2026-10-31T14:04:36.943Z level=INFO msg="meter epoch closed" epoch_id=01a19246-d08c-798d-b465-f90fd415f295 reason=final_sample event_seq=0
Oct 31 14:04:36 5698bce62483 fvpn-agent[1655]: time=2026-10-31T14:04:36.968Z level=INFO msg="usage status available"
Oct 31 14:04:37 5698bce62483 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:04:37 5698bce62483 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:04:37 5698bce62483 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:04:38 5698bce62483 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:04:38 5698bce62483 fvpn-agent[2027]: time=2026-10-31T14:04:38.496Z level=INFO msg="meter epoch opened" epoch_id=01a19246-d6de-7971-9043-3698ea7337da reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:04:38 5698bce62483 fvpn-agent[2027]: time=2026-10-31T14:04:38.503Z level=INFO msg="usage checkpoint applied"
Oct 31 14:04:38 5698bce62483 fvpn-agent[2027]: time=2026-10-31T14:04:38.505Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 14:04:38 5698bce62483 fvpn-agent[2027]: time=2026-10-31T14:04:38.551Z level=INFO msg="meter epoch closed" epoch_id=01a19246-d6de-7971-9043-3698ea7337da reason=final_sample event_seq=0
Oct 31 14:04:38 5698bce62483 fvpn-agent[2027]: time=2026-10-31T14:04:38.575Z level=INFO msg="usage status available"
Oct 31 14:04:39 5698bce62483 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:04:39 5698bce62483 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:04:39 5698bce62483 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:04:39 5698bce62483 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:04:39 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:39.165Z level=INFO msg="meter epoch opened" epoch_id=01a19246-d97b-73fe-99b6-66241334fb6b reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:04:39 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:39.171Z level=INFO msg="usage checkpoint applied"
Oct 31 14:04:39 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:39.363Z level=INFO msg="meter epoch closed" epoch_id=01a19246-d97b-73fe-99b6-66241334fb6b reason=final_sample event_seq=0
Oct 31 14:04:39 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:39.396Z level=INFO msg="usage status available"
Oct 31 14:04:44 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:44.228Z level=INFO msg="meter epoch opened" epoch_id=01a19246-ed43-714c-9c3e-f78c13a49db2 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:04:49 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:49.253Z level=INFO msg="meter epoch closed" epoch_id=01a19246-ed43-714c-9c3e-f78c13a49db2 reason=final_sample event_seq=0
Oct 31 14:04:49 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:04:49.370Z level=INFO msg="meter epoch opened" epoch_id=01a19247-0158-7d67-a6a1-f1ccc9751c6e reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:05:04 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:05:04.265Z level=INFO msg="meter epoch closed" epoch_id=01a19247-0158-7d67-a6a1-f1ccc9751c6e reason=final_sample event_seq=0
Oct 31 14:05:04 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:05:04.374Z level=INFO msg="meter epoch opened" epoch_id=01a19247-3bf5-78ec-9ea0-863dddfba2b8 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:05:14 5698bce62483 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:05:14 5698bce62483 fvpn-agent[2265]: time=2026-10-31T14:05:14.756Z level=INFO msg="meter epoch closed" epoch_id=01a19247-3bf5-78ec-9ea0-863dddfba2b8 reason=final_sample event_seq=0
Oct 31 14:05:14 5698bce62483 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:05:14 5698bce62483 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:05:14 5698bce62483 systemd[1]: fvpn-agent.service: Consumed 1.127s CPU time, 30.1M memory peak, 0B memory swap peak.
Oct 31 14:05:16 5698bce62483 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:05:16 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:05:16.491Z level=INFO msg="meter epoch opened" epoch_id=01a19247-6b49-7600-a8a5-18a5dd57e485 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:05:16 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:05:16.499Z level=INFO msg="usage checkpoint applied"
Oct 31 14:05:16 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:05:16.517Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=1
Oct 31 14:05:16 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:05:16.625Z level=INFO msg="usage status available"
Oct 31 14:05:21 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:05:21.592Z level=INFO msg="meter epoch closed" epoch_id=01a19247-6b49-7600-a8a5-18a5dd57e485 reason=final_sample event_seq=0
Oct 31 14:05:21 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:05:21.695Z level=INFO msg="meter epoch opened" epoch_id=01a19247-7f9e-79ee-82d8-9ba97c08488c reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:07:23 5698bce62483 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:07:23 5698bce62483 fvpn-agent[3847]: time=2026-10-31T14:07:23.443Z level=INFO msg="meter epoch closed" epoch_id=01a19247-7f9e-79ee-82d8-9ba97c08488c reason=final_sample event_seq=3
Oct 31 14:07:23 5698bce62483 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:07:23 5698bce62483 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:07:23 5698bce62483 systemd[1]: fvpn-agent.service: Consumed 3.698s CPU time, 32.1M memory peak, 0B memory swap peak.
Oct 31 14:07:24 5698bce62483 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:07:24 5698bce62483 fvpn-agent[7604]: time=2026-10-31T14:07:24.605Z level=INFO msg="meter epoch opened" epoch_id=01a19249-5fba-7b29-befc-8c555307440e reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:07:24 5698bce62483 fvpn-agent[7604]: time=2026-10-31T14:07:24.611Z level=WARN msg="usage batch unavailable" status=503
Oct 31 14:07:24 5698bce62483 fvpn-agent[7604]: time=2026-10-31T14:07:24.611Z level=WARN msg="usage upload retry" backoff=1s err="usageupload: ledger unavailable"
Oct 31 14:07:24 5698bce62483 fvpn-agent[7604]: time=2026-10-31T14:07:24.612Z level=INFO msg="usage checkpoint applied"
Oct 31 14:07:24 5698bce62483 fvpn-agent[7604]: time=2026-10-31T14:07:24.734Z level=INFO msg="usage status available"
Oct 31 14:07:25 5698bce62483 fvpn-agent[7604]: time=2026-10-31T14:07:25.635Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=3

- central_observation: {"report": {"lease": {"lease_id": "f5e7507a-d64c-47da-bb9c-6d2fefe93f5d", "expires_at": "2026-11-01T14:07:24.622777999Z"}, "usage": {"blocked_seq": null, "ledger_epoch": "0376b40d-abbd-4bb0-b6f9-1e5badf32f5d", "meter_epoch_id": "01a19249-5fba-7b29-befc-8c555307440e", "oldest_unacked_at": "2026-10-31T14:07:21.719273208Z", "acked_archive_bytes": "654", "unacked_event_count": 2, "central_acked_through": "1", "unacked_payload_bytes": "1306", "last_durable_event_seq": "3"}, "checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "qos", "detail": "read_at=2026-10-31T14:07:24.724061395Z source=live candidate=6 verified=true {\"generation\":\"6\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"8bb10eba90f35353cb5558c9ad2e78793b6eb1f17e4745d71276f66075353a82\"}", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:07:24.648842511Z"}], "errors": [], "update": {"stage": null, "state": null, "job_id": null, "generation": null}, "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a", "netbird": {"backend": "kernel", "peer_id": null, "version": null}, "node_id": "01a19246-b349-7974-9962-fee669622101", "report_id": "01a19249-603d-7e35-812f-16265d9804a5", "data_health": "healthy", "observed_at": "2026-10-31T14:07:24.733904411Z", "tc_evidence": {"phase": "verify", "sha256": "8bb10eba90f35353cb5558c9ad2e78793b6eb1f17e4745d71276f66075353a82", "read_at": "2026-10-31T14:07:24.723968163Z", "rejected": null, "verified": true, "truncated": false, "error_code": null, "attachments": [{"objects": [{"kind": "fq_codel", "handle": "6d59:", "parent": "1:6d59"}, {"kind": "htb", "root": true, "handle": "1:"}], "interface": "fvmb0", "attachment": "qdisc"}, {"objects": [{"kind": "fq_codel", "handle": "6d59:", "parent": "1:6d59"}, {"kind": "htb", "root": true, "handle": "1:"}], "interface": "fvup0", "attachment": "qdisc"}, {"objects": [], "interface": "fvmb0", "attachment": "1:1"}, {"objects": [], "interface": "fvmb0", "attachment": "1:6d59"}, {"objects": [], "interface": "fvmb0", "attachment": "1:fffe"}, {"objects": [], "interface": "fvmb0", "attachment": "6d59:"}, {"objects": [], "interface": "fvmb0", "attachment": "egress"}, {"objects": [], "interface": "fvmb0", "attachment": "ingress"}, {"objects": [], "interface": "fvmb0", "attachment": "root"}, {"objects": [], "interface": "fvup0", "attachment": "1:1"}, {"objects": [], "interface": "fvup0", "attachment": "1:6d59"}, {"objects": [], "interface": "fvup0", "attachment": "1:fffe"}, {"objects": [], "interface": "fvup0", "attachment": "6d59:"}, {"objects": [], "interface": "fvup0", "attachment": "egress"}, {"objects": [], "interface": "fvup0", "attachment": "ingress"}, {"objects": [], "interface": "fvup0", "attachment": "root"}], "reads_total": 16, "objects_total": 4, "candidate_generation": "6"}, "agent_version": "1.1.0", "safety_status": "passed", "current_release_id": "01a19246-7581-7226-bc63-2d51bcab5202", "provisioning_state": "ready", "applied_policy_refs": {"dns_generation": "1", "fup_generation": "1", "authorization_generation": "1"}, "dns_policy_evidence": {"feeds": [], "outcome": "applied", "applied_at": "2026-10-31T14:07:24.645513907Z", "error_code": null, "generation": "1", "document_id": "01a19246-b310-7e11-ac25-8733086e7a37", "content_sha256": "f7407dbbc15d93dbc29cc447b14d72f6d247852bf05f7f67d3cb7cb43608fa8e"}, "node_incarnation_id": "01a19246-cb9e-70d2-9e31-e611a924c6a6", "last_good_release_id": "01a19246-7581-7226-bc63-2d51bcab5202", "applied_binding_revision": "1", "forwarding_block_reasons": [], "applied_config_generation": "6", "member_forwarding_enabled": true}, "received_at": "2026-10-31T14:07:24.742534+00:00"}
- agent_active: true
