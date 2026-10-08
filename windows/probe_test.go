package main

import "testing"

// On the Mac: the direct check finds this Mac's own Desk Buddy when it isn't skipped as "mine".
func TestProbe(t *testing.T) {
	f := probeSubnets(map[string]bool{})
	for _, x := range f {
		t.Log("found", x.Name, x.Addr)
	}
	if len(f) == 0 {
		t.Skip("no Desk Buddy on port 8787 on this network")
	}
}
