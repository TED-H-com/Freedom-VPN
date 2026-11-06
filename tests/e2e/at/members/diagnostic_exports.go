package members

import "github.com/TED-H-com/Freedom-VPN/pkg/api"

func summarizeNodeDetail(node api.NodeDetail) map[string]any {
	applied, _ := node.ConfigGeneration.Applied.Get()
	forwarding, _ := node.MemberForwardingObserved.Get()
	checkMap := func(target api.NodeCheckName) map[string]any {
		for _, c := range node.Checks {
			if c.Name == target {
				code, _ := c.ErrorCode.Get()
				detail := ""
				if c.Detail != nil {
					detail = *c.Detail
				}
				return map[string]any{"status": c.Status, "error_code": code, "detail": detail, "observed_at": c.ObservedAt}
			}
		}
		return nil
	}
	var recentErrors []map[string]any
	for _, e := range node.RecentErrors {
		recentErrors = append(recentErrors, map[string]any{"stage": e.Stage, "code": e.Code, "message": e.Message, "observed_at": e.ObservedAt})
	}
	out := map[string]any{
		"desired_generation":         node.ConfigGeneration.Desired,
		"applied_generation":         applied,
		"provisioning_state":         node.ProvisioningState,
		"publication_intent":         node.PublicationIntent,
		"publication_observed":       node.PublicationObserved,
		"safety_status":              node.SafetyStatus,
		"member_forwarding_observed": forwarding,
		"lease_check":                checkMap(api.NodeCheckNameLease),
		"qos_check":                  checkMap(api.NodeCheckNameQos),
		"source_binding_check":       checkMap(api.NodeCheckNameSourceBinding),
		"recent_errors":              recentErrors,
	}
	if node.ServingRead.IsSpecified() {
		out["serving_read"] = nil
		if !node.ServingRead.IsNull() {
			value, _ := node.ServingRead.Get()
			out["serving_read"] = value
		}
	}
	return out
}
