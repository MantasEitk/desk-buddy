package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// "On this PC" against a throwaway server (DB_TEST_SERVER / DB_TEST_KEY, started with --key):
// a Kindle asks to connect, the page shows it with its code, Allow, the Kindle gets the key.
func TestLocalAllow(t *testing.T) {
	server, key := os.Getenv("DB_TEST_SERVER"), os.Getenv("DB_TEST_KEY")
	if server == "" {
		t.Skip("set DB_TEST_SERVER and DB_TEST_KEY")
	}
	os.Setenv("APPDATA", t.TempDir())
	saveConfig(Config{Mode: "local", Server: server, Key: key})
	startPage()
	// a Kindle asks (no key)
	resp, err := http.Post(server+"/api/pair", "application/json", bytes.NewReader([]byte(`{"name":"Kindle"}`)))
	if err != nil {
		t.Fatal(err)
	}
	var ask struct{ ID, Code string }
	json.NewDecoder(resp.Body).Decode(&ask)
	go watchPairing()
	for i := 0; i < 30 && len(pairs()) == 0; i++ {
		time.Sleep(200 * time.Millisecond)
	}
	if len(pairs()) != 1 || pairs()[0].Code != ask.Code {
		t.Fatal("the request didn't show up", pairs())
	}
	rec := httptest.NewRecorder()
	page(rec, httptest.NewRequest("GET", "/?t="+token, nil))
	body := rec.Body.String()
	if !strings.Contains(body, "Kindle wants to connect") || !strings.Contains(body, ask.Code) || !strings.Contains(body, `class="qr"`) {
		t.Fatal("page doesn't show the request, its code and the QR")
	}
	form := url.Values{"t": {token}, "do": {"allow"}, "id": {ask.ID}}
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	page(httptest.NewRecorder(), req)
	r2, _ := http.Get(server + "/api/pair/" + ask.ID)
	var got struct{ State, Key string }
	json.NewDecoder(r2.Body).Decode(&got)
	if got.State != "ok" || got.Key != key {
		t.Fatal("the Kindle didn't get the key:", got.State)
	}
	t.Log("request shown with code", ask.Code, "+ QR; Allow; the Kindle got the key")
}
