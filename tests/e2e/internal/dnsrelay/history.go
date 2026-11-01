package dnsrelay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"reflect"
)

// Request joins a start to its later completion across client query boundaries.
// A nil Completion is outstanding at the snapshot, even after client timeout.
type Request struct {
	Start      Event  `json:"start"`
	Completion *Event `json:"completion"`
}

// History reads the entire append-only stream, including other closed cases
// when checking sequence continuity. Missing starts, partial writes, duplicate
// completions make the snapshot incomplete, not a cache hit. Each process stream
// starts at sequence one; retained stream IDs expose deliberate relay restarts.
func History(raw []byte) ([]Request, bool) {
	requests := []Request{}
	indexes := map[string]int{}
	sequences := map[string]uint64{}
	complete := len(raw) == 0 || raw[len(raw)-1] == '\n'
	scan := bufio.NewScanner(bytes.NewReader(raw))
	for scan.Scan() {
		var e Event
		if json.Unmarshal(scan.Bytes(), &e) != nil || e.RequestID == "" || e.StreamID == "" || e.Started.IsZero() {
			complete = false
			continue
		}
		if e.Sequence != sequences[e.StreamID]+1 {
			complete = false
		}
		sequences[e.StreamID] = e.Sequence
		switch e.Phase {
		case "started":
			if _, exists := indexes[e.RequestID]; exists || !e.Completed.IsZero() {
				complete = false
				continue
			}
			indexes[e.RequestID] = len(requests)
			requests = append(requests, Request{Start: e})
		case "completed":
			i, exists := indexes[e.RequestID]
			if !exists || requests[i].Completion != nil || e.Completed.Before(e.Started) {
				complete = false
				continue
			}
			start := requests[i].Start
			identity := e
			identity.ReplySHA256, identity.ReplyBytes = start.ReplySHA256, start.ReplyBytes
			identity.Phase, identity.Sequence, identity.Completed = start.Phase, start.Sequence, start.Completed
			identity.RCode, identity.TTLs, identity.Classes = start.RCode, start.TTLs, start.Classes
			identity.AuthoritySerial, identity.Success = start.AuthoritySerial, start.Success
			if !reflect.DeepEqual(identity, start) {
				complete = false
				continue
			}
			requests[i].Completion = &e
		default:
			complete = false
		}
	}
	return requests, complete && scan.Err() == nil
}
