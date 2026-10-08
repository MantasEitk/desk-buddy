package main

// Updates over Wi-Fi: the backend says which build is current (its SHA-256). If this
// Kindle's copy differs, download and verify it as desk.new; the header then offers
// **Update**, and only a tap installs it (desk.new → desk, keeping desk.old) and exits
// with code 3 so run.sh starts the new one. run.sh itself is kept up to date the same
// way, so the Kindle never needs USB for updates.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const exitUpdate = 3

// buildVersion is set by build.sh (the build time), shown in the brightness panel.
var buildVersion = "dev"

type Release struct {
	SHA256  string      `json:"sha256"`
	Size    int64       `json:"size"`
	File    string      `json:"file"` // path on the backend, e.g. /kindle/desk-1a2b….bin
	RunSHA  string      `json:"run_sha256"`
	RunFile string      `json:"run_file"`
	Notes   []NoteEntry `json:"notes"` // the newest patch notes (patchnotes.go)
}

// lastNotes: the newest patch notes from the last release check.
var lastNotes []NoteEntry

func fileSHA(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))
}

// download fetches path into dst (via dst.part), checking its size and SHA-256.
func (c *Client) download(path, dst string, size int64, sha string) error {
	dl := *c.hc
	dl.Timeout = 0 // big file on slow Wi-Fi: no overall limit (a stalled connection still times out)
	r, err := dl.Get(c.Base() + path)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("download: %s", r.Status)
	}
	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), r.Body)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && ((size > 0 && n != size) || hex.EncodeToString(h.Sum(nil)) != sha) {
		err = fmt.Errorf("download didn't match (got %d bytes)", n)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	os.Chmod(tmp, 0o755)
	return os.Rename(tmp, dst)
}

// checkUpdate brings run.sh up to date and downloads a newer app build as desk.new.
// Returns true when one is ready to install.
func (c *Client) checkUpdate(dir string) (bool, error) {
	resp, err := c.hc.Get(c.url("/api/kindle/release"))
	if err != nil {
		return false, err
	}
	var rel Release
	err = json.NewDecoder(resp.Body).Decode(&rel)
	resp.Body.Close()
	if err != nil || rel.SHA256 == "" {
		return false, err
	}
	lastNotes = rel.Notes
	// run.sh first: small, and it's what starts the new build.
	if rel.RunSHA != "" && fileSHA(filepath.Join(dir, "run.sh")) != rel.RunSHA {
		if err := c.download(rel.RunFile, filepath.Join(dir, "run.sh"), 0, rel.RunSHA); err != nil {
			return false, fmt.Errorf("run.sh: %w", err)
		}
	}
	installed := filepath.Join(dir, "desk")
	if fileSHA(installed) == rel.SHA256 {
		os.Remove(filepath.Join(dir, "desk.new")) // nothing pending
		return false, nil                         // up to date
	}
	next := filepath.Join(dir, "desk.new")
	if fileSHA(next) != rel.SHA256 {
		if err := c.download(rel.File, next, rel.Size, rel.SHA256); err != nil {
			return false, err
		}
	}
	return true, nil // downloaded and verified; installed only when the user taps Update
}

// installDownloaded puts the verified desk.new in place (keeping desk.old). The app runs
// from a copy in /tmp, so replacing /mnt/us/desk/desk is safe.
func installDownloaded(dir string) error {
	installed, next, old := filepath.Join(dir, "desk"), filepath.Join(dir, "desk.new"), filepath.Join(dir, "desk.old")
	if _, err := os.Stat(next); err != nil {
		return err
	}
	os.Remove(old)
	if err := os.Rename(installed, old); err != nil {
		return err
	}
	if err := os.Rename(next, installed); err != nil {
		os.Rename(old, installed) // put the old one back
		return err
	}
	syncFS()
	return nil
}
