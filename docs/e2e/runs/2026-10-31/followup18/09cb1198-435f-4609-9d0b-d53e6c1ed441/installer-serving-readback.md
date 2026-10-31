# installer-serving-readback

D-116 four-chain verifier and separate hookless marker after traffic restoration

Status: pass

Run: 09cb1198-435f-4609-9d0b-d53e6c1ed441

Source: 84b6a342970ffc651aafda81d485451529902233

Spec: ENR-05, D-116, AT07

- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 26009500482404
wall_unix_ns 1793442471984111293
policy_expires_mono_ns 29459539251600
baseline_ok 1
serving 1
generation 6
reason serving

- table_fvpn: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 279, "packets": 35, "bytes": 3270}}, {"counter": {"family": "inet", "name": "nd_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 280, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 281, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a1917e2af07cda883c736214db2cc1", "table": "fvpn", "handle": 282, "packets": 50, "bytes": 3664}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"], "elem": [{"elem": {"val": "100.119.232.150", "timeout": 3449, "expires": 3446}}]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 1209, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 1210, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 1211, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 1212, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1213, "comment": "fvpn_generation=6 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1214, "comment": "fvpn_class=01a1917e-1662-7571-8a9a-60da4e86666d:a753", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1215, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1216, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1217, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": "invalid"}}, {"counter": "ctinv"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1218, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": "invalid"}}, {"counter": "ctinv"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1219, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "!=", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}}, {"counter": "ctunku"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1220, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.119.232.150"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 53}}, {"counter": "nu_01a1917e2af07cda883c736214db2cc1"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:a753"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1221, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.119.232.150"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "dport"}}, "right": 53}}, {"counter": "nu_01a1917e2af07cda883c736214db2cc1"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:a753"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1222, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"counter": "ctbadns"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1223, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@denied_v4"}}, {"counter": "ctdeny"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1224, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@host_v4"}}, {"counter": "cthost"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1225, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.119.232.150"}}, {"counter": "u_01a1917e2af07cda883c736214db2cc1"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:a753"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1226, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "!=", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@allowed_v4"}}, {"counter": "ctunkd"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1228, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "!=", "left": {"ct": {"key": "state"}}, "right": {"set": ["established", "related"]}}}, {"counter": "ctnodn"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1229, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.119.232.150"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "sport"}}, "right": 53}}, {"counter": "nd_01a1917e2af07cda883c736214db2cc1"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:a753"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1230, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.119.232.150"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "sport"}}, "right": 53}}, {"counter": "nd_01a1917e2af07cda883c736214db2cc1"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:a753"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1231, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.119.232.150"}}, {"counter": "d_01a1917e2af07cda883c736214db2cc1"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:a753"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 1232, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 1233, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "postrouting", "handle": 1234, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}}, {"snat": {"family": "ip", "addr": "192.0.3.24"}}]}}]}

- table_fvpn_install: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 4}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}]}

- installer_check: {
  "status": "healthy",
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
      "stderr": "{\"time\":\"2026-10-31T10:27:51.052708785Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a1917d-cdde-777b-9e4c-6caa1495d8d8\"}\n"
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
    "pid": 7417,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 2600926
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
  "drifts": []
}

- agent_journal: Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.126Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-efbb-78df-b1b9-67bc2dd60c34 reason=baseline boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.135Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.136Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.163Z level=INFO msg="bootstrap stored"
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.204Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-efbb-78df-b1b9-67bc2dd60c34 reason=final_sample event_seq=0
Oct 31 10:25:12 9cd9049d833e fvpn-agent[1389]: time=2026-10-31T10:25:12.232Z level=INFO msg="usage status available"
Oct 31 10:25:12 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:12 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:12 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:13 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.341Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-f476-744f-b05b-7f8d3dc41e7b reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.350Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.351Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.417Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-f476-744f-b05b-7f8d3dc41e7b reason=final_sample event_seq=0
Oct 31 10:25:13 9cd9049d833e fvpn-agent[1651]: time=2026-10-31T10:25:13.452Z level=INFO msg="usage status available"
Oct 31 10:25:13 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:13 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:13 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:14 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.056Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-fb2b-724d-9fb8-9e1d9f578333 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.064Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.066Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.122Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-fb2b-724d-9fb8-9e1d9f578333 reason=final_sample event_seq=0
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2023]: time=2026-10-31T10:25:15.158Z level=INFO msg="usage status available"
Oct 31 10:25:15 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:15 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:15 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:15 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:15.835Z level=INFO msg="meter epoch opened" epoch_id=01a1917d-fe38-76d6-82aa-7a35e86b2b26 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:15 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:15.843Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:16 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:16.058Z level=INFO msg="meter epoch closed" epoch_id=01a1917d-fe38-76d6-82aa-7a35e86b2b26 reason=final_sample event_seq=0
Oct 31 10:25:16 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:16.091Z level=INFO msg="usage status available"
Oct 31 10:25:20 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:20.962Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-123d-7966-9b08-e41ffa842006 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:25 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:25.993Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-123d-7966-9b08-e41ffa842006 reason=final_sample event_seq=0
Oct 31 10:25:26 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:26.135Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-2675-7f2a-a15c-4142d3d24682 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:30 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:30.917Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-2675-7f2a-a15c-4142d3d24682 reason=final_sample event_seq=0
Oct 31 10:25:31 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:31.029Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-3993-7268-a783-2cb02862cd76 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:41 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:25:41 9cd9049d833e fvpn-agent[2263]: time=2026-10-31T10:25:41.326Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-3993-7268-a783-2cb02862cd76 reason=final_sample event_seq=0
Oct 31 10:25:41 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:25:41 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:43 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.493Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-6a41-711e-951f-20611c2e502e reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.500Z level=INFO msg="usage checkpoint applied"
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.524Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=1
Oct 31 10:25:43 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:43.646Z level=INFO msg="usage status available"
Oct 31 10:25:48 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:48.595Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-6a41-711e-951f-20611c2e502e reason=final_sample event_seq=0
Oct 31 10:25:48 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:25:48.760Z level=INFO msg="meter epoch opened" epoch_id=01a1917e-7ea7-75b6-9e76-1127f6b5762b reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:27:50 9cd9049d833e systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 10:27:50 9cd9049d833e fvpn-agent[3661]: time=2026-10-31T10:27:50.499Z level=INFO msg="meter epoch closed" epoch_id=01a1917e-7ea7-75b6-9e76-1127f6b5762b reason=final_sample event_seq=3
Oct 31 10:27:50 9cd9049d833e systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 10:27:50 9cd9049d833e systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:27:50 9cd9049d833e systemd[1]: fvpn-agent.service: Consumed 3.785s CPU time, 32.1M memory peak, 0B memory swap peak.
Oct 31 10:27:51 9cd9049d833e systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.814Z level=INFO msg="meter epoch opened" epoch_id=01a19180-5f83-7315-bc3e-9d2785649a80 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.819Z level=WARN msg="usage batch unavailable" status=503
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.820Z level=WARN msg="usage upload retry" backoff=1s err="usageupload: ledger unavailable"
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.821Z level=INFO msg="usage checkpoint applied"
Oct 31 10:27:51 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:51.950Z level=INFO msg="usage status available"
Oct 31 10:27:52 9cd9049d833e fvpn-agent[7417]: time=2026-10-31T10:27:52.852Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=3

- central_observation: {"report": {"lease": {"lease_id": "6c07c9db-19a1-4e62-8a49-cb457e96da87", "expires_at": "2026-11-01T10:27:51.831706623Z"}, "usage": {"blocked_seq": null, "ledger_epoch": "629370d2-858f-4740-be83-be70a3fbc16d", "meter_epoch_id": "01a19180-5f83-7315-bc3e-9d2785649a80", "oldest_unacked_at": "2026-10-31T10:27:48.714238318Z", "acked_archive_bytes": "654", "unacked_event_count": 2, "central_acked_through": "1", "unacked_payload_bytes": "1306", "last_durable_event_seq": "3"}, "checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "qos", "detail": "read_at=2026-10-31T10:27:51.938232213Z source=live candidate=6 verified=true {\"generation\":\"6\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"367b409ca4634def688bfab1fce33363a0f5e8b4db01488c5d671dde10f0cf26\"}", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:27:51.85584429Z"}], "errors": [], "update": {"stage": null, "state": null, "job_id": null, "generation": null}, "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a", "netbird": {"backend": "kernel", "peer_id": null, "version": null}, "node_id": "01a1917d-e899-7f0f-a666-35b7f0ddd137", "report_id": "01a19180-600e-754b-aa94-f9afd71f2f43", "data_health": "healthy", "observed_at": "2026-10-31T10:27:51.950314609Z", "agent_version": "1.1.0", "safety_status": "passed", "current_release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8", "provisioning_state": "ready", "applied_policy_refs": {"dns_generation": "1", "fup_generation": "1", "authorization_generation": "1"}, "dns_policy_evidence": {"feeds": [], "outcome": "applied", "applied_at": "2026-10-31T10:27:51.852463013Z", "error_code": null, "generation": "1", "document_id": "01a1917d-e87e-7e9c-b84b-36083252f8a6", "content_sha256": "203c623718175faa4b42c42be65e6e2d185bd409a93f6ebae443b6d286bf8468"}, "node_incarnation_id": "01a1917d-eedf-75ab-a77e-babbc7fb81dc", "last_good_release_id": "01a1917d-cdde-777b-9e4c-6caa1495d8d8", "applied_binding_revision": "1", "forwarding_block_reasons": [], "applied_config_generation": "6", "member_forwarding_enabled": true}, "received_at": "2026-10-31T10:27:51.95668+00:00"}
- agent_active: true
