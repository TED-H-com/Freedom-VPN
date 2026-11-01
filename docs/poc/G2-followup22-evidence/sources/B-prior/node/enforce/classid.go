package enforce

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

const (
	minorMin uint16 = 0x0010
	minorMax uint16 = 0xEFF0
	// Inclusive count of [minorMin, minorMax]. The old difference excluded
	// one minor, so a full table could not reach a free class (§11.4).
	minorSpan uint32 = uint32(minorMax-minorMin) + 1
)

// PreferredMinor is the stable first choice for a member. It is a pure
// function of the member id.
func PreferredMinor(id uuid.UUID) uint16 {
	var sum uint32
	for _, b := range id {
		sum = sum*33 + uint32(b)
	}
	return minorMin + uint16(sum%minorSpan)
}

// ClassID is the HTB class id and the skb->priority value for a leaf.
// Kernel sch_htb selects a level-0 class when skb->priority equals its
// classid, so no tc filter and no fwmark are required (§11.4).
func ClassID(minor uint16) uint32 {
	return (HTBMajor << 16) | uint32(minor)
}

// AssignClasses returns a minor per member. Existing assignments are kept.
// A new member takes PreferredMinor, then the next free minor. Two members
// never share a minor. The result is deterministic for the same inputs.
func AssignClasses(members []uuid.UUID, existing map[uuid.UUID]uint16) (map[uuid.UUID]uint16, error) {
	out := make(map[uuid.UUID]uint16, len(members))
	used := make(map[uint16]uuid.UUID, len(members))
	for id, minor := range existing {
		if minor < minorMin || minor > minorMax || minor == HTBDefaultMinor || minor == 0 {
			return nil, fmt.Errorf("class minor 0x%x for %s is outside the allocatable range", minor, id)
		}
		if other, ok := used[minor]; ok && other != id {
			return nil, fmt.Errorf("class minor 0x%x is assigned to both %s and %s", minor, other, id)
		}
		used[minor] = id
	}
	var fresh []uuid.UUID
	seenMember := map[uuid.UUID]bool{}
	for _, id := range members {
		if id == uuid.Nil {
			return nil, fmt.Errorf("member id is nil")
		}
		if seenMember[id] {
			return nil, fmt.Errorf("duplicate member %s", id)
		}
		seenMember[id] = true
		if minor, ok := existing[id]; ok {
			out[id] = minor
			continue
		}
		fresh = append(fresh, id)
	}
	sort.Slice(fresh, func(i, j int) bool {
		return fresh[i].String() < fresh[j].String()
	})
	for _, id := range fresh {
		minor, err := firstFree(PreferredMinor(id), used)
		if err != nil {
			return nil, fmt.Errorf("member %s: %w", id, err)
		}
		used[minor] = id
		out[id] = minor
	}
	return out, nil
}

func firstFree(start uint16, used map[uint16]uuid.UUID) (uint16, error) {
	minor := start
	for i := uint32(0); i < minorSpan; i++ {
		if _, taken := used[minor]; !taken {
			return minor, nil
		}
		minor++
		if minor > minorMax {
			minor = minorMin
		}
	}
	return 0, fmt.Errorf("no free HTB class minor")
}

func classHandle(minor uint16) string {
	return fmt.Sprintf("%x:%x", HTBMajor, minor)
}
