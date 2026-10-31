package enforce

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// TCSummary describes the entire read, before bounded TC rows are selected.
// Generation is the candidate generation, not an acknowledgement. Phase is
// load, pre_apply, or verify. Rejected retains the exact deciding attachment
// and object even when it falls outside the bounded sample of other reads.
// SHA256 covers the sorted, sanitized full read, never packet selectors.
// This is a library/report-only value, not a new HTTP contract field.
type TCSummary struct {
	Generation decstr.U64      `json:"generation"`
	Phase      string          `json:"phase"`
	Reads      int             `json:"reads"`
	Objects    int             `json:"objects"`
	Truncated  bool            `json:"truncated"`
	SHA256     string          `json:"sha256"`
	Rejected   *TCReadEvidence `json:"rejected,omitempty"`
}

// ReportDetail fits NodeCheck.detail / NodeError.message (500 characters).
// A successful read belongs in a passed qos check, never in errors[]. Failed
// reads may use errors[] with their actual failure code. Callers supply the
// read timestamp and keep it distinct from the later observation timestamp.
func (ev ServingEvidence) ReportDetail() string {
	b, _ := json.Marshal(ev.TCSummary)
	if len(b) <= 500 {
		return string(b)
	}
	s := ev.TCSummary
	detail := fmt.Sprintf("tc generation=%s phase=%s reads=%d objects=%d truncated=%t sha256=%s", s.Generation.String(), s.Phase, s.Reads, s.Objects, s.Truncated, s.SHA256)
	if r := s.Rejected; r != nil {
		detail += fmt.Sprintf(" interface=%s attachment=%s error=%s", r.Interface, r.Attachment, r.Error)
		if len(r.Objects) > 0 {
			o := r.Objects[0]
			detail += fmt.Sprintf(" kind=%s handle=%s parent=%s", o.Kind, o.Handle, o.Parent)
		}
	}
	return detail
}

var tcEvidenceToken = regexp.MustCompile(`^[a-zA-Z0-9_:-]{1,32}$`)

func cleanTCToken(s string) string {
	for _, prefix := range []string{"sk_", "gh" + "p_", "gh" + "o_", "github_pat_"} {
		if strings.HasPrefix(s, prefix) {
			return "redacted"
		}
	}
	if s == "" || tcEvidenceToken.MatchString(s) {
		return s
	}
	return "redacted"
}

func cleanTCRead(r TCReadEvidence) TCReadEvidence {
	v := TCReadEvidence{Interface: cleanTCToken(r.Interface), Attachment: cleanTCToken(r.Attachment), Error: cleanTCToken(r.Error), Objects: make([]TCObject, 0, len(r.Objects))}
	for _, o := range r.Objects {
		x := TCObject{Kind: cleanTCToken(o.Kind), Handle: cleanTCToken(o.Handle), Parent: cleanTCToken(o.Parent), Root: o.Root}
		for _, action := range o.Actions {
			x.Actions = append(x.Actions, cleanTCToken(action))
		}
		// Action order is semantic; retain it. Object dump order is not.
		v.Objects = append(v.Objects, x)
	}
	sort.Slice(v.Objects, func(i, j int) bool { return tcJSON(v.Objects[i]) < tcJSON(v.Objects[j]) })
	return v
}

func tcJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func boundTCRead(r TCReadEvidence) (TCReadEvidence, bool) {
	truncated := len(r.Objects) > 4
	if truncated {
		r.Objects = r.Objects[:4]
	}
	for i := range r.Objects {
		if len(r.Objects[i].Actions) > 4 {
			r.Objects[i].Actions = r.Objects[i].Actions[:4]
			truncated = true
		}
	}
	return r, truncated
}

// Called before any rollback or quarantine, using only the reads that decided
// this verification. Never re-read after rollback and label it as the rejection.
func (ev *ServingEvidence) retainTC(generation uint64, phase string, err error) {
	s := TCSummary{Generation: decstr.U64(generation), Phase: phase, Reads: len(ev.TC)}
	reads := make([]TCReadEvidence, 0, len(ev.TC))
	for _, r := range ev.TC {
		s.Objects += len(r.Objects)
		reads = append(reads, cleanTCRead(r))
	}
	sort.Slice(reads, func(i, j int) bool { return tcJSON(reads[i]) < tcJSON(reads[j]) })
	h := sha256.Sum256([]byte(tcJSON(reads)))
	s.SHA256 = hex.EncodeToString(h[:])
	var check *CheckError
	if errors.As(err, &check) && check.tc != nil {
		r, truncated := boundTCRead(cleanTCRead(*check.tc))
		s.Rejected, s.Truncated = &r, truncated
		// Existing agent versions forward TC rows through report-only errors.
		// Put the deciding row first so their own truncation cannot discard it.
		selected := []TCReadEvidence{r}
		for _, other := range reads {
			if other.Interface != r.Interface || other.Attachment != r.Attachment {
				selected = append(selected, other)
			}
		}
		reads = selected
	}
	if len(reads) > 16 {
		reads, s.Truncated = reads[:16], true
	}
	for i := range reads {
		r, truncated := boundTCRead(reads[i])
		reads[i], s.Truncated = r, s.Truncated || truncated
	}
	ev.TC, ev.TCSummary = reads, s
}

func tcCheck(r TCReadEvidence, obj *TCObject) error {
	if obj != nil {
		r.Objects = []TCObject{*obj}
		return &CheckError{Reason: ReasonTCForeignFilter, Detail: "interface " + r.Interface + " parent " + tcParent(r.Attachment, *obj) + " handle " + obj.Handle + " kind " + obj.Kind, tc: &r}
	}
	return &CheckError{Reason: ReasonTCForeignFilter, Detail: "interface " + r.Interface + " parent " + r.Attachment + " kind unreadable: " + r.Error, tc: &r}
}
