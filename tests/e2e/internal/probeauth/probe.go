// Package probeauth implements a bounded TEST ONLY synthetic UDP echo packet.
// No payload or address is logged. Request/reply domains prevent reflection.
package probeauth

import (
	"crypto/hmac"
	"crypto/sha256"

	"github.com/google/uuid"
)

const Size = 1 + 16 + sha256.Size

func Seal(key []byte, id uuid.UUID, reply bool) []byte {
	raw := make([]byte, Size)
	raw[0] = 1
	if reply {
		raw[0] = 2
	}
	copy(raw[1:17], id[:])
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw[:17])
	copy(raw[17:], mac.Sum(nil))
	return raw
}
func Open(key, raw []byte, reply bool) (uuid.UUID, bool) {
	var id uuid.UUID
	kind := byte(1)
	if reply {
		kind = 2
	}
	if len(raw) != Size || raw[0] != kind {
		return id, false
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw[:17])
	if !hmac.Equal(raw[17:], mac.Sum(nil)) {
		return id, false
	}
	copy(id[:], raw[1:17])
	return id, id != uuid.Nil
}
