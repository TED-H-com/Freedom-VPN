package gen

// revision14Files adds deterministic interoperability fixtures. These describe
// service outcomes; the runtime owners still implement D-105 and AT29.
func revision14Files(out map[string][]byte) {
	const current = "01000000-0000-4000-8000-000000000014"
	event := withHash(mustCanon(usageEvent), "payload_sha256")
	var cases []any
	for _, name := range []string{"accepted", "after-retirement-window", "idempotent-replay"} {
		ev := event
		status, ack, blocked := "accepted", "1234", any(nil)
		var eventError any
		if name == "after-retirement-window" {
			ev = withHash(setMember(event, []string{"interval_end"}, `"2026-10-28T14:00:31Z"`), "payload_sha256")
			status, ack, blocked = "rejected", "1233", "1234"
			eventError = map[string]any{"code": "VALIDATION_FAILED", "message": "Event ends after the retired stream hand-off window.", "fields": []any{}}
		} else if name == "idempotent-replay" {
			status = "duplicate"
		}
		request := map[string]any{"node_id": idNode, "node_incarnation_id": current, "origin_incarnation_id": idIncarnation, "ledger_epoch": idLedgerEpoch, "events": []any{rawJSON(ev)}}
		response := map[string]any{"node_id": idNode, "node_incarnation_id": idIncarnation, "ledger_epoch": idLedgerEpoch, "highest_contiguous_seq": ack, "blocked_seq": blocked,
			"results": []any{map[string]any{"event_seq": "1234", "status": status, "error": eventError}}, "received_at": TestNow}
		cases = append(cases, map[string]any{"name": name, "request": request, "response": response, "http_status": 200, "audited": name == "after-retirement-window"})
		if name == "after-retirement-window" {
			out["examples/semantic-invalid/UsageBatchRequest.retired-window.json"] = pretty(request)
		} else {
			out["examples/valid/UsageBatchRequest.handoff-"+name+".json"] = pretty(request)
		}
		out["examples/valid/UsageBatchResult.handoff-"+name+".json"] = pretty(response)
	}
	out["vectors/usage-retired-handoff.json"] = pretty(map[string]any{
		"_comment": TestOnlyLabel + ". D-105 semantic scenarios; accepted and replay requests are byte-identical.",
		"context":  map[string]any{"node_id": idNode, "retired_incarnation_id": idIncarnation, "retired_at": "2026-10-28T14:00:00Z", "sampling_interval_seconds": 30, "presenting_incarnation_id": current, "presenting_enrolled_at": "2026-10-28T14:00:01Z", "presenting_is_current": true, "previous_highest_contiguous_seq": "1233"},
		"cases":    cases,
	})

	var tcCases []any
	for _, name := range []string{"verified", "rejected", "truncated"} {
		obj := map[string]any{"kind": "fq_codel", "handle": "1:", "root": true}
		read := map[string]any{"interface": "fvwan0", "attachment": "qdisc", "objects": []any{obj}}
		rows := []any{read}
		reads, objects, truncated := 1, 1, false
		var rejected, code any
		if name == "rejected" {
			obj = map[string]any{"kind": "matchall", "parent": "ffff:", "actions": []string{"mirred"}}
			read = map[string]any{"interface": "fvwan0", "attachment": "ingress", "objects": []any{obj}}
			rows, rejected, code = []any{read}, read, "TC_FOREIGN_FILTER"
		} else if name == "truncated" {
			reads, objects, truncated = 17, 17, true
			for len(rows) < 16 {
				rows = append(rows, read)
			}
		}
		projection := pretty(map[string]any{"reads": reads, "objects": objects, "truncated": truncated, "tc": rows})
		canon := mustCanon(string(projection))
		evidence := map[string]any{"candidate_generation": "18446744073709551615", "read_at": "2026-10-28T15:00:29Z", "phase": "verify", "verified": name != "rejected", "error_code": code, "reads_total": reads, "objects_total": objects, "truncated": truncated, "sha256": hexSHA(canon), "attachments": rows, "rejected": rejected}
		out["examples/valid/TCReadbackEvidence."+name+".json"] = pretty(evidence)
		tcCases = append(tcCases, map[string]any{"name": name, "evidence": evidence, "canonical_projection": string(canon)})
	}
	out["vectors/tc-readback-evidence.json"] = pretty(map[string]any{"_comment": TestOnlyLabel + ". D-106 digest uses the existing netlab reads/objects/truncated/tc projection.", "cases": tcCases})

	operation := map[string]any{"operation_id": "8089f821-cd9b-4c3f-a526-20ee33e68ad5", "kind": "member.grant_revoke", "status": "pending", "resource": map[string]any{"type": "entitlement_grant", "id": "01000000-0000-4000-8000-000000000029"}, "resource_version": "1", "desired_generation": nil, "observed_generation": nil, "steps": []any{}, "error": nil, "request_id": "a12d4141-4f3e-487b-b638-e1488be3d4ee", "created_at": TestNow, "updated_at": TestNow, "completed_at": nil}
	out["examples/valid/Operation.grant-withdrawal.json"] = pretty(operation)
	out["vectors/member-grant-revoke.json"] = pretty(map[string]any{
		"_comment":     TestOnlyLabel + ". AT29 manual withdrawal is separate from expiry. Same-key replay returns the same operation.",
		"operation_id": "revokeMemberGrant", "method": "POST", "path": "/api/v1/admin/members/" + idMemberA + "/grants/01000000-0000-4000-8000-000000000029/revoke",
		"headers": map[string]any{"Idempotency-Key": "01000000-0000-4000-8000-000000000030"},
		"request": map[string]any{"reason": "TEST ONLY manual withdrawal."}, "response": operation, "http_status": 202, "audit_action": "member.grant_revoke",
		"cases": []any{map[string]any{"name": "finite", "http_status": 202}, map[string]any{"name": "open-ended", "http_status": 202}, map[string]any{"name": "same-key-replay", "http_status": 202, "same_operation": true}, map[string]any{"name": "unknown-or-other-member", "http_status": 404, "error_code": "RESOURCE_NOT_FOUND"}, map[string]any{"name": "already-revoked", "http_status": 409, "error_code": "STATE_CONFLICT"}, map[string]any{"name": "expired", "http_status": 409, "error_code": "STATE_CONFLICT"}},
	})
}
