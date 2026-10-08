package main

// The Garden theme: a botanical-engraving garden drawn around the screens (no image files):
// vines climbing the edges and dividers, leaves with a midrib, five-petal flowers, buds,
// curling tendrils and grass. Decoration stays in the margins and on dividers, never over
// text or buttons, and is the same on every redraw (a fixed seed), so it never flashes.
// A ticked-off task's circle becomes a small black flower.

import (
	"image"
	"math"
	"math/rand"
)

// themeName: "" (plain) or "garden". (Settings › Screen › Theme style, once it's a setting.)
var themeName = ""

func garden() bool { return themeName == "garden" }

func ellipsePts(cx, cy, rx, ry, angle float64, n int) [][2]float64 {
	pts := make([][2]float64, n)
	ca, sa := math.Cos(angle), math.Sin(angle)
	for i := range pts {
		t := 2 * math.Pi * float64(i) / float64(n)
		x, y := rx*math.Cos(t), ry*math.Sin(t)
		pts[i] = [2]float64{cx + x*ca - y*sa, cy + x*sa + y*ca}
	}
	return pts
}

func outline(img *image.Gray, pts [][2]float64, w float64, c uint8) {
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		line(img, a[0], a[1], b[0], b[1], w, c)
	}
}

// leaf: an almond-shaped leaf from (x, y) pointing at angle, grey with a black edge and midrib.
func leaf(img *image.Gray, x, y, angle, length, width float64) {
	var pts [][2]float64
	ca, sa := math.Cos(angle), math.Sin(angle)
	at := func(u, v float64) [2]float64 { return [2]float64{x + u*ca - v*sa, y + u*sa + v*ca} }
	const n = 14
	for i := 0; i <= n; i++ { // one side, base to tip
		t := float64(i) / n
		pts = append(pts, at(t*length, math.Sin(t*math.Pi)*width/2*(1-0.25*t)))
	}
	for i := n - 1; i > 0; i-- { // and back along the other side
		t := float64(i) / n
		pts = append(pts, at(t*length, -math.Sin(t*math.Pi)*width/2*(1-0.25*t)))
	}
	fillPolygon(img, pts, grey3)
	outline(img, pts, 1.2, ink)
	m0, m1 := at(0, 0), at(length*0.85, 0)
	line(img, m0[0], m0[1], m1[0], m1[1], 1, ink)
}

// flower: five petals around a centre. solid: black petals and a white centre (a ticked task);
// otherwise white petals outlined in black around a black centre.
func flower(img *image.Gray, cx, cy, r, turn float64, solid bool) {
	for i := 0; i < 5; i++ {
		a := turn + 2*math.Pi*float64(i)/5
		p := ellipsePts(cx+math.Cos(a)*r*0.52, cy+math.Sin(a)*r*0.52, r*0.5, r*0.31, a, 18)
		if solid {
			fillPolygon(img, p, ink)
		} else {
			fillPolygon(img, p, paper)
			outline(img, p, 1.2, ink)
		}
	}
	if solid {
		fillCircle(img, cx, cy, r*0.27, paper)
		fillCircle(img, cx, cy, r*0.12, ink)
	} else {
		fillCircle(img, cx, cy, r*0.24, ink)
	}
}

// bigFlower: a layered bloom (eight outer petals, five inner, a dotted centre).
func bigFlower(img *image.Gray, cx, cy, r, turn float64) {
	for i := 0; i < 8; i++ {
		a := turn + 2*math.Pi*float64(i)/8
		p := ellipsePts(cx+math.Cos(a)*r*0.58, cy+math.Sin(a)*r*0.58, r*0.46, r*0.22, a, 20)
		fillPolygon(img, p, grey4)
		outline(img, p, 1.2, ink)
	}
	for i := 0; i < 5; i++ {
		a := turn + 0.3 + 2*math.Pi*float64(i)/5
		p := ellipsePts(cx+math.Cos(a)*r*0.34, cy+math.Sin(a)*r*0.34, r*0.3, r*0.18, a, 18)
		fillPolygon(img, p, paper)
		outline(img, p, 1.2, ink)
	}
	fillCircle(img, cx, cy, r*0.2, ink)
	for i := 0; i < 6; i++ {
		a := 2 * math.Pi * float64(i) / 6
		fillCircle(img, cx+math.Cos(a)*r*0.1, cy+math.Sin(a)*r*0.1, 1.2, paper)
	}
}

// bud: a closed flower on a short stalk.
func bud(img *image.Gray, x, y, angle, size float64) {
	tx, ty := x+math.Cos(angle)*size, y+math.Sin(angle)*size
	line(img, x, y, tx, ty, 1.4, ink)
	p := ellipsePts(tx+math.Cos(angle)*size*0.45, ty+math.Sin(angle)*size*0.45, size*0.55, size*0.32, angle, 16)
	fillPolygon(img, p, grey2)
	outline(img, p, 1.1, ink)
}

// tendril: a little curl.
func tendril(img *image.Gray, x, y, angle, size float64, dir float64) {
	px, py := x, y
	for i := 1; i <= 30; i++ {
		t := float64(i) / 30
		a := angle + dir*t*3.6
		rr := size * (1 - t*0.7)
		nx, ny := px+math.Cos(a)*rr*0.12, py+math.Sin(a)*rr*0.12
		line(img, px, py, nx, ny, 1.1, ink)
		px, py = nx, ny
	}
}

// leafMin: the smallest leaf (the standby frame grows them bigger than the dividers do).
var leafMin = 12.0

// butterfly: two pairs of wings and a body, resting at (x, y).
func butterfly(img *image.Gray, x, y, s float64) {
	for _, side := range []float64{-1, 1} {
		up := ellipsePts(x+side*s*0.55, y-s*0.3, s*0.6, s*0.42, side*0.5, 20)
		lo := ellipsePts(x+side*s*0.42, y+s*0.42, s*0.4, s*0.28, -side*0.6, 18)
		fillPolygon(img, up, grey4)
		outline(img, up, 1.2, ink)
		fillPolygon(img, lo, grey3)
		outline(img, lo, 1.2, ink)
		fillCircle(img, x+side*s*0.6, y-s*0.35, s*0.12, ink)   // a spot on each wing
		line(img, x, y-s*0.5, x+side*s*0.35, y-s*1.05, 1, ink) // antennae
		fillCircle(img, x+side*s*0.35, y-s*1.05, 1.6, ink)
	}
	line(img, x, y-s*0.5, x, y+s*0.6, 3, ink)
}

// vine: a stem along path(t), t from 0 to 1, with leaves, buds, tendrils and the odd flower.
// keep reports whether a point is free to draw on (out of the way of text and buttons).
func vine(img *image.Gray, rng *rand.Rand, path func(t float64) (float64, float64), steps int, every float64, flowers bool, keep func(x, y float64) bool) {
	px, py := path(0)
	dist, side, count := 0.0, 1.0, 0
	for i := 1; i <= steps; i++ {
		x, y := path(float64(i) / float64(steps))
		line(img, px, py, x, y, 2, ink) // the stem runs unbroken; leaves and flowers keep clear
		dist += math.Hypot(x-px, y-py)
		if dist >= every {
			dist = 0
			a := math.Atan2(y-py, x-px) + side*(0.9+rng.Float64()*0.4)
			count++
			switch {
			case flowers && count%7 == 0 && keep(x+math.Cos(a)*12, y+math.Sin(a)*12):
				fx, fy := x+math.Cos(a)*11, y+math.Sin(a)*11
				line(img, x, y, fx, fy, 1.4, ink)
				flower(img, fx, fy, 8+rng.Float64()*3, rng.Float64(), false)
			case count%5 == 0 && keep(x+math.Cos(a)*14, y+math.Sin(a)*14):
				bud(img, x, y, a, 7+rng.Float64()*3)
			case count%4 == 0 && keep(x+math.Cos(a)*10, y+math.Sin(a)*10):
				tendril(img, x, y, a, 18, side)
			default:
				l := leafMin + rng.Float64()*leafMin*0.6
				if keep(x+math.Cos(a)*l, y+math.Sin(a)*l) {
					leaf(img, x, y, a, l, l*0.48)
				}
			}
			side = -side
		}
		px, py = x, y
	}
}

// gardenStandby: the standby clock framed by a garden: vines up both sides, hanging from the
// top corners, two tall blooms at the bottom corners and grass along the bottom.
func gardenStandby(img *image.Gray) {
	rng := rand.New(rand.NewSource(7))
	free := func(x, y float64) bool { return true }
	w, h := float64(LW), float64(LH)
	leafMin = 15
	defer func() { leafMin = 12 }()
	// up the sides
	vine(img, rng, func(t float64) (float64, float64) { return 28 + 11*math.Sin(t*9), h - 40 - t*(h-90) }, 320, 20, true, free)
	vine(img, rng, func(t float64) (float64, float64) { return w - 28 + 11*math.Sin(t*8+1), h - 40 - t*(h-90) }, 320, 20, true, free)
	// hanging from the top corners, curving inwards
	vine(img, rng, func(t float64) (float64, float64) { return 28 + t*290, 26 + 34*math.Sin(t*math.Pi) }, 180, 19, true, free)
	vine(img, rng, func(t float64) (float64, float64) { return w - 28 - t*290, 26 + 34*math.Sin(t*math.Pi) }, 180, 19, true, free)
	butterfly(img, w-210, 112, 15)
	// grass along the bottom
	for x := 6.0; x < w-6; x += 5 + rng.Float64()*4 {
		hgt := 10 + rng.Float64()*26
		if math.Abs(x-w/2) < 90 { // short in the middle, clear of the page dots
			hgt = 4 + rng.Float64()*6
		}
		lean := (rng.Float64() - 0.5) * 14
		px, py := x, h-4
		for i := 1; i <= 6; i++ {
			t := float64(i) / 6
			nx, ny := x+lean*t*t, h-4-hgt*t
			line(img, px, py, nx, ny, 1.6-t, ink)
			px, py = nx, ny
		}
	}
	// two tall blooms on curving stems
	for _, side := range []float64{1, -1} {
		bx := w/2 + side*(w/2-70)
		stem := func(t float64) (float64, float64) { return bx - side*30*math.Sin(t*2.4), h - 6 - t*190 }
		px, py := stem(0)
		for i := 1; i <= 60; i++ {
			x, y := stem(float64(i) / 60)
			line(img, px, py, x, y, 2.4, ink)
			if i == 18 || i == 34 {
				leaf(img, x, y, -math.Pi/2+side*float64(map[bool]int{true: 1, false: -1}[i == 18])*1.1, 30, 14)
			}
			px, py = x, y
		}
		bigFlower(img, px, py, 38, side*0.4)
		bud(img, px+side*20, py+52, -math.Pi/2-side*0.7, 13)
		// a smaller bloom leaning in beside it
		sx, sy := px-side*58, py+92
		fx, fy := sx-side*10, sy-58
		line(img, sx, h-6, sx, sy, 2, ink)
		line(img, sx, sy, fx, fy, 2, ink)
		leaf(img, sx, sy+40, -math.Pi/2-side*0.9, 24, 11)
		bigFlower(img, fx, fy, 22, side*-0.3)
	}
}

// gardenTasks: on the Tasks screen, a vine climbing the line between the tasks and the side
// panel, and one along the bottom of the top bar (leaves kept off the text and buttons).
func gardenTasks(v *View, divX int) {
	img := v.Img
	rng := rand.New(rand.NewSource(3))
	busy := func(x, y float64) bool { // over a button or something to tap: keep off it
		p := image.Pt(int(x), int(y))
		for _, t := range v.Targets {
			if p.In(t.Show.Inset(-3)) {
				return false
			}
		}
		return true
	}
	// along the divider, up from the bottom (when there is one; else the top bar's vine starts mid-way)
	x0 := float64(divX)
	if divX > 0 {
		vine(img, rng, func(t float64) (float64, float64) {
			return x0 + 7*math.Sin(t*14), float64(LH) - 4 - t*float64(LH-barH-14)
		}, 300, 30, true,
			func(x, y float64) bool { return math.Abs(x-x0) < 22 && busy(x, y) })
	} else {
		x0 = float64(LW) * 0.62
	}
	// along the bottom of the top bar, from the divider out to both sides
	yb := float64(barH + 1)
	near := func(x, y float64) bool { return y > yb-12 && y < yb+12 && busy(x, y) }
	vine(img, rng, func(t float64) (float64, float64) { return x0 - t*(x0-8), yb + 3*math.Sin(t*11) }, 220, 34, false, near)
	vine(img, rng, func(t float64) (float64, float64) { return x0 + t*(float64(LW)-8-x0), yb + 3*math.Sin(t*7) }, 120, 34, false, near)
}
