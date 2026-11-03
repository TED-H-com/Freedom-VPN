package isolation

import (
	_ "embed"
	"encoding/json"
)

//go:embed expected-blocked.json
var expectedBlockedJSON []byte

type ExpectedBlock struct {
	Case          string `json:"case"`
	Row           string `json:"row"`
	Reason        string `json:"reason"`
	Owner         string `json:"owner"`
	EstablishedBy string `json:"established_by"`
}

func ExpectedBlocks() []ExpectedBlock {
	var entries []ExpectedBlock
	if json.Unmarshal(expectedBlockedJSON, &entries) != nil {
		return nil
	}
	return entries
}

// ScheduledCaseAcceptable maps only the scheduled job's exit. It never changes
// a Record or Check. Real Go test errors remain errors regardless of row status.
func ScheduledCaseAcceptable(r Record) bool {
	if len(r.Checks) == 0 || (r.Status != StatusPass && r.Status != StatusBlocked) {
		return false
	}
	seen := map[string]bool{}
	for _, c := range r.Checks {
		if seen[c.Name] {
			return false
		}
		seen[c.Name] = true
		if c.Status == StatusPass {
			if !c.Positive.Valid() {
				return false
			}
			continue
		}
		if c.Status != StatusBlocked {
			return false
		}
		matched := false
		for _, e := range ExpectedBlocks() {
			if e.Case == r.ID && e.Row == c.Name && e.Owner != "" && e.Reason != "" && e.EstablishedBy != "" {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
