package main

// The background work, like the Mac's helper.py: tell the server when this PC is being used,
// wait for Deck taps from the Kindle (open the app), keep the app list current, and chime
// when a focus or break ends.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const inUse = 240 // seconds without input before the PC counts as not in use (the server allows 300)

// heartbeats: says when the PC starts or stops being used, or light / dark changes, and every
// 2 minutes otherwise (the server forgets a computer after 3). Looking is local, every 10 s.
func heartbeats() {
	var last time.Time
	var wasUsed, wasDark, sent bool
	for {
		c := current()
		if connected() && !c.Paused {
			idle := idleSeconds()
			dark, haveDark := darkMode()
			used := idle < inUse
			if !sent || time.Since(last) >= 2*time.Minute || used != wasUsed || dark != wasDark {
				body := map[string]any{"machine": "pc", "idle": idle}
				if haveDark {
					body["dark"] = dark
				}
				var r struct{ Profile string }
				if err := call(quick, "POST", "/api/heartbeat", body, &r); err != nil {
					setStatus("Can't reach the server, retrying")
					last = time.Now().Add(-90 * time.Second) // try again in 30 s
				} else {
					setStatus(map[string]string{"home": "Running · the Kindle shows the Windows list", "work": "Running · the Kindle shows the Mac list"}[r.Profile])
					last, sent, wasUsed, wasDark = time.Now(), true, used, dark
				}
			}
		} else {
			sent = false
		}
		time.Sleep(10 * time.Second)
	}
}

// ---- the Deck ----

type deckItem struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	Win      string    `json:"win"`
	URL      *string   `json:"url"`
	Media    string    `json:"media"`
	Switch   string    `json:"switch"`
	Pomodoro *pomoInfo `json:"pomodoro"`
	Wake     bool      `json:"wake"`
}

type pomoInfo struct {
	State string  `json:"state"`
	Ends  float64 `json:"ends"`
	Sound bool    `json:"sound"`
	Now   float64 `json:"now"`
}

// deck: one request at a time waits (up to 20 s) at the server for a tap meant for this PC.
func deck() {
	for {
		c := current()
		if !connected() || c.Paused {
			time.Sleep(3 * time.Second)
			continue
		}
		var it deckItem
		if err := call(long, "GET", "/api/deck/next?machine=pc&wait=20", nil, &it); err != nil {
			time.Sleep(5 * time.Second) // offline or an old server: don't spin
			continue
		}
		scheduleChime(it.Pomodoro)
		if it.ID == "" || it.Wake || it.Switch != "" {
			continue
		}
		var err string
		switch {
		case it.Media != "":
			err = "music controls aren't on Windows yet"
		default:
			err = openItem(it)
		}
		setLast(it.Label, err)
		call(quick, "POST", "/api/deck/done", map[string]any{"id": it.ID, "ok": err == "", "error": err}, nil)
	}
}

// Only meeting links from these services open (the Kindle's Join), never anything else.
var meeting = regexp.MustCompile(`(?i)^https://(?:[a-z0-9-]+\.)*(?:zoom\.us|meet\.google\.com|teams\.microsoft\.com|teams\.live\.com|webex\.com|whereby\.com|meet\.jit\.si)(?:/[^\s"'<>()]*)?$`)

// openItem opens a Deck tile: an app from the Start menu by its name, or a meeting link.
// Never runs commands.
func openItem(it deckItem) string {
	if it.URL != nil {
		if !meeting.MatchString(*it.URL) {
			return "not a meeting link"
		}
		if e := shellOpen(*it.URL); e != nil {
			return "couldn't open it"
		}
		return ""
	}
	name := it.Win
	if name == "" {
		name = it.Label
	}
	path, ok := findApp(name)
	if !ok {
		return "no app called " + name + " in the Start menu"
	}
	if e := shellOpen(path); e != nil {
		return "couldn't open " + name
	}
	return ""
}

// ---- the apps (Start menu shortcuts), for the Kindle's "All apps" ----

var (
	appsMu sync.Mutex
	apps   = map[string]string{} // name → shortcut
)

var safeName = regexp.MustCompile(`^[\w .&'+()-]{1,60}$`)

func scanApps() map[string]string {
	found := map[string]string{}
	for _, dir := range []string{
		filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs`),
		filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`),
	} {
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".lnk") {
				return nil
			}
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			low := strings.ToLower(name)
			if !safeName.MatchString(name) || strings.Contains(low, "uninstall") || strings.Contains(low, "readme") || strings.Contains(low, "help") {
				return nil
			}
			if _, dup := found[name]; !dup {
				found[name] = p
			}
			return nil
		})
	}
	return found
}

func findApp(name string) (string, bool) {
	appsMu.Lock()
	defer appsMu.Unlock()
	if p, ok := apps[name]; ok {
		return p, true
	}
	for n, p := range apps { // "chrome" for "Google Chrome"
		if strings.EqualFold(n, name) {
			return p, true
		}
	}
	return "", false
}

// appList sends the app names when they change (looked at every 10 minutes), and every 6 hours.
func appList() {
	var sent string
	var at time.Time
	for {
		if connected() && !current().Paused {
			found := scanApps()
			appsMu.Lock()
			apps = found
			appsMu.Unlock()
			names := make([]string, 0, len(found))
			for n := range found {
				names = append(names, n)
			}
			sort.Strings(names)
			sig := current().Server + "|" + strings.Join(names, "|")
			if sig != sent || time.Since(at) > 6*time.Hour {
				if err := call(quick, "POST", "/api/deck/icons", map[string]any{"machine": "pc", "icons": map[string]string{}, "apps": names}, nil); err == nil {
					sent, at = sig, time.Now()
				}
			}
		}
		time.Sleep(10 * time.Minute)
	}
}

// ---- the chime at the end of a focus or break (on the server's clock) ----

var (
	chimeMu    sync.Mutex
	chimeEnds  float64
	chimeTimer *time.Timer
)

func scheduleChime(p *pomoInfo) {
	ends := 0.0
	if p != nil && p.Sound && (p.State == "focus" || p.State == "break") {
		ends = p.Ends
	}
	chimeMu.Lock()
	defer chimeMu.Unlock()
	if ends == chimeEnds {
		return
	}
	if chimeTimer != nil {
		chimeTimer.Stop()
		chimeTimer = nil
	}
	chimeEnds = ends
	if ends > 0 {
		delay := ends - p.Now
		if delay > -30 { // just ended and we only now heard: still chime
			chimeTimer = time.AfterFunc(time.Duration(max(0, delay)*float64(time.Second)), chimeSound)
		}
	}
}
