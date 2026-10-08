package main

// Working without the server (e.g. local mode while the Mac sleeps):
//
//   - The last desk is kept in last-desk.json, so the Kindle starts on the last-known
//     screen instead of "Connecting…".
//   - Task changes that couldn't be sent wait in outbox.json and go out, in order, once the
//     server answers again. Until then they're applied on top of whatever the Kindle shows.
//     Only task and setting changes wait: opening apps, music, meetings and the Pomodoro are
//     about "now" and are dropped (the Deck incident: late taps must never fire).
//   - In local mode, when the computer can't be reached, the Kindle looks on the Wi-Fi for
//     it again (its address may have changed) and follows it.

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// canWait: changes that may be sent later.
func canWait(act string) bool {
	switch act {
	case "toggle", "pin", "unpin", "move", "addtask", "edittask", "deltask", "cleartasks", "setting":
		return true
	}
	return false
}

// netError: the request never got an answer (as opposed to the server saying no).
func netError(err error) bool {
	var ne net.Error
	var ue *url.Error
	return errors.As(err, &ne) || errors.As(err, &ue)
}

type Outbox struct {
	Items    []Target `json:"items"`
	NextTemp int      `json:"next_temp"` // tasks added offline get ids -1, -2, … until sent
	path     string
}

func loadOutbox(dir string) *Outbox {
	o := &Outbox{path: filepath.Join(dir, "outbox.json")}
	if b, err := os.ReadFile(o.path); err == nil {
		json.Unmarshal(b, o)
	}
	return o
}

func (o *Outbox) save() {
	if o.path == "" {
		return
	}
	if len(o.Items) == 0 {
		os.Remove(o.path)
		o.NextTemp = 0
		return
	}
	b, _ := json.Marshal(o)
	if os.WriteFile(o.path+".new", b, 0o644) == nil {
		os.Rename(o.path+".new", o.path)
	}
}

// add queues a change; a new task gets a temporary id so it can be ticked or edited offline.
func (o *Outbox) add(t Target) Target {
	if t.Act == "addtask" {
		o.NextTemp--
		t.Temp = o.NextTemp
	}
	o.Items = append(o.Items, t)
	o.save()
	return t
}

// applyAll shows the waiting changes on a desk fresh from the server (or the cache).
func (o *Outbox) applyAll(d *Desk) {
	for _, t := range o.Items {
		apply(d, t)
	}
}

// flush sends the waiting changes in order. It stops at the first one that gets no answer
// (still offline) and drops ones the server refuses (e.g. a task deleted elsewhere).
// It returns how many items were dealt with (the caller removes them) and the ids the
// server gave tasks added offline (temporary id → real), for the items still waiting.
func (o *Outbox) flush(c *Client, items []Target) (done int, ids map[int]int, err error) {
	ids = map[int]int{}
	real := func(id int) int {
		if n, ok := ids[id]; ok {
			return n
		}
		return id
	}
	for _, t := range items {
		t.ID = real(t.ID)
		if t.ID < 0 { // a change to a task whose adding was refused: nothing to send
			done++
			continue
		}
		var serr error
		if t.Act == "addtask" {
			var id int
			id, serr = c.addTask(t)
			if serr == nil && t.Temp != 0 {
				ids[t.Temp] = id
			}
		} else {
			serr = c.send(t)
		}
		if serr != nil && netError(serr) {
			return done, ids, serr
		}
		done++
	}
	return done, ids, nil
}

// ---------- the last desk, kept for the next start ----------

type lastDesk struct {
	path  string
	saved []byte
	at    time.Time
}

func (l *lastDesk) load() (*Desk, time.Time) {
	b, err := os.ReadFile(l.path)
	if err != nil {
		return nil, time.Time{}
	}
	var d Desk
	if json.Unmarshal(b, &d) != nil {
		return nil, time.Time{}
	}
	fi, _ := os.Stat(l.path)
	l.saved = b
	return &d, fi.ModTime()
}

// keep writes the desk when it changed, at most once a minute (spare the Kindle's storage).
func (l *lastDesk) keep(d *Desk) {
	if d == nil || time.Since(l.at) < time.Minute {
		return
	}
	c := *d
	c.Now, c.Music = 0, nil // the time and the song change constantly; not worth a write
	b, err := json.Marshal(&c)
	if err != nil || string(b) == string(l.saved) {
		return
	}
	if os.WriteFile(l.path+".new", b, 0o644) == nil && os.Rename(l.path+".new", l.path) == nil {
		l.saved, l.at = b, time.Now()
	}
}

// refind looks on the Wi-Fi for the computer we were using (by name; or the only one that
// answers) and returns its address if it moved and our key works there.
func refind(c *Client, name, current string) (string, string, bool) {
	found := findServers()
	var pick *FoundServer
	for i := range found {
		if found[i].Name == name {
			pick = &found[i]
		}
	}
	if pick == nil && len(found) == 1 {
		pick = &found[0]
	}
	if pick == nil || pick.Addr == current || tryServer(c, pick.Addr) != nil {
		return "", "", false
	}
	return pick.Name, pick.Addr, true
}
