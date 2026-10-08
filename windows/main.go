// Desk Buddy for Windows: connects this PC to Desk Buddy. A tray icon (bottom right) with a
// small menu; the window is a page in your browser. It tells the Kindle when you're on this PC
// (so it shows the Windows list), opens apps from the Kindle's Deck, and chimes when a focus or
// break ends. Built on the Mac: GOOS=windows GOARCH=amd64 (see build.sh).
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/systray"
)

const version = "0.1"

//go:embed icon.ico
var iconICO []byte

var (
	stateMu  sync.Mutex
	status   = "Starting…"
	lastDeck string
	mStatus  *systray.MenuItem
)

func setStatus(s string) {
	if s == "" {
		return
	}
	stateMu.Lock()
	status = s
	stateMu.Unlock()
	if mStatus != nil {
		mStatus.SetTitle(s)
	}
}

func getStatus() (string, string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return status, lastDeck
}

func setLast(label, err string) {
	stateMu.Lock()
	if err == "" {
		lastDeck = label + " · opened " + time.Now().Format("15:04")
	} else {
		lastDeck = label + " · " + err
	}
	stateMu.Unlock()
	if err != "" {
		report("deck: " + label + ": " + err)
	}
}

func logf(format string, a ...any) { log.Printf(format, a...) }

func main() {
	if !onlyOne() {
		openPage() // already running: just show it
		return
	}
	os.MkdirAll(configDir(), 0o700)
	logPath := filepath.Join(configDir(), "desk-buddy.log")
	if st, err := os.Stat(logPath); err == nil && st.Size() > 500_000 {
		os.Remove(logPath)
	}
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		log.SetOutput(f)
	}
	defer func() {
		if r := recover(); r != nil {
			report(fmt.Sprintf("crashed: %v", r))
			time.Sleep(2 * time.Second)
		}
	}()
	loadConfig()
	startPage()
	go heartbeats()
	go deck()
	go appList()
	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetIcon(iconICO)
	systray.SetTitle("Desk Buddy")
	systray.SetTooltip("Desk Buddy")
	mStatus = systray.AddMenuItem(status, "")
	mStatus.Disable()
	systray.AddSeparator()
	mOpen := systray.AddMenuItem("Open Desk Buddy…", "Settings and status")
	mPause := systray.AddMenuItemCheckbox("Pause", "Stop telling the Kindle about this PC", current().Paused)
	mLogin := systray.AddMenuItemCheckbox("Start with Windows", "", startsWithWindows())
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit Desk Buddy", "")
	if !connected() {
		setStatus("Not connected yet")
		openPage()
	}
	for {
		select {
		case <-mOpen.ClickedCh:
			openPage()
		case <-mPause.ClickedCh:
			setPaused(!current().Paused)
			if current().Paused {
				mPause.Check()
			} else {
				mPause.Uncheck()
			}
		case <-mLogin.ClickedCh:
			on := !startsWithWindows()
			if err := setStartWithWindows(on); err != nil {
				report("start with Windows: " + err.Error())
			}
			if startsWithWindows() {
				mLogin.Check()
			} else {
				mLogin.Uncheck()
			}
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func setPaused(p bool) {
	c := current()
	c.Paused = p
	saveConfig(c)
	if p {
		setStatus("Paused · the Kindle won't follow this PC")
	} else {
		setStatus("Running")
	}
}
