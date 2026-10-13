package main

// The full-bleed cover: the album cover fills the whole screen (cropped top and bottom,
// since the screen is wider than it is tall), with the controls in a column on the right.
// Tap the picture to hide the controls; tap again to bring them back.

import (
	"image"

	xdraw "golang.org/x/image/draw"
)

// coverStyle: how the controls sit on the picture. "frost" lightens a strip down the
// right (a see-through panel); "circles" puts white round buttons straight on the picture.
var coverStyle = "circles"

const stripW = 112 // the frosted strip's width

func coverFullView(v *View, d *Desk, ui *UI) {
	img := v.Img
	m := d.Music
	W, H := img.Rect.Dx(), img.Rect.Dy() // the canvas, in pixels
	full := fullArt(ui, W)
	off := (full.Rect.Dy() - H) / 2 // crop the middle
	for y := 0; y < H; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+W], full.Pix[(y+off)*full.Stride:])
	}
	v.add(Target{R: image.Rect(0, 0, LW, LH), Act: "coverbare"}) // first, so the buttons on top win
	cx := float64(LW - stripW/2)
	if !ui.CoverBare && coverStyle == "frost" {
		sw := px(stripW, scaleOf(img))
		for y := 0; y < H; y++ {
			row := img.Pix[y*img.Stride+W-sw : y*img.Stride+W]
			for i, p := range row {
				row[i] = 255 - uint8(int(255-p)*30/100) // 70 % white over the picture
			}
		}
	}
	bayer16(img)
	if !ui.CoverBare {
		type btn struct {
			y         float64
			act, arg  string
			main      bool
			drawGlyph func(c uint8)
		}
		ys := [5]float64{58, float64(LH)/2 - 104, float64(LH) / 2, float64(LH)/2 + 104, float64(LH) - 58}
		for _, b := range []btn{
			{ys[0], "coverfull", "", false, func(c uint8) { cornersGlyph(img, cx, ys[0], 13, true, c) }},
			{ys[1], "music", "previous", false, func(c uint8) { transportIcon(img, "prev", cx, ys[1], m.Playing, c) }},
			{ys[2], "music", "playpause", true, func(c uint8) { transportIcon(img, "playpause", cx, ys[2], m.Playing, c) }},
			{ys[3], "music", "next", false, func(c uint8) { transportIcon(img, "next", cx, ys[3], m.Playing, c) }},
			{ys[4], "refresh", "", false, func(c uint8) { refreshIcon(img, cx, ys[4], 11) }},
		} {
			rad := 32.0
			if b.main {
				rad = 38
			}
			r := image.Rect(int(cx-rad), int(b.y-rad), int(cx+rad), int(b.y+rad))
			switch {
			case b.main:
				fillCircle(img, cx, b.y, rad, ink)
				b.drawGlyph(paper)
			case coverStyle == "circles":
				fillCircle(img, cx, b.y, rad, paper)
				ringCircle(img, cx, b.y, rad, 1.5, grey3)
				b.drawGlyph(ink)
			default:
				b.drawGlyph(ink)
			}
			v.add(Target{R: image.Rect(LW-stripW, int(b.y)-50, LW, int(b.y)+50), Show: r, Act: b.act, Arg: b.arg})
		}
	}
	if ui.Dark { // a photo stays a photo: undo the dark-mode inversion that follows
		invertRect(img, img.Rect)
	}
}

// fullArt is the cover enlarged to the screen's width w in pixels (not yet dithered), made once
// per song.
func fullArt(ui *UI, w int) *image.Gray {
	if ui.ArtFullKey != ui.ArtKey || ui.ArtFull == nil || ui.ArtFull.Rect.Dx() != w {
		g := image.NewGray(image.Rect(0, 0, w, w))
		xdraw.CatmullRom.Scale(g, g.Rect, ui.Art, ui.Art.Rect, xdraw.Src, nil)
		ui.ArtFull, ui.ArtFullKey = g, ui.ArtKey
	}
	return ui.ArtFull
}
