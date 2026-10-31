// Command fvpn-node-install is the Ubuntu node installer (§6.3, PKG01).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
	"golang.org/x/sys/unix"
)

func main() {
	unix.Umask(0o077)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if err := installer.RejectTokenArgv(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return installer.ExitUsage
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: fvpn-node-install <install|check|upgrade|remove|reenroll|image-prepare|firstboot|verify-archive|netbird-up|version>")
		return installer.ExitUsage
	}
	switch args[0] {
	case "version", "-h", "--help", "help":
		fmt.Fprintln(stdout, "fvpn-node-install dev")
		return installer.ExitOK
	case "install":
		return cmdInstall(args[1:], stdout, stderr)
	case "check":
		return cmdCheck(args[1:], stdout, stderr)
	case "upgrade":
		return cmdUpgrade(args[1:], stdout, stderr)
	case "remove":
		return cmdRemove(args[1:], stdout, stderr)
	case "reenroll":
		return cmdReenroll(args[1:], stdout, stderr)
	case "image-prepare":
		return cmdImage(args[1:], stdout, stderr)
	case "firstboot":
		return cmdFirst(args[1:], stdout, stderr)
	case "verify-archive":
		return cmdVerify(args[1:], stdout, stderr)
	case "netbird-up":
		return cmdNetbird(args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "usage: unknown command %q\n", args[0])
		return installer.ExitUsage
	}
}

func envOf(stdout, stderr io.Writer) *installer.Env {
	return &installer.Env{Stdout: stdout, Stderr: stderr, AllowHost: true}
}

func printResult(stdout io.Writer, asJSON bool, r *installer.Result) {
	if r == nil {
		return
	}
	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		return
	}
	fmt.Fprint(stdout, r.Text())
}

func fail(stderr io.Writer, err error) int {
	ee := installer.AsError(err)
	fmt.Fprintln(stderr, ee.Error())
	if ee.Exit == 0 {
		return installer.ExitFailed
	}
	return ee.Exit
}

func cmdInstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifest := fs.String("manifest", "", "signed release manifest")
	artifact := fs.String("artifact", "", "release archive for this architecture")
	trust := fs.String("trust", "", "release trust root")
	desired := fs.String("desired-trust", "", "desired-state trust root (defaults to the release trust)")
	token := fs.String("token-file", "", "enrollment token file, mode 0600")
	control := fs.String("control-url", "", "https control URL")
	ca := fs.String("server-ca", "", "control server CA PEM")
	bundle := fs.String("trust-bundle", "", "control trust bundle")
	reserve := fs.Uint64("host-reserve-bps", 0, "host traffic reserve in bits per second (required, 待實測)")
	member := fs.String("datapath-member", "fvmb0", "member ingress interface")
	uplink := fs.String("datapath-uplink", "fvup0", "uplink interface")
	snat := fs.String("snat", "", "SNAT IPv4 address")
	nb := fs.String("netbird-management-url", "", "https NetBird management URL")
	fault := fs.String("fault-after", "", "TEST ONLY: directories")
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	env := envOf(stdout, stderr)
	res, err := installer.Install(context.Background(), env, installer.InstallRequest{
		ManifestPath: *manifest, ArtifactPath: *artifact, TrustPath: *trust,
		DesiredTrustPath: *desired, TokenFile: *token, ControlURL: *control,
		ServerCA: *ca, TrustBundle: *bundle, HostReserveBPS: *reserve,
		MemberIngress: *member, Uplink: *uplink, SNAT: *snat,
		NetbirdManagementURL: *nb, FaultAfter: *fault,
	})
	if res != nil {
		printResult(stdout, *asJSON, res)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	rep, err := installer.Check(context.Background(), envOf(stdout, stderr))
	if rep != nil {
		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep)
		} else {
			fmt.Fprint(stdout, rep.Text())
		}
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdUpgrade(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifest := fs.String("manifest", "", "signed release manifest")
	artifact := fs.String("artifact", "", "release archive")
	trust := fs.String("trust", "", "release trust root")
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	res, err := installer.Upgrade(context.Background(), envOf(stdout, stderr), *manifest, *artifact, *trust)
	if res != nil {
		printResult(stdout, *asJSON, res)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	purge := fs.Bool("purge", false, "also remove identity, the usage store and the update journal")
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	res, err := installer.Remove(context.Background(), envOf(stdout, stderr), *purge)
	if res != nil {
		printResult(stdout, *asJSON, res)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdReenroll(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reenroll", flag.ContinueOnError)
	fs.SetOutput(stderr)
	token := fs.String("token-file", "", "enrollment token file, mode 0600")
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	if *token == "" {
		fmt.Fprintln(stderr, "usage: reenroll requires --token-file")
		return installer.ExitUsage
	}
	res, err := installer.Reenroll(context.Background(), envOf(stdout, stderr), *token)
	if res != nil {
		printResult(stdout, *asJSON, res)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdImage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("image-prepare", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("i-am-building-an-image", false, "strip an enrolled identity while building a master image")
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	res, err := installer.ImagePrepare(context.Background(), envOf(stdout, stderr), *force)
	if res != nil {
		printResult(stdout, *asJSON, res)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdFirst(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("firstboot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	res, err := installer.FirstBoot(context.Background(), envOf(stdout, stderr))
	if res != nil {
		printResult(stdout, *asJSON, res)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}

func cmdVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify-archive", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifest := fs.String("manifest", "", "signed release manifest")
	artifact := fs.String("artifact", "", "release archive")
	trust := fs.String("trust", "", "release trust root")
	asJSON := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	v, err := installer.VerifyArchive(*manifest, *artifact, *trust, runtime.GOARCH, time.Now())
	if err != nil {
		return fail(stderr, err)
	}
	res := &installer.Result{Status: "verified", ReleaseID: v.Manifest.ReleaseId.String(), ManifestSHA256: v.SHA}
	printResult(stdout, *asJSON, res)
	return installer.ExitOK
}

func cmdNetbird(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("netbird-up", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return installer.ExitUsage
	}
	if err := installer.NetbirdUp(context.Background(), envOf(nil, stderr)); err != nil {
		return fail(stderr, err)
	}
	return installer.ExitOK
}
