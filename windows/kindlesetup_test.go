package main

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A pretend Kindle drive: Put Desk Buddy on it copies the files, set up for this PC's server,
// keeps the old rotation, and the manual zip has the same files.
func TestKindleSetup(t *testing.T) {
	root := t.TempDir()
	drive := filepath.Join(root, "KINDLE")
	os.MkdirAll(filepath.Join(drive, "documents"), 0o755)
	os.MkdirAll(filepath.Join(drive, "system"), 0o755)
	os.MkdirAll(filepath.Join(drive, "desk"), 0o755)
	os.WriteFile(filepath.Join(drive, "desk", "config.json"), []byte(`{"rotation": 270}`), 0o644)
	os.WriteFile(filepath.Join(drive, "desk", "cover.png"), []byte("png"), 0o644)
	pub := filepath.Join(root, "pub")
	os.MkdirAll(pub, 0o755)
	bin := []byte("pretend kindle app")
	os.WriteFile(filepath.Join(pub, "desk-abc.bin"), bin, 0o644)
	os.WriteFile(filepath.Join(pub, "run-def.sh"), []byte("#!/bin/sh"), 0o644)
	ksRoots = func() []string { return []string{filepath.Join(root, "pub"), drive} }
	kindlePub = func() string { return pub }
	cfgMu.Lock()
	old := cfg
	cfg = Config{Mode: "remote", Server: "http://192.168.1.23:8787", Key: "k"}
	cfgMu.Unlock()
	defer func() { cfgMu.Lock(); cfg = old; cfgMu.Unlock(); ksRoots = kindleRoots; ks = kSetup{State: "waiting"} }()

	if st := kindleSetupState(); st.State != "found" || st.Drive != drive || st.Has {
		t.Fatalf("should find the Kindle without Desk Buddy on it: %+v", st)
	}
	installKindle()
	for i := 0; i < 50 && ks.State != "done" && ks.State != "failed"; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if ks.State != "done" {
		t.Fatalf("install: %+v", ks)
	}
	got, _ := os.ReadFile(filepath.Join(drive, "desk", "desk"))
	if sha256.Sum256(got) != sha256.Sum256(bin) {
		t.Error("desk/desk wasn't copied")
	}
	if item, err := os.ReadFile(filepath.Join(drive, "documents", "Desk Buddy.sh")); err != nil {
		t.Error("no library item")
	} else if !strings.Contains(string(item), "# Icon: /mnt/us/desk/cover.png\n# DontUseFBInk") {
		t.Errorf("the library item lost its cover line:\n%s", item)
	}
	var c map[string]any
	b, _ := os.ReadFile(filepath.Join(drive, "desk", "config.json"))
	json.Unmarshal(b, &c)
	if c["server"] != "http://192.168.1.23:8787" || c["key"] != "k" || c["rotation"] != 270.0 {
		t.Errorf("config: %v", c)
	}
}
