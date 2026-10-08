package main

// Runs against a Desk Buddy server given in DB_TEST_SERVER with key DB_TEST_KEY (a throwaway
// local one): connect with a one-time code, check in as the PC, send the app list, and a Deck tap.

import (
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
	// the app list
	if err := call(quick, "POST", "/api/deck/icons", map[string]any{"machine": "pc", "icons": map[string]string{}, "apps": []string{"Notepad", "Spotify"}}, nil); err != nil {
		t.Fatal("apps:", err)
	}
	// a meeting link check
	if openItem(deckItem{URL: ptr("https://evil.example.com/x")}) == "" || openItem(deckItem{URL: ptr("https://zoom.us/j/123")}) != "" {
		t.Fatal("meeting links: only real meeting services")
	}
	t.Log("connected, checked in (list:", hb.Profile+"), app list sent, links checked")
}

func ptr(s string) *string { return &s }
