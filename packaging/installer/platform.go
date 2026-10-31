package installer

import (
	"os"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

// Supported baseline (§3.2). Anything else is UNSUPPORTED_PLATFORM.
const (
	SupportedOSID      = "ubuntu"
	SupportedOSVersion = "24.04"
)

// Platform is the host the installer identified.
type Platform struct {
	OSID      string
	OSVersion string
	Arch      string
	Kernel    string
}

// Identify reads os-release and the kernel. Arch is runtime.GOARCH unless
// Env.Arch is set (tests).
func Identify(env *Env) (Platform, error) {
	if env == nil {
		env = &Env{}
	}
	// layout() falls back to DefaultLayout. The CLI does not fill Layout.
	raw, err := os.ReadFile(env.layout().OSRelease)
	if err != nil {
		return Platform{}, errf(ExitPlatform, "UNSUPPORTED_PLATFORM", "cannot read os-release")
	}
	id, version := parseOSRelease(string(raw))
	arch := env.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	kernel := env.Kernel
	if kernel == "" {
		kernel = readKernel()
	}
	p := Platform{OSID: id, OSVersion: version, Arch: arch, Kernel: kernel}
	if id != SupportedOSID || version != SupportedOSVersion || (arch != "amd64" && arch != "arm64") {
		return p, errf(ExitPlatform, "UNSUPPORTED_PLATFORM", "need Ubuntu %s amd64 or arm64, have %s %s %s", SupportedOSVersion, id, version, arch)
	}
	if kernel != "" && !strings.HasPrefix(kernel, "Linux") {
		return p, errf(ExitPlatform, "UNSUPPORTED_PLATFORM", "kernel is not Linux")
	}
	return p, nil
}

func parseOSRelease(text string) (id, version string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, `"'`)
		switch key {
		case "ID":
			id = val
		case "VERSION_ID":
			version = val
		}
	}
	return id, version
}

func readKernel() string {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return ""
	}
	sys := cstr(u.Sysname[:])
	rel := cstr(u.Release[:])
	if sys == "" {
		return ""
	}
	return sys + " " + rel
}

func cstr(b []byte) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n])
}
