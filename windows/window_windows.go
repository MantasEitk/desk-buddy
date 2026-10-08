//go:build windows

package main

// The app window: Windows' own Edge engine (WebView2, on every Windows 10 / 11 PC) showing
// Desk Buddy's interface (ui.html), like the Mac app's window. One at a time; closing it keeps
// Desk Buddy running in the taskbar. Without WebView2 it opens in the browser instead.

import (
	"path/filepath"
	"runtime"
	"sync"

	"github.com/jchv/go-webview2"
)

var (
	winMu   sync.Mutex
	winOpen webview2.WebView
)

func openWindow() {
	winMu.Lock()
	if winOpen != nil {
		w := winOpen
		winMu.Unlock()
		w.Dispatch(func() { showWindow(w) })
		return
	}
	winMu.Unlock()
	go func() {
		runtime.LockOSThread() // a window and its messages live on one thread
		w := webview2.NewWithOptions(webview2.WebViewOptions{
			AutoFocus:     true,
			DataPath:      filepath.Join(configDir(), "window"),
			WindowOptions: webview2.WindowOptions{Title: "Desk Buddy", Width: 960, Height: 700, IconId: 1, Center: true},
		})
		if w == nil {
			report("window: WebView2 isn't available, opening the browser instead")
			openPage()
			return
		}
		winMu.Lock()
		winOpen = w
		winMu.Unlock()
		w.SetSize(760, 520, webview2.HintMin)
		w.Navigate(uiURL())
		w.Run()
		w.Destroy()
		winMu.Lock()
		winOpen = nil
		winMu.Unlock()
	}()
}

// showWindow brings the window to the front (it may be behind others or minimised).
func showWindow(w webview2.WebView) {
	h := uintptr(w.Window())
	procShowWindow.Call(h, 9) // SW_RESTORE
	procSetForegroundWindow.Call(h)
}

var (
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)
