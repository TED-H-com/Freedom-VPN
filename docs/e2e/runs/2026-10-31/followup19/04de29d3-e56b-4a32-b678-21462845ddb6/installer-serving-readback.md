# installer-serving-readback

D-116 four-chain verifier and separate hookless marker after traffic restoration

Status: pass

Run: 04de29d3-e56b-4a32-b678-21462845ddb6

Source: fb5b8fcee72bd71b3bd0a4536af9049df4055096

Spec: ENR-05, D-116, AT07

- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39665547440496
wall_unix_ns 1793456128031069452
policy_expires_mono_ns 40137043202016
baseline_ok 1
serving 1
generation 6
reason serving

- table_fvpn: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn", "handle": 3}}, {"counter": {"family": "inet", "name": "ctbase", "table": "fvpn", "handle": 1, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctv6", "table": "fvpn", "handle": 2, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctin", "table": "fvpn", "handle": 3, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctbadns", "table": "fvpn", "handle": 159, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctdeny", "table": "fvpn", "handle": 160, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "cthost", "table": "fvpn", "handle": 161, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctinv", "table": "fvpn", "handle": 162, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctnodn", "table": "fvpn", "handle": 163, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunkd", "table": "fvpn", "handle": 164, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "ctunku", "table": "fvpn", "handle": 165, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "d_01a1924e8bf57b59bc65ece94dd391ca", "table": "fvpn", "handle": 411, "packets": 36, "bytes": 3347}}, {"counter": {"family": "inet", "name": "nd_01a1924e8bf57b59bc65ece94dd391ca", "table": "fvpn", "handle": 412, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "nu_01a1924e8bf57b59bc65ece94dd391ca", "table": "fvpn", "handle": 413, "packets": 0, "bytes": 0}}, {"counter": {"family": "inet", "name": "u_01a1924e8bf57b59bc65ece94dd391ca", "table": "fvpn", "handle": 414, "packets": 50, "bytes": 3701}}, {"set": {"family": "inet", "name": "allowed_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 4, "flags": ["timeout"], "elem": [{"elem": {"val": "100.119.193.232", "timeout": 471, "expires": 468}}]}}, {"set": {"family": "inet", "name": "denied_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 166, "flags": ["interval"], "elem": [{"prefix": {"addr": "0.0.0.0", "len": 8}}, {"prefix": {"addr": "10.0.0.0", "len": 8}}, {"prefix": {"addr": "100.64.0.0", "len": 10}}, {"prefix": {"addr": "127.0.0.0", "len": 8}}, {"prefix": {"addr": "169.254.0.0", "len": 16}}, {"prefix": {"addr": "172.16.0.0", "len": 12}}, {"prefix": {"addr": "192.0.0.0", "len": 24}}, {"prefix": {"addr": "192.0.2.0", "len": 24}}, {"prefix": {"addr": "192.88.99.0", "len": 24}}, {"prefix": {"addr": "192.168.0.0", "len": 16}}, {"prefix": {"addr": "198.18.0.0", "len": 15}}, {"prefix": {"addr": "198.51.100.0", "len": 24}}, {"prefix": {"addr": "203.0.113.0", "len": 24}}, {"prefix": {"addr": "224.0.0.0", "len": 3}}]}}, {"set": {"family": "inet", "name": "host_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 167, "flags": ["interval"], "elem": ["192.0.3.24"]}}, {"set": {"family": "inet", "name": "resolver_v4", "table": "fvpn", "type": "ipv4_addr", "handle": 168, "elem": ["192.0.3.53"]}}, {"chain": {"family": "inet", "table": "fvpn", "name": "forward", "handle": 1385, "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "input", "handle": 1386, "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "output", "handle": 1387, "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}}, {"chain": {"family": "inet", "table": "fvpn", "name": "postrouting", "handle": 1388, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1389, "comment": "fvpn_generation=6 fvpn_metering=forward_admission_v1", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1390, "comment": "fvpn_class=01a1924e-4e61-7626-bb8b-147951a6220d:eba7", "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "192.0.2.255"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "192.0.2.255"}}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1391, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1392, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "nfproto"}}, "right": "ipv6"}}, {"counter": "ctv6"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1393, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": "invalid"}}, {"counter": "ctinv"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1394, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "in", "left": {"ct": {"key": "state"}}, "right": "invalid"}}, {"counter": "ctinv"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1395, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "!=", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}}, {"counter": "ctunku"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1396, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.119.193.232"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 53}}, {"counter": "nu_01a1924e8bf57b59bc65ece94dd391ca"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:eba7"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1397, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.119.193.232"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "dport"}}, "right": 53}}, {"counter": "nu_01a1924e8bf57b59bc65ece94dd391ca"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:eba7"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1398, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@resolver_v4"}}, {"counter": "ctbadns"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1399, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@denied_v4"}}, {"counter": "ctdeny"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1400, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@host_v4"}}, {"counter": "cthost"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1401, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "100.119.193.232"}}, {"counter": "u_01a1924e8bf57b59bc65ece94dd391ca"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:eba7"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1402, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "!=", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "@allowed_v4"}}, {"counter": "ctunkd"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1404, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "!=", "left": {"ct": {"key": "state"}}, "right": {"set": ["established", "related"]}}}, {"counter": "ctnodn"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1405, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.119.193.232"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "tcp", "field": "sport"}}, "right": 53}}, {"counter": "nd_01a1924e8bf57b59bc65ece94dd391ca"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:eba7"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1406, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@resolver_v4"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.119.193.232"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "udp", "field": "sport"}}, "right": 53}}, {"counter": "nd_01a1924e8bf57b59bc65ece94dd391ca"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:eba7"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "forward", "handle": 1407, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvmb0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "daddr"}}, "right": "100.119.193.232"}}, {"counter": "d_01a1924e8bf57b59bc65ece94dd391ca"}, {"mangle": {"key": {"meta": {"key": "priority"}}, "value": "1:eba7"}}, {"accept": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 1408, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "iifname"}}, "right": "fvmb0"}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "input", "handle": 1409, "expr": [{"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "100.64.0.0", "len": 10}}}}, {"counter": "ctin"}, {"drop": null}]}}, {"rule": {"family": "inet", "table": "fvpn", "chain": "postrouting", "handle": 1410, "expr": [{"match": {"op": "==", "left": {"meta": {"key": "oifname"}}, "right": "fvup0"}}, {"match": {"op": "==", "left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}}, {"snat": {"family": "ip", "addr": "192.0.3.24"}}]}}]}

- table_fvpn_install: {"nftables": [{"metainfo": {"version": "1.0.9", "release_name": "Old Doc Yak #3", "json_schema_version": 1}}, {"table": {"family": "inet", "name": "fvpn_install", "handle": 4}}, {"chain": {"family": "inet", "table": "fvpn_install", "name": "owner", "handle": 1, "comment": "managed-by=fvpn-node-install"}}]}

- installer_check: {
  "status": "healthy",
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
      "stderr": "{\"time\":\"2026-10-31T14:15:27.111985093Z\",\"level\":\"INFO\",\"msg\":\"verified installer baseline seeded\",\"release_id\":\"01a1924d-93c5-7e6e-879f-528a14cc7a18\"}\n"
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
    "state": "",
    "recovery_required": false,
    "seeded_baseline": {
      "release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18",
      "manifest_sha256": "0f32fde0fd482890b50ebf9888470e789b312ff89391d763d4c1060f721b371f"
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
    "pid": 7986,
    "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a",
    "start_time": 3966530
  },
  "identity": {
    "present": true,
    "incarnation_id": "01a1924d-b2ed-7dae-875c-392d39febfa2",
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
  "drifts": []
}

- agent_journal: Oct 31 14:12:08 914aebfc9693 fvpn-agent[1387]: time=2026-10-31T14:12:08.311Z level=INFO msg="usage status available"
Oct 31 14:12:18 914aebfc9693 fvpn-agent[1387]: time=2026-10-31T14:12:18.250Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 14:12:18 914aebfc9693 fvpn-agent[1387]: time=2026-10-31T14:12:18.259Z level=INFO msg="bootstrap stored"
Oct 31 14:12:18 914aebfc9693 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:12:18 914aebfc9693 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:12:18 914aebfc9693 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:12:19 914aebfc9693 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:12:19 914aebfc9693 fvpn-agent[1665]: time=2026-10-31T14:12:19.305Z level=INFO msg="meter epoch opened" epoch_id=01a1924d-dee7-7a3f-b26c-3ef73dc317f1 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:12:19 914aebfc9693 fvpn-agent[1665]: time=2026-10-31T14:12:19.312Z level=INFO msg="usage checkpoint applied"
Oct 31 14:12:19 914aebfc9693 fvpn-agent[1665]: time=2026-10-31T14:12:19.315Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 14:12:19 914aebfc9693 fvpn-agent[1665]: time=2026-10-31T14:12:19.365Z level=INFO msg="meter epoch closed" epoch_id=01a1924d-dee7-7a3f-b26c-3ef73dc317f1 reason=final_sample event_seq=0
Oct 31 14:12:19 914aebfc9693 fvpn-agent[1665]: time=2026-10-31T14:12:19.397Z level=INFO msg="usage status available"
Oct 31 14:12:19 914aebfc9693 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:12:19 914aebfc9693 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:12:19 914aebfc9693 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:12:20 914aebfc9693 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:12:20 914aebfc9693 fvpn-agent[2043]: time=2026-10-31T14:12:20.923Z level=INFO msg="meter epoch opened" epoch_id=01a1924d-e538-7e37-9311-c1cbd7dc382a reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:12:20 914aebfc9693 fvpn-agent[2043]: time=2026-10-31T14:12:20.930Z level=INFO msg="usage checkpoint applied"
Oct 31 14:12:20 914aebfc9693 fvpn-agent[2043]: time=2026-10-31T14:12:20.931Z level=WARN msg="desired state" err="controlclient: http 503"
Oct 31 14:12:20 914aebfc9693 fvpn-agent[2043]: time=2026-10-31T14:12:20.983Z level=INFO msg="meter epoch closed" epoch_id=01a1924d-e538-7e37-9311-c1cbd7dc382a reason=final_sample event_seq=0
Oct 31 14:12:21 914aebfc9693 fvpn-agent[2043]: time=2026-10-31T14:12:21.014Z level=INFO msg="usage status available"
Oct 31 14:12:21 914aebfc9693 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:12:21 914aebfc9693 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:12:21 914aebfc9693 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:12:21 914aebfc9693 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:12:21 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:21.625Z level=INFO msg="meter epoch opened" epoch_id=01a1924d-e7f7-7b28-9cdb-bf132411ac13 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:12:21 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:21.632Z level=INFO msg="usage checkpoint applied"
Oct 31 14:12:21 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:21.828Z level=INFO msg="meter epoch closed" epoch_id=01a1924d-e7f7-7b28-9cdb-bf132411ac13 reason=final_sample event_seq=0
Oct 31 14:12:21 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:21.857Z level=INFO msg="usage status available"
Oct 31 14:12:26 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:26.685Z level=INFO msg="meter epoch opened" epoch_id=01a1924d-fbbb-7414-ad73-5f207121e90f reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:12:51 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:51.721Z level=INFO msg="meter epoch closed" epoch_id=01a1924d-fbbb-7414-ad73-5f207121e90f reason=final_sample event_seq=0
Oct 31 14:12:51 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:12:51.832Z level=INFO msg="meter epoch opened" epoch_id=01a1924e-5df7-74f3-9894-36d5893c4442 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:13:06 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:13:06.721Z level=INFO msg="meter epoch closed" epoch_id=01a1924e-5df7-74f3-9894-36d5893c4442 reason=final_sample event_seq=0
Oct 31 14:13:06 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:13:06.828Z level=INFO msg="meter epoch opened" epoch_id=01a1924e-988a-77ea-b509-0e5434af89ab reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:13:17 914aebfc9693 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:13:17 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:13:17.350Z level=WARN msg=apply err=TC_FOREIGN_FILTER
Oct 31 14:13:17 914aebfc9693 fvpn-agent[2282]: time=2026-10-31T14:13:17.452Z level=INFO msg="meter epoch closed" epoch_id=01a1924e-988a-77ea-b509-0e5434af89ab reason=final_sample event_seq=0
Oct 31 14:13:17 914aebfc9693 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:13:17 914aebfc9693 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:13:17 914aebfc9693 systemd[1]: fvpn-agent.service: Consumed 1.649s CPU time, 30.0M memory peak, 0B memory swap peak.
Oct 31 14:13:19 914aebfc9693 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:13:19 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:19.507Z level=INFO msg="meter epoch opened" epoch_id=01a1924e-ca0e-7494-bb07-92d459332ab4 reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:13:19 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:19.521Z level=INFO msg="usage checkpoint applied"
Oct 31 14:13:19 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:19.543Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=1
Oct 31 14:13:19 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:19.593Z level=INFO msg="meter epoch closed" epoch_id=01a1924e-ca0e-7494-bb07-92d459332ab4 reason=final_sample event_seq=0
Oct 31 14:13:19 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:19.618Z level=INFO msg="usage status available"
Oct 31 14:13:24 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:24.688Z level=WARN msg=apply err="apply failed"
Oct 31 14:13:24 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:13:24.713Z level=INFO msg="meter epoch opened" epoch_id=01a1924e-de67-7540-9fd3-9e198af8adef reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:15:26 914aebfc9693 systemd[1]: Stopping fvpn-agent.service - Freedom VPN node agent...
Oct 31 14:15:26 914aebfc9693 fvpn-agent[4292]: time=2026-10-31T14:15:26.527Z level=INFO msg="meter epoch closed" epoch_id=01a1924e-de67-7540-9fd3-9e198af8adef reason=final_sample event_seq=3
Oct 31 14:15:26 914aebfc9693 systemd[1]: fvpn-agent.service: Deactivated successfully.
Oct 31 14:15:26 914aebfc9693 systemd[1]: Stopped fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:15:26 914aebfc9693 systemd[1]: fvpn-agent.service: Consumed 3.731s CPU time, 34.3M memory peak, 0B memory swap peak.
Oct 31 14:15:27 914aebfc9693 systemd[1]: Started fvpn-agent.service - Freedom VPN node agent.
Oct 31 14:15:27 914aebfc9693 fvpn-agent[7986]: time=2026-10-31T14:15:27.850Z level=INFO msg="meter epoch opened" epoch_id=01a19250-bf68-7647-966f-72fe064badbc reason=after_final_sample boot_id=97c018d7-a925-4723-9127-83c097de1c8a event_seq=0
Oct 31 14:15:27 914aebfc9693 fvpn-agent[7986]: time=2026-10-31T14:15:27.856Z level=WARN msg="usage batch unavailable" status=503
Oct 31 14:15:27 914aebfc9693 fvpn-agent[7986]: time=2026-10-31T14:15:27.856Z level=WARN msg="usage upload retry" backoff=1s err="usageupload: ledger unavailable"
Oct 31 14:15:27 914aebfc9693 fvpn-agent[7986]: time=2026-10-31T14:15:27.857Z level=INFO msg="usage checkpoint applied"
Oct 31 14:15:27 914aebfc9693 fvpn-agent[7986]: time=2026-10-31T14:15:27.996Z level=INFO msg="usage status available"
Oct 31 14:15:28 914aebfc9693 fvpn-agent[7986]: time=2026-10-31T14:15:28.873Z level=INFO msg="usage batch acknowledged" highest_contiguous_seq=3

- central_observation: {"report": {"lease": {"lease_id": "fd57945f-1e6d-44df-9bbe-b3cf59636634", "expires_at": "2026-11-01T14:15:27.867361727Z"}, "usage": {"blocked_seq": null, "ledger_epoch": "3c6e8f2a-10e3-4a46-a451-d3c7dbfda9e2", "meter_epoch_id": "01a19250-bf68-7647-966f-72fe064badbc", "oldest_unacked_at": "2026-10-31T14:15:24.823743362Z", "acked_archive_bytes": "654", "unacked_event_count": 2, "central_acked_through": "1", "unacked_payload_bytes": "1305", "last_durable_event_seq": "3"}, "checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "qos", "detail": "read_at=2026-10-31T14:15:27.98443721Z source=live candidate=6 verified=true {\"generation\":\"6\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"dd0003f5b5177898b14a6e102262d598da5e6f003d4551f82053a8e654137265\"}", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T14:15:27.894039498Z"}], "errors": [], "update": {"stage": null, "state": null, "job_id": null, "generation": null}, "boot_id": "97c018d7-a925-4723-9127-83c097de1c8a", "netbird": {"backend": "kernel", "peer_id": null, "version": null}, "node_id": "01a1924d-ad8a-75f0-9fab-9ac72ddcc32f", "report_id": "01a19250-bffc-776e-99ac-63d2ebc09bec", "data_health": "healthy", "observed_at": "2026-10-31T14:15:27.996463508Z", "tc_evidence": {"phase": "verify", "sha256": "dd0003f5b5177898b14a6e102262d598da5e6f003d4551f82053a8e654137265", "read_at": "2026-10-31T14:15:27.984359806Z", "rejected": null, "verified": true, "truncated": false, "error_code": null, "attachments": [{"objects": [{"kind": "fq_codel", "handle": "eba7:", "parent": "1:eba7"}, {"kind": "htb", "root": true, "handle": "1:"}], "interface": "fvmb0", "attachment": "qdisc"}, {"objects": [{"kind": "fq_codel", "handle": "eba7:", "parent": "1:eba7"}, {"kind": "htb", "root": true, "handle": "1:"}], "interface": "fvup0", "attachment": "qdisc"}, {"objects": [], "interface": "fvmb0", "attachment": "1:1"}, {"objects": [], "interface": "fvmb0", "attachment": "1:eba7"}, {"objects": [], "interface": "fvmb0", "attachment": "1:fffe"}, {"objects": [], "interface": "fvmb0", "attachment": "eba7:"}, {"objects": [], "interface": "fvmb0", "attachment": "egress"}, {"objects": [], "interface": "fvmb0", "attachment": "ingress"}, {"objects": [], "interface": "fvmb0", "attachment": "root"}, {"objects": [], "interface": "fvup0", "attachment": "1:1"}, {"objects": [], "interface": "fvup0", "attachment": "1:eba7"}, {"objects": [], "interface": "fvup0", "attachment": "1:fffe"}, {"objects": [], "interface": "fvup0", "attachment": "eba7:"}, {"objects": [], "interface": "fvup0", "attachment": "egress"}, {"objects": [], "interface": "fvup0", "attachment": "ingress"}, {"objects": [], "interface": "fvup0", "attachment": "root"}], "reads_total": 16, "objects_total": 4, "candidate_generation": "6"}, "agent_version": "1.1.0", "safety_status": "passed", "current_release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18", "provisioning_state": "ready", "applied_policy_refs": {"dns_generation": "1", "fup_generation": "1", "authorization_generation": "1"}, "dns_policy_evidence": {"feeds": [], "outcome": "applied", "applied_at": "2026-10-31T14:15:27.8904246Z", "error_code": null, "generation": "1", "document_id": "01a1924d-ad93-7056-80f8-c1493d454c5c", "content_sha256": "33da065995882eb632947d896eb2725ba5077945fa3f9583d7302947bbe0f14d"}, "node_incarnation_id": "01a1924d-b2ed-7dae-875c-392d39febfa2", "last_good_release_id": "01a1924d-93c5-7e6e-879f-528a14cc7a18", "applied_binding_revision": "1", "forwarding_block_reasons": [], "applied_config_generation": "6", "member_forwarding_enabled": true}, "received_at": "2026-10-31T14:15:28.00518+00:00"}
- agent_active: true
