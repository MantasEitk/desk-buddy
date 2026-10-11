package main

// App icons for the Kindle's Deck, made the way the Mac app makes them (IconSync.swift's
// png()): 96×96 greyscale on white, the darkest tone stretched to black so it reads on e-ink.
// One difference: Windows icons fill their whole square, where the Mac's leave a margin that
// the Kindle trims to, so here the icon is first cropped to what's drawn and fitted inside
// that margin. Windows' part (getting the picture) is in apps_windows.go.

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/png"
	"math"
)

const (
	iconSide   = 96
	iconMargin = 13        // px on each side: the Kindle trims icons to a rounded square ~9.5% in
	iconMax    = 40000     // base64 characters per icon (the server drops bigger ones)
	iconsTotal = 1_400_000 // all of them together (the cloud's database keeps up to 2 MB in one value)
	iconsCount = 320       // the server keeps this many
	iconName   = 40        // the server cuts names to this; longer ones would never be found
)

// bitmapImage turns Windows' 32-bit top-down BGRA pixels into an image. The colours are
// usually premultiplied by alpha; if any colour is brighter than its alpha they aren't; and
// if every alpha is 0 the bitmap has no alpha at all (it's opaque).
func bitmapImage(bgra []byte, w, h int) image.Image {
	if w <= 0 || h <= 0 || len(bgra) < w*h*4 {
		return nil
	}
	straight, anyAlpha := false, false
	for i := 0; i < w*h*4; i += 4 {
		a := bgra[i+3]
		anyAlpha = anyAlpha || a != 0
		straight = straight || bgra[i] > a || bgra[i+1] > a || bgra[i+2] > a
	}
	pix := make([]byte, w*h*4)
	for i := 0; i < w*h*4; i += 4 {
		pix[i], pix[i+1], pix[i+2], pix[i+3] = bgra[i+2], bgra[i+1], bgra[i], bgra[i+3]
		if !anyAlpha {
			pix[i+3] = 255
		}
	}
	r := image.Rect(0, 0, w, h)
	if straight || !anyAlpha {
		return &image.NRGBA{Pix: pix, Stride: w * 4, Rect: r}
	}
	return &image.RGBA{Pix: pix, Stride: w * 4, Rect: r}
}

// iconPNG makes the Kindle's icon from an app's picture (nil if there's nothing to see).
func iconPNG(src image.Image) []byte {
	if src == nil || src.Bounds().Empty() {
		return nil
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy())) // premultiplied, from 0,0
	draw.Draw(rgba, rgba.Rect, src, b.Min, draw.Src)

	// what's drawn: the pixels that aren't (nearly) see-through
	crop := image.Rectangle{}
	for y := 0; y < rgba.Rect.Dy(); y++ {
		for x := 0; x < rgba.Rect.Dx(); x++ {
			if rgba.Pix[rgba.PixOffset(x, y)+3] > 16 {
				crop = crop.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if crop.Empty() {
		return nil
	}

	// fitted inside the margin, keeping its shape; each pixel is the average of 4×4 samples
	inner := float64(iconSide - 2*iconMargin)
	scale := math.Min(inner/float64(crop.Dx()), inner/float64(crop.Dy()))
	dw, dh := int(math.Round(float64(crop.Dx())*scale)), int(math.Round(float64(crop.Dy())*scale))
	ox, oy := (iconSide-dw)/2, (iconSide-dh)/2
	lum := make([]float64, iconSide*iconSide)
	alpha := make([]float64, iconSide*iconSide)
	for i := range lum {
		lum[i] = 255
	}
	const k = 4
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var r, g, bl, a float64
			for sy := 0; sy < k; sy++ {
				for sx := 0; sx < k; sx++ {
					px := float64(crop.Min.X) + (float64(x)+(float64(sx)+0.5)/k)/scale
					py := float64(crop.Min.Y) + (float64(y)+(float64(sy)+0.5)/k)/scale
					c := bilinear(rgba, px, py)
					r, g, bl, a = r+c[0], g+c[1], bl+c[2], a+c[3]
				}
			}
			r, g, bl, a = r/(k*k), g/(k*k), bl/(k*k), a/(k*k)
			i := (oy+y)*iconSide + ox + x
			// over white (premultiplied: colour + white × what shows through)
			lum[i] = 0.299*(r+255-a) + 0.587*(g+255-a) + 0.114*(bl+255-a)
			alpha[i] = a
		}
	}

	darkest := 255.0
	for _, l := range lum {
		darkest = math.Min(darkest, l)
	}
	if darkest > 200 { // a white icon (made for a coloured tile): invisible on white, so its shape in black
		darkest = 255
		for i, a := range alpha {
			lum[i] = 255 - a
			darkest = math.Min(darkest, lum[i])
		}
	}
	gray := image.NewGray(image.Rect(0, 0, iconSide, iconSide))
	span := math.Max(1, 255-darkest)
	for i, l := range lum {
		gray.Pix[i] = uint8(math.Max(0, math.Min(255, (l-darkest)/span*255)))
	}
	var buf bytes.Buffer
	if (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, gray) != nil {
		return nil
	}
	return buf.Bytes()
}

// bilinear: the premultiplied colour at (x, y) in pixels; outside the picture is see-through.
func bilinear(img *image.RGBA, x, y float64) [4]float64 {
	x, y = x-0.5, y-0.5
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	var out [4]float64
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			w := (1 - math.Abs(float64(i)-fx)) * (1 - math.Abs(float64(j)-fy))
			px, py := x0+i, y0+j
			if w == 0 || px < 0 || py < 0 || px >= img.Rect.Dx() || py >= img.Rect.Dy() {
				continue
			}
			o := img.PixOffset(px, py)
			for c := 0; c < 4; c++ {
				out[c] += w * float64(img.Pix[o+c])
			}
		}
	}
	return out
}

// collectIcons: what to upload, keyed like the Mac's: each favourite tile by its label, then
// every app by its name, within the server's limits. icon gives an app's PNG by its target
// (nil if Windows has none).
func collectIcons(tiles []deckItem, apps map[string]string, icon func(target string) []byte) map[string]string {
	out := map[string]string{}
	total := 0
	add := func(key, target string) {
		if target == "" || len(key) > iconName || len(out) >= iconsCount {
			return
		}
		if _, have := out[key]; have {
			return
		}
		p := icon(target)
		if p == nil {
			return
		}
		s := base64.StdEncoding.EncodeToString(p)
		if len(s) >= iconMax || total+len(s) > iconsTotal {
			return
		}
		out[key] = s
		total += len(s)
	}
	for _, t := range tiles {
		if t.Label != "" && t.URL == nil {
			add(t.Label, tileApp(t, apps))
		}
	}
	for _, n := range sortedNames(apps) {
		add(n, apps[n])
	}
	return out
}
