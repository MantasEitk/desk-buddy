package main

// The window: a small page served only to this PC (127.0.0.1) and opened in the browser, with
// a random token in its address so other websites can't use it. Connect (server + the Mac's
// one-time code), status, Pause, Start with Windows, Disconnect.

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
)

//go:embed buddy.png
var buddyPNG []byte

var (
	pageURL string
	token   string
)

func startPage() {
	b := make([]byte, 16)
	rand.Read(b)
	token = hex.EncodeToString(b)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		report("page: " + err.Error())
		return
	}
	pageURL = fmt.Sprintf("http://%s/?t=%s", ln.Addr(), token)
	mux := http.NewServeMux()
	mux.HandleFunc("/", page)
	go http.Serve(ln, mux)
}

func openPage() {
	if pageURL != "" {
		shellOpen(pageURL)
	}
}

func page(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.FormValue("t") != token {
		http.Error(w, "Open Desk Buddy from its icon in the taskbar.", 403)
		return
	}
	note := ""
	if r.Method == "POST" {
		switch r.FormValue("do") {
		case "connect":
			server := strings.TrimSpace(r.FormValue("server"))
			if server != "" && !strings.HasPrefix(server, "http") {
				if strings.Contains(server, ".workers.dev") || !strings.Contains(server, ":") {
					server = "https://" + server
				} else {
					server = "http://" + server
				}
			}
			server = strings.TrimRight(server, "/")
			key, err := claim(server, r.FormValue("code"))
			if err != nil {
				note = err.Error()
				break
			}
			c := current()
			c.Server, c.Key, c.Paused = server, key, false
			if err := saveConfig(c); err != nil {
				note = "Couldn't save the settings: " + err.Error()
				break
			}
			setStatus("Connected")
		case "pause":
			setPaused(!current().Paused)
		case "login":
			setStartWithWindows(!startsWithWindows())
		case "disconnect":
			saveConfig(Config{})
			setStatus("Not connected")
		}
		if note == "" {
			http.Redirect(w, r, "/?t="+token, http.StatusSeeOther)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, pageHTML(note, r.FormValue("server")))
}

func pageHTML(note, server string) string {
	c := current()
	st, last := getStatus()
	esc := html.EscapeString
	buddy := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buddyPNG)
	form := func(do, label string, solid bool) string {
		cls := "pill"
		if solid {
			cls += " solid"
		}
		return `<form method="post"><input type="hidden" name="t" value="` + token + `"><input type="hidden" name="do" value="` + do + `"><button class="` + cls + `">` + label + `</button></form>`
	}
	var body string
	if !connected() {
		if server == "" {
			server = c.Server
		}
		body = `<h2>Connect this PC</h2>
<ol><li>On your Mac, open <b>Desk Buddy</b> › <b>This Mac</b> › <b>Connect a computer…</b></li>
<li>Type the server and the code it shows:</li></ol>
<form method="post" class="connect"><input type="hidden" name="t" value="` + token + `"><input type="hidden" name="do" value="connect">
<label>Server<input name="server" value="` + esc(server) + `" placeholder="https://….workers.dev" required></label>
<label>Code<input name="code" placeholder="ABCD-1234" required autocomplete="off"></label>
<button class="pill solid">Connect</button></form>`
		if note != "" {
			body += `<p class="note">` + esc(note) + `</p>`
		}
	} else {
		pause, login := "Pause", "Off"
		if c.Paused {
			pause = "Resume"
		}
		if startsWithWindows() {
			login = "On"
		}
		lastRow := ""
		if last != "" {
			lastRow = `<div class="row"><div><b>Last from the Kindle</b><span>` + esc(last) + `</span></div></div>`
		}
		body = `<p class="status">` + esc(st) + `</p>
<div class="row"><div><b>Server</b><span>` + esc(c.Server) + `</span></div>` + form("disconnect", "Disconnect", false) + `</div>
<div class="row"><div><b>Desk Buddy</b><span>Tells the Kindle you're on this PC, opens apps from the Deck, chimes when a focus ends</span></div>` + form("pause", pause, false) + `</div>
<div class="row"><div><b>Start with Windows</b><span>` + login + `</span></div>` + form("login", map[string]string{"On": "Turn off", "Off": "Turn on"}[login], false) + `</div>` + lastRow + `
<p class="small">Coming next: music, volume and calendar. Close this tab any time: Desk Buddy keeps running in the taskbar (bottom right).</p>`
	}
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Desk Buddy</title>` + refresh(connected()) + `<style>
:root{--ink:#111;--paper:#fff;--grey:#666;--line:#ddd}
@media (prefers-color-scheme:dark){:root{--ink:#eee;--paper:#161616;--grey:#aaa;--line:#333}.buddy{filter:invert(1)}}
body{margin:0;background:var(--paper);color:var(--ink);font:15px/1.45 "Segoe UI",system-ui,sans-serif}
main{max-width:620px;margin:40px auto;padding:0 24px}
header{display:flex;align-items:center;gap:16px;margin-bottom:24px}.buddy{width:56px}
h1{font-size:26px;margin:0}h1 small{display:block;font-size:13px;font-weight:400;color:var(--grey)}
h2{font-size:18px;margin:8px 0}ol{padding-left:20px;color:var(--grey)}ol b{color:var(--ink)}
.connect{display:grid;gap:12px;margin-top:12px}label{display:grid;gap:4px;font-weight:600;font-size:13px}
input{font:inherit;padding:8px 2px;border:0;border-bottom:2px solid var(--ink);background:none;color:var(--ink);outline:none}
.pill{font:600 14px "Segoe UI",system-ui,sans-serif;padding:8px 18px;border:1.5px solid var(--ink);border-radius:8px;background:var(--paper);color:var(--ink);cursor:pointer}
.pill.solid{background:var(--ink);color:var(--paper)}.connect .pill{justify-self:start;margin-top:6px}
.row{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:14px 0;border-bottom:1px solid var(--line)}
.row b{display:block}.row span{color:var(--grey);font-size:13px}form{margin:0}
.status{font-weight:700;font-size:17px}.note{font-weight:700}.small{color:var(--grey);font-size:12px;margin-top:20px}
</style></head><body><main><header><img class="buddy" src="` + buddy + `" alt=""><h1>Desk Buddy<small>for Windows · ` + version + `</small></h1></header>` + body + `</main></body></html>`
}

func refresh(on bool) string {
	if on {
		return `<meta http-equiv="refresh" content="15">`
	}
	return ""
}
