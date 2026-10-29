package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

type hashMode int

const (
	hashRaw hashMode = iota
	hashCanonical
)

const downloadChunk = 32 * 1024

// errContentChanged means a resume is not valid for the bytes already on
// disk (§13.4, AT26). The partial is discarded and the whole file is fetched
// again, then re-verified. It is not a network retry.
var errContentChanged = errors.New("updater: download content changed")

type partialMeta struct {
	URL  string `json:"url"`
	Hash string `json:"hash"`
	Mode string `json:"mode"`
	ETag string `json:"etag"`
	Size int64  `json:"size"`
}

func (m hashMode) String() string {
	if m == hashCanonical {
		return "canonical"
	}
	return "raw"
}

// download fetches dest with resume. A partial file is valid only for the
// same URL, content hash and mode. A 200 after a range request, a mismatched
// Content-Range, or a hash mismatch discards the partial and downloads the
// whole file again, bounded by ContentRetries (§13.4). Permanent hash and
// size failures are returned without further attempts.
func (e *Engine) download(ctx context.Context, rawURL, dest, want string, mode hashMode, size int64, poll func() error) error {
	if err := checkURL(rawURL, e.AllowLoopback); err != nil {
		return err
	}
	if e.Faults.DiskFullDownload {
		_ = os.Remove(dest + ".partial")
		return transient(apierr.DiskSpaceLow, "download")
	}
	if ok, err := fileMatches(dest, want, mode, size); err != nil {
		return err
	} else if ok {
		return nil
	}

	cfg := e.Download.normalize()
	partial := dest + ".partial"
	metaPath := partial + ".meta"
	contentTries := 0
	netTries := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if poll != nil {
			if err := poll(); err != nil {
				return err
			}
		}
		err := e.downloadOnce(ctx, rawURL, partial, metaPath, want, mode, size, poll)
		if err == nil {
			ok, herr := fileMatches(partial, want, mode, size)
			if herr != nil {
				_ = os.Remove(partial)
				_ = os.Remove(metaPath)
				return herr
			}
			if !ok {
				contentTries++
				_ = os.Remove(partial)
				_ = os.Remove(metaPath)
				if contentTries > cfg.ContentRetries {
					return permanent(apierr.HashMismatch, "artifact")
				}
				continue
			}
			if err := os.Rename(partial, dest); err != nil {
				return wrapCode(apierr.StoreUnavailable, err, true)
			}
			_ = os.Remove(metaPath)
			if err := fsutil.SyncDir(filepath.Dir(dest)); err != nil {
				return wrapCode(apierr.CheckpointFailed, err, true)
			}
			return nil
		}
		if errors.Is(err, errContentChanged) {
			contentTries++
			_ = os.Remove(partial)
			_ = os.Remove(metaPath)
			if contentTries > cfg.ContentRetries {
				return permanent(apierr.HashMismatch, "content changed during resume")
			}
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		var op *OpError
		if errors.As(err, &op) && !op.Retry {
			_ = os.Remove(partial)
			_ = os.Remove(metaPath)
			return op
		}
		if fsutil.IsNoSpace(err) {
			_ = os.Remove(partial)
			return transient(apierr.DiskSpaceLow, "download")
		}
		netTries++
		e.log().Info("download retry", "url", safeURL(rawURL), "attempt", netTries, "err", errCode(err))
		if netTries >= cfg.MaxAttempts {
			if errors.Is(err, context.DeadlineExceeded) {
				return transient(apierr.DownloadTimeout, "attempt")
			}
			return transient(apierr.StoreUnavailable, "download")
		}
		if err := e.sleepCtx(ctx, e.backoff(netTries)); err != nil {
			return err
		}
	}
}

func (e *Engine) downloadOnce(ctx context.Context, rawURL, partial, metaPath, want string, mode hashMode, size int64, poll func() error) error {
	meta, _ := readMeta(metaPath)
	if meta.URL != "" && (meta.URL != rawURL || meta.Hash != want || meta.Mode != mode.String() || (size > 0 && meta.Size > 0 && meta.Size != size)) {
		return errContentChanged
	}
	var offset int64
	if st, err := os.Stat(partial); err == nil {
		offset = st.Size()
		if size > 0 && offset > size {
			return errContentChanged
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if offset == 0 {
		_ = os.Remove(partial)
	}

	cfg := e.Download.normalize()
	cctx, cancel := context.WithTimeout(ctx, cfg.AttemptTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return permanent(apierr.SignatureInvalid, "artifact url")
	}
	req.Header.Set("User-Agent", "fvpn-updater/"+Version)
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		if meta.ETag != "" {
			req.Header.Set("If-Range", meta.ETag)
		}
	}
	resp, err := e.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if offset > 0 {
			return errContentChanged
		}
		offset = 0
		_ = os.Remove(partial)
	case http.StatusPartialContent:
		start, _, total, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok || start != offset {
			return errContentChanged
		}
		if size > 0 && total >= 0 && total != size {
			return errContentChanged
		}
	case http.StatusRequestedRangeNotSatisfiable:
		return errContentChanged
	default:
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusRequestTimeout {
			return transient(apierr.StoreUnavailable, resp.Status)
		}
		return transient(apierr.StoreUnavailable, resp.Status)
	}

	f, err := os.OpenFile(partial, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if offset == 0 {
		if err := f.Truncate(0); err != nil {
			return err
		}
	} else if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	buf := make([]byte, downloadChunk)
	written := offset
	for {
		if poll != nil {
			if err := poll(); err != nil {
				_ = f.Sync()
				return err
			}
		}
		if e.Faults.DiskFullDownload {
			return transient(apierr.DiskSpaceLow, "download")
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if size > 0 && written+int64(n) > size {
				return permanent(apierr.SizeMismatch, "artifact longer than manifest")
			}
			if _, werr := f.Write(buf[:n]); werr != nil {
				if fsutil.IsNoSpace(werr) {
					return transient(apierr.DiskSpaceLow, "download")
				}
				return werr
			}
			written += int64(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			_ = f.Sync()
			return rerr
		}
	}
	if size > 0 && written != size {
		// Short body: keep the partial so a later range request can resume,
		// unless the server claimed this was the whole object (200).
		if resp.StatusCode == http.StatusOK {
			return io.ErrUnexpectedEOF
		}
		return io.ErrUnexpectedEOF
	}
	if err := f.Sync(); err != nil {
		return err
	}
	etag := resp.Header.Get("ETag")
	meta = partialMeta{URL: rawURL, Hash: want, Mode: mode.String(), ETag: etag, Size: size}
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(metaPath, raw, 0o600)
}

func fileMatches(path, want string, mode hashMode, size int64) (bool, error) {
	st, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if size > 0 && st.Size() != size {
		return false, nil
	}
	got, err := hashPath(path, mode)
	if err != nil {
		if mode == hashCanonical {
			return false, permanent(apierr.HashMismatch, "manifest is not canonical JSON")
		}
		return false, err
	}
	if !strings.EqualFold(got, want) {
		return false, nil
	}
	return true, nil
}

func hashPath(path string, mode hashMode) (string, error) {
	if mode == hashCanonical {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		c, err := canonjson.Canonicalize(b)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(c)
		return hex.EncodeToString(sum[:]), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readMeta(path string) (partialMeta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return partialMeta{}, err
	}
	var m partialMeta
	err = json.Unmarshal(b, &m)
	return m, err
}

func parseContentRange(s string) (start, end, total int64, ok bool) {
	// bytes START-END/TOTAL or bytes START-END/*
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(s), "bytes ") {
		return 0, 0, 0, false
	}
	rest := strings.TrimSpace(s[6:])
	dash := strings.IndexByte(rest, '-')
	slash := strings.IndexByte(rest, '/')
	if dash < 0 || slash < 0 || slash < dash {
		return 0, 0, 0, false
	}
	var err error
	start, err = strconv.ParseInt(rest[:dash], 10, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	end, err = strconv.ParseInt(rest[dash+1:slash], 10, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	tot := rest[slash+1:]
	if tot == "*" {
		return start, end, -1, true
	}
	total, err = strconv.ParseInt(tot, 10, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return start, end, total, true
}

func (e *Engine) backoff(attempt int) time.Duration {
	cfg := e.Download.normalize()
	d := cfg.Initial
	for i := 1; i < attempt; i++ {
		if d > cfg.MaxWait/2 {
			d = cfg.MaxWait
			break
		}
		d *= 2
	}
	if d > cfg.MaxWait {
		d = cfg.MaxWait
	}
	j := 1.0
	if e.Jitter != nil {
		j = e.Jitter()
	} else {
		j = e.jitter()
	}
	if j < 0 {
		j = 0
	}
	if j > 1 {
		j = 1
	}
	return time.Duration(float64(d) * j)
}

func (e *Engine) sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	if e.Sleep != nil {
		e.Sleep(d)
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func errCode(err error) string {
	if err == nil {
		return ""
	}
	if op := asOp(err); op != nil {
		return string(op.Code)
	}
	return err.Error()
}
