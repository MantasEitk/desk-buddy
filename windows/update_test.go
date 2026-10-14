package main

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"0.4", "0.3", true}, {"0.10", "0.9", true}, {"0.3", "0.3", false}, {"0.3", "0.4", false}, {"1.0", "0.9.9", true}, {"0.3.1", "0.3", true}} {
		if newer(c.a, c.b) != c.want {
			t.Errorf("newer(%s, %s) should be %v", c.a, c.b, c.want)
		}
	}
}

// The newest patch notes are for this version, newest first, each with something in it.
func TestPatchNotes(t *testing.T) {
	n := patchNotes()
	if len(n) == 0 || n[0].Version != version {
		t.Fatalf("patchnotes.json should start with version %s", version)
	}
	for i, e := range n {
		if len(e.Items) == 0 || e.Date == "" {
			t.Errorf("version %s needs a date and items", e.Version)
		}
		if i > 0 && !newer(n[i-1].Version, e.Version) {
			t.Errorf("%s should come before %s", e.Version, n[i-1].Version)
		}
	}
}

// A GitHub update replaces every file in the folder, keeps the running programs aside as
// .old.exe, and leaves files that aren't in the new version alone.
func TestSwapFolder(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "app")
	os.MkdirAll(filepath.Join(app, "public", "kindle"), 0o755)
	os.WriteFile(filepath.Join(app, "Desk Buddy.exe"), []byte("old app"), 0o644)
	os.WriteFile(filepath.Join(app, "desk-buddy-server.exe"), []byte("old server"), 0o644)
	os.WriteFile(filepath.Join(app, "public", "kindle", "desk-old.bin"), []byte("old kindle"), 0o644)

	zp := filepath.Join(dir, "update.zip")
	f, _ := os.Create(zp)
	w := zip.NewWriter(f)
	for name, body := range map[string]string{
		"Desk Buddy for Windows/":                           "",
		"Desk Buddy for Windows/Desk Buddy.exe":             "new app",
		"Desk Buddy for Windows/desk-buddy-server.exe":      "new server",
		"Desk Buddy for Windows/public/kindle/desk-new.bin": "new kindle",
	} {
		e, _ := w.Create(name)
		e.Write([]byte(body))
	}
	w.Close()
	f.Close()

	if err := swapFolder(zp, filepath.Join(dir, "staging"), app); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"Desk Buddy.exe": "new app", "desk-buddy-server.exe": "new server",
		"Desk Buddy.old.exe": "old app", "desk-buddy-server.old.exe": "old server",
		"public/kindle/desk-new.bin": "new kindle", "public/kindle/desk-old.bin": "old kindle",
	} {
		if b, _ := os.ReadFile(filepath.Join(app, filepath.FromSlash(name))); string(b) != want {
			t.Errorf("%s = %q, want %q", name, b, want)
		}
	}
}

// A zip with a path outside the folder is refused before anything is replaced.
func TestSwapFolderRefusesEscapes(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "bad.zip")
	f, _ := os.Create(zp)
	w := zip.NewWriter(f)
	e, _ := w.Create("Desk Buddy for Windows/../../evil.exe")
	e.Write([]byte("x"))
	w.Close()
	f.Close()
	if err := swapFolder(zp, filepath.Join(dir, "staging"), filepath.Join(dir, "app")); err == nil {
		t.Fatal("should refuse")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.exe")); err == nil {
		t.Fatal("wrote outside the folder")
	}
}

// The GitHub check reads the Windows part of desk-buddy-latest.json and offers it when newer.
func TestCheckGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag":"v9.0","mac":{"version":"9.0"},"windows":{"version":"9.0",
			"url":"https://github.com/deskbuddy-app/desk-buddy/releases/download/v9.0/Desk-Buddy-for-Windows.zip",
			"sha256":"abc","size":3,"patchnotes":[{"version":"9.0","date":"2026-10-14","items":["x"]}]}}`))
	}))
	defer srv.Close()
	was := latestURL
	latestURL = srv.URL
	defer func() { latestURL = was }()
	if err := checkUpdate(); err != nil {
		t.Fatal(err)
	}
	r, _, _ := updateState()
	if r == nil || r.Version != "9.0" || r.URL == "" || len(r.PatchNotes) != 1 {
		t.Fatalf("should offer 9.0 from GitHub, got %+v", r)
	}
}
