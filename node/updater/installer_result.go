package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// prepareInstallerResult implements D-114 only for unaccepted installer results.
// All evidence is retained; only the job/result identities and covering hash
// change. Unknown members may carry a seal we cannot recreate, so those bytes
// get the ordinary single submission and terminal 409 handling instead.
func (e *Engine) prepareInstallerResult(ctx context.Context, ent updatejournal.OutboxEntry) (updatejournal.OutboxEntry, error) {
	var result api.UpdateResult
	if err := json.Unmarshal(ent.Payload, &result); err != nil {
		return ent, err
	}
	if result.Source == nil || *result.Source != api.UpdateResultSourceInstaller {
		return ent, nil
	}
	if result.RequestedReleaseId == uuid.Nil || result.IncarnationId == uuid.Nil || !validSHA256(result.ManifestSha256) {
		return ent, permanent(apierr.ValidationFailed, "installer receipt identity")
	}
	want := installerjob.ID(result.RequestedReleaseId, result.ManifestSha256, result.IncarnationId)
	if result.JobId == want {
		return ent, nil // New seals replay byte-for-byte.
	}
	hash, err := canonjson.HashHex(ent.Payload, "payload_hash")
	if err != nil {
		return ent, err
	}
	if hash != ent.PayloadHash || hash != result.PayloadHash || ent.JobID != result.JobId.String() || ent.ResultID != result.ResultId.String() {
		return ent, permanent(apierr.AckConflict, "installer receipt integrity")
	}
	if !locallyRecreatableResult(ent.Payload) {
		return ent, nil
	}
	// Work on raw members to preserve null/absent distinctions and all evidence.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(ent.Payload, &fields); err != nil {
		return ent, err
	}
	id := resultID(want.String(), result.Attempt, result.State)
	fields["job_id"], _ = json.Marshal(want)
	fields["result_id"], _ = json.Marshal(id)
	raw, err := json.Marshal(fields)
	if err != nil {
		return ent, err
	}
	hash, err = canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		return ent, err
	}
	fields["payload_hash"], _ = json.Marshal(hash)
	raw, err = json.Marshal(fields)
	if err != nil {
		return ent, err
	}
	payload, err := canonjson.Canonicalize(raw)
	if err != nil {
		return ent, err
	}
	next := updatejournal.OutboxEntry{ResultID: id.String(), JobID: want.String(), Attempt: ent.Attempt,
		Generation: ent.Generation, State: ent.State, Payload: payload, PayloadHash: hash}
	next, err = e.Journal.ReplaceInstallerResult(ctx, ent, next)
	if errors.Is(err, updatejournal.ErrResultConflict) {
		// An earlier receipt already owns D-110's bounded identity. Keep it
		// unchanged and let central decide this legacy seal's terminal outcome.
		return ent, nil
	}
	return next, err
}

func locallyRecreatableResult(payload []byte) bool {
	var result api.UpdateResult
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	return decoder.Decode(&result) == nil
}
