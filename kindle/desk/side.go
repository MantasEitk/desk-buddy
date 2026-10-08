package main

// The Tasks tab's right column: calendar (now / up next), Claude, today.

import (
	"fmt"
	"image"
	"time"
)

func sideColumn(v *View, d *Desk, st Status) {
	img := v.Img
	x0, x1 := sideX+22, LW-22
	w := x1 - x0
	y := barH + 2 + 14

	y = agendaCard(img, d, st, x0, y, w)

	c := d.Claude
	ago := ""
	if c != nil {
		ago = agoText(st.Now.Sub(time.Unix(int64(c.At), 0)))
	}
	switch {
	case c == nil || c.State == "idle":
		y = card(img, x0, y, w, "CLAUDE", "No sessions running", "", false)
	case c.State == "input":
		y = card(img, x0, y, w, "CLAUDE · NEEDS YOU", "Waiting for your input", c.Text+" · "+ago, true)
	case c.State == "working":
		y = card(img, x0, y, w, "CLAUDE", "Working: "+c.Text, "started "+ago, false)
	default:
		y = card(img, x0, y, w, "CLAUDE", "Finished: "+c.Text, ago, false)
	}

	open := "Nothing open in Work or Home"
	if d.OpenTotal == 1 {
		open = "1 task open across Work and Home"
	} else if d.OpenTotal > 1 {
		open = fmt.Sprintf("%d tasks open across Work and Home", d.OpenTotal)
	}
	if y < LH-140 {
		card(img, x0, y, w, "TODAY", st.Now.Format("Monday 2 January"), open, false)
	}
	drawText(img, face("ui", 13), x0, LH-22, "Hold the clock 3 s, or press power, to leave", grey3)
}

// agendaCard: what's on now, what's next and when, then a couple more.
func agendaCard(img *image.Gray, d *Desk, st Status, x, y, w int) int {
	if d.Calendars == 0 && len(d.Agenda) == 0 {
		return card(img, x, y, w, "CALENDAR", "No calendar yet", "Add one on your phone page", false)
	}
	now := st.Now.Unix()
	today := st.Now.Format("2006-01-02")
	var cur *Event
	var next []Event
	var rows []string
	for i, e := range d.Agenda {
		switch {
		case e.AllDay:
			if time.Unix(e.Start, 0).UTC().Format("2006-01-02") == today {
				rows = append(rows, "All day · "+e.Title)
			}
		case e.Start <= now && e.End > now && cur == nil:
			cur = &d.Agenda[i]
		case e.Start > now:
			next = append(next, e)
		}
	}
	if cur != nil {
		y = card(img, x, y, w, "NOW · UNTIL "+hm(cur.End, st), cur.Title, cur.Location, false)
	}
	if len(next) == 0 {
		big := "Nothing else today"
		if cur == nil && len(rows) == 0 {
			big = "Nothing on today"
		}
		return card(img, x, y, w, "CALENDAR", big, "", false, rows...)
	}
	e := next[0]
	small := relative(e.Start, st)
	if e.Location != "" {
		small += " · " + e.Location
	}
	for _, n := range next[1:min(len(next), 3)] {
		rows = append(rows, dayPrefix(n.Start, st)+hm(n.Start, st)+"  "+n.Title)
	}
	return card(img, x, y, w, "NEXT UP", dayPrefix(e.Start, st)+hm(e.Start, st)+" "+e.Title, small, false, rows...)
}

func hm(t int64, st Status) string { return time.Unix(t, 0).In(st.Now.Location()).Format("15:04") }

func dayPrefix(t int64, st Status) string {
	if time.Unix(t, 0).In(st.Now.Location()).Format("2006-01-02") == st.Now.Format("2006-01-02") {
		return ""
	}
	return "Tomorrow "
}

// relative: "in 25 min", "in 1 h 10 min", or "Tomorrow".
func relative(t int64, st Status) string {
	if dayPrefix(t, st) != "" {
		return "Tomorrow"
	}
	m := int((t - st.Now.Unix() + 59) / 60)
	switch {
	case m < 60:
		return fmt.Sprintf("in %d min", m)
	case m%60 == 0:
		return fmt.Sprintf("in %d h", m/60)
	default:
		return fmt.Sprintf("in %d h %d min", m/60, m%60)
	}
}

func agoText(d time.Duration) string {
	m := int(d.Minutes())
	switch {
	case m < 1:
		return "just now"
	case m < 60:
		return fmt.Sprintf("%d min ago", m)
	default:
		return fmt.Sprintf("%d h %d min ago", m/60, m%60)
	}
}

// card draws a side-column card and returns the y below it. rows are extra small lines.
func card(img *image.Gray, x, y, w int, lbl, big, small string, alert bool, rows ...string) int {
	bf, sf, rf := face("serifB", 21), face("ui", 14), face("ui", 15)
	lines := wrap(bf, big, w-28, 3)
	bottom := y + 54 + 26*(len(lines)-1) // baseline of the last big line
	smallY := bottom + 24
	if small != "" {
		bottom = smallY
	}
	rowsY := bottom + 26
	if len(rows) > 0 {
		bottom = rowsY + 24*(len(rows)-1)
	}
	h := bottom - y + 18
	lc, bc, sc := grey2, ink, grey2
	if alert {
		fillRoundRect(img, image.Rect(x-14, y, x+w+6, y+h), 9, ink)
		lc, bc, sc = grey4, paper, grey4
	}
	label(img, x, y+24, lbl, lc)
	for i, l := range lines {
		drawText(img, bf, x, y+54+26*i, l, bc)
	}
	if small != "" {
		drawText(img, sf, x, smallY, ellipsize(sf, small, w-8), sc)
	}
	for i, r := range rows {
		drawText(img, rf, x, rowsY+24*i, ellipsize(rf, r, w-8), bc)
	}
	if alert {
		return y + h + 12
	}
	fillRect(img, image.Rect(x, y+h-2, x+w, y+h), grey4)
	return y + h + 6
}
