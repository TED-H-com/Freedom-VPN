package agent

import (
	"math"
	"net/netip"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

// Input is everything Decide needs. It does not read the clock or the disk.
type Input struct {
	Now            time.Time
	MonoNow        time.Duration
	Datapath       enforce.Datapath
	Desired        *api.DesiredState
	DesiredOK      bool
	DesiredExpired bool
	Lease          *api.ServingLease
	LeaseFresh     bool
	LeaseMonoSent  time.Duration
	AnchorOK       bool
	LatchBlocked   bool
	RecoverBlocked bool
	Backend        string
	ReleaseID      uuid.UUID
	Assignments    map[uuid.UUID]uint16
	TestMode       bool
	ExemptV4       []netip.Prefix
	ExtraDenied    []netip.Prefix
	HostV4         []netip.Prefix
	HostReserveBPS uint64
	Revoked        bool
	Enrolled       bool
	BaselineOK     bool
	StorageOK      bool
	BootstrapLost  bool
	OutboxFull     bool
	// MeterFailed is a meter-path failure. Only reason meter_path_invalid
	// sets it. Usage caps and persist failures use the fields below so
	// METER_PATH_INVALID keeps that meaning (§10.3, §10.8).
	MeterFailed bool
	// UsageOutbox is a full usage outbox (age, bytes, or archive budget).
	UsageOutbox bool
	// UsageDisk is the usage store disk floor.
	UsageDisk bool
	// UsagePersist is a usage sample that could not be stored.
	UsagePersist bool
}

// Decision is the forwarding choice for one step (§14.7, §16.3).
type Decision struct {
	Serve         bool
	Reasons       []api.ForwardingBlockReason
	Model         enforce.Model
	Checks        []api.NodeCheck
	Errors        []api.NodeError
	Safety        api.SafetyStatus
	Provisioning  api.ProvisioningState
	Publication   api.PublicationIntent
	PolicyExpires time.Duration
}

// Decide chooses member forwarding. IPv6 is always block: desired state has
// no mode field (§8.3). A null capacity cannot be shaped, so forwarding stays
// off. The kernel backend is required; userspace is METER_PATH_INVALID (§10.3).
func Decide(in Input) Decision {
	at := in.Now
	if at.IsZero() {
		at = time.Unix(0, 0).UTC()
	}
	var reasons []api.ForwardingBlockReason
	add := func(r api.ForwardingBlockReason) {
		for _, have := range reasons {
			if have == r {
				return
			}
		}
		reasons = append(reasons, r)
	}
	if !in.Enrolled || in.Revoked {
		add(api.ForwardingBlockReasonIdentityInvalid)
	}
	if in.BootstrapLost {
		add(api.ForwardingBlockReasonProvisioningIncomplete)
	}
	if !in.AnchorOK {
		add(api.ForwardingBlockReasonClockUntrusted)
	}
	if !in.StorageOK || in.UsagePersist {
		add(api.ForwardingBlockReasonMeterStorageFailed)
	}
	if in.OutboxFull || in.UsageOutbox {
		add(api.ForwardingBlockReasonOutboxFull)
	}
	if in.UsageDisk {
		add(api.ForwardingBlockReasonDiskLow)
	}
	if in.LatchBlocked || in.RecoverBlocked {
		add(api.ForwardingBlockReasonRecoveryRequired)
	}
	if in.Backend != "kernel" || in.MeterFailed {
		add(api.ForwardingBlockReasonMeterPathInvalid)
	}
	if !in.BaselineOK {
		add(api.ForwardingBlockReasonSafetyCheckFailed)
	}
	pub := api.PublicationIntentUnpublished
	var releaseOK, bindingOK, qosOK, leaseOK bool
	bindingOK = true
	if in.Desired == nil || !in.DesiredOK {
		if in.DesiredExpired {
			add(api.ForwardingBlockReasonPolicyExpired)
		} else {
			add(api.ForwardingBlockReasonPolicyMissing)
		}
	} else {
		pub = in.Desired.PublicationIntent
		if pub != api.PublicationIntentPublished {
			add(api.ForwardingBlockReasonNotPublished)
		}
		releaseOK = in.ReleaseID != uuid.Nil && containsID(in.Desired.ServingReleaseIds, in.ReleaseID)
		if !releaseOK {
			add(api.ForwardingBlockReasonReleaseNotSupported)
		}
	}
	if !in.LeaseFresh || in.Lease == nil || in.LeaseMonoSent <= 0 {
		add(api.ForwardingBlockReasonLeaseExpired)
	} else {
		leaseOK = true
	}

	d := Decision{Publication: pub, Reasons: reasons}
	model, expires, qerr := buildModel(in)
	if qerr != nil {
		qosOK = false
		bindingOK = false
		add(api.ForwardingBlockReasonSafetyCheckFailed)
		d.Errors = append(d.Errors, api.NodeError{
			Code: api.ErrorCodeVALIDATIONFAILED, Message: "safety policy could not be rendered",
			ObservedAt: at, Stage: "qos",
		})
	} else if leaseOK && in.DesiredOK {
		qosOK = true
		if len(in.Desired.DeviceBindings) > 0 && len(model.Bindings) == 0 {
			add(api.ForwardingBlockReasonLeaseExpired)
			leaseOK = false
		}
	}
	d.Reasons = reasons
	d.Model = model
	d.PolicyExpires = expires
	d.Serve = len(d.Reasons) == 0
	if !d.Serve {
		d.Model = enforce.Model{}
		d.PolicyExpires = 0
	}
	d.Checks = checks(in, at, releaseOK, bindingOK, qosOK, leaseOK && d.Serve)
	d.Safety = safetyOf(d)
	d.Provisioning = provisioningOf(in)
	// ISOLATION_BASELINE_MISSING is report-only (§8.2, revision 3). It is
	// carried on the observation and is not an HTTP response.
	if !in.BaselineOK {
		d.Errors = append(d.Errors, api.NodeError{
			Code: api.ErrorCodeISOLATIONBASELINEMISSING, Message: "ISOLATION_BASELINE_MISSING",
			ObservedAt: at, Stage: "isolation",
		})
	}
	if in.Backend != "kernel" || in.MeterFailed {
		d.Errors = append(d.Errors, api.NodeError{
			Code: api.ErrorCodeMETERPATHINVALID, Message: "METER_PATH_INVALID",
			ObservedAt: at, Stage: "meter",
		})
	}
	if in.UsagePersist {
		d.Errors = append(d.Errors, api.NodeError{
			Code: api.ErrorCodeSTOREUNAVAILABLE, Message: "persist_failed",
			ObservedAt: at, Stage: "usage",
		})
	}
	if in.UsageDisk {
		d.Errors = append(d.Errors, api.NodeError{
			Code: api.ErrorCodeDISKSPACELOW, Message: "disk_space",
			ObservedAt: at, Stage: "usage",
		})
	}
	return d
}

func checks(in Input, at time.Time, releaseOK, bindingOK, qosOK, leaseOK bool) []api.NodeCheck {
	return []api.NodeCheck{
		chk(api.NodeCheckNameDnsPolicy, in.DesiredOK, api.ErrorCodeAUTHORIZATIONEXPIRED, at),
		chk(api.NodeCheckNameInternetOnly, in.BaselineOK, api.ErrorCodeISOLATIONBASELINEMISSING, at),
		chk(api.NodeCheckNameLease, leaseOK, api.ErrorCodeLEASEEXPIRED, at),
		chk(api.NodeCheckNameMeterPath, in.Backend == "kernel" && !in.MeterFailed, api.ErrorCodeMETERPATHINVALID, at),
		chk(api.NodeCheckNamePolicySignature, in.DesiredOK, api.ErrorCodeSIGNATUREINVALID, at),
		chk(api.NodeCheckNameQos, qosOK && in.DesiredOK, api.ErrorCodeVALIDATIONFAILED, at),
		chk(api.NodeCheckNameReleaseSupported, releaseOK, api.ErrorCodeMINIMUMVERSIONVIOLATION, at),
		chk(api.NodeCheckNameSourceBinding, bindingOK, api.ErrorCodeVALIDATIONFAILED, at),
		chk(api.NodeCheckNameStorage, in.StorageOK && !in.UsagePersist && !in.UsageDisk, api.ErrorCodeSTOREUNAVAILABLE, at),
		{
			Name: api.NodeCheckNameTunnelProbe, Status: api.HealthCheckStatusNotRun,
			ObservedAt: at, ErrorCode: presence.NewNullNullable[api.ErrorCode](),
		},
	}
}

func chk(name api.NodeCheckName, ok bool, code api.ErrorCode, at time.Time) api.NodeCheck {
	if ok {
		return api.NodeCheck{
			Name: name, Status: api.HealthCheckStatusPassed, ObservedAt: at,
			ErrorCode: presence.NewNullNullable[api.ErrorCode](),
		}
	}
	return api.NodeCheck{
		Name: name, Status: api.HealthCheckStatusFailed, ObservedAt: at,
		ErrorCode: presence.NewNullableWithValue(code),
	}
}

func safetyOf(d Decision) api.SafetyStatus {
	if d.Serve {
		return api.SafetyStatusPassed
	}
	for _, r := range d.Reasons {
		if r == api.ForwardingBlockReasonPolicyExpired || r == api.ForwardingBlockReasonLeaseExpired {
			return api.SafetyStatusExpired
		}
	}
	for _, r := range d.Reasons {
		if r == api.ForwardingBlockReasonSafetyCheckFailed || r == api.ForwardingBlockReasonMeterPathInvalid {
			return api.SafetyStatusFailed
		}
	}
	return api.SafetyStatusUnknown
}

func provisioningOf(in Input) api.ProvisioningState {
	if in.Revoked || in.BootstrapLost {
		return api.ProvisioningStateFailed
	}
	if !in.Enrolled {
		return api.ProvisioningStatePending
	}
	return api.ProvisioningStateReady
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func buildModel(in Input) (enforce.Model, time.Duration, error) {
	if in.Desired == nil || in.Lease == nil || !in.DesiredOK || !in.LeaseFresh {
		return enforce.Model{}, 0, nil
	}
	cap, err := in.Desired.NodeCapacity.Get()
	if err != nil || cap.UpBps <= 0 || cap.DownBps <= 0 {
		return enforce.Model{}, 0, errCapacity
	}
	host := in.HostReserveBPS
	if host == 0 {
		// The configuration rejects zero before startup. A zero that still
		// reaches Decide fails closed and is not replaced with a stand-in.
		return enforce.Model{}, 0, errCapacity
	}
	upParent := uint64(cap.UpBps)
	downParent := uint64(cap.DownBps)
	if host >= upParent || host >= downParent {
		return enforce.Model{}, 0, errCapacity
	}
	var upCap, downCap uint64
	if v, err := cap.MemberUpBpsCap.Get(); err == nil {
		if v <= 0 {
			return enforce.Model{}, 0, errCapacity
		}
		upCap = uint64(v)
	}
	if v, err := cap.MemberDownBpsCap.Get(); err == nil {
		if v <= 0 {
			return enforce.Model{}, 0, errCapacity
		}
		downCap = uint64(v)
	}

	elapsed := in.MonoNow - in.LeaseMonoSent
	if in.LeaseMonoSent <= 0 || elapsed < 0 {
		return enforce.Model{}, 0, errCapacity
	}
	now := in.Lease.IssuedAt.Add(elapsed)
	nodeWindow := in.Lease.ExpiresAt.Sub(in.Lease.IssuedAt)
	nodeDeadline, ok := timeanchor.MemberDeadline(in.LeaseMonoSent, nodeWindow)
	if !ok {
		return enforce.Model{}, 0, errCapacity
	}
	policyExpires := nodeDeadline

	byMember := map[uuid.UUID]api.MemberLease{}
	for _, m := range in.Lease.Members {
		byMember[m.MemberId] = m
	}
	var bindings []enforce.Binding
	var leases []enforce.MemberLease
	seenLease := map[uuid.UUID]bool{}
	seenAddr := map[netip.Addr]bool{}
	admitted := map[uuid.UUID]uint64{}
	for _, b := range in.Desired.DeviceBindings {
		ml, ok := byMember[b.MemberId]
		if !ok {
			continue
		}
		if !b.ValidFrom.IsZero() && now.Before(b.ValidFrom) {
			continue
		}
		if !b.ValidUntil.IsZero() && !now.Before(b.ValidUntil) {
			continue
		}
		window := timeanchor.Window(in.Lease.IssuedAt, ml.LeaseEnd, ml.EffectiveServiceEnd)
		if nw := timeanchor.Window(in.Lease.IssuedAt, in.Lease.ExpiresAt, ml.EffectiveServiceEnd); nw < window {
			window = nw
		}
		deadline, ok := timeanchor.MemberDeadline(in.LeaseMonoSent, window)
		if !ok {
			continue
		}
		rem := timeanchor.Remaining(deadline, in.MonoNow)
		sec := uint64(rem / time.Second)
		if sec < 1 {
			continue
		}
		if deadline < policyExpires {
			policyExpires = deadline
		}
		prefs, err := hostPrefixes(b.AllowedIpv4)
		if err != nil || b.PeerId == "" {
			return enforce.Model{}, 0, errCapacity
		}
		for _, p := range prefs {
			if seenAddr[p.Addr()] {
				return enforce.Model{}, 0, errCapacity
			}
			seenAddr[p.Addr()] = true
		}
		bindings = append(bindings, enforce.Binding{
			MemberID: b.MemberId, DeviceID: b.DeviceId, PeerID: b.PeerId,
			AllowedV4: prefs, BindingRevision: uint64(b.BindingRevision),
			ValidFrom: b.ValidFrom, ValidUntil: b.ValidUntil,
		})
		admitted[b.MemberId] = sec
		if !seenLease[b.MemberId] {
			seenLease[b.MemberId] = true
			end := in.Lease.IssuedAt.Add(window)
			leases = append(leases, enforce.MemberLease{
				MemberID: b.MemberId, EffectiveServiceEnd: end, NodeLeaseEnd: end,
				SignedServerTime: in.Lease.IssuedAt, MaxOfflineSeconds: uint64(in.Lease.MaxOfflineSeconds),
				TimeoutCeilSeconds: sec,
			})
		}
	}
	qos, err := buildQoS(in, upParent, downParent, upCap, downCap, host, admitted, now)
	if err != nil {
		return enforce.Model{}, 0, err
	}
	var denied []netip.Prefix
	if in.TestMode && len(in.ExemptV4) > 0 {
		denied = enforce.Without(enforce.SpecDeniedV4(), in.ExemptV4...)
	}
	resolvers := make([]netip.Addr, 0, len(in.Desired.Dns.ResolverIpv4))
	for _, s := range in.Desired.Dns.ResolverIpv4 {
		a, err := netip.ParseAddr(s)
		if err != nil || !a.Is4() {
			return enforce.Model{}, 0, errCapacity
		}
		resolvers = append(resolvers, a)
	}
	return enforce.Model{
		Generation:    uint64(in.Desired.ConfigGeneration),
		IPv6Mode:      enforce.IPv6Block,
		Now:           now,
		Datapath:      in.Datapath,
		Bindings:      bindings,
		Leases:        leases,
		Resolvers:     resolvers,
		DeniedV4:      denied,
		ExtraDeniedV4: in.ExtraDenied,
		HostV4:        in.HostV4,
		QoS:           qos,
	}, policyExpires, nil
}

var errCapacity = errString("agent: capacity")

type errString string

func (e errString) Error() string { return string(e) }

func hostPrefixes(in []string) ([]netip.Prefix, error) {
	if len(in) == 0 {
		return nil, errCapacity
	}
	out := make([]netip.Prefix, 0, len(in))
	for _, s := range in {
		p, err := netip.ParsePrefix(s)
		if err != nil || !p.IsValid() || p.Bits() != 32 {
			return nil, errCapacity
		}
		out = append(out, p)
	}
	return out, nil
}

func buildQoS(in Input, upParent, downParent, upCap, downCap, host uint64, admitted map[uuid.UUID]uint64, now time.Time) (enforce.QoS, error) {
	n := len(admitted)
	q := enforce.QoS{
		Up:          enforce.Rate{RateBPS: upParent, CeilBPS: upParent},
		Down:        enforce.Rate{RateBPS: downParent, CeilBPS: downParent},
		HostUp:      enforce.Rate{RateBPS: host, CeilBPS: upParent},
		HostDown:    enforce.Rate{RateBPS: host, CeilBPS: downParent},
		Assignments: in.Assignments,
	}
	if n == 0 {
		return q, nil
	}
	fairUp := (upParent - host) / uint64(n)
	fairDown := (downParent - host) / uint64(n)
	if fairUp == 0 || fairDown == 0 {
		return enforce.QoS{}, errCapacity
	}
	ids := make([]uuid.UUID, 0, n)
	for id := range admitted {
		ids = append(ids, id)
	}
	sortUUIDs(ids)
	for _, id := range ids {
		pol, found := pickPolicy(in.Desired.MemberRatePolicies, id, now)
		shadow := !found || pol.Mode == api.MemberRatePolicyModeShadow
		var upE, downE presence.Nullable[int64]
		if found {
			upE, downE = pol.UpBps, pol.DownBps
		} else {
			upE = presence.NewNullNullable[int64]()
			downE = presence.NewNullNullable[int64]()
		}
		upRate, upCeil, ok := classRate(upE, shadow, upParent, upCap, fairUp)
		if !ok {
			return enforce.QoS{}, errCapacity
		}
		downRate, downCeil, ok := classRate(downE, shadow, downParent, downCap, fairDown)
		if !ok {
			return enforce.QoS{}, errCapacity
		}
		q.Members = append(q.Members, enforce.MemberQoS{
			MemberID: id,
			Up:       enforce.Rate{RateBPS: upRate, CeilBPS: upCeil},
			Down:     enforce.Rate{RateBPS: downRate, CeilBPS: downCeil},
		})
	}
	return q, nil
}

func classRate(explicit presence.Nullable[int64], shadow bool, parent, cap, fair uint64) (rate, ceil uint64, ok bool) {
	ceil = parent
	if cap > 0 && cap < ceil {
		ceil = cap
	}
	if !shadow && explicit.IsSpecified() && !explicit.IsNull() {
		v, err := explicit.Get()
		if err != nil || v <= 0 {
			return 0, 0, false
		}
		if uint64(v) < ceil {
			ceil = uint64(v)
		}
	}
	rate = fair
	if rate > ceil {
		rate = ceil
	}
	if rate == 0 || ceil == 0 {
		return 0, 0, false
	}
	return rate, ceil, true
}

func pickPolicy(ps []api.MemberRatePolicy, id uuid.UUID, now time.Time) (api.MemberRatePolicy, bool) {
	var best api.MemberRatePolicy
	found := false
	bestCeil := int64(math.MaxInt64)
	for _, p := range ps {
		if p.MemberId != id || now.Before(p.ValidFrom) || !now.Before(p.ValidUntil) {
			continue
		}
		ceil := int64(math.MaxInt64)
		if v, err := p.UpBps.Get(); err == nil && v < ceil {
			ceil = v
		}
		if v, err := p.DownBps.Get(); err == nil && v < ceil {
			ceil = v
		}
		if !found || p.Priority > best.Priority || (p.Priority == best.Priority && ceil < bestCeil) {
			best, bestCeil, found = p, ceil, true
		}
	}
	return best, found
}

func sortUUIDs(ids []uuid.UUID) {
	for i := 1; i < len(ids); i++ {
		j := i
		for j > 0 && ids[j].String() < ids[j-1].String() {
			ids[j], ids[j-1] = ids[j-1], ids[j]
			j--
		}
	}
}
