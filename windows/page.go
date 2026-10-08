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
	"github.com/skip2/go-qrcode"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	// for a second copy started later: it asks this one to show its window (see main)
	os.WriteFile(filepath.Join(configDir(), "running.url"), []byte(pageURL), 0o600)
	mux := http.NewServeMux()
	mux.HandleFunc("/show", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == token {
			openWindow()
		}
	})
	mux.HandleFunc("/", page)
	uiRoutes(mux)
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
			c.Mode, c.Server, c.Key, c.Paused = "remote", server, key, false
			if err := saveConfig(c); err != nil {
				note = "Couldn't save the settings: " + err.Error()
				break
			}
			setStatus("Connected")
		case "local": // run Desk Buddy on this PC
			c := current()
			if c.Mode != "local" || c.Key == "" {
				c.Key = newKey()
			}
			c.Mode, c.Server, c.Paused = "local", "http://127.0.0.1:8787", false
			if err := saveConfig(c); err != nil {
				note = "Couldn't save the settings: " + err.Error()
				break
			}
			stop = false
			setStatus("Starting Desk Buddy on this PC…")
		case "allow", "deny":
			answerPair(r.FormValue("id"), r.FormValue("do") == "allow")
		case "pause":
			setPaused(!current().Paused)
		case "login":
			setStartWithWindows(!startsWithWindows())
		case "disconnect":
			if local() {
				stopServer() // the data stays in %APPDATA%\DeskBuddy\server
			}
			saveConfig(Config{})
			setStatus("Not set up")
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
		body = `<h2>Set up Desk Buddy</h2>
<div class="choice"><div><b>Run Desk Buddy on this PC</b><span>If this is your only computer, or the first one you set up. Nothing else to set up: your Kindle finds this PC on the Wi-Fi.</span></div>` + form("local", "Set up here", true) + `</div>
<h2 style="margin-top:28px">Or join Desk Buddy on another computer</h2>
<ol><li>On that computer, open <b>Desk Buddy</b> › <b>This Mac</b> › <b>Connect a computer…</b></li>
<li>Type the server and the code it shows:</li></ol>
<form method="post" class="connect"><input type="hidden" name="t" value="` + token + `"><input type="hidden" name="do" value="connect">
<label>Server<input name="server" value="` + esc(server) + `" placeholder="https://….workers.dev or 192.168.1.23:8787" required></label>
<label>Code<input name="code" placeholder="ABCD-1234" required autocomplete="off"></label>
<button class="pill">Join</button></form>`
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
		where, phone := c.Server, c.Server
		if c.Mode == "local" {
			where, phone = "On this PC ("+lanAddress()+")", lanAddress()
		}
		asks := ""
		for _, p := range pairs() {
			idf := `<input type="hidden" name="id" value="` + esc(p.ID) + `">`
			asks += `<div class="ask"><b>` + esc(p.Name) + ` wants to connect</b><span>Check it shows the code <b class="code">` + esc(p.Code) + `</b>. Allowing it lets it show and change your tasks. Only allow a device you own.</span>
<div class="buttons"><form method="post"><input type="hidden" name="t" value="` + token + `"><input type="hidden" name="do" value="allow">` + idf + `<button class="pill solid">Allow</button></form>
<form method="post"><input type="hidden" name="t" value="` + token + `"><input type="hidden" name="do" value="deny">` + idf + `<button class="pill">Don't Allow</button></form></div></div>`
		}
		qr := ""
		if png, err := qrcode.Encode(phone+"/app?key="+c.Key, qrcode.Medium, 220); err == nil {
			qr = `<img class="qr" src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(png) + `" alt="QR code for the phone page" title="Only show it to your own phone">`
		}
		body = asks + `<p class="status">` + esc(st) + `</p>
<div class="row"><div><b>Desk Buddy runs</b><span>` + esc(where) + `</span></div>` + form("disconnect", map[bool]string{true: "Stop and reset", false: "Disconnect"}[c.Mode == "local"], false) + `</div>
<div class="row"><div><b>Phone page</b><span>Scan with your phone's camera to see and change your tasks there.` + map[bool]string{true: " Works on this Wi-Fi while this PC is on.", false: ""}[c.Mode == "local"] + `</span></div>` + qr + `</div>
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
.status{font-weight:700;font-size:17px}.choice{display:flex;gap:16px;align-items:center;justify-content:space-between;padding:14px;border:1.5px solid var(--ink);border-radius:10px}.choice b{display:block}.choice span{color:var(--grey);font-size:13px}
.ask{border:2px solid var(--ink);border-radius:10px;padding:14px;margin-bottom:18px}.ask b{display:block}.ask span{color:var(--grey);font-size:13px}.ask .code{display:inline;font-size:20px;color:var(--ink);letter-spacing:2px}.buttons{display:flex;gap:10px;margin-top:10px}
.qr{width:120px;height:120px;background:#fff;padding:6px;border-radius:6px}.note{font-weight:700}.small{color:var(--grey);font-size:12px;margin-top:20px}
</style></head><body><main><header><img class="buddy" src="` + buddy + `" alt=""><h1>Desk Buddy<small>for Windows · ` + version + `</small></h1></header>` + body + `</main></body></html>`
}

func refresh(on bool) string {
	if on {
		if len(pairs()) > 0 {
			return `<meta http-equiv="refresh" content="30">`
		}
		return `<meta http-equiv="refresh" content="5">`
	}
	return ""
}
