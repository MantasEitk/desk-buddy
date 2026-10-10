package main

import "testing"

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
