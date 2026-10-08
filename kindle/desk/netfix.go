package main

// When the backend can't be reached: say why in plain words, and fix what the Kindle can
// fix itself. Its Wi-Fi may be off (the Kindle's own UI, which we pause, normally turns it
// back on), or its clock may be wrong after the battery ran flat (then every https
// connection fails, as certificates look "not valid yet").

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// wifiState is wifid's connection state ("CONNECTED" when on a network), "" off a Kindle.
func wifiState() string {
	out, err := exec.Command("lipc-get-prop", "-s", "com.lab126.wifid", "cmState").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// wifiOff: Wi-Fi is switched off (wifid's "enable" is 0). Only then do we turn it on: while
// it's on but still searching or joining (slow on a weak signal), poking it restarts the
// join and it may never connect (seen: the Kindle stopped reaching Wi-Fi from the desk).
func wifiOff() bool {
	out, err := exec.Command("lipc-get-prop", "-i", "com.lab126.wifid", "enable").Output()
	return err == nil && strings.TrimSpace(string(out)) == "0"
}

// wifiOn asks the Kindle to turn Wi-Fi on and join its saved network.
func wifiOn() { exec.Command("lipc-set-prop", "-i", "com.lab126.wifid", "enable", "1").Run() }

// fixClock sets the Kindle's clock from the server's own clock, read over plain http (which
// doesn't need the clock to be right). Returns true if it changed it.
func fixClock(base string) bool {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return false
	}
	hc := &http.Client{Timeout: 6 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := hc.Get("http://" + u.Host + "/")
	if err != nil {
		return false
	}
	resp.Body.Close()
	t, err := http.ParseTime(resp.Header.Get("Date"))
	if err != nil || t.Sub(time.Now()).Abs() < 5*time.Minute {
		return false
	}
	tv := syscall.NsecToTimeval(t.UnixNano())
	return syscall.Settimeofday(&tv) == nil
}

// clockError: the connection failed because the Kindle's clock is off.
func clockError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "x509") && (strings.Contains(s, "not yet valid") || strings.Contains(s, "expired"))
}

// explainNetErr turns a failed request into a short, plain reason.
func explainNetErr(err error, wifi string) string {
	var dns *net.DNSError
	switch {
	case wifi != "" && wifi != "CONNECTED":
		return "The Kindle isn't on Wi-Fi yet (still joining, or out of range)."
	case clockError(err):
		return "The Kindle's clock is wrong (it says " + time.Now().Format("2 Jan 2006") + "). Setting it…"
	case errors.As(err, &dns):
		return "Can't look up the server's address. Is the Wi-Fi connected to the internet?"
	case strings.Contains(err.Error(), "Timeout") || strings.Contains(err.Error(), "timeout"):
		return "The server didn't answer in time."
	case strings.Contains(err.Error(), "connection refused"):
		return "The server isn't running there."
	case strings.Contains(err.Error(), "401"):
		return "The server doesn't accept this Kindle's key."
	}
	return "Couldn't connect: " + err.Error()
}
