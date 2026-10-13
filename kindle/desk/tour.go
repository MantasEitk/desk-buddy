package main

// The first-run tour: three pages over the real Tasks screen. Everything stays where it is,
// faded; what's being explained is lit up, with a line to a short label. Shown once on the
// first start (state.json "tour_done"), and again from Settings › About › Show the tour.

import (
	"image"
)

const tourPages = 3

// tourOverlay draws page ui.Tour (1–3) over the screen that's already drawn in v.
func tourOverlay(v *View, d *Desk, ui *UI) {
	img := v.Img
	orig := image.NewGray(img.Rect)
	copy(orig.Pix, img.Pix)
	targets := v.Targets                                    // the screen's tap areas, to find what to point at
	find := func(act, arg string) (image.Rectangle, bool) { // a tap area on screen, by its action
		for _, t := range targets {
			if t.Act == act && (arg == "" || t.Arg == arg) {
				return t.Show, true
			}
		}
		return image.Rectangle{}, false
	}
	var lit []image.Rectangle
	type note struct {
		title, text string
	}
	// a grey veil over everything; the parts being explained are copied back afterwards
	for i, p := range img.Pix {
		img.Pix[i] = uint8((int(p)*35 + 172*65) / 100) // white → light grey, black → mid grey
	}
	v.Targets = nil

	tf, xf := face("uiBold", 17), face("ui", 15)
	dot := func(p image.Point) { fillCircle(img, float64(p.X), float64(p.Y), 4.5, ink) }
	// labelAt draws a title + text block; align "left" (text starts at x) or "right" (ends at x)
	labelAt := func(x, y int, n note, align string) {
		tw := max(textW(tf, n.title), textW(xf, n.text))
		if align == "right" {
			x -= tw
		}
		bg := image.Rect(x-12, y-capHeight(tf)-10, x+tw+12, y+31) // a white bubble
		fillRoundRect(img, bg.Inset(-2), 10, ink)
		fillRoundRect(img, bg, 9, paper)
		drawText(img, tf, x, y, n.title, ink)
		drawText(img, xf, x, y+22, n.text, grey2)
	}

	switch ui.Tour {
	case 1: // the top bar
		// left group: lines go down then right to labels from x 500 (rows top to bottom
		// from the rightmost item, so no line crosses another)
		left := []struct {
			act, arg string
			n        note
		}{
			{"tab", "weather", note{"Weather", "Today, the next hours and the week"}},
			{"tab", "calendar", note{"Calendar", "Your month; add events with +"}},
			{"tab", "deck", note{"Deck", "Tiles that open apps on your computer"}},
			{"tab", "tasks", note{"Tasks", "Your tasks, with today's calendar beside it"}},
			{"menu", "", note{"Menu", "The computer to follow; feedback and Settings"}},
		}
		for i, it := range left {
			r, ok := find(it.act, it.arg)
			if !ok {
				continue
			}
			lit = append(lit, r)
			x, rowY := (r.Min.X+r.Max.X)/2, 126+i*58
			line(img, float64(x), float64(r.Max.Y+3), float64(x), float64(rowY), 2, ink)
			line(img, float64(x), float64(rowY), 462, float64(rowY), 2, ink)
			dot(image.Pt(x, r.Max.Y+3))
			labelAt(470, rowY+6, it.n, "left")
		}
		// right group: lines go down then left to labels ending at x 740 (rows from the
		// leftmost item down)
		right := []struct {
			act string
			n   note
		}{
			{"lightmenu", note{"Battery", "Tap: brightness, light or dark"}},
			{"volopen", note{"Volume", "Shows while music plays; tap to change it"}},
			{"refresh", note{"Refresh", "Redraw the screen and update now"}},
			{"dark", note{"Light / dark", "Swap black and white (Auto is in Settings)"}},
			{"standby", note{"Clock", "Tap: standby clock. Hold 3 s: leave the app"}},
		}
		i := 0
		for _, it := range right {
			r, ok := find(it.act, "")
			if !ok {
				continue // e.g. no volume while nothing plays
			}
			lit = append(lit, r)
			x, rowY := (r.Min.X+r.Max.X)/2, 414+i*56
			i++
			line(img, float64(x), float64(r.Max.Y+3), float64(x), float64(rowY), 2, ink)
			line(img, float64(x), float64(rowY), 748, float64(rowY), 2, ink)
			dot(image.Pt(x, r.Max.Y+3))
			labelAt(740, rowY+6, it.n, "right")
		}
	case 2: // the page
		callout := func(r image.Rectangle, box image.Point, n note, align string) {
			lit = append(lit, r)
			ex, ey := (r.Min.X+r.Max.X)/2, r.Max.Y+3
			line(img, float64(ex), float64(ey), float64(ex), float64(box.Y-24), 2, ink)
			dot(image.Pt(ex, ey))
			labelAt(box.X, box.Y, n, align)
		}
		if r, ok := find("addnew", "add"); ok {
			callout(r, image.Pt(r.Max.X, r.Max.Y+300), note{"Add a task", "Then type it on the keyboard"}, "right")
		}
		if r, ok := firstTask(targets); ok {
			callout(r, image.Pt(40, min(r.Max.Y+130, 560)), note{"A task", "Tap: open it. Hold: edit it. Circle: tick it off. Swipe left: delete"}, "left")
		}
		if r, ok := find("widget", "day"); ok {
			r2, _ := find("widget", "info")
			all := r.Union(r2)
			callout(all, image.Pt(690, 330), note{"Side panel", "Day, weather, music, info"}, "left")
		}
		if r, ok := find("sidehide", ""); ok {
			callout(r, image.Pt(r.Max.X, 230), note{"Hide it", "More room for tasks"}, "right")
		}
		labelAt(40, 630, note{"Swipes", "Up / down: more tasks. Three fingers sideways: the next tab. In Settings, swipe right to go back"}, "left")
	case 3: // your computer and Settings
		box := image.Rect(LW/2-360, barH+40, LW/2+360, barH+40+410)
		fillRoundRect(img, box.Inset(-3), 16, paper)
		strokeRoundRect(img, box, 14, 2, ink)
		x := box.Min.X + 36
		y := box.Min.Y + 52
		hf, bf, kf := face("serifB", 28), face("serif", 19), face("uiBold", 17)
		drawText(img, hf, x, y, "On your computer", ink)
		// TODO before release: a QR code to the helper download (helperURL) here.
		y += 30
		drawText(img, bf, x, y, "Install Desk Buddy on your computer: it connects the two.", ink)
		y += 12
		for _, it := range [][2]string{
			{"Calendar", "your computer's calendar shows here"},
			{"Music", "what's playing, and the play / skip buttons"},
			{"Deck", "tiles open apps on your computer"},
			{"Auto", "with more than one computer connected, it follows the one you use"},
		} {
			y += 30
			nx := drawText(img, kf, x+20, y, it[0], ink)
			drawText(img, bf, nx+10, y, it[1], grey2)
			fillCircle(img, float64(x+7), float64(y-capHeight(kf)/2), 3, ink)
		}
		y += 58
		drawText(img, hf, x, y, "Settings", ink)
		name := listName(d)
		if name == "" {
			name = "Tasks"
		}
		for _, l := range []string{
			"Tap “" + name + "” at the top left, then Settings. You can also",
			"change them from your computer, in Desk Buddy.",
			"To see this tour again: Settings › About.",
		} {
			y += 30
			drawText(img, bf, x, y, l, ink)
		}
	}
	// light up the parts being explained
	for _, r := range lit {
		r = r.Inset(-4).Intersect(image.Rect(0, 0, LW, LH))
		p := pxRect(img, r) // the pixels to bring back from the undimmed picture
		for yy := p.Min.Y; yy < p.Max.Y; yy++ {
			copy(img.Pix[yy*img.Stride+p.Min.X:yy*img.Stride+p.Max.X], orig.Pix[yy*orig.Stride+p.Min.X:yy*orig.Stride+p.Max.X])
		}
		strokeRoundRect(img, r.Inset(-2), 10, 2, ink)
	}

	// a white strip along the bottom: "Welcome", page dots, Skip, Next / Done
	fillRect(img, image.Rect(0, LH-72, LW, LH), paper)
	fillRect(img, image.Rect(0, LH-72, LW, LH-70), ink)
	by := LH - 60
	drawBuddy(img, pad+14, by+6, 34, ink)
	label(img, pad+38, by+30, "WELCOME TO DESK BUDDY", grey2)
	for i := 0; i < tourPages; i++ {
		cx := float64(pad + 38 + spacedW(face("uiBold", 13), "WELCOME TO DESK BUDDY", 2) + 24 + i*18) // after the label (it starts 38 in, past Desk Buddy)
		if i+1 == ui.Tour {
			fillCircle(img, cx, float64(by+25), 5, ink)
		} else {
			ringCircle(img, cx, float64(by+25), 5, 1.5, grey2)
		}
	}
	next := "Next"
	if ui.Tour == tourPages {
		next = "Done"
	}
	nr := pillAt(img, image.Rect(LW-pad-130, by, LW-pad, by+48), next, true)
	v.add(Target{R: nr.Inset(-6), Show: nr, Act: "tour", Arg: "next"})
	if ui.Tour < tourPages {
		sr := pillAt(img, image.Rect(nr.Min.X-12-110, by, nr.Min.X-12, by+48), "Skip", false)
		v.add(Target{R: sr.Inset(-6), Show: sr, Act: "tour", Arg: "skip"})
	}
}

// firstTask: the first task row in the list (below the "+" on the TODAY line), by its tap area.
func firstTask(targets []Target) (image.Rectangle, bool) {
	top := 0
	for _, t := range targets {
		if t.Act == "addnew" && t.Arg == "add" {
			top = t.Show.Max.Y
		}
	}
	for _, t := range targets {
		if t.Act == "select" && t.Show.Min.Y > top {
			return t.Show, true
		}
	}
	return image.Rectangle{}, false
}
