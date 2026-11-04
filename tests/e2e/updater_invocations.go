package e2e

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
)

// UpdaterInvocation retains combined CLI output, including a missing final
// newline, with the invocation's own clock and process/exit receipt. A nil
// exit_code and empty ended_at mean that no completion receipt was observed.
type UpdaterInvocation struct {
	ID        string `json:"invocation_id"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	PID       int    `json:"pid,omitempty"`
	ExitCode  *int   `json:"exit_code"`
	Output    string `json:"output"`
}

func ReadUpdaterInvocations(src io.Reader) ([]UpdaterInvocation, error) {
	entries := map[string]map[string][]byte{}
	reader := tar.NewReader(src)
	for {
		h, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		dir, file := path.Split(path.Clean(h.Name))
		id := path.Base(dir)
		if path.Dir(path.Clean(dir)) != "updater-invocations" {
			return nil, fmt.Errorf("unexpected invocation path")
		}
		if entries[id] == nil {
			entries[id] = map[string][]byte{}
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		entries[id][file] = data
	}
	out := make([]UpdaterInvocation, 0, len(entries))
	for id, files := range entries {
		row := UpdaterInvocation{ID: id}
		if err := json.Unmarshal(files["start.json"], &row); err != nil {
			return nil, fmt.Errorf("invocation start: %w", err)
		}
		if result, ok := files["result.json"]; ok {
			if err := json.Unmarshal(result, &row); err != nil {
				return nil, fmt.Errorf("invocation result: %w", err)
			}
			if row.ID != id || row.ExitCode == nil || row.EndedAt == "" {
				return nil, fmt.Errorf("incomplete invocation result")
			}
		}
		row.Output = string(files["output.txt"])
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt < out[j].StartedAt })
	return out, nil
}
