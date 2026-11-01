package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

type containmentEvidence struct {
	Generation        uint64                      `json:"-"`
	ReadAt            time.Time                   `json:"read_at"`
	InitialQuarantine *enforce.QuarantineEvidence `json:"initial_quarantine,omitempty"`
	Code              api.ErrorCode               `json:"code"`
	TC                []enforce.TCReadEvidence    `json:"tc,omitempty"`
	Quarantine        *enforce.QuarantineEvidence `json:"quarantine,omitempty"`
	ApplyQuarantine   *enforce.QuarantineEvidence `json:"apply_quarantine,omitempty"`
	Error             string                      `json:"error,omitempty"`
}

func (e *Engine) recordForeign(ev enforce.ServingEvidence) {
	if e.containment == nil {
		e.containment = &containmentEvidence{}
	}
	e.containment.Code = api.ErrorCodeTCFOREIGNFILTER
	// Retain attachment/action metadata only. Never serialize raw tc output,
	// selectors, action parameters, routes, or the counters on ServingEvidence.
	var reads []enforce.TCReadEvidence
	for _, r := range ev.TC {
		if len(reads) == 16 {
			break
		}
		if len(r.Objects) == 0 && r.Error == "" {
			continue
		}
		v := enforce.TCReadEvidence{Interface: evidenceToken(r.Interface), Attachment: evidenceToken(r.Attachment), Error: evidenceToken(r.Error)}
		for _, o := range r.Objects {
			if len(v.Objects) == 4 {
				break
			}
			x := enforce.TCObject{Kind: evidenceToken(o.Kind), Handle: evidenceToken(o.Handle), Parent: evidenceToken(o.Parent), Root: o.Root}
			for _, a := range o.Actions {
				if len(x.Actions) < 4 {
					x.Actions = append(x.Actions, evidenceToken(a))
				}
			}
			v.Objects = append(v.Objects, x)
		}
		reads = append(reads, v)
	}
	if len(reads) != 0 && len(e.containment.TC) == 0 {
		e.containment.TC = reads
	}
}

var evidenceTokenRE = regexp.MustCompile(`^[a-zA-Z0-9_:-]{1,32}$`)

func evidenceToken(s string) string {
	if s == "" {
		return ""
	}
	if !evidenceTokenRE.MatchString(s) {
		return "redacted"
	}
	return *sanitizeDetail(s)
}

func cleanQuarantine(q enforce.QuarantineEvidence) *enforce.QuarantineEvidence {
	q.Links = append([]enforce.QuarantineLink(nil), q.Links...)
	for i := range q.Links {
		l := &q.Links[i]
		l.Interface, l.Error = evidenceToken(l.Interface), evidenceToken(l.Error)
		if l.Before != nil {
			s := *l.Before
			s.OperState = evidenceToken(s.OperState)
			l.Before = &s
		}
		if l.After != nil {
			s := *l.After
			s.OperState = evidenceToken(s.OperState)
			l.After = &s
		}
	}
	q.Baseline.Error = evidenceToken(q.Baseline.Error)
	return &q
}

func (e *Engine) recordApplyQuarantine(res enforce.ApplyResult, err error) {
	hadEvidence := e.containment != nil
	e.recordForeign(res.Evidence)
	if !hadEvidence {
		code := api.ErrorCodeHEALTHCHECKFAILED
		var check *enforce.CheckError
		if errors.As(err, &check) && api.ErrorCode(check.Reason).Valid() {
			code = api.ErrorCode(check.Reason)
		}
		e.containment.Code = code
	}
	e.containment.ApplyQuarantine = cleanQuarantine(*res.Quarantine)
	e.quarantinePending, e.memberPathPending = true, true
}

func (e *Engine) quarantine(ctx context.Context) {
	if budget, ok := ctx.Value(localBudgetKey{}).(*localBudget); ok {
		if budget.quarantined {
			return
		}
		budget.quarantined = true
	}
	ctx, cancel := context.WithTimeout(localContext(ctx), 5*time.Second)
	defer cancel()
	e.serving, e.policyExpires = false, 0
	e.expectServing.Store(false)
	if e.containment == nil {
		e.containment = &containmentEvidence{Code: api.ErrorCodeTCFOREIGNFILTER}
	}
	if e.appliedModel == nil {
		e.containment.Quarantine = nil
		e.containment.Error = "retained_model_unavailable"
		return // Apply's own quarantine evidence is still reported.
	}
	if !e.planeLocked {
		lock, err := nodelock.Acquire(ctx, e.cfg.LockPath, 3, 50*time.Millisecond)
		if err != nil {
			e.containment.Quarantine = nil
			e.containment.Error = "lock_unavailable"
			return
		}
		defer lock.Release()
	}
	var q enforce.QuarantineEvidence
	err := e.baselineCall("foreign_tc_quarantine", func() error {
		var err error
		q, err = e.plane.Quarantine(ctx, *e.appliedModel)
		return err
	})
	e.containment.Quarantine = cleanQuarantine(q)
	if e.containment.InitialQuarantine == nil {
		e.containment.InitialQuarantine = cleanQuarantine(q)
	}
	e.containment.Error = ""
	if err != nil {
		e.containment.Error = "quarantine_incomplete"
	}
	e.baselineOK = q.Baseline.Verified
	e.lastObs = time.Time{}
}

func (e *Engine) containmentJSON() string {
	if e.containment == nil {
		return ""
	}
	b, _ := json.Marshal(e.containment)
	return string(b)
}

func (e *Engine) attachContainment(rep *api.ObservationReport, at time.Time) {
	if e.containment == nil {
		return
	}
	// Existing report-only error summaries carry structured sanitized evidence;
	// no HTTP contract extension is necessary.
	add := func(stage string, value any) {
		b, _ := json.Marshal(value)
		observed := at
		if stage == "tc_evidence" && !e.containment.ReadAt.IsZero() {
			observed = e.containment.ReadAt
		}
		rep.Errors = append(rep.Errors, api.NodeError{Code: e.containment.Code, Stage: stage, Message: clipMsg(string(b)), ObservedAt: observed})
	}
	if len(e.containment.TC) > 0 {
		add("tc_evidence", map[string]string{"candidate_generation": fmt.Sprint(e.containment.Generation), "read_at": e.containment.ReadAt.Format(time.RFC3339Nano), "source": "rejected"})
	}
	for _, r := range e.containment.TC {
		if len(r.Objects) == 0 {
			add("tc_evidence", r)
			continue
		}
		for _, obj := range r.Objects {
			part := r
			part.Objects = []enforce.TCObject{obj}
			add("tc_evidence", part)
		}
	}
	if e.containment.InitialQuarantine != nil {
		add("initial_quarantine", e.containment.InitialQuarantine)
	}
	if e.containment.Quarantine != nil {
		add("quarantine", e.containment.Quarantine)
	}
	if e.containment.ApplyQuarantine != nil {
		add("apply_quarantine", e.containment.ApplyQuarantine)
	}
	if e.containment.Error != "" {
		add("quarantine", e.containment.Error)
	}
}
