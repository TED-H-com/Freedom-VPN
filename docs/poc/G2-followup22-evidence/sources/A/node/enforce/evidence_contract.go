package enforce

import (
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Keep the original read separate from later rollback/quarantine annotations
// and from callers modifying the public diagnostic rows.
type tcReadbackSnapshot struct{ evidence api.TCReadbackEvidence }

// TCReadbackEvidence returns a detached D-106 snapshot of the actual read.
// Nil means no retained read (for example, Apply failed before reading tc).
// Conversion performs no I/O and never refreshes read_at. Verified describes
// the complete serving verifier, not merely a clean tc scan or Apply's eventual
// success. Callers must still honor Apply/VerifyServing's returned error.
// Attach the value to ObservationReport.tc_evidence; keep historical rejection
// errors and applied-generation acknowledgements separate.
func (ev ServingEvidence) TCReadbackEvidence() *api.TCReadbackEvidence {
	if ev.tcReadback == nil {
		return nil
	}
	s := ev.tcReadback.evidence
	// All mutable slices, pointers and nullable values belong to the returned
	// value. Publishing or editing it cannot change the retained read.
	s.Attachments = append([]api.TCReadAttachment{}, s.Attachments...)
	for i := range s.Attachments {
		r := &s.Attachments[i]
		r.Objects = cloneTCObjects(r.Objects)
		if r.Error != nil {
			v := *r.Error
			r.Error = &v
		}
	}
	s.ErrorCode.SetNull()
	if v, err := ev.tcReadback.evidence.ErrorCode.Get(); err == nil {
		s.ErrorCode.Set(v)
	}
	s.Rejected.SetNull()
	if r, err := ev.tcReadback.evidence.Rejected.Get(); err == nil {
		r.Objects = cloneTCObjects(r.Objects)
		if r.Error != nil {
			v := *r.Error
			r.Error = &v
		}
		s.Rejected.Set(r)
	}
	return &s
}

func cloneTCObjects(objects []api.TCReadObject) []api.TCReadObject {
	out := append([]api.TCReadObject{}, objects...)
	for i := range out {
		o := &out[i]
		if o.Handle != nil {
			v := *o.Handle
			o.Handle = &v
		}
		if o.Parent != nil {
			v := *o.Parent
			o.Parent = &v
		}
		if o.Root != nil {
			v := *o.Root
			o.Root = &v
		}
		if o.Actions != nil {
			v := append([]api.TCToken{}, (*o.Actions)...)
			o.Actions = &v
		}
	}
	return out
}

func contractTCRead(r TCReadEvidence) api.TCReadAttachment {
	a := api.TCReadAttachment{Interface: r.Interface, Attachment: r.Attachment, Objects: []api.TCReadObject{}}
	if r.Error != "" {
		v := api.TCReadAttachmentError(r.Error)
		a.Error = &v
	}
	for _, o := range r.Objects {
		x := api.TCReadObject{Kind: o.Kind}
		if o.Handle != "" {
			x.Handle = &o.Handle
		}
		if o.Parent != "" {
			x.Parent = &o.Parent
		}
		if o.Root {
			x.Root = &o.Root
		}
		if len(o.Actions) > 0 {
			v := append([]api.TCToken{}, o.Actions...)
			x.Actions = &v
		}
		a.Objects = append(a.Objects, x)
	}
	return a
}

func newTCReadbackSnapshot(ev ServingEvidence, readAt time.Time, err error) *tcReadbackSnapshot {
	s := ev.TCSummary
	v := api.TCReadbackEvidence{
		CandidateGeneration: s.Generation, ReadAt: readAt,
		Phase: api.TCReadbackEvidencePhase(s.Phase), Verified: s.Phase == "verify" && err == nil,
		ReadsTotal: s.Reads, ObjectsTotal: s.Objects, Truncated: s.Truncated, Sha256: s.SHA256,
		Attachments: []api.TCReadAttachment{},
	}
	v.ErrorCode.SetNull()
	var check *CheckError
	if errors.As(err, &check) {
		if code := api.ErrorCode(check.Reason); code.Valid() {
			v.ErrorCode.Set(code)
		}
	}
	for _, r := range ev.TC {
		v.Attachments = append(v.Attachments, contractTCRead(r))
	}
	v.Rejected.SetNull()
	if s.Rejected != nil {
		r := contractTCRead(*s.Rejected)
		rejected := api.TCRejectedAttachment{Interface: r.Interface, Attachment: r.Attachment, Objects: r.Objects}
		if r.Error != nil {
			v := api.TCRejectedAttachmentError(*r.Error)
			rejected.Error = &v
		}
		v.Rejected.Set(rejected)
	}
	return &tcReadbackSnapshot{evidence: v}
}
