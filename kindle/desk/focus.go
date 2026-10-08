package main

// The focus screen: a whole screen for a Pomodoro. A big ring with the minutes left on
// the left, the task (and what's next) on the right, and the controls under it.

import (
	"fmt"
	"image"
	"math"
	"strings"
)

func focusView(v *View, d *Desk, st Status) {
	img := v.Img
	p := d.Pomodoro
	now := st.Now.Unix()

	// top bar: back, and the time
	bf := face("uiBold", 17)
	back := image.Rect(pad-8, 16, pad+170, 60)
	chevron(img, float64(back.Min.X+18), 38, 7, "left", ink)
	drawText(img, bf, back.Min.X+34, 38+capHeight(bf)/2, "Back to desk", ink)
	v.add(Target{R: back.Inset(-6), Show: back, Act: "focusview"})
	drawRight(img, face("serifB", 26), LW-pad, 38+capHeight(face("serifB", 26))/2, st.Now.Format("15:04"), ink)
	if st.Battery >= 0 {
		drawRight(img, face("ui", 15), LW-pad-90, 38+capHeight(face("ui", 15))/2, itoa(st.Battery)+"%", grey2)
	}

	// the ring
	cx, cy, r := 290.0, 400.0, 205.0
	ringCircle(img, cx, cy, r, 16, grey4)
	left := p.remaining(now)
	frac := 0.0
	if p.Length > 0 {
		frac = math.Max(0, math.Min(1, float64(p.Length-left)/float64(p.Length)))
	}
	arc(img, cx, cy, r-8, frac, 16, ink)
	mins := (left + 59) / 60
	big := itoa(int(mins))
	unit := "min left"
	if left < 60 {
		big, unit = "<1", "min left"
	}
	if mins == 1 && left >= 60 {
		unit = "min left"
	}
	nf := face("display", 150)
	drawCentered(img, nf, int(cx), int(cy)+52, big, ink)
	drawCentered(img, face("serif", 26), int(cx), int(cy)+100, unit, grey2)
	state := map[string]string{"focus": "FOCUS", "break": "BREAK", "paused": "PAUSED"}[p.State]
	lf := face("uiBold", 15)
	drawSpaced(img, lf, int(cx)-spacedW(lf, state, 3)/2, int(cy)-118, state, 3, grey2)

	// right: what you're on, and what's next
	x0, x1 := 560, LW-pad
	y := 150
	var cur, next *Task
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if t.ID == p.Task {
			cur = t
		}
	}
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if !t.Done && t.Parent == nil && (cur == nil || t.ID != cur.ID) {
			next = t
			break
		}
	}
	if p.State == "break" || (p.State == "paused" && p.Was == "break") {
		label(img, x0, y, "BREAK", grey2)
		f := face("serifB", 40)
		drawText(img, f, x0, y+52, "Step away", ink)
		drawText(img, face("serif", 22), x0, y+92, "Stretch, have some water,", grey2)
		drawText(img, face("serif", 22), x0, y+122, "look at something far away.", grey2)
		y += 170
		if cur != nil {
			label(img, x0, y, "UP NEXT", grey2)
			for i, l := range wrap(face("serifB", 26), cur.Text, x1-x0, 2) {
				drawText(img, face("serifB", 26), x0, y+40+i*32, l, ink)
			}
		}
	} else {
		label(img, x0, y, "WORKING ON", grey2)
		f := face("serifB", 34)
		title := "No task"
		if cur != nil {
			title = cur.Text
		} else if p.Title != "" {
			title = p.Title
		}
		lines := wrap(f, title, x1-x0, 2)
		for i, l := range lines {
			drawText(img, f, x0, y+46+i*40, l, ink)
		}
		y += 46 + (len(lines)-1)*40 + 10
		if cur != nil {
			var kids, done int
			for _, t := range d.Tasks {
				if t.Parent != nil && *t.Parent == cur.ID {
					kids++
					if t.Done {
						done++
					}
				}
			}
			if kids > 0 {
				drawText(img, face("ui", 17), x0, y+18, fmt.Sprintf("%d of %d subtasks done", done, kids), grey2)
				y += 28
			}
		}
	}

	// Up next: the following unfinished tasks; tap one to switch to it (the timer carries on).
	var upNext []Task
	for _, t := range d.Tasks {
		if !t.Done && t.Parent == nil && (cur == nil || t.ID != cur.ID) {
			upNext = append(upNext, t)
		}
	}
	if len(upNext) > 0 {
		y = max(y+30, 300)
		label(img, x0, y, "UP NEXT · TAP TO SWITCH", grey2)
		y += 14
		rf := face("serif", 21)
		for k, t := range upNext {
			if k == 4 || y+50 > 540 {
				break
			}
			row := image.Rect(x0-10, y, x1+6, y+50)
			ringCircle(img, float64(x0+11), float64(y+25), 10, 2, grey3)
			drawText(img, rf, x0+34, y+25+capHeight(rf)/2, ellipsize(rf, t.Text, x1-x0-44), ink)
			fillRect(img, image.Rect(x0+34, y+49, x1, y+50), grey4)
			v.add(Target{R: row, Show: row, Act: "pomo", Arg: "switch", ID: t.ID})
			y += 50
		}
	}
	_ = next
	_ = strings.ToUpper

	// controls
	by := 560
	type btn struct {
		label, act, arg string
		id              int
		solid           bool
	}
	var btns []btn
	if cur != nil && p.State != "break" {
		btns = append(btns, btn{"Done", "toggle", "", cur.ID, true})
	}
	if p.State == "paused" {
		btns = append(btns, btn{"Resume", "pomo", "resume", 0, cur == nil})
	} else {
		btns = append(btns, btn{"Pause", "pomo", "pause", 0, false})
	}
	btns = append(btns, btn{"Stop", "pomo", "stop", 0, false})
	bx := x0
	for _, b := range btns {
		w := textW(face("uiBold", 16), b.label) + 52
		rr := pillAt(img, image.Rect(bx, by, bx+w, by+56), b.label, b.solid)
		v.add(Target{R: rr.Inset(-5), Show: rr, Act: b.act, ID: b.id, Arg: b.arg})
		bx = rr.Max.X + 12
	}
	// sound
	sr := image.Rect(x0, by+80, x0+190, by+124)
	strokeRoundRect(img, sr, 22, 2, ink)
	speakerGlyph(img, float64(sr.Min.X+28), float64(sr.Min.Y+sr.Max.Y)/2, p.Sound, ink)
	sl := "Sound off"
	if p.Sound {
		sl = "Sound on"
	}
	sf := face("uiBold", 16)
	drawText(img, sf, sr.Min.X+50, (sr.Min.Y+sr.Max.Y)/2+capHeight(sf)/2, sl, ink)
	arg := "sound:1"
	if p.Sound {
		arg = "sound:0"
	}
	v.add(Target{R: sr.Inset(-4), Show: sr, Act: "pomo", Arg: arg})
}

// arc draws the part of a ring from 12 o'clock, clockwise, for frac (0–1) of the way round.
func arc(img *image.Gray, cx, cy, r, frac, w float64, c uint8) {
	if frac <= 0 {
		return
	}
	steps := int(120 * frac)
	px, py := cx, cy-r
	for i := 1; i <= max(steps, 1); i++ {
		a := -math.Pi/2 + 2*math.Pi*frac*float64(i)/float64(max(steps, 1))
		x, y := cx+r*math.Cos(a), cy+r*math.Sin(a)
		line(img, px, py, x, y, w, c)
		px, py = x, y
	}
}
