package main

// Runs against a Desk Buddy server given in DB_TEST_SERVER with key DB_TEST_KEY (a throwaway
// local one): connect with a one-time code, check in as the PC, send the app list, and a Deck tap.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestFlow(t *testing.T) {
	server, key := os.Getenv("DB_TEST_SERVER"), os.Getenv("DB_TEST_KEY")
	if server == "" {
		t.Skip("set DB_TEST_SERVER and DB_TEST_KEY")
	}
	os.Setenv("APPDATA", t.TempDir())
	// the Mac app makes a code
	saveConfig(Config{Server: server, Key: key})
	var offer struct{ Code string }
	if err := call(quick, "POST", "/api/connect/offer", map[string]any{}, &offer); err != nil {
		t.Fatal("offer:", err)
	}
	saveConfig(Config{})
	// the PC connects with it through its page
	startPage()
	form := url.Values{"t": {token}, "do": {"connect"}, "server": {strings.TrimPrefix(server, "http://")}, "code": {strings.ToLower(offer.Code)}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // as a browser sends it
	page(rec, req)
	if rec.Code != http.StatusSeeOther || current().Key != key {
		t.Fatalf("connect: %d, key ok %v, body %.200s", rec.Code, current().Key == key, rec.Body.String())
	}
	// wrong token: refused
	rec = httptest.NewRecorder()
	page(rec, httptest.NewRequest("GET", "/?t=nope", nil))
	if rec.Code != 403 {
		t.Fatal("page without the token should be refused")
	}
	// a check-in as the PC
	var hb struct{ Profile string }
	if err := call(quick, "POST", "/api/heartbeat", map[string]any{"machine": "pc", "idle": 5.0, "dark": false}, &hb); err != nil || hb.Profile == "" {
		t.Fatal("heartbeat:", err, hb)
	}
	// the app list (an old helper's, then Windows' whole "All apps", which picks the PC's
	// first favourites from it, once, and leaves the Mac's alone)
	type tile struct{ Label, Win, App string }
	var deck, before struct{ Work, Home []tile }
	call(quick, "GET", "/api/deck", nil, &before)
	var r struct{ Seeded bool }
	if err := call(quick, "POST", "/api/deck/icons", map[string]any{"machine": "pc", "icons": map[string]string{}, "apps": []string{"Notepad", "Spotify"}}, &r); err != nil || r.Seeded {
		t.Fatal("apps:", err, r.Seeded)
	}
	list := map[string]any{"machine": "pc", "icons": map[string]string{}, "apps": []string{"Notepad", "Spotify", "Google Chrome", "Paint 3D", "Outlook (new)"}, "all_apps": true}
	if err := call(quick, "POST", "/api/deck/icons", list, &r); err != nil || !r.Seeded {
		t.Fatal("apps:", err, r.Seeded)
	}
	call(quick, "GET", "/api/deck", nil, &deck)
	want := []tile{{"Spotify", "Spotify", ""}, {"Chrome", "Google Chrome", ""}, {"Outlook", "Outlook (new)", ""}, {"Notepad", "Notepad", ""}}
	if fmt.Sprint(deck.Home) != fmt.Sprint(want) || fmt.Sprint(deck.Work) != fmt.Sprint(before.Work) {
		t.Fatalf("first favourites: %v (Mac's %d, was %d)", deck.Home, len(deck.Work), len(before.Work))
	}
	if err := call(quick, "POST", "/api/deck/icons", list, &r); err != nil || r.Seeded {
		t.Fatal("favourites should only be picked once:", err)
	}
	// a meeting link check
	if openItem(deckItem{URL: ptr("https://evil.example.com/x")}) == "" || openItem(deckItem{URL: ptr("https://zoom.us/j/123")}) != "" {
		t.Fatal("meeting links: only real meeting services")
	}
	t.Log("connected, checked in (list:", hb.Profile+"), app list sent, links checked")
}

func ptr(s string) *string { return &s }
