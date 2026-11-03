package isolation

import (
	"regexp"
	"strconv"
	"strings"
)

// ForwardView is the inet fvpn forward chain on a node (§8.2, §19 AT10).
// A member accept is an fvmb0 accept. NetBird fvnb0 accepts in the same
// chain do not count, and neither does policy accept on another chain.
// D-146 adds two non-verdict TCP SYN MSS clamps per admitted address (MSSClamp)
// and D-147 adds three RELATED ICMPv4 fragmentation-needed accepts per admitted
// address (PMTUAccept). LabMemberAccept counts admitted-member accept rules on
// either fvmb0 or lab fv-fh while excluding synthetic-probe sets.
type ForwardView struct {
	Present         bool
	Policy          string
	Baseline        bool
	Generation      int
	MemberAccept    int
	LabMemberAccept int
	MSSClamp        int
	PMTUAccept      int
	Excerpt         string
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
		if mssClampRule(trim) {
			view.MSSClamp++
		}
		if pmtuAcceptRule(trim) {
			view.PMTUAccept++
		}
		if admittedMemberAcceptRule(trim) {
			view.LabMemberAccept++
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

func admittedMemberAcceptRule(line string) bool {
	if strings.Contains(line, "policy ") || strings.Contains(line, "drop") || !strings.Contains(line, "accept") {
		return false
	}
	if strings.Contains(line, "@probe_v4") || strings.Contains(line, "@probe_dst_v4") {
		return false
	}
	if !strings.Contains(line, `"fvmb0"`) && !strings.Contains(line, `"fv-fh"`) {
		return false
	}
	return strings.Contains(line, "iifname") || strings.Contains(line, "oifname") || strings.Contains(line, "ip saddr") || strings.Contains(line, "ip daddr")
}

func mssClampRule(line string) bool {
	if !strings.Contains(line, `"fvmb0"`) && !strings.Contains(line, `"fv-fh"`) {
		return false
	}
	return (strings.Contains(line, "tcp flags & syn == syn") || strings.Contains(line, "tcp flags & (syn) == syn")) &&
		strings.Contains(line, "tcp option maxseg size > 1160") &&
		strings.Contains(line, "tcp option maxseg size set 1160")
}

func pmtuAcceptRule(line string) bool {
	if !admittedMemberAcceptRule(line) {
		return false
	}
	return strings.Contains(line, "ct state related") &&
		strings.Contains(line, "icmp type destination-unreachable") &&
		strings.Contains(line, "icmp code frag-needed")
}
