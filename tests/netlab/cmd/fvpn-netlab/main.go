// Command fvpn-netlab applies a JSON enforcement model inside the lab container.
// It is not a node agent. The agent calls node/enforce directly.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "agent-trace":
		err = agentTraceCmd(os.Args[2:])
	case "apply":
		err = applyCmd(os.Args[2:])
	case "baseline":
		err = baselineCmd(os.Args[2:])
	case "quarantine":
		err = quarantineCmd(os.Args[2:])
	case "verify":
		err = verifyCmd(os.Args[2:])
	case "counters":
		err = countersCmd(os.Args[2:])
	case "hold":
		err = holdCmd(os.Args[2:])
	case "render":
		err = renderCmd(os.Args[2:])
	case "meter-path":
		err = meterPathCmd(os.Args[2:])
	case "inject":
		err = injectCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%v\n", err)
	var check *enforce.CheckError
	if errors.As(err, &check) {
		os.Exit(3)
	}
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: fvpn-netlab agent-trace|apply|baseline|quarantine|verify|counters|hold|render|meter-path|inject\n")
}

func applyCmd(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	path := fs.String("f", "-", "model JSON path or - for stdin")
	prevPath := fs.String("prev", "", "optional previous counter JSON (§10.7)")
	tracePath := fs.String("trace", "", "TEST ONLY: append Apply/guard command brackets as JSONL")
	deadlineWindow := fs.Duration("deadline-window", 0, "TEST ONLY: map member deadline onto CLOCK_MONOTONIC immediately before Apply")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := loadModel(*path)
	if err != nil {
		return err
	}
	prev, err := loadPrev(*prevPath)
	if err != nil {
		return err
	}
	if *deadlineWindow > 0 {
		m.Now = time.Now().UTC()
		m.NowMono, err = enforce.MonotonicNow()
		if err != nil {
			return err
		}
		deadline := m.NowMono + *deadlineWindow
		for i := range m.Leases {
			m.Leases[i].DeadlineMono = &deadline
		}
	}
	var res enforce.ApplyResult
	if *tracePath != "" {
		f, openErr := os.OpenFile(*tracePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if openErr != nil {
			return openErr
		}
		defer f.Close()
		res, err = tracedApply(context.Background(), m, prev, f)
	} else {
		res, err = enforce.Apply(context.Background(), enforce.System{}, m, prev)
	}
	if printErr := writeResult(os.Stdout, m, res); printErr != nil && err == nil {
		return printErr
	}
	return err
}

func baselineCmd(args []string) error {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	path := fs.String("f", "", "optional model JSON; a nil probe removes the exception")
	ingress := fs.String("ingress", "fv-fh", "member ingress interface")
	uplink := fs.String("uplink", "fv-ih", "uplink interface")
	snat := fs.String("snat", "192.0.2.37", "snat address, required by the datapath check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path != "" {
		m, err := loadModel(*path)
		if err != nil {
			return err
		}
		return enforce.ApplyBaselineModel(context.Background(), enforce.System{}, m)
	}
	addr, err := netip.ParseAddr(*snat)
	if err != nil {
		return err
	}
	dp := enforce.Datapath{MemberIngress: *ingress, Uplink: *uplink, SNAT: addr}
	return enforce.ApplyBaseline(context.Background(), enforce.System{}, dp)
}

func quarantineCmd(args []string) error {
	fs := flag.NewFlagSet("quarantine", flag.ContinueOnError)
	path := fs.String("f", "-", "retained applied model JSON path or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := loadModel(*path)
	if err != nil {
		return err
	}
	evidence, err := enforce.Quarantine(context.Background(), enforce.System{}, m)
	if printErr := json.NewEncoder(os.Stdout).Encode(evidence); err == nil {
		return printErr
	}
	return err
}

func verifyCmd(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	path := fs.String("f", "-", "model JSON path or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := loadModel(*path)
	if err != nil {
		return err
	}
	ctx := context.Background()
	plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
	if err != nil {
		return err
	}
	evidence, err := enforce.VerifyServing(ctx, enforce.System{}, m, plan.Assignments)
	doc := struct {
		enforce.ServingEvidence
		TCEvidence *api.TCReadbackEvidence `json:"tc_evidence"`
	}{evidence, evidence.TCReadbackEvidence()}
	if printErr := json.NewEncoder(os.Stdout).Encode(doc); err == nil {
		return printErr
	}
	return err
}

func countersCmd(args []string) error {
	fs := flag.NewFlagSet("counters", flag.ContinueOnError)
	path := fs.String("f", "", "optional model JSON used to join samples")
	if err := fs.Parse(args); err != nil {
		return err
	}
	raw, err := enforce.System{}.Output(context.Background(), "nft", "-j", "list", "table", enforce.TableFamily, enforce.TableName)
	if err != nil {
		return err
	}
	rs, err := enforce.ParseRuleset(raw)
	if err != nil {
		return err
	}
	out := map[string]any{"counters": rs.Counters}
	if *path != "" {
		m, err := loadModel(*path)
		if err != nil {
			return err
		}
		samples, err := enforce.Samples(m, rs.Counters)
		if err != nil {
			return err
		}
		out["samples"] = samples
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func holdCmd(args []string) error {
	fs := flag.NewFlagSet("hold", flag.ContinueOnError)
	path := fs.String("f", "-", "model JSON path or - for stdin")
	ready := fs.String("ready", "/scratch/hold.ready", "file created after a successful apply")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := loadModel(*path)
	if err != nil {
		return err
	}
	res, err := enforce.Apply(context.Background(), enforce.System{}, m, nil)
	if err != nil {
		_ = writeResult(os.Stdout, m, res)
		return err
	}
	if err := writeResult(os.Stdout, m, res); err != nil {
		return err
	}
	if err := os.WriteFile(*ready, []byte("ok\n"), 0o644); err != nil {
		return err
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	// Leave the rules in place. A dead agent must not open an allow window.
	return nil
}

func renderCmd(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	path := fs.String("f", "-", "model JSON path or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := loadModel(*path)
	if err != nil {
		return err
	}
	plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, plan.NFT)
	fmt.Fprint(os.Stdout, plan.TC)
	return nil
}

func meterPathCmd(args []string) error {
	fs := flag.NewFlagSet("meter-path", flag.ContinueOnError)
	path := fs.String("f", "-", "model JSON path or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	m, err := loadModel(*path)
	if err != nil {
		return err
	}
	return enforce.EvaluateMeterPath(m.Bindings, fs.Args())
}

func loadPrev(path string) (map[string]enforce.CounterValue, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc map[string]json.RawMessage
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	payload := raw
	if wrapped, ok := doc["counters"]; ok {
		payload = wrapped
	}
	var counters map[string]enforce.CounterValue
	if err := json.Unmarshal(payload, &counters); err != nil {
		return nil, err
	}
	return counters, nil
}

func loadModel(path string) (enforce.Model, error) {
	var r io.Reader
	if path == "" || path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return enforce.Model{}, err
		}
		defer f.Close()
		r = f
	}
	dec := json.NewDecoder(r)
	dec.UseNumber()
	var m enforce.Model
	if err := dec.Decode(&m); err != nil {
		return enforce.Model{}, err
	}
	if m.Now.IsZero() {
		m.Now = time.Now().UTC()
	}
	return m, nil
}

func writeResult(w io.Writer, m enforce.Model, res enforce.ApplyResult) error {
	assign := map[string]string{}
	for id, minor := range res.Plan.Assignments {
		assign[id.String()] = fmt.Sprintf("%x", minor)
	}
	doc := struct {
		Leases      []enforce.MemberLease       `json:"leases"`
		Deadline    enforce.DeadlineEvidence    `json:"deadline"`
		Verified    bool                        `json:"verified"`
		Evidence    enforce.ServingEvidence     `json:"evidence"`
		TCEvidence  *api.TCReadbackEvidence     `json:"tc_evidence"`
		Quarantine  *enforce.QuarantineEvidence `json:"quarantine,omitempty"`
		Generation  uint64                      `json:"generation"`
		Admitted    int                         `json:"admitted"`
		Resets      []enforce.Reset             `json:"resets"`
		Finals      []enforce.Sample            `json:"finals"`
		Samples     []enforce.Sample            `json:"samples"`
		Assignments map[string]string           `json:"assignments"`
	}{
		Leases:      m.Leases,
		Deadline:    res.Deadline,
		Verified:    res.Verified,
		Evidence:    res.Evidence,
		TCEvidence:  res.Evidence.TCReadbackEvidence(),
		Quarantine:  res.Quarantine,
		Generation:  m.Generation,
		Admitted:    res.Plan.Admitted,
		Resets:      res.Resets,
		Finals:      res.Plan.Finals,
		Samples:     res.Samples,
		Assignments: assign,
	}
	if doc.Resets == nil {
		doc.Resets = []enforce.Reset{}
	}
	if doc.Finals == nil {
		doc.Finals = []enforce.Sample{}
	}
	if doc.Samples == nil {
		doc.Samples = []enforce.Sample{}
	}
	enc := json.NewEncoder(w)
	return enc.Encode(doc)
}
