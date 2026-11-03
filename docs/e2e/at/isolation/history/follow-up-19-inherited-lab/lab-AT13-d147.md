# lab-AT13-d147

Mechanism: D-147 RELATED IPv4 fragmentation-needed per source set and overlap

Status: fail

Run: 58fb86e8-92a2-47e2-9a06-b183df343d9c; started UTC: 2026-11-03T21:57:13Z; source dirty: false

Spec: §19 AT13, ISO01, ISO02, MTR01, D-146, D-147

UTC: 2026-11-03T21:57:37Z

Build: 9493950a6308c86a729ddc7fff0fe0ff97bb370f

Image digests:

- fvpn-w4-isolation-lab:lab sha256:fc54a3d39b52f06e91e5da325f2c9824e4eab8db52c4c2c64eb2b1d30df456a5

Policy revision: generation 4200

Node incarnation: mechanism-lab

Expected: Genuine router IPv4 ICMP fragmentation-needed (type 3, code 4, MTU 1000) from resolver_v4-only, denied_v4-only, host_v4-only, and denied_v4+host_v4 overlap sources reaches the admitted member when conntrack marks it RELATED, is counted once in d_<device> at the device's assigned priority (1:2f), and matches independent captures; time-exceeded is dropped at ctdenyd, unsolicited fragmentation-needed at ctinv, and member upload to that source is dropped with 0 upload usage.

Actual: d147 runner failed: exit status 1
stderr:

stdout:
{
  "passed": false,
  "member_ns": "fv-a1",
  "member_ip": "100.64.1.1",
  "device_id": "019a00000000700080000000000000a1",
  "snat_ip": "192.0.2.37",
  "origin_ip": "203.0.113.34",
  "origin_port": 8080,
  "path_mtu": 1000,
  "upload_bytes": 64064,
  "assigned_priority": "0x0001002f",
  "assigned_tc_class": "1:2f",
  "rows": [
    {
      "id": "resolver_v4_only",
      "source_ip": "203.0.113.53",
      "expected_sets": [
        "resolver_v4"
      ],
      "set_membership": {
        "in_resolver_v4": true,
        "in_denied_v4": false,
        "in_host_v4": false,
        "matched_sets": [
          "resolver_v4"
        ]
      },
      "set_membership_verified": true,
      "upload_drop_counter": "ctdns",
      "passed": false,
      "subcases": [
        {
          "name": "related-frag-needed",
          "passed": false,
          "pmtu_before": null,
          "transfer": {
            "ok": false,
            "error": "[Errno 32] Broken pipe",
            "socket_pmtu": null,
            "socket_mss": null,
            "elapsed_ms": 1.67,
            "pmtu": 1000
          },
          "captures": {
            "router_fv_ii": {
              "packets": 10,
              "ip_bytes": 5760,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "node_up_egress_fv_ih": {
              "packets": 27,
              "ip_bytes": 24372,
              "tcp_packets": 27,
              "tcp_ip_bytes": 24372,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1160
              ],
              "icmp": []
            },
            "node_down_egress_fv_fh": {
              "packets": 21,
              "ip_bytes": 6328,
              "tcp_packets": 11,
              "tcp_ip_bytes": 568,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 60,
              "syn_mss": [
                1160
              ],
              "icmp": [
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "member_fv_wg0": {
              "packets": 48,
              "ip_bytes": 30700,
              "tcp_packets": 38,
              "tcp_ip_bytes": 24940,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1380,
                1160
              ],
              "icmp": [
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 27,
              "bytes": 24372
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 21,
              "bytes": 6328
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          },
          "usage_reconciliation": {
            "device_id": "019a00000000700080000000000000a1",
            "assigned_priority": "0x0001002f",
            "assigned_tc_class": "1:2f",
            "upload_counter_matches_capture": true,
            "download_counter_matches_capture": true,
            "download_tcp_ip_bytes": 568,
            "download_frag_needed_icmp_packets": 10,
            "download_frag_needed_icmp_ip_bytes": 5760,
            "download_total_ip_bytes": 6328,
            "tc_upload_leaf": {
              "packets": 27,
              "bytes": 24750,
              "drops": 0
            },
            "tc_download_leaf": {
              "packets": 21,
              "bytes": 6622,
              "drops": 0
            },
            "tc_upload_all": {
              "1:1": {
                "packets": 28,
                "bytes": 24792,
                "drops": 0
              },
              "1:2f": {
                "packets": 27,
                "bytes": 24750,
                "drops": 0
              },
              "1:fffe": {
                "packets": 1,
                "bytes": 42,
                "drops": 0
              }
            },
            "tc_download_all": {
              "1:1": {
                "packets": 22,
                "bytes": 6664,
                "drops": 0
              },
              "1:2f": {
                "packets": 21,
                "bytes": 6622,
                "drops": 0
              },
              "1:fffe": {
                "packets": 1,
                "bytes": 42,
                "drops": 0
              }
            }
          }
        },
        {
          "name": "time-exceeded-dropped",
          "passed": true,
          "dropped_at_counter": "ctdenyd",
          "captures": {
            "router_fv_ii": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.53",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 1,
              "bytes": 84
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 1,
              "bytes": 112
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "unsolicited-frag-needed-dropped",
          "passed": true,
          "dropped_at_counter": "ctinv",
          "captures": {
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 60,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 60,
              "max_ip_len": 60,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.53",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 60
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 1,
              "bytes": 60
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "member-upload-to-source-dropped",
          "passed": false,
          "dropped_at_counter": "ctdns",
          "captures": {
            "node_ingress_fv_fh": {
              "packets": 1,
              "ip_bytes": 84,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 84,
              "max_ip_len": 84,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "100.64.1.1",
                  "dst": "203.0.113.53",
                  "icmp_type": 8,
                  "icmp_code": 0,
                  "mtu": 1,
                  "ip_len": 84
                }
              ]
            },
            "node_egress_fv_ih": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "router_fv_ii": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        }
      ]
    },
    {
      "id": "denied_v4_only",
      "source_ip": "192.168.34.2",
      "expected_sets": [
        "denied_v4"
      ],
      "set_membership": {
        "in_resolver_v4": false,
        "in_denied_v4": true,
        "in_host_v4": false,
        "matched_sets": [
          "denied_v4"
        ]
      },
      "set_membership_verified": true,
      "upload_drop_counter": "ctdeny",
      "passed": false,
      "subcases": [
        {
          "name": "related-frag-needed",
          "passed": false,
          "pmtu_before": null,
          "transfer": {
            "ok": false,
            "error": "[Errno 104] Connection reset by peer",
            "socket_pmtu": null,
            "socket_mss": null,
            "elapsed_ms": 1.43,
            "pmtu": 1000
          },
          "captures": {
            "router_fv_ii": {
              "packets": 10,
              "ip_bytes": 5760,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "node_up_egress_fv_ih": {
              "packets": 27,
              "ip_bytes": 24372,
              "tcp_packets": 27,
              "tcp_ip_bytes": 24372,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1160
              ],
              "icmp": []
            },
            "node_down_egress_fv_fh": {
              "packets": 16,
              "ip_bytes": 6128,
              "tcp_packets": 6,
              "tcp_ip_bytes": 368,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 60,
              "syn_mss": [
                1160
              ],
              "icmp": [
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "member_fv_wg0": {
              "packets": 43,
              "ip_bytes": 30500,
              "tcp_packets": 33,
              "tcp_ip_bytes": 24740,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1380,
                1160
              ],
              "icmp": [
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 27,
              "bytes": 24372
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 16,
              "bytes": 6128
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          },
          "usage_reconciliation": {
            "device_id": "019a00000000700080000000000000a1",
            "assigned_priority": "0x0001002f",
            "assigned_tc_class": "1:2f",
            "upload_counter_matches_capture": true,
            "download_counter_matches_capture": true,
            "download_tcp_ip_bytes": 368,
            "download_frag_needed_icmp_packets": 10,
            "download_frag_needed_icmp_ip_bytes": 5760,
            "download_total_ip_bytes": 6128,
            "tc_upload_leaf": {
              "packets": 27,
              "bytes": 24750,
              "drops": 0
            },
            "tc_download_leaf": {
              "packets": 16,
              "bytes": 6352,
              "drops": 0
            },
            "tc_upload_all": {
              "1:1": {
                "packets": 27,
                "bytes": 24750,
                "drops": 0
              },
              "1:2f": {
                "packets": 27,
                "bytes": 24750,
                "drops": 0
              },
              "1:fffe": {
                "packets": 0,
                "bytes": 0,
                "drops": 0
              }
            },
            "tc_download_all": {
              "1:1": {
                "packets": 16,
                "bytes": 6352,
                "drops": 0
              },
              "1:2f": {
                "packets": 16,
                "bytes": 6352,
                "drops": 0
              },
              "1:fffe": {
                "packets": 0,
                "bytes": 0,
                "drops": 0
              }
            }
          }
        },
        {
          "name": "time-exceeded-dropped",
          "passed": true,
          "dropped_at_counter": "ctdenyd",
          "captures": {
            "router_fv_ii": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.2",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 1,
              "bytes": 84
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 1,
              "bytes": 112
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "unsolicited-frag-needed-dropped",
          "passed": true,
          "dropped_at_counter": "ctinv",
          "captures": {
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 60,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 60,
              "max_ip_len": 60,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.2",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 60
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 1,
              "bytes": 60
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "member-upload-to-source-dropped",
          "passed": true,
          "dropped_at_counter": "ctdeny",
          "captures": {
            "node_ingress_fv_fh": {
              "packets": 1,
              "ip_bytes": 84,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 84,
              "max_ip_len": 84,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "100.64.1.1",
                  "dst": "192.168.34.2",
                  "icmp_type": 8,
                  "icmp_code": 0,
                  "mtu": 1,
                  "ip_len": 84
                }
              ]
            },
            "node_egress_fv_ih": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "router_fv_ii": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 1,
              "bytes": 84
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        }
      ]
    },
    {
      "id": "host_v4_only",
      "source_ip": "203.0.113.33",
      "expected_sets": [
        "host_v4"
      ],
      "set_membership": {
        "in_resolver_v4": false,
        "in_denied_v4": false,
        "in_host_v4": true,
        "matched_sets": [
          "host_v4"
        ]
      },
      "set_membership_verified": true,
      "upload_drop_counter": "cthost",
      "passed": false,
      "subcases": [
        {
          "name": "related-frag-needed",
          "passed": false,
          "pmtu_before": null,
          "transfer": {
            "ok": false,
            "error": "[Errno 104] Connection reset by peer",
            "socket_pmtu": null,
            "socket_mss": null,
            "elapsed_ms": 1.95,
            "pmtu": 1000
          },
          "captures": {
            "router_fv_ii": {
              "packets": 10,
              "ip_bytes": 5760,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "node_up_egress_fv_ih": {
              "packets": 43,
              "ip_bytes": 40372,
              "tcp_packets": 43,
              "tcp_ip_bytes": 40372,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1160
              ],
              "icmp": []
            },
            "node_down_egress_fv_fh": {
              "packets": 40,
              "ip_bytes": 7136,
              "tcp_packets": 30,
              "tcp_ip_bytes": 1376,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 60,
              "syn_mss": [
                1160
              ],
              "icmp": [
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "member_fv_wg0": {
              "packets": 83,
              "ip_bytes": 47508,
              "tcp_packets": 73,
              "tcp_ip_bytes": 41748,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1380,
                1160
              ],
              "icmp": [
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 43,
              "bytes": 40372
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 40,
              "bytes": 7136
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          },
          "usage_reconciliation": {
            "device_id": "019a00000000700080000000000000a1",
            "assigned_priority": "0x0001002f",
            "assigned_tc_class": "1:2f",
            "upload_counter_matches_capture": true,
            "download_counter_matches_capture": true,
            "download_tcp_ip_bytes": 1376,
            "download_frag_needed_icmp_packets": 10,
            "download_frag_needed_icmp_ip_bytes": 5760,
            "download_total_ip_bytes": 7136,
            "tc_upload_leaf": {
              "packets": 43,
              "bytes": 40974,
              "drops": 0
            },
            "tc_download_leaf": {
              "packets": 40,
              "bytes": 7696,
              "drops": 0
            },
            "tc_upload_all": {
              "1:1": {
                "packets": 43,
                "bytes": 40974,
                "drops": 0
              },
              "1:2f": {
                "packets": 43,
                "bytes": 40974,
                "drops": 0
              },
              "1:fffe": {
                "packets": 0,
                "bytes": 0,
                "drops": 0
              }
            },
            "tc_download_all": {
              "1:1": {
                "packets": 40,
                "bytes": 7696,
                "drops": 0
              },
              "1:2f": {
                "packets": 40,
                "bytes": 7696,
                "drops": 0
              },
              "1:fffe": {
                "packets": 0,
                "bytes": 0,
                "drops": 0
              }
            }
          }
        },
        {
          "name": "time-exceeded-dropped",
          "passed": true,
          "dropped_at_counter": "ctdenyd",
          "captures": {
            "router_fv_ii": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.33",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 1,
              "bytes": 84
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 1,
              "bytes": 112
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "unsolicited-frag-needed-dropped",
          "passed": true,
          "dropped_at_counter": "ctinv",
          "captures": {
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 60,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 60,
              "max_ip_len": 60,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "203.0.113.33",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 60
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 1,
              "bytes": 60
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "member-upload-to-source-dropped",
          "passed": true,
          "dropped_at_counter": "cthost",
          "captures": {
            "node_ingress_fv_fh": {
              "packets": 1,
              "ip_bytes": 84,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 84,
              "max_ip_len": 84,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "100.64.1.1",
                  "dst": "203.0.113.33",
                  "icmp_type": 8,
                  "icmp_code": 0,
                  "mtu": 1,
                  "ip_len": 84
                }
              ]
            },
            "node_egress_fv_ih": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "router_fv_ii": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 1,
              "bytes": 84
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        }
      ]
    },
    {
      "id": "denied_v4_and_host_v4_overlap",
      "source_ip": "192.168.34.1",
      "expected_sets": [
        "denied_v4",
        "host_v4"
      ],
      "set_membership": {
        "in_resolver_v4": false,
        "in_denied_v4": true,
        "in_host_v4": true,
        "matched_sets": [
          "denied_v4",
          "host_v4"
        ]
      },
      "set_membership_verified": true,
      "upload_drop_counter": "ctdeny",
      "passed": false,
      "subcases": [
        {
          "name": "related-frag-needed",
          "passed": false,
          "pmtu_before": null,
          "transfer": {
            "ok": false,
            "error": "[Errno 104] Connection reset by peer",
            "socket_pmtu": null,
            "socket_mss": null,
            "elapsed_ms": 0.83,
            "pmtu": 1000
          },
          "captures": {
            "router_fv_ii": {
              "packets": 10,
              "ip_bytes": 5760,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "node_up_egress_fv_ih": {
              "packets": 29,
              "ip_bytes": 26372,
              "tcp_packets": 29,
              "tcp_ip_bytes": 26372,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1160
              ],
              "icmp": []
            },
            "node_down_egress_fv_fh": {
              "packets": 23,
              "ip_bytes": 6420,
              "tcp_packets": 13,
              "tcp_ip_bytes": 660,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 576,
              "max_tcp_payload": 60,
              "syn_mss": [
                1160
              ],
              "icmp": [
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            },
            "member_fv_wg0": {
              "packets": 52,
              "ip_bytes": 32792,
              "tcp_packets": 42,
              "tcp_ip_bytes": 27032,
              "icmp_packets": 10,
              "icmp_ip_bytes": 5760,
              "max_ip_len": 1200,
              "max_tcp_payload": 1148,
              "syn_mss": [
                1380,
                1160
              ],
              "icmp": [
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                },
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 576
                }
              ]
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 29,
              "bytes": 26372
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 23,
              "bytes": 6420
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          },
          "usage_reconciliation": {
            "device_id": "019a00000000700080000000000000a1",
            "assigned_priority": "0x0001002f",
            "assigned_tc_class": "1:2f",
            "upload_counter_matches_capture": true,
            "download_counter_matches_capture": true,
            "download_tcp_ip_bytes": 660,
            "download_frag_needed_icmp_packets": 10,
            "download_frag_needed_icmp_ip_bytes": 5760,
            "download_total_ip_bytes": 6420,
            "tc_upload_leaf": {
              "packets": 29,
              "bytes": 26778,
              "drops": 0
            },
            "tc_download_leaf": {
              "packets": 23,
              "bytes": 6742,
              "drops": 0
            },
            "tc_upload_all": {
              "1:1": {
                "packets": 29,
                "bytes": 26778,
                "drops": 0
              },
              "1:2f": {
                "packets": 29,
                "bytes": 26778,
                "drops": 0
              },
              "1:fffe": {
                "packets": 0,
                "bytes": 0,
                "drops": 0
              }
            },
            "tc_download_all": {
              "1:1": {
                "packets": 23,
                "bytes": 6742,
                "drops": 0
              },
              "1:2f": {
                "packets": 23,
                "bytes": 6742,
                "drops": 0
              },
              "1:fffe": {
                "packets": 0,
                "bytes": 0,
                "drops": 0
              }
            }
          }
        },
        {
          "name": "time-exceeded-dropped",
          "passed": true,
          "dropped_at_counter": "ctdenyd",
          "captures": {
            "router_fv_ii": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 112,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 112,
              "max_ip_len": 112,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.1",
                  "dst": "192.0.2.37",
                  "icmp_type": 11,
                  "icmp_code": 0,
                  "mtu": 0,
                  "ip_len": 112
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 1,
              "bytes": 84
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 1,
              "bytes": 112
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "unsolicited-frag-needed-dropped",
          "passed": true,
          "dropped_at_counter": "ctinv",
          "captures": {
            "node_ingress_fv_ih": {
              "packets": 1,
              "ip_bytes": 60,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 60,
              "max_ip_len": 60,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "192.168.34.1",
                  "dst": "100.64.1.1",
                  "icmp_type": 3,
                  "icmp_code": 4,
                  "mtu": 1000,
                  "ip_len": 60
                }
              ]
            },
            "node_egress_fv_fh": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "member_fv_wg0": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 0,
              "bytes": 0
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 1,
              "bytes": 60
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        },
        {
          "name": "member-upload-to-source-dropped",
          "passed": true,
          "dropped_at_counter": "ctdeny",
          "captures": {
            "node_ingress_fv_fh": {
              "packets": 1,
              "ip_bytes": 84,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 1,
              "icmp_ip_bytes": 84,
              "max_ip_len": 84,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": [
                {
                  "src": "100.64.1.1",
                  "dst": "192.168.34.1",
                  "icmp_type": 8,
                  "icmp_code": 0,
                  "mtu": 1,
                  "ip_len": 84
                }
              ]
            },
            "node_egress_fv_ih": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            },
            "router_fv_ii": {
              "packets": 0,
              "ip_bytes": 0,
              "tcp_packets": 0,
              "tcp_ip_bytes": 0,
              "icmp_packets": 0,
              "icmp_ip_bytes": 0,
              "max_ip_len": 0,
              "max_tcp_payload": 0,
              "syn_mss": [],
              "icmp": []
            }
          },
          "counters": {
            "u_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "d_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "du_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "dd_019a00000000700080000000000000a1": {
              "packets": 0,
              "bytes": 0
            },
            "ctdns": {
              "packets": 0,
              "bytes": 0
            },
            "ctdeny": {
              "packets": 1,
              "bytes": 84
            },
            "cthost": {
              "packets": 0,
              "bytes": 0
            },
            "ctdenyd": {
              "packets": 0,
              "bytes": 0
            },
            "ctinv": {
              "packets": 0,
              "bytes": 0
            },
            "ctnodown": {
              "packets": 0,
              "bytes": 0
            },
            "ctunkd": {
              "packets": 0,
              "bytes": 0
            }
          }
        }
      ]
    }
  ]
}


Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`
