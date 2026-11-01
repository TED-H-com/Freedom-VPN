package enforce

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// TCDev is what we already installed on one interface. Render uses it so a
// rate change updates classes in place instead of replacing the root qdisc
// (§11.4).
type TCDev struct {
	RootHTB bool
	// Leaves records fq_codel qdiscs keyed by the parent class minor.
	Leaves map[uint16]bool
	// Classes records every child HTB class minor except the parent 1:1 and
	// the host default 1:fffe. A minor stays occupied until render deletes it.
	Classes map[uint16]bool
}

// TCState is keyed by interface name.
type TCState struct {
	Devices map[string]TCDev
}

func renderTC(m Model, assignments map[uuid.UUID]uint16, state TCState) (string, error) {
	type side struct {
		iface  string
		parent Rate
		host   Rate
		pick   func(MemberQoS) Rate
	}
	sides := []side{
		{m.Datapath.Uplink, m.QoS.Up, m.QoS.HostUp, func(q MemberQoS) Rate { return q.Up }},
		{m.Datapath.MemberIngress, m.QoS.Down, m.QoS.HostDown, func(q MemberQoS) Rate { return q.Down }},
	}
	members := append([]MemberQoS{}, m.QoS.Members...)
	sort.Slice(members, func(i, j int) bool {
		return members[i].MemberID.String() < members[j].MemberID.String()
	})
	var b strings.Builder
	b.WriteString("# HTB parent per direction, one leaf class per member, fq_codel under each (§11.4).\n")
	b.WriteString("# Classification is skb->priority set to the leaf classid. No fwmark, no filter.\n")
	for _, s := range sides {
		dev := state.Devices[s.iface]
		assigned := map[uint16]bool{}
		for _, mem := range members {
			minor, ok := assignments[mem.MemberID]
			if !ok {
				return "", fmt.Errorf("no class assignment for %s", mem.MemberID)
			}
			assigned[minor] = true
		}
		if !dev.RootHTB {
			fmt.Fprintf(&b, "qdisc add dev %s root handle %x: htb default %x\n", s.iface, HTBMajor, HTBDefaultMinor)
		} else {
			// Drop leaves and classes the new generation no longer uses before
			// replacing the classes that stay. A removed member must not keep
			// a guaranteed rate on the parent (§11.4).
			for _, minor := range orphanMinors(dev, assigned) {
				if dev.Leaves[minor] {
					fmt.Fprintf(&b, "qdisc del dev %s parent %s\n", s.iface, classHandle(minor))
				}
				fmt.Fprintf(&b, "class del dev %s classid %s\n", s.iface, classHandle(minor))
			}
		}
		writeClass(&b, s.iface, fmt.Sprintf("%x:", HTBMajor), classHandle(1), s.parent, 0)
		writeClass(&b, s.iface, classHandle(1), classHandle(HTBDefaultMinor), s.host, 0)
		for _, mem := range members {
			minor := assignments[mem.MemberID]
			writeClass(&b, s.iface, classHandle(1), classHandle(minor), s.pick(mem), 1)
			if !dev.Leaves[minor] {
				fmt.Fprintf(&b, "qdisc add dev %s parent %s handle %x: fq_codel limit 10240 flows 1024 quantum 1514 target 5ms interval 100ms\n",
					s.iface, classHandle(minor), minor)
			}
		}
	}
	return b.String(), nil
}

// reserveUnmappedClasses keeps a kernel class minor out of the free set when
// the nft class map no longer names its member. AssignClasses then cannot
// hand that minor to someone else while the class still exists.
func reserveUnmappedClasses(m Model, tc TCState) Model {
	occupied := map[uint16]struct{}{}
	for _, dev := range tc.Devices {
		for minor := range dev.Classes {
			if classMinorRecorded(minor) {
				occupied[minor] = struct{}{}
			}
		}
	}
	if len(occupied) == 0 {
		return m
	}
	owned := map[uint16]struct{}{}
	for _, minor := range m.QoS.Assignments {
		owned[minor] = struct{}{}
	}
	assigned := make(map[uuid.UUID]uint16, len(m.QoS.Assignments)+len(occupied))
	for id, minor := range m.QoS.Assignments {
		assigned[id] = minor
	}
	added := false
	for minor := range occupied {
		if _, ok := owned[minor]; ok {
			continue
		}
		assigned[kernelClassPlaceholder(minor)] = minor
		added = true
	}
	if !added {
		return m
	}
	m.QoS.Assignments = assigned
	return m
}

func classMinorRecorded(minor uint16) bool {
	return minor != 0 && minor != 1 && minor != HTBDefaultMinor
}

func kernelClassPlaceholder(minor uint16) uuid.UUID {
	var id uuid.UUID
	id[0], id[1], id[2], id[3] = 0xff, 0xff, 0xff, 0xff
	id[4], id[5] = 0xff, 0xff
	id[6], id[7] = 0x4f, 0xff
	id[8], id[9] = 0x8f, 0xff
	id[14] = byte(minor >> 8)
	id[15] = byte(minor)
	return id
}

func orphanMinors(dev TCDev, assigned map[uint16]bool) []uint16 {
	seen := map[uint16]bool{}
	var out []uint16
	add := func(minor uint16) {
		if !classMinorRecorded(minor) || assigned[minor] || seen[minor] {
			return
		}
		seen[minor] = true
		out = append(out, minor)
	}
	for minor := range dev.Classes {
		add(minor)
	}
	for minor := range dev.Leaves {
		add(minor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func childClassMinors(view QdiscView) map[uint16]bool {
	out := map[uint16]bool{}
	for handle := range view.Classes {
		major, minor, ok := splitHandle(handle)
		if !ok || uint32(major) != HTBMajor || !classMinorRecorded(minor) {
			continue
		}
		out[minor] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeClass(b *strings.Builder, dev, parent, classid string, rate Rate, prio int) {
	burst := burstBytes(rate.CeilBPS)
	fmt.Fprintf(b, "class replace dev %s parent %s classid %s htb rate %s ceil %s burst %d cburst %d prio %d quantum 1514\n",
		dev, parent, classid, formatRate(rate.RateBPS), formatRate(rate.CeilBPS), burst, burst, prio)
}

func formatRate(bps uint64) string {
	switch {
	case bps%1_000_000 == 0:
		return fmt.Sprintf("%dmbit", bps/1_000_000)
	case bps%1_000 == 0:
		return fmt.Sprintf("%dkbit", bps/1_000)
	default:
		return fmt.Sprintf("%dbit", bps)
	}
}

// burstBytes is about 10 ms of the ceil, and at least one full packet.
// A one-second burst would dominate the short lab window (§11.7).
func burstBytes(bps uint64) uint64 {
	b := bps / 8 / 100
	if b < 1600 {
		b = 1600
	}
	if b > 256*1024 {
		b = 256 * 1024
	}
	return b
}

func retiredSamples(m Model, state State, devices []admittedDevice) []Sample {
	inModel := map[uuid.UUID]bool{}
	for _, d := range devices {
		inModel[d.Binding.DeviceID] = true
	}
	for _, b := range m.Bindings {
		inModel[b.DeviceID] = true
	}
	var out []Sample
	for name, val := range state.Counters {
		id, dir, ns, ok := parseDeviceCounter(name)
		if !ok || inModel[id] {
			continue
		}
		out = append(out, Sample{
			DeviceID:  id,
			Direction: dir,
			Packets:   val.Packets,
			Bytes:     val.Bytes,
			Namespace: ns,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DeviceID.String() != out[j].DeviceID.String() {
			return out[i].DeviceID.String() < out[j].DeviceID.String()
		}
		return out[i].Direction < out[j].Direction
	})
	return out
}
