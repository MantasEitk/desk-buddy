package main

// Talking to the Desk Buddy server: the key goes as an X-Key header (never in the address), and
// every request has a User-Agent (Cloudflare blocks some default ones).

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	quick = &http.Client{Timeout: 20 * time.Second}
	long  = &http.Client{Timeout: 45 * time.Second} // the Deck's waiting request (up to 20 s)
)

const userAgent = "DeskBuddy-windows/" + version

func call(hc *http.Client, method, path string, body, out any) error {
	c := current()
	if c.Server == "" {
		return fmt.Errorf("not connected")
	}
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.Server, "/")+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("X-Key", c.Key)
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return errKeyRefused
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("server said %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// errKeyRefused: the server is there but doesn't take this PC's key any more (Desk Buddy on
// the other computer was reinstalled, or its key changed): this PC has to join again.
var errKeyRefused = errors.New("the server doesn't know this PC any more")

// claim trades the Mac app's one-time code for the key (no key needed for this one).
func claim(server, code string) (string, error) {
	b, _ := json.Marshal(map[string]string{"code": code})
	req, err := http.NewRequest("POST", strings.TrimRight(server, "/")+"/api/connect", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := quick.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't reach that server: check the address")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return "", fmt.Errorf("%s", strings.TrimSpace(string(msg)))
	}
	var r struct{ Key string }
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || r.Key == "" {
		return "", fmt.Errorf("that doesn't look like a Desk Buddy server")
	}
	return r.Key, nil
}

// report sends an error to the server's helper log (Desk Buddy on the Mac can show it), once
// per message per hour so a broken loop doesn't flood it.
var (
	reportMu sync.Mutex
	reported = map[string]time.Time{}
)

func report(text string) {
	logf("%s", text)
	reportMu.Lock()
	t, ok := reported[text]
	if !ok || time.Since(t) >= time.Hour {
		reported[text] = time.Now()
	}
	reportMu.Unlock()
	if ok && time.Since(t) < time.Hour {
		return
	}
	go call(quick, "POST", "/api/helper/log", map[string]string{"machine": "pc", "text": text}, nil)
}
