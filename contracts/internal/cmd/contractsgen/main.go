// Command contractsgen writes the generated contract files: JSON Schemas
// derived from contracts/openapi.yaml, test vectors and the signed or hashed
// examples. Run it from the repository root:
//
//	go run ./contracts/internal/cmd/contractsgen
//
// With -check it writes nothing and exits non-zero when a committed file
// differs from the generated output.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/TED-H-com/Freedom-VPN/contracts/internal/gen"
)

func main() {
	dir := flag.String("dir", "contracts", "contracts directory")
	check := flag.Bool("check", false, "only check that committed files are up to date")
	flag.Parse()
	if err := run(*dir, *check); err != nil {
		fmt.Fprintln(os.Stderr, "contractsgen:", err)
		os.Exit(1)
	}
}

func run(dir string, check bool) error {
	spec, err := os.ReadFile(filepath.Join(dir, "openapi.yaml"))
	if err != nil {
		return err
	}
	files, err := gen.Files(spec)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	stale := 0
	for _, n := range names {
		p := filepath.Join(dir, filepath.FromSlash(n))
		old, _ := os.ReadFile(p)
		if bytes.Equal(old, files[n]) {
			continue
		}
		if check {
			fmt.Fprintln(os.Stderr, "stale:", p)
			stale++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, files[n], 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", p)
	}
	if stale > 0 {
		return fmt.Errorf("%d generated files are stale; run make -f mk/contracts.mk contracts-generate", stale)
	}
	return nil
}
