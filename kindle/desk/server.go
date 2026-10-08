package main

// Which backend the Kindle talks to: Cloud (Cloudflare) or a computer on the same Wi-Fi
// running the local server ("This Mac" in the Mac helper). Settings › Server switches;
// the Kindle finds computers by asking on the Wi-Fi (UDP broadcast "desk-buddy?" to port
// 8788, answered with {name, port}). The choice is saved in config.json.
//
// While offline, a small icon in the header says so; tapping it explains (for a local
// server: the computer has to be awake, on this Wi-Fi, with the helper open).

import (
	"encoding/json"
	"fmt"
	"image"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// FoundServer is a computer on the Wi-Fi running the local server.
type FoundServer struct {
	Name, Addr string // "My MacBook", "http://192.168.1.23:8787"
}

var serverMu sync.RWMutex

// SetBase points the client at another backend.
func (c *Client) SetBase(base string) {
	serverMu.Lock()
	c.base = strings.TrimRight(base, "/")
	serverMu.Unlock()
}

func (c *Client) Base() string {
	serverMu.RLock()
	defer serverMu.RUnlock()
	return c.base
}

// isLocalServer: anything that isn't https is a computer at home.
func isLocalServer(base string) bool { return !strings.HasPrefix(base, "https://") }

// findServers asks on the Wi-Fi for local servers, for about 1.5 s.
func findServers() []FoundServer {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil
	}
	defer conn.Close()
	targets := []string{"255.255.255.255"}
	if ifs, err := net.InterfaceAddrs(); err == nil { // also this network's own broadcast address
		for _, a := range ifs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil && !n.IP.IsLoopback() {
				ip, mask := n.IP.To4(), n.Mask
				b := make(net.IP, 4)
				for i := range b {
					b[i] = ip[i] | ^mask[len(mask)-4+i]
				}
				targets = append(targets, b.String())
			}
		}
	}
	for _, t := range targets {
		conn.WriteToUDP([]byte("desk-buddy?"), &net.UDPAddr{IP: net.ParseIP(t), Port: 8788})
	}
	seen := map[string]FoundServer{}
	conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
	buf := make([]byte, 512)
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var r struct {
			Name string `json:"name"`
			Port int    `json:"port"`
		}
		if json.Unmarshal(buf[:n], &r) == nil && r.Port > 0 {
			addr := fmt.Sprintf("http://%s:%d", from.IP, r.Port)
			seen[addr] = FoundServer{Name: r.Name, Addr: addr}
		}
	}
	var out []FoundServer
	for _, s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

var errWrongKey = fmt.Errorf("it has a different key")

func (c *Client) Key() string {
	serverMu.RLock()
	defer serverMu.RUnlock()
	return c.key
}

// SetKey: the key a local server handed over when pairing, or the server's new key.
func (c *Client) SetKey(key string) {
	serverMu.Lock()
	c.key = key
	serverMu.Unlock()
}

// pairWith asks a local server to let this Kindle in: it shows the code (via shown) while
// the user clicks Allow in Desk Buddy on that computer, then returns the key.
func pairWith(c *Client, base string, shown func(code string)) (string, error) {
	hc := &http.Client{Timeout: 8 * time.Second, Transport: c.hc.Transport}
	base = strings.TrimRight(base, "/")
	resp, err := hc.Post(base+"/api/pair", "application/json", strings.NewReader(`{"name":"Kindle"}`))
	if err != nil {
		return "", fmt.Errorf("can't reach it")
	}
	var req struct{ ID, Code string }
	json.NewDecoder(resp.Body).Decode(&req)
	resp.Body.Close()
	if resp.StatusCode != 200 || req.ID == "" {
		return "", fmt.Errorf("it doesn't take new Kindles (update Desk Buddy)")
	}
	shown(req.Code)
	for end := time.Now().Add(5 * time.Minute); time.Now().Before(end); time.Sleep(2 * time.Second) {
		resp, err := hc.Get(base + "/api/pair/" + req.ID)
		if err != nil {
			continue
		}
		var r struct{ State, Key string }
		json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		switch r.State {
		case "ok":
			return r.Key, nil
		case "denied":
			return "", fmt.Errorf("it said no")
		case "expired":
			return "", fmt.Errorf("it took too long; try again")
		}
	}
	return "", fmt.Errorf("it took too long; try again")
}

// tryServer checks a backend answers with our key.
func tryServer(c *Client, base string) error {
	hc := &http.Client{Timeout: 8 * time.Second, Transport: c.hc.Transport}
	resp, err := hc.Get(strings.TrimRight(base, "/") + "/api/where") // hc uses c's transport: the key goes as a header
	if err != nil {
		return fmt.Errorf("can't reach it")
	}
	resp.Body.Close()
	if resp.StatusCode == 401 {
		return errWrongKey
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("it said %s", resp.Status)
	}
	return nil
}

// saveServer writes the new server into config.json, keeping everything else in it
// (and the key, when pairing gave a new one).
func saveServer(path, base, name, cloud string, key ...string) error {
	m := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &m)
	}
	m["server"], m["server_name"] = base, name
	if cloud != "" {
		m["cloud"] = cloud
	}
	if len(key) > 0 && key[0] != "" {
		m["key"] = key[0]
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	tmp := path + ".new"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// autoFind (Settings › Server › Find automatically): the server stopped answering, so look for
// the live one: the same computer at a new address, the cloud, or another computer on the
// Wi-Fi. "Live" = it answers with our key and has no "moved to" note (a retired copy).
func autoFind(c *Client, name, current, cloud string) (string, string, bool) {
	live := func(addr string) bool {
		if tryServer(c, addr) != nil {
			return false
		}
		d, err := NewClient(addr, c.Key()).Desk()
		return err == nil && (d.MoveTo == nil || d.MoveTo.Server == "")
	}
	if isLocalServer(current) {
		if n, a, ok := refind(c, name, current); ok && live(a) {
			return n, a, true
		}
	}
	if cloud != "" && cloud != current && live(cloud) {
		return "", cloud, true
	}
	for _, f := range findServers() {
		if f.Addr != current && live(f.Addr) {
			return f.Name, f.Addr, true
		}
	}
	return "", "", false
}

// serverRow: Settings › Server: where the Kindle is connected. Switching happens only on the
// computer (Desk Buddy for Mac / Windows); the Kindle follows by itself (see autoFind and the
// "moved to" notes), so there's nothing to pick here.
func serverRow(v *View, ui *UI, x0, x1, y int) {
	img := v.Img
	label(img, x0, y, "SERVER", grey2)
	where, sub := "Cloud", "Works while your computer sleeps"
	if ui.ServerLocal {
		where, sub = "Your computer", "Updates while it's on, on this Wi-Fi"
		if ui.ServerName != "" {
			where = ui.ServerName
		}
	}
	drawText(img, face("serifB", 20), x0, y+48, ellipsize(face("serifB", 20), where, x1-x0), ink)
	drawText(img, face("ui", 13), x0, y+70, sub, grey2)
	fillRect(img, image.Rect(x0, y+86, x1, y+87), grey4)
	f := face("ui", 15)
	for i, l := range []string{
		"To change this, use Desk Buddy on your computer.",
		"The Kindle follows by itself, and finds it again if it moves.",
	} {
		drawText(img, f, x0, y+122+i*24, l, grey2)
	}
}

// serverList: the places Desk Buddy can run (Cloud, computers found on this Wi-Fi, or asking
// the Mac to run it), the current one ticked; returns the y below it. Used by the Server page
// and, on first setup, by the pop-up.
func serverList(v *View, ui *UI, x0, x1, y int) int {
	img := v.Img
	const rowH = 64
	row := func(title, sub string, current bool, act, arg string) {
		r := image.Rect(x0-12, y, x1+12, y+rowH)
		tf := face("uiBold", 19)
		if !current {
			tf = face("ui", 19)
		}
		drawText(img, tf, x0, y+28, title, ink)
		drawText(img, face("ui", 14), x0, y+50, sub, grey2)
		if current {
			cy := float64(y + rowH/2)
			line(img, float64(x1-18), cy, float64(x1-11), cy+7, 2.6, ink)
			line(img, float64(x1-11), cy+7, float64(x1), cy-7, 2.6, ink)
		} else if act != "" {
			chevron(img, float64(x1-6), float64(y+rowH/2), 6, "right", ink)
			v.add(Target{R: r, Show: r, Act: act, Arg: arg})
		}
		fillRect(img, image.Rect(x0, y+rowH-1, x1, y+rowH), grey4)
		y += rowH
	}
	if !ui.NoCloud {
		row("Cloud", "Cloudflare · works while your computer sleeps", !ui.ServerLocal, "serverset", "cloud")
	}
	switch {
	case ui.Finding:
		row("Looking on this Wi-Fi…", "For a computer running Desk Buddy", false, "", "")
	case len(ui.Found) == 0 && ui.MacSeen && !ui.ServerLocal: // on the cloud: ask the Mac to run it
		row("Your Mac", "Ask it to run Desk Buddy on this Wi-Fi", false, "serverask", "local")
	case len(ui.Found) == 0:
		row("No computer found", "Turn on “This Mac” in Desk Buddy, then tap to look again", false, "serverfind", "")
	default:
		for _, s := range ui.Found {
			host := strings.TrimPrefix(s.Addr, "http://")
			row(s.Name, host+" · updates while it's on", ui.ServerLocal && ui.ServerAddr == s.Addr, "serverset", s.Name+"|"+s.Addr)
		}
	}
	if ui.PairCode != "" { // waiting for Allow on the computer: the code to compare
		cy := y + 22
		drawText(img, face("ui", 16), x0, cy+24, "Click Allow on the computer, and check it", grey2)
		drawText(img, face("ui", 16), x0, cy+46, "shows this code:", grey2)
		drawRight(img, face("uiBold", 44), x1, cy+52, ui.PairCode, ink)
		y += 96
	}
	if ui.ServerNote != "" {
		drawText(img, face("uiBold", 15), x0, y+30, ellipsize(face("uiBold", 15), ui.ServerNote, x1-x0), ink)
		y += 40
	}
	return y
}

// serverModal: "Where's Desk Buddy running?" over the faded Settings screen.
func serverModal(v *View, ui *UI) {
	img := v.Img
	for i, p := range img.Pix {
		img.Pix[i] = 255 - uint8(int(255-p)*28/100)
	}
	v.Targets = nil
	rows := max(1, len(ui.Found))
	if !ui.NoCloud {
		rows++
	}
	const rowH = 64
	box := image.Rect(LW/2-330, 0, LW/2+330, 0)
	extra := 104
	if ui.ServerNote != "" {
		extra += 30
	}
	if ui.PairCode != "" {
		extra += 96
	}
	box.Min.Y = LH/2 - (extra+rows*rowH)/2
	box.Max.Y = box.Min.Y + extra + rows*rowH
	fillRoundRect(img, box.Inset(-3), 16, paper)
	strokeRoundRect(img, box, 14, 2, ink)
	x0, x1 := box.Min.X+32, box.Max.X-32
	drawText(img, face("serifB", 28), x0, box.Min.Y+56, "Where's Desk Buddy running?", ink)
	cancel := pillAt(img, image.Rect(x1-110, box.Min.Y+20, x1, box.Min.Y+62), "Cancel", false)
	v.add(Target{R: cancel.Inset(-4), Show: cancel, Act: "serverpick"})
	serverList(v, ui, x0, x1, box.Min.Y+84)
}

// offlineInfo: what "offline" means here, after tapping the header's icon.
func offlineInfo(v *View, ui *UI, st Status) {
	img := v.Img
	for i, p := range img.Pix {
		img.Pix[i] = 255 - uint8(int(255-p)*28/100)
	}
	v.Targets = nil
	box := image.Rect(LW/2-320, LH/2-140, LW/2+320, LH/2+140)
	fillRoundRect(img, box.Inset(-3), 16, paper)
	strokeRoundRect(img, box, 14, 2, ink)
	x0 := box.Min.X + 32
	since := ""
	if !st.LastOK.IsZero() {
		since = " since " + st.LastOK.Format("15:04")
	}
	title, lines := "Can't reach the cloud"+since, []string{
		"The Kindle can't get online. Check its Wi-Fi.",
		"It keeps trying every 15 seconds.",
		"Task changes are kept here and sent once it's back.",
	}
	if ui.ServerLocal {
		name := ui.ServerName
		if name == "" {
			name = "your computer"
		}
		title = "Can't reach " + name + since
		lines = []string{
			"Desk Buddy runs on " + name + ". It needs to be awake,",
			"on this Wi-Fi, with Desk Buddy open.",
			"Task changes are kept here and sent once it's back.",
			"If its address changed, the Kindle finds it again.",
		}
	}
	drawText(img, face("serifB", 26), x0, box.Min.Y+56, ellipsize(face("serifB", 26), title, box.Dx()-64), ink)
	for i, l := range lines {
		drawText(img, face("ui", 18), x0, box.Min.Y+100+i*28, l, grey2)
	}
	by0, by1 := box.Max.Y-24-46, box.Max.Y-24
	ok := pillAt(img, image.Rect(box.Max.X-32-100, by0, box.Max.X-32, by1), "OK", true)
	v.add(Target{R: img.Rect, Show: ok, Act: "offlineinfo"}) // OK, or tap anywhere else
	retry := pillAt(img, image.Rect(x0, by0, x0+140, by1), "Try again", false)
	v.add(Target{R: retry.Inset(-4), Show: retry, Act: "retry"})
	restart := pillAt(img, image.Rect(retry.Max.X+10, by0, retry.Max.X+10+160, by1), "Restart app", false)
	v.add(Target{R: restart.Inset(-4), Show: restart, Act: "restartapp"})
}

// offlineGlyph: a cloud with a line through it.
func offlineGlyph(img *image.Gray, cx, cy float64, c uint8) {
	fillRoundRect(img, image.Rect(int(cx)-11, int(cy)-1, int(cx)+12, int(cy)+8), 4.5, c)
	fillCircle(img, cx-4, cy, 5, c)
	fillCircle(img, cx+3, cy-3, 7, c)
	line(img, cx-13, cy-12, cx+13, cy+12, 5, paper^c^ink) // a gap in the background colour…
	line(img, cx-13, cy-12, cx+13, cy+12, 2.2, c)         // …and the line itself
}
