package main

// Updating itself: the Desk Buddy server this PC is connected to (your Mac, another computer, or
// your own cloud) offers the newest Windows app at /windows/release.json (windows/release.sh puts
// it there). When it's newer, the window shows "Update Desk Buddy"; clicking it downloads the
// new Desk Buddy.exe, checks it, swaps it in (Windows lets a running program be renamed) and
// starts it. Once the repo is public this can also come from GitHub Releases.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type release struct {
	Version string   `json:"version"`
	SHA256  string   `json:"sha256"`
	Size    int64    `json:"size"`
	File    string   `json:"file"`
	Notes   []string `json:"notes"`
}

var (
	updMu    sync.Mutex
	updReady *release
	updNote  string
	updBusy  bool
)

// newer: "0.10" > "0.9" > "0.3".
func newer(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(pa), len(pb)); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func checkUpdate() error {
	if !connected() {
		return fmt.Errorf("not connected")
	}
	var r release
	if err := call(quick, "GET", "/windows/release.json", nil, &r); err != nil {
		return err
	}
	updMu.Lock()
	defer updMu.Unlock()
	if r.Version != "" && r.SHA256 != "" && newer(r.Version, version) {
		updReady = &r
	} else {
		updReady = nil
	}
	return nil
}

// updates looks a minute after starting and then every 6 hours.
func updates() {
	time.Sleep(time.Minute)
	for {
		checkUpdate()
		time.Sleep(6 * time.Hour)
	}
}

func updateState() (*release, string, bool) {
	updMu.Lock()
	defer updMu.Unlock()
	return updReady, updNote, updBusy
}

func setUpdNote(s string, busy bool) {
	updMu.Lock()
	updNote, updBusy = s, busy
	updMu.Unlock()
}

// applyUpdate downloads, checks and swaps in the new Desk Buddy.exe, then restarts.
func applyUpdate() {
	r, _, busy := updateState()
	if r == nil || busy {
		return
	}
	setUpdNote("Downloading…", true)
	fail := func(msg string) { setUpdNote(msg, false); report("update: " + msg) }
	c := current()
	req, _ := http.NewRequest("GET", strings.TrimRight(c.Server, "/")+r.File, nil)
	req.Header.Set("X-Key", c.Key)
	req.Header.Set("User-Agent", userAgent)
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil || resp.StatusCode != 200 {
		fail("Couldn't download it. Try again in a moment.")
		return
	}
	defer resp.Body.Close()
	dir := filepath.Join(configDir(), "update")
	os.MkdirAll(dir, 0o700)
	tmp := filepath.Join(dir, "Desk Buddy.exe.part")
	f, err := os.Create(tmp)
	if err != nil {
		fail("Couldn't save it: " + err.Error())
		return
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), resp.Body)
	f.Close()
	if err != nil || (r.Size > 0 && n != r.Size) || hex.EncodeToString(h.Sum(nil)) != r.SHA256 {
		os.Remove(tmp)
		fail("The download didn't check out. Try again.")
		return
	}
	setUpdNote("Installing…", true)
	exe, _ := os.Executable()
	old := strings.TrimSuffix(exe, ".exe") + ".old.exe"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil { // a running program can be renamed, not overwritten
		fail("Couldn't replace Desk Buddy: " + err.Error())
		return
	}
	if err := copyFile(tmp, exe); err != nil {
		os.Rename(old, exe) // put it back
		fail("Couldn't replace Desk Buddy: " + err.Error())
		return
	}
	os.Remove(tmp)
	setUpdNote("Restarting…", true)
	if err := exec.Command(exe, "--after-update").Start(); err != nil {
		fail("Updated, but couldn't restart: open Desk Buddy again")
		return
	}
	quitApp()
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// cleanAfterUpdate removes the previous version's file once it has stopped.
func cleanAfterUpdate() {
	exe, _ := os.Executable()
	old := strings.TrimSuffix(exe, ".exe") + ".old.exe"
	for i := 0; i < 30; i++ {
		if _, err := os.Stat(old); err != nil || os.Remove(old) == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}
