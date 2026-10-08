package main

import (
	"image"
	"testing"
)

// Tapping a Deck tile changes the tile and the status line: two areas, not one big one.
func TestChangedKeepsSeparateAreasApart(t *testing.T) {
	a, b := newCanvas(LW, LH), newCanvas(LW, LH)
	fillRect(b, image.Rect(520, 552, 750, 732), ink) // a bottom tile
	fillRect(b, image.Rect(700, 100, 996, 130), ink) // the status text, top right
	got := changed(a, b, nil, nil)
	if len(got) != 2 {
		t.Fatalf("want 2 areas, got %v", got)
	}
	for _, r := range got {
		if r.Dx()*r.Dy() > LW*LH/4 {
			t.Errorf("area %v is too big", r)
		}
	}
	if len(changed(a, a, nil, nil)) != 0 {
		t.Error("identical pictures should have no changes")
	}
}
