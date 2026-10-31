package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
)

// Journal records each host change before it happens (§6.3). Phase open
// means a crash should undo records[Commit:]. Seal moves Commit forward so
// a later service-start failure does not delete an identity whose enrollment
// token was already consumed (§6.2). The journal directory itself is created
// before the first record, because the record has to land somewhere, and
// rollback never removes it.
type Journal struct {
	env  *Env
	path string
	file journalFile
}

type journalFile struct {
	Phase   string   `json:"phase"`
	Commit  int      `json:"commit"`
	Undone  bool     `json:"undone"`
	Records []Record `json:"records"`
}

// Record is one intended host change and the previous value.
type Record struct {
	Action   string `json:"action"`
	Path     string `json:"path"`
	Prev     string `json:"prev,omitempty"`
	PrevMode string `json:"prev_mode,omitempty"`
	Mode     string `json:"mode,omitempty"`
	Backup   string `json:"backup,omitempty"`
	Data     string `json:"data,omitempty"`
	Sticky   bool   `json:"sticky,omitempty"`
	Note     string `json:"note,omitempty"`
}

func (j *Journal) flush() error {
	raw, err := json.MarshalIndent(j.file, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return fsutil.WriteFileAtomic(j.path, raw, 0o600)
}

// prepareJournal rolls back an open attempt and rotates the file so the
// next attempt starts clean. A committed or already-rolled-back file is
// rotated and left as evidence.
func prepareJournal(env *Env) error {
	l := env.layout()
	path := l.journalFile()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f journalFile
	if json.Unmarshal(raw, &f) != nil || f.Phase != "open" || f.Undone {
		return nil
	}
	j := &Journal{env: env, path: path, file: f}
	if err := j.Rollback(context.Background()); err != nil {
		return err
	}
	return rotateJournal(env, path)
}

func rotateJournal(env *Env, path string) error {
	stamp := env.now().UTC().Format("20060102T150405.000000000Z")
	dest := filepath.Join(env.layout().InstallDir, "journal-"+stamp+".json")
	if err := os.Rename(path, dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func openJournal(env *Env) (*Journal, error) {
	l := env.layout()
	if err := os.MkdirAll(l.InstallDir, 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(l.journalFile()); err == nil {
		if err := rotateJournal(env, l.journalFile()); err != nil {
			return nil, err
		}
	}
	if err := os.Chmod(l.InstallDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(l.blobDir(), 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(l.blobDir(), 0o700); err != nil {
		return nil, err
	}
	j := &Journal{env: env, path: l.journalFile(), file: journalFile{Phase: "open"}}
	if err := j.flush(); err != nil {
		return nil, err
	}
	return j, nil
}

func (j *Journal) record(rec Record) error {
	j.file.Records = append(j.file.Records, rec)
	return j.flush()
}

// Seal keeps every record so far across a later rollback (§6.2).
func (j *Journal) Seal() error {
	j.file.Commit = len(j.file.Records)
	j.file.Phase = "open"
	return j.flush()
}

// Finish marks the attempt committed. Nothing is undone after this.
func (j *Journal) Finish() error {
	j.file.Commit = len(j.file.Records)
	j.file.Phase = "committed"
	j.file.Undone = false
	return j.flush()
}

// Rollback undoes records after Commit, once. Missing files are ignored.
// Paths outside the installer layout are never removed (§6.3).
func (j *Journal) Rollback(ctx context.Context) error {
	if j.file.Undone {
		return nil
	}
	var first error
	for i := len(j.file.Records) - 1; i >= j.file.Commit; i-- {
		if err := j.undo(ctx, j.file.Records[i]); err != nil && first == nil {
			first = err
		}
	}
	j.file.Undone = true
	j.file.Phase = "rolled_back"
	if err := j.flush(); err != nil && first == nil {
		first = err
	}
	return first
}

func (j *Journal) undo(ctx context.Context, rec Record) error {
	switch rec.Action {
	case "mkdir":
		if rec.Prev != "absent" || rec.Sticky {
			return nil
		}
		if !j.env.layout().owns(rec.Path) {
			return errf(ExitFailed, "failed", "rollback refused a path outside the installer layout")
		}
		if err := os.RemoveAll(rec.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "chmod":
		mode, err := parseMode(rec.PrevMode)
		if err != nil {
			return err
		}
		if err := os.Chmod(rec.Path, mode); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "write":
		if !j.env.layout().owns(rec.Path) {
			return errf(ExitFailed, "failed", "rollback refused a path outside the installer layout")
		}
		if rec.Prev == "absent" {
			if err := os.Remove(rec.Path); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		if rec.Backup == "" {
			return nil
		}
		return copyFile(rec.Backup, rec.Path, parseModeOr(rec.PrevMode, 0o600))
	case "symlink":
		if !j.env.layout().owns(rec.Path) {
			return errf(ExitFailed, "failed", "rollback refused a path outside the installer layout")
		}
		if err := os.Remove(rec.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		if rec.Prev != "" && rec.Prev != "absent" {
			return os.Symlink(rec.Prev, rec.Path)
		}
		return nil
	case "nft-table":
		// Legacy journals named the agent's table. D-116 leaves it intact.
		return nil
	case "nft-marker-table":
		if rec.Data != j.env.layout().markerTable() {
			return errf(ExitFailed, "failed", "rollback refused an unknown nft table")
		}
		return removeMarker(ctx, j.env)
	case "netns":
		_ = j.env.run(ctx, "ip", "netns", "delete", rec.Data)
		return nil
	case "link":
		_ = j.env.run(ctx, "ip", "link", "delete", rec.Data)
		return nil
	case "start":
		_ = j.env.services().Stop(ctx, rec.Data)
		return nil
	case "stop":
		_ = j.env.services().Start(ctx, rec.Data)
		return nil
	case "package":
		if rec.Data == "" {
			return nil
		}
		return j.env.run(ctx, "apt-get", "remove", "-y", rec.Data)
	default:
		return nil
	}
}

// Chmod records the previous mode and sets mode. A matching mode is a no-op,
// so a re-run can repair a drifted directory without a new journal record (§6.3).
func (j *Journal) Chmod(path string, mode os.FileMode) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.Mode().Perm() == mode.Perm() {
		return nil
	}
	if err := j.record(Record{
		Action: "chmod", Path: path,
		PrevMode: fmt.Sprintf("%04o", st.Mode().Perm()),
		Mode:     fmt.Sprintf("%04o", mode.Perm()),
	}); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// MkdirAll records every missing path, then creates it. sticky is set only
// on VarDir and InstallDir, which rollback keeps as the evidence of the attempt.
func (j *Journal) MkdirAll(path string, mode os.FileMode) error {
	path = filepath.Clean(path)
	var missing []string
	for cur := path; ; {
		_, err := os.Lstat(cur)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, cur)
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	l := j.env.layout()
	for i := len(missing) - 1; i >= 0; i-- {
		p := missing[i]
		sticky := p == l.VarDir || p == l.InstallDir
		if err := j.record(Record{
			Action: "mkdir", Path: p, Prev: "absent",
			Mode: fmt.Sprintf("%04o", mode.Perm()), Sticky: sticky,
		}); err != nil {
			return err
		}
		if err := os.Mkdir(p, mode); err != nil && !os.IsExist(err) {
			return err
		}
		if err := os.Chmod(p, mode); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile records the previous bytes, then replaces the file.
func (j *Journal) WriteFile(path string, data []byte, mode os.FileMode) error {
	return j.replaceFile(path, mode, func() error { return fsutil.WriteFileAtomic(path, data, mode) })
}

func (j *Journal) replaceFile(path string, mode os.FileMode, write func() error) error {
	prev := "absent"
	prevMode := ""
	backup := ""
	if st, err := os.Lstat(path); err == nil {
		if st.Mode()&os.ModeSymlink != 0 {
			return errf(ExitFailed, "failed", "refusing to replace a symlink")
		}
		prev = "present"
		prevMode = fmt.Sprintf("%04o", st.Mode().Perm())
		backup = filepath.Join(j.env.layout().blobDir(), fmt.Sprintf("%d-%s", len(j.file.Records), filepath.Base(path)))
		if err := copyFile(path, backup, 0o600); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := j.record(Record{
		Action: "write", Path: path, Prev: prev, PrevMode: prevMode,
		Mode: fmt.Sprintf("%04o", mode.Perm()), Backup: backup,
	}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := write(); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// Symlink records the previous target, then points link at target.
// target is stored verbatim (§13.5 relative current link).
func (j *Journal) Symlink(link, target string) error {
	prev := "absent"
	if t, err := os.Readlink(link); err == nil {
		prev = t
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := j.record(Record{Action: "symlink", Path: link, Prev: prev, Data: target}); err != nil {
		return err
	}
	return fsutil.SwitchSymlink(link, target)
}

func (j *Journal) Note(action, path, data, note string) error {
	return j.record(Record{Action: action, Path: path, Data: data, Note: note, Prev: "absent"})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(dst, mode)
}

func parseMode(s string) (os.FileMode, error) {
	var m uint32
	if _, err := fmt.Sscanf(s, "%o", &m); err != nil {
		return 0, err
	}
	return os.FileMode(m), nil
}

func parseModeOr(s string, def os.FileMode) os.FileMode {
	m, err := parseMode(s)
	if err != nil {
		return def
	}
	return m
}
