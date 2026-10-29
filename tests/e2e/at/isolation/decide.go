//go:build e2e

package isolation

import "strings"

type productObs struct {
	stop             string
	nodeID           string
	incarnation      string
	policyRevision   string
	publication      string
	operationStatus  string
	operationKind    string
	steps            string
	operationError   string
	idempotentStatus string
	bindings         int
	resolvers        int
	forward          ForwardView
	safety           string
	provisioning     string
	control          string
	dataHealth       string
	memberRoute      string
	targets          string
	mechanism        string
	notes            []Observation
}

func emptyStage(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", ",")
	return clip(s, 400)
}

// exitProbe is what a member observed. A destination counts only when its
// route uses fvnb0 (§19 AT10). The zero value means the probe did not run.
type exitProbe struct {
	lines           []string
	protectedVia    int
	protectedAllows int
	protectedRefuse int
	internet        string
	flap            string
	rate            string
	ipv6            string
	dns             string
	members         string
	agentKill       string
	netbird         string
	reboot          string
	updater         string
	fail            string
}

type caseResult struct {
	status string
	actual string
}

func decideProduct(obs productObs, probe exitProbe) (at10, at11, at12, at13 caseResult) {
	blocked := func(extra string) caseResult {
		text := obs.stop
		if extra != "" {
			text = strings.TrimSpace(text + " " + extra)
		}
		if text == "" {
			text = "serving preconditions were not met"
		}
		return caseResult{StatusBlocked, text}
	}
	if obs.stop != "" {
		return blocked(""),
			blocked("ipv6=" + emptyStage(probe.ipv6)),
			blocked("members=" + emptyStage(probe.members)),
			blocked("reboot and updater were not run")
	}
	at10 = caseResult{StatusBlocked, "exit path was up but the allow and deny halves were not both proved"}
	if probe.fail != "" {
		at10 = caseResult{StatusFail, probe.fail}
	} else if probe.protectedVia > 0 && probe.protectedAllows == 0 && strings.Contains(probe.internet, "fvnb0 connected") && strings.Contains(probe.flap, "fvnb0 denied") && rateAtLeast(probe.rate, 30) {
		at10 = caseResult{StatusPass, "protected destinations via fvnb0 were denied, origin connected, flap " + probe.flap + ", " + probe.rate}
	} else {
		at10.actual = fmtAttempts(probe)
	}
	if strings.Contains(probe.ipv6, "no global") || probe.ipv6 == "" {
		at11 = caseResult{StatusBlocked, "dev stack member has no global IPv6 underlay and no IPv6 target; mode block was not exercised on the product path. " + probe.ipv6}
	} else if probe.fail != "" {
		at11 = caseResult{StatusFail, probe.fail}
	} else {
		at11 = caseResult{StatusBlocked, probe.ipv6}
	}
	at12 = caseResult{StatusBlocked, probe.members + "; a second member and a second device are not admitted on this stack. " + probe.dns}
	at13 = caseResult{StatusBlocked, strings.TrimSpace(probe.agentKill + " " + probe.netbird + " " + probe.reboot + " " + probe.updater)}
	if probe.fail != "" && (strings.Contains(probe.fail, "SIGKILL") || strings.Contains(probe.fail, "NetBird")) {
		at13 = caseResult{StatusFail, probe.fail}
	}
	return at10, at11, at12, at13
}

func fmtAttempts(probe exitProbe) string {
	parts := []string{
		"protected_via_fvnb0=" + itoa(probe.protectedVia),
		"delivered=" + itoa(probe.protectedAllows),
		"refused=" + itoa(probe.protectedRefuse),
		"internet=" + emptyStage(probe.internet),
		"flap=" + emptyStage(probe.flap),
		"rate=" + emptyStage(probe.rate),
	}
	if len(probe.lines) > 0 {
		parts = append(parts, strings.Join(probe.lines, "; "))
	}
	return strings.Join(parts, " ")
}

func rateAtLeast(rate string, min int) bool {
	n := 0
	for _, c := range rate {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n >= min && strings.Contains(rate, "attempts")
}
