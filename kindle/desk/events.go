package main

// Adding calendar events and changing the weather place on the Kindle, both without a
// computer: events are kept on the backend (they show here and on the phone page, not
// in the Mac's Calendar app); the place is looked up on the backend (Open-Meteo).
//
// Both start with the keyboard (keyboard.go): an event's name, then a second step for
// when it is; a place's name, then a list of matching places to pick from.

import (
	"encoding/json"
	"fmt"
	"image"
	"net/url"
	"strings"
	"time"
)

type Place struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
	Area string  `json:"area"`
}

// Places looks a town or city up on the backend: up to 5 matches.
func (c *Client) Places(q string) ([]Place, error) {
	resp, err := c.hc.Get(c.url("/api/weather/places?q=" + url.QueryEscape(q)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server said %s", resp.Status)
	}
	var ps []Place
	return ps, json.NewDecoder(resp.Body).Decode(&ps)
}

// openEventEditor starts a new event on day: its name first.
func openEventEditor(ui *UI, day, now time.Time) {
	h := 9
	if day.Format("2006-01-02") == now.Format("2006-01-02") {
		h = min(23, now.Hour()+1) // today: the next full hour
	}
	ui.Edit = &Editor{Mode: "event", Shift: true, Day: day, Hour: h, Length: 60}
	ui.Menu, ui.LightMenu = false, false
}

// eventTarget is the backend action for the event being made, plus the event itself
// (so it can show straight away).
func eventTarget(e *Editor) (Target, Event) {
	b := map[string]any{"title": strings.TrimSpace(e.Text), "date": e.Day.Format("2006-01-02"), "all_day": e.AllDay}
	ev := Event{Title: strings.TrimSpace(e.Text), AllDay: e.AllDay, Cal: "Added on the Kindle"}
	if e.AllDay {
		ev.Start = time.Date(e.Day.Year(), e.Day.Month(), e.Day.Day(), 0, 0, 0, 0, time.UTC).Unix()
		ev.End = ev.Start + 86400
	} else {
		b["time"], b["minutes"] = fmt.Sprintf("%02d:%02d", e.Hour, e.Min), e.Length
		ev.Start = time.Date(e.Day.Year(), e.Day.Month(), e.Day.Day(), e.Hour, e.Min, 0, 0, e.Day.Location()).Unix()
		ev.End = ev.Start + int64(e.Length)*60
	}
	j, _ := json.Marshal(b)
	return Target{Act: "addevent", Arg: string(j)}, ev
}

// eventTimeView: step two of a new event, when it is. The desk is already faded.
func eventTimeView(v *View, ui *UI) {
	img := v.Img
	e := ui.Edit
	card := image.Rect(150, barH+30, LW-150, LH-70)
	fillRoundRect(img, card.Inset(-3), 16, paper)
	strokeRoundRect(img, card, 14, 2, ink)
	x0, x1 := card.Min.X+28, card.Max.X-28
	y := card.Min.Y

	label(img, x0, y+34, "NEW EVENT · "+strings.ToUpper(e.Day.Format("Monday 2 January")), grey2)
	cancel := pillAt(img, image.Rect(x1-110, y+16, x1, y+58), "Cancel", false)
	v.add(Target{R: cancel.Inset(-6), Show: cancel, Act: "kbcancel"})

	// the name, with Rename to go back to the keyboard
	tf := face("serifB", 30)
	rf := face("uiBold", 16)
	rename := image.Rect(x1-textW(rf, "Rename")-32, y+76, x1, y+120)
	drawText(img, tf, x0, y+98+capHeight(tf)/2, ellipsize(tf, strings.TrimSpace(e.Text), rename.Min.X-16-x0), ink)
	pillAt(img, rename, "Rename", false)
	v.add(Target{R: rename.Inset(-6), Show: rename, Act: "evtitle"})
	y += 146

	allDay := 0
	if e.AllDay {
		allDay = 1
	}
	segmented(v, image.Rect(x0, y, x0+260, y+44), []string{"At a time", "All day"}, []string{"0", "1"}, allDay, "evallday")
	y += 76

	if e.AllDay {
		drawText(img, face("serif", 24), x0, y+20, "All day · "+e.Day.Format("Monday 2 January"), grey2)
	} else {
		label(img, x0, y, "STARTS", grey2)
		y += 16
		bf := face("uiBold", 17)
		btn := func(r image.Rectangle, s, arg string) {
			pillAt(img, r, "", false)
			drawCentered(img, bf, (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2+capHeight(bf)/2, s, ink)
			v.add(Target{R: r.Inset(-4), Show: r, Act: "evtime", Arg: arg})
		}
		const bw, bh, gap = 84, 56, 10
		btn(image.Rect(x0, y, x0+bw, y+bh), "−1 h", "-60")
		btn(image.Rect(x0+bw+gap, y, x0+2*bw+gap, y+bh), "−15", "-15")
		btn(image.Rect(x1-2*bw-gap, y, x1-bw-gap, y+bh), "+15", "15")
		btn(image.Rect(x1-bw, y, x1, y+bh), "+1 h", "60")
		big := face("serifB", 46)
		drawCentered(img, big, (x0+x1)/2, y+bh/2+capHeight(big)/2, fmt.Sprintf("%02d:%02d", e.Hour, e.Min), ink)
		y += bh + 34

		label(img, x0, y, "HOW LONG", grey2)
		y += 16
		lens := []int{15, 30, 45, 60, 90, 120, 180}
		names := []string{"15 m", "30 m", "45 m", "1 h", "1½ h", "2 h", "3 h"}
		args := make([]string, len(lens))
		sel := -1
		for i, l := range lens {
			args[i] = itoa(l)
			if l == e.Length {
				sel = i
			}
		}
		segmented(v, image.Rect(x0, y, x1, y+44), names, args, sel, "evlen")
		y += 84
		endM := e.Hour*60 + e.Min + e.Length
		ends := fmt.Sprintf("%02d:%02d", endM/60%24, endM%60)
		if endM >= 24*60 {
			ends += " the next day"
		}
		drawText(img, face("serif", 22), x0, y, fmt.Sprintf("%02d:%02d – %s", e.Hour, e.Min, ends), grey2)
	}

	add := image.Rect(x1-180, card.Max.Y-24-50, x1, card.Max.Y-24)
	pillAt(img, add, "Add event", true)
	v.add(Target{R: add.Inset(-6), Show: add, Act: "evadd"})
}

// placeResults: the matching places, in a card above the keyboard's text card.
func placeResults(v *View, e *Editor, card image.Rectangle) {
	img := v.Img
	if !e.Searching && e.SearchErr == "" && e.Places == nil {
		return
	}
	const rowH = 50
	n := max(1, min(3, len(e.Places)))
	r := image.Rect(card.Min.X, card.Min.Y-14-n*rowH-16, card.Max.X, card.Min.Y-14)
	fillRoundRect(img, r.Inset(-3), 16, paper)
	strokeRoundRect(img, r, 14, 2, ink)
	x0 := r.Min.X + 24
	msg := ""
	switch {
	case e.Searching:
		msg = "Looking for “" + strings.TrimSpace(e.Text) + "”…"
	case e.SearchErr != "":
		msg = "Couldn't search: " + e.SearchErr
	case len(e.Places) == 0:
		msg = "No place called “" + strings.TrimSpace(e.Text) + "”. Try another spelling."
	}
	if msg != "" {
		f := face("ui", 18)
		drawText(img, f, x0, r.Min.Y+8+rowH/2+capHeight(f)/2, ellipsize(f, msg, r.Dx()-48), grey2)
		return
	}
	nf, af := face("uiBold", 19), face("ui", 16)
	for i, p := range e.Places[:n] {
		row := image.Rect(r.Min.X+8, r.Min.Y+8+i*rowH, r.Max.X-8, r.Min.Y+8+(i+1)*rowH)
		if i > 0 {
			fillRect(img, image.Rect(x0, row.Min.Y, r.Max.X-24, row.Min.Y+1), grey4)
		}
		cy := (row.Min.Y + row.Max.Y) / 2
		nx := drawText(img, nf, x0, cy+capHeight(nf)/2, p.Name, ink)
		drawText(img, af, nx+12, cy+capHeight(af)/2, ellipsize(af, p.Area, r.Max.X-24-nx-12-60), grey2)
		chevron(img, float64(r.Max.X-36), float64(cy), 6, "right", ink)
		v.add(Target{R: row, Show: row, Act: "setplace", Arg: fmt.Sprintf("%g|%g|%s", p.Lat, p.Lon, p.Name)})
	}
}

// mapPinGlyph: a location pin (change the weather place).
func mapPinGlyph(img *image.Gray, cx, cy float64, c uint8) {
	fillPolygon(img, [][2]float64{{cx - 7.5, cy - 3}, {cx, cy + 11}, {cx + 7.5, cy - 3}}, c)
	fillCircle(img, cx, cy-5, 8, c)
	fillCircle(img, cx, cy-5, 3.2, paper^c^ink) // the hole: the background colour
}

// addGlyphButton: a 48×44 button without an outline, holding a plus.
func addGlyphButton(v *View, r image.Rectangle, act string) {
	plusGlyph(v.Img, float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2, 9, ink)
	v.add(Target{R: r.Inset(-4), Show: r, Act: act})
}
