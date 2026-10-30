package isolation

import (
	"regexp"
	"strconv"
	"strings"
)

// ForwardView is the inet fvpn forward chain on a node (§8.2, §19 AT10).
// A member accept is an fvmb0 accept. NetBird fvnb0 accepts in the same
// chain do not count, and neither does policy accept on another chain.
type ForwardView struct {
	Present      bool
	Policy       string
	Baseline     bool
	Generation   int
	MemberAccept int
	Excerpt      string
}

var generationRE = regexp.MustCompile(`fvpn_generation=([0-9]+)`)

// ParseForwardChain reads one nft list of the forward chain.
func ParseForwardChain(text string) ForwardView {
	start := strings.Index(text, "chain forward")
	if start < 0 {
		return ForwardView{Generation: -1}
	}
	body := text[start:]
	if end := strings.Index(body, "\n\t}"); end > 0 {
		body = body[:end]
	}
	view := ForwardView{Present: true, Generation: -1}
	var kept []string
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || trim == "chain forward {" {
			continue
		}
		if strings.Contains(trim, "hook forward") {
			switch {
			case strings.Contains(trim, "policy drop"):
				view.Policy = "drop"
			case strings.Contains(trim, "policy accept"):
				view.Policy = "accept"
			}
		}
		if strings.Contains(trim, "fvpn_baseline=1") {
			view.Baseline = true
		}
		if m := generationRE.FindStringSubmatch(trim); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				view.Generation = n
			}
		}
		if memberAcceptRule(trim) {
			view.MemberAccept++
		}
		if len(kept) < 8 {
			kept = append(kept, trim)
		}
	}
	view.Excerpt = strings.Join(kept, " | ")
	if len(view.Excerpt) > 500 {
		view.Excerpt = view.Excerpt[:500]
	}
	return view
}

func memberAcceptRule(line string) bool {
	if strings.Contains(line, "policy ") || strings.Contains(line, "drop") || !strings.Contains(line, "accept") {
		return false
	}
	if !strings.Contains(line, `"fvmb0"`) {
		return false
	}
	return strings.Contains(line, "iifname") || strings.Contains(line, "oifname") || strings.Contains(line, "ip saddr") || strings.Contains(line, "ip daddr")
}
