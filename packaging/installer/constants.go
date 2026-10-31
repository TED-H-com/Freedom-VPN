package installer

// NetBirdPin is the client version docs/netbird/VERSIONS.md locks.
// The installer installs this exact package and refuses any other.
const NetBirdPin = "0.82.0"

// Unit names installed from the release archive. The watchdog has no
// PartOf, BindsTo or Requires on the agent (§17.1, "Agent 卡住或退出").
const (
	UnitWatchdog  = "fvpn-watchdog.service"
	UnitAgent     = "fvpn-agent.service"
	UnitUpdater   = "fvpn-updater.service"
	UnitTimer     = "fvpn-updater.timer"
	UnitRecover   = "fvpn-updater-recover.service"
	UnitNetbird   = "fvpn-netbird.service"
	UnitNetbirdUp = "fvpn-node-netbird-up.service"
	UnitFirstboot = "fvpn-node-firstboot.service"
)

// requiredUnits are copied out of the release tree and enabled.
var requiredUnits = []string{
	UnitWatchdog, UnitAgent, UnitUpdater, UnitTimer,
	UnitRecover, UnitNetbird, UnitNetbirdUp, UnitFirstboot,
}

// startOrder is watchdog, agent, NetBird, then the updater (§6.3).
// fvpn-updater-recover runs first so an open update journal is reconciled
// before member services (§13.5). netbird-up waits for the setup key file
// the agent writes; the key is never placed on a command line (§18.2).
var startOrder = []string{
	UnitRecover,
	UnitWatchdog,
	UnitAgent,
	UnitNetbird,
	UnitNetbirdUp,
	UnitTimer,
}

// stopOrder is the reverse, plus the timer's oneshot and first-boot.
var stopOrder = []string{
	UnitTimer,
	UnitUpdater,
	UnitNetbirdUp,
	UnitAgent,
	UnitNetbird,
	UnitWatchdog,
	UnitRecover,
	UnitFirstboot,
}

// expectActive is what check treats as healthy. The updater oneshot is
// inactive between timer fires; that is not drift. first-boot is inactive
// once the node is enrolled.
var expectActive = []string{
	UnitWatchdog,
	UnitAgent,
	UnitNetbird,
	UnitRecover,
	UnitTimer,
	UnitNetbirdUp,
}

const (
	statusInstalled       = "installed"
	statusAlreadyCurrent  = "already_current"
	statusRepaired        = "repaired"
	statusStaged          = "staged"
	statusRemoved         = "removed"
	statusReenrolled      = "reenrolled"
	statusPrepared        = "prepared"
	statusEnrolled        = "enrolled"
	statusAlreadyEnrolled = "already_enrolled"
	statusHealthy         = "healthy"
	statusDrifted         = "drifted"
)
