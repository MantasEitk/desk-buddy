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
