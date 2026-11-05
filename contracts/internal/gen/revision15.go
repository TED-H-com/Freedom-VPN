package gen

import "strings"

// revision15Files adds deterministic D-164 ServingReadEvidence examples,
// ObservationReport/NodeDetail serving_read presence fixtures, invalid
// schema fixtures, and the shared serving-read-evidence vector.
func revision15Files(out map[string][]byte, reasons map[string]Reason) {
	applyVerified := map[string]any{
		"read_at":              "2026-10-28T15:00:29Z",
		"source":               "apply",
		"candidate_generation": "9007199254740993",
		"verified":             true,
		"diagnostic":           `read_at=2026-10-28T15:00:29Z source=apply candidate=9007199254740993 verified=true {"generation":"9007199254740993","phase":"verify","reads":2,"objects":4,"truncated":false,"sha256":"dc031120dc666039402e235c0416643a9d76d09c41ef57f4b2cfef65378033a1"}`,
	}
	liveVerified := map[string]any{
		"read_at":              "2026-10-28T15:00:30Z",
		"source":               "live",
		"candidate_generation": "18446744073709551615",
		"verified":             true,
		"diagnostic":           `read_at=2026-10-28T15:00:30Z source=live candidate=18446744073709551615 verified=true {"generation":"18446744073709551615","phase":"verify","reads":2,"objects":4,"truncated":false,"sha256":"dc031120dc666039402e235c0416643a9d76d09c41ef57f4b2cfef65378033a1"}`,
	}
	applyRejected := map[string]any{
		"read_at":              "2026-10-28T15:00:10Z",
		"source":               "apply",
		"candidate_generation": "18446744073709551615",
		"verified":             false,
		"diagnostic":           `read_at=2026-10-28T15:00:10Z source=apply candidate=18446744073709551615 verified=false {"generation":"18446744073709551615","phase":"verify","reads":2,"objects":5,"truncated":false,"sha256":"6d78e62925fc23e6d6864f986fa220987447329d5b109c7b71aa0958a2d57c91","rejected":{"interface":"fv0","attachment":"ingress","objects":[{"kind":"flower","handle":"0x1","parent":"ffff:fff1","actions":["gact"]}]}}`,
	}
	liveRejectedTC := map[string]any{
		"read_at":              "2026-10-28T15:00:15Z",
		"source":               "live",
		"candidate_generation": "9007199254740993",
		"verified":             false,
		"diagnostic":           `read_at=2026-10-28T15:00:15Z source=live candidate=9007199254740993 verified=false {"generation":"9007199254740993","phase":"verify","reads":17,"objects":68,"truncated":true,"sha256":"f2606d3330f3202863fd0a38fa3be177220bc78596c0032b00ff4936a4a3f1e5","rejected":{"interface":"fv0","attachment":"ingress","objects":[{"kind":"u32","handle":"800::800","parent":"ffff:fff1","actions":["mirred"]}]}}`,
	}

	out["examples/valid/ServingReadEvidence.apply-verified.json"] = pretty(applyVerified)
	out["examples/valid/ServingReadEvidence.live-verified.json"] = pretty(liveVerified)
	out["examples/valid/ServingReadEvidence.apply-rejected.json"] = pretty(applyRejected)
	out["examples/valid/ServingReadEvidence.live-rejected-tc.json"] = pretty(liveRejectedTC)

	obsWith := func(hasServingRead bool, sr any) map[string]any {
		m := map[string]any{
			"report_id":                 "9d93e97b-dc01-40c0-8de6-0ba0b2f6e14d",
			"node_id":                   idNode,
			"node_incarnation_id":       idIncarnation,
			"boot_id":                   idBoot,
			"observed_at":               "2026-10-28T15:00:30Z",
			"agent_version":             "1.1.0",
			"current_release_id":        idReleaseR42,
			"last_good_release_id":      idReleaseR42,
			"applied_config_generation": "128",
			"applied_policy_refs": map[string]any{
				"authorization_generation": "52",
				"fup_generation":           "9",
				"dns_generation":           "12",
			},
			"applied_binding_revision":  "407",
			"provisioning_state":        "ready",
			"safety_status":             "passed",
			"data_health":               "healthy",
			"member_forwarding_enabled": false,
			"forwarding_block_reasons":  []string{"not_published"},
			"lease": map[string]any{
				"lease_id":   idLease,
				"expires_at": "2026-10-29T15:00:00Z",
			},
			"netbird": map[string]any{
				"peer_id": "peer-test-node-01",
				"version": "0.76.2",
				"backend": "kernel",
			},
			"usage": map[string]any{
				"ledger_epoch":           idLedgerEpoch,
				"meter_epoch_id":         idMeterEpoch15,
				"last_durable_event_seq": "12060",
				"central_acked_through":  "12058",
				"blocked_seq":            nil,
				"unacked_event_count":    2,
				"unacked_payload_bytes":  "1840",
				"oldest_unacked_at":      "2026-10-28T15:00:10Z",
				"acked_archive_bytes":    "7340032",
			},
			"update": map[string]any{
				"job_id":     nil,
				"generation": nil,
				"state":      nil,
				"stage":      nil,
			},
			"checks": []any{
				map[string]any{
					"name":        "internet_only",
					"status":      "passed",
					"observed_at": "2026-10-28T15:00:30Z",
					"error_code":  nil,
				},
			},
			"errors": []any{},
		}
		if hasServingRead {
			m["serving_read"] = sr
		}
		return m
	}

	obsAbsent := obsWith(false, nil)
	obsNull := obsWith(true, nil)
	obsApplyVerified := obsWith(true, applyVerified)
	obsLiveRejected := obsWith(true, map[string]any{
		"read_at":              "2026-10-28T15:00:10Z",
		"source":               "live",
		"candidate_generation": "18446744073709551615",
		"verified":             false,
		"diagnostic":           `read_at=2026-10-28T15:00:10Z source=live candidate=18446744073709551615 verified=false {"generation":"18446744073709551615","phase":"verify","reads":2,"objects":5,"truncated":false,"sha256":"6d78e62925fc23e6d6864f986fa220987447329d5b109c7b71aa0958a2d57c91","rejected":{"interface":"fv0","attachment":"ingress","objects":[{"kind":"flower","handle":"0x1","parent":"ffff:fff1","actions":["gact"]}]}}`,
	})

	out["examples/valid/ObservationReport.serving-read-null.json"] = pretty(obsNull)
	out["examples/valid/ObservationReport.serving-read-apply-verified.json"] = pretty(obsApplyVerified)
	out["examples/valid/ObservationReport.serving-read-live-rejected.json"] = pretty(obsLiveRejected)

	nodeWith := func(hasServingRead bool, sr any) map[string]any {
		m := map[string]any{
			"node_id":                    idNode,
			"exit_id":                    "caba5aec-4b2b-432c-9512-211a7b3e96db",
			"name":                       "Tokyo 1",
			"region":                     "jp-tokyo",
			"member_group_ids":           []string{"63575bae-2d23-45c0-ae11-ba6bb9ace34a"},
			"lifecycle_state":            "registered",
			"provisioning_state":         "ready",
			"publication_intent":         "disabled",
			"publication_observed":       "withdrawn",
			"control_status":             "online",
			"data_health":                "healthy",
			"safety_status":              "passed",
			"member_forwarding_observed": false,
			"config_generation": map[string]any{
				"desired": "128",
				"applied": "127",
			},
			"releases": map[string]any{
				"current_release_id":   idReleaseR41,
				"desired_release_id":   idReleaseR42,
				"last_good_release_id": idReleaseR41,
			},
			"update": map[string]any{
				"batch_id":         idBatch,
				"job_id":           idJob,
				"generation":       "7",
				"state":            "downloading",
				"last_result_at":   "2026-10-28T14:06:00Z",
				"last_observed_at": "2026-10-28T14:06:30Z",
				"last_error_code":  nil,
			},
			"last_seen_at":           "2026-10-28T14:06:30Z",
			"resource_version":       "42",
			"created_at":             "2026-10-01T09:00:00Z",
			"current_incarnation_id": idIncarnation,
			"platform": map[string]any{
				"os":         "ubuntu",
				"os_version": "24.04",
				"arch":       "arm64",
			},
			"capacity": map[string]any{
				"down_bps":            1000000000,
				"up_bps":              1000000000,
				"member_down_bps_cap": 200000000,
				"member_up_bps_cap":   100000000,
			},
			"notes":              "Pre-installed box; source noted for reference only.",
			"netbird_peer_id":    "peer-test-node-01",
			"observed_exit_ipv4": "203.0.113.10",
			"observed_exit_ipv6": nil,
			"checks": []any{
				map[string]any{
					"name":        "internet_only",
					"status":      "passed",
					"observed_at": "2026-10-28T14:06:30Z",
					"error_code":  nil,
				},
				map[string]any{
					"name":        "meter_path",
					"status":      "passed",
					"observed_at": "2026-10-28T14:06:30Z",
					"error_code":  nil,
				},
			},
			"recent_errors": []any{},
			"drift": []any{
				map[string]any{
					"field":    "config_generation",
					"desired":  "128",
					"observed": "127",
				},
			},
			"retired_at": nil,
		}
		if hasServingRead {
			m["serving_read"] = sr
		}
		return m
	}

	nodeAbsent := nodeWith(false, nil)
	nodeNull := nodeWith(true, nil)
	nodeApplyVerified := nodeWith(true, map[string]any{
		"read_at":              "2026-10-28T14:06:29Z",
		"source":               "apply",
		"candidate_generation": "9007199254740993",
		"verified":             true,
		"diagnostic":           `read_at=2026-10-28T14:06:29Z source=apply candidate=9007199254740993 verified=true {"generation":"9007199254740993","phase":"verify","reads":2,"objects":4,"truncated":false,"sha256":"dc031120dc666039402e235c0416643a9d76d09c41ef57f4b2cfef65378033a1"}`,
	})
	nodeLiveRejected := nodeWith(true, map[string]any{
		"read_at":              "2026-10-28T14:06:10Z",
		"source":               "live",
		"candidate_generation": "18446744073709551615",
		"verified":             false,
		"diagnostic":           `read_at=2026-10-28T14:06:10Z source=live candidate=18446744073709551615 verified=false {"generation":"18446744073709551615","phase":"verify","reads":2,"objects":5,"truncated":false,"sha256":"6d78e62925fc23e6d6864f986fa220987447329d5b109c7b71aa0958a2d57c91","rejected":{"interface":"fv0","attachment":"ingress","objects":[{"kind":"flower","handle":"0x1","parent":"ffff:fff1","actions":["gact"]}]}}`,
	})

	out["examples/valid/NodeDetail.serving-read-null.json"] = pretty(nodeNull)
	out["examples/valid/NodeDetail.serving-read-apply-verified.json"] = pretty(nodeApplyVerified)
	out["examples/valid/NodeDetail.serving-read-live-rejected.json"] = pretty(nodeLiveRejected)

	cloneEvidence := func() map[string]any {
		return map[string]any{
			"read_at":              "2026-10-28T15:00:29Z",
			"source":               "apply",
			"candidate_generation": "9007199254740993",
			"verified":             true,
			"diagnostic":           `read_at=2026-10-28T15:00:29Z source=apply candidate=9007199254740993 verified=true {"generation":"9007199254740993","phase":"verify","reads":2,"objects":4,"truncated":false,"sha256":"dc031120dc666039402e235c0416643a9d76d09c41ef57f4b2cfef65378033a1"}`,
		}
	}

	invalidSpecs := []struct {
		name    string
		pointer string
		keyword string
		note    string
		mutate  func(map[string]any)
	}{
		{
			name: "missing-read-at", pointer: "", keyword: "required",
			note:   "ServingReadEvidence requires read_at (D-164).",
			mutate: func(m map[string]any) { delete(m, "read_at") },
		},
		{
			name: "missing-source", pointer: "", keyword: "required",
			note:   "ServingReadEvidence requires source (D-164).",
			mutate: func(m map[string]any) { delete(m, "source") },
		},
		{
			name: "missing-candidate-generation", pointer: "", keyword: "required",
			note:   "ServingReadEvidence requires candidate_generation (D-164).",
			mutate: func(m map[string]any) { delete(m, "candidate_generation") },
		},
		{
			name: "missing-verified", pointer: "", keyword: "required",
			note:   "ServingReadEvidence requires verified (D-164).",
			mutate: func(m map[string]any) { delete(m, "verified") },
		},
		{
			name: "missing-diagnostic", pointer: "", keyword: "required",
			note:   "ServingReadEvidence requires diagnostic (D-164).",
			mutate: func(m map[string]any) { delete(m, "diagnostic") },
		},
		{
			name: "unknown-source", pointer: "/source", keyword: "enum",
			note:   "ServingReadEvidence.source allows only apply or live (D-164).",
			mutate: func(m map[string]any) { m["source"] = "rollback" },
		},
		{
			name: "unexpected-error-code", pointer: "", keyword: "additionalProperties",
			note:   "ServingReadEvidence has no error_code member and sets additionalProperties: false (D-164).",
			mutate: func(m map[string]any) { m["error_code"] = "GENERATION_MISMATCH" },
		},
		{
			name: "non-utc-read-at", pointer: "/read_at", keyword: "pattern",
			note:   "ServingReadEvidence.read_at must be UTC RFC 3339 with Z.",
			mutate: func(m map[string]any) { m["read_at"] = "2026-10-28T15:00:29+00:00" },
		},
		{
			name: "numeric-candidate-generation", pointer: "/candidate_generation", keyword: "type",
			note:   "ServingReadEvidence.candidate_generation must be a decimal string, never a JSON number (§10.4).",
			mutate: func(m map[string]any) { m["candidate_generation"] = 128 },
		},
		{
			name: "leading-zero-candidate-generation", pointer: "/candidate_generation", keyword: "pattern",
			note:   "ServingReadEvidence.candidate_generation forbids leading zeros (§10.4).",
			mutate: func(m map[string]any) { m["candidate_generation"] = "01" },
		},
		{
			name: "too-long-candidate-generation", pointer: "/candidate_generation", keyword: "maxLength",
			note:   "ServingReadEvidence.candidate_generation has at most 20 digits (§14.1).",
			mutate: func(m map[string]any) { m["candidate_generation"] = "100000000000000000000" },
		},
		{
			name: "diagnostic-too-long", pointer: "/diagnostic", keyword: "maxLength",
			note:   "ServingReadEvidence.diagnostic is bounded to 500 characters (D-164).",
			mutate: func(m map[string]any) { m["diagnostic"] = strings.Repeat("x", 501) },
		},
		{
			name: "unknown-property", pointer: "", keyword: "additionalProperties",
			note:   "ServingReadEvidence sets additionalProperties: false (D-164).",
			mutate: func(m map[string]any) { m["raw_stderr"] = "forbidden" },
		},
	}

	var invalidCases []any
	for _, spec := range invalidSpecs {
		m := cloneEvidence()
		spec.mutate(m)
		fileName := "ServingReadEvidence." + spec.name + ".json"
		out["examples/invalid/"+fileName] = pretty(m)
		reasons[fileName] = Reason{
			Kind:    "schema",
			Pointer: spec.pointer,
			Keyword: spec.keyword,
			Note:    spec.note,
		}
		invalidCases = append(invalidCases, map[string]any{
			"name":     spec.name,
			"kind":     "schema",
			"pointer":  spec.pointer,
			"keyword":  spec.keyword,
			"evidence": m,
		})
	}

	overflowEvidence := cloneEvidence()
	overflowEvidence["candidate_generation"] = "18446744073709551616"
	invalidCases = append(invalidCases, map[string]any{
		"name":     "overflow-candidate-generation",
		"kind":     "uint64_overflow",
		"pointer":  "/candidate_generation",
		"keyword":  "decstr.U64",
		"evidence": overflowEvidence,
	})

	out["vectors/serving-read-evidence.json"] = pretty(map[string]any{
		"_comment": TestOnlyLabel + ". D-164 retained actual complete VerifyServing read outcome (ServingReadEvidence) and nullable presence on ObservationReport and NodeDetail.",
		"cases": []any{
			map[string]any{"name": "apply-verified", "evidence": applyVerified},
			map[string]any{"name": "live-verified", "evidence": liveVerified},
			map[string]any{"name": "apply-rejected", "evidence": applyRejected},
			map[string]any{"name": "live-rejected-tc", "evidence": liveRejectedTC},
		},
		"observation_reports": []any{
			map[string]any{"name": "absent", "state": "absent", "report": obsAbsent},
			map[string]any{"name": "null", "state": "null", "report": obsNull},
			map[string]any{"name": "apply-verified", "state": "present", "report": obsApplyVerified},
			map[string]any{"name": "live-rejected-retained", "state": "present", "report": obsLiveRejected},
		},
		"node_details": []any{
			map[string]any{"name": "absent", "state": "absent", "node": nodeAbsent},
			map[string]any{"name": "null", "state": "null", "node": nodeNull},
			map[string]any{"name": "apply-verified", "state": "present", "node": nodeApplyVerified},
			map[string]any{"name": "live-rejected-retained", "state": "present", "node": nodeLiveRejected},
		},
		"invalid_cases": invalidCases,
	})
}
