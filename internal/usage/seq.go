package usage

import (
	"crypto/sha256"
	"encoding/binary"
	"slices"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// SeqRange is an inclusive range of event sequences [Start, End] (contract
// SeqRange and ReplayUnavailableRange are inclusive on both ends).
type SeqRange struct {
	Start, End uint64
}

// AdvanceContiguous returns the highest contiguous sequence reachable from
// hcs given the committed sequences above it (§10.6): hcs+1, hcs+2, ... as
// long as each is present. Sequences at or below hcs are ignored. The
// result never depends on the largest sequence seen, only on presence.
func AdvanceContiguous(hcs uint64, committed []uint64) uint64 {
	s := sortedAbove(hcs, committed)
	for _, v := range s {
		if v != hcs+1 {
			break
		}
		hcs = v
	}
	return hcs
}

// MissingRanges lists every sequence range absent between hcs and the
// largest committed sequence (§17.3: all gaps, never a sample). committed
// may be unsorted and contain sequences at or below hcs.
func MissingRanges(hcs uint64, committed []uint64) []SeqRange {
	s := sortedAbove(hcs, committed)
	var out []SeqRange
	prev := hcs
	for _, v := range s {
		if v > prev+1 {
			out = append(out, SeqRange{Start: prev + 1, End: v - 1})
		}
		prev = v
	}
	return out
}

func sortedAbove(hcs uint64, committed []uint64) []uint64 {
	s := make([]uint64, 0, len(committed))
	for _, v := range committed {
		if v > hcs {
			s = append(s, v)
		}
	}
	slices.Sort(s)
	return slices.Compact(s)
}

// ExtendPrefixProof folds the payload hashes of newly contiguous events into
// the stream's prefix digest: proof' = SHA-256(proof || seq_be64 || hash).
// The zero-length proof is the empty prefix. It lets an operator verify that
// the receipts 1..hcs were not replaced (§15.4 usage_checkpoints prefix
// proof).
func ExtendPrefixProof(proof []byte, seq uint64, payloadHash []byte) []byte {
	h := sha256.New()
	h.Write(proof)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], seq)
	h.Write(b[:])
	h.Write(payloadHash)
	return h.Sum(nil)
}

func toAPIRanges(rs []SeqRange) []api.SeqRange {
	out := make([]api.SeqRange, 0, len(rs))
	for _, r := range rs {
		out = append(out, api.SeqRange{StartSeq: decstr.U64(r.Start), EndSeq: decstr.U64(r.End)})
	}
	return out
}
