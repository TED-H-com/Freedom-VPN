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
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "apply":
		err = applyCmd(os.Args[2:])
	case "baseline":
		err = baselineCmd(os.Args[2:])
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
	fmt.Fprintf(os.Stderr, "usage: fvpn-netlab apply|baseline|verify|counters|hold|render|meter-path|inject\n")
}

func applyCmd(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	path := fs.String("f", "-", "model JSON path or - for stdin")
	prevPath := fs.String("prev", "", "optional previous counter JSON (§10.7)")
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
	res, err := enforce.Apply(context.Background(), enforce.System{}, m, prev)
	if printErr := writeResult(os.Stdout, m.Generation, res); printErr != nil && err == nil {
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
	raw, err := enforce.System{}.Output(ctx, "nft", "-j", "list", "table", enforce.TableFamily, enforce.TableName)
	if err != nil {
		return err
	}
	plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
	if err != nil {
		return err
	}
	if err := enforce.VerifyServing(raw, m, plan.Assignments); err != nil {
		return err
	}
	qos, err := readQoS(ctx, m)
	if err != nil {
		return err
	}
	return enforce.VerifyQoS(qos, m, plan.Assignments)
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
		_ = writeResult(os.Stdout, m.Generation, res)
		return err
	}
	if err := writeResult(os.Stdout, m.Generation, res); err != nil {
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

func writeResult(w io.Writer, generation uint64, res enforce.ApplyResult) error {
	assign := map[string]string{}
	for id, minor := range res.Plan.Assignments {
		assign[id.String()] = fmt.Sprintf("%x", minor)
	}
	doc := struct {
		Generation  uint64            `json:"generation"`
		Admitted    int               `json:"admitted"`
		Resets      []enforce.Reset   `json:"resets"`
		Finals      []enforce.Sample  `json:"finals"`
		Samples     []enforce.Sample  `json:"samples"`
		Assignments map[string]string `json:"assignments"`
	}{
		Generation:  generation,
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

func readQoS(ctx context.Context, m enforce.Model) (map[string][2][]byte, error) {
	out := map[string][2][]byte{}
	for _, ifc := range []string{m.Datapath.Uplink, m.Datapath.MemberIngress} {
		q, err := enforce.System{}.Output(ctx, "tc", "-j", "qdisc", "show", "dev", ifc)
		if err != nil {
			return nil, err
		}
		c, err := enforce.System{}.Output(ctx, "tc", "-j", "class", "show", "dev", ifc)
		if err != nil {
			return nil, err
		}
		out[ifc] = [2][]byte{q, c}
	}
	return out, nil
}
