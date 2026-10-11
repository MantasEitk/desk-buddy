//go:build windows

package main

// The few Windows calls Desk Buddy needs: how long since the last keyboard / mouse input,
// light or dark mode, a sound, opening an app or link, one copy at a time, start with Windows.

import (
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	winmm                = windows.NewLazySystemDLL("winmm.dll")
	procGetLastInputInfo = user32.NewProc("GetLastInputInfo")
	procGetTickCount     = kernel32.NewProc("GetTickCount")
	procPlaySound        = winmm.NewProc("PlaySoundW")
)

// idleSeconds: time since the last keyboard or mouse input (only the time, never what it was).
func idleSeconds() float64 {
	var li struct{ cbSize, dwTime uint32 }
	li.cbSize = uint32(unsafe.Sizeof(li))
	if r, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&li))); r == 0 {
		return 0
	}
	now, _, _ := procGetTickCount.Call()
	return float64(uint32(now)-li.dwTime) / 1000
}

// darkMode: Windows' "app mode" is dark.
func darkMode() (bool, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false, false
	}
	return v == 0, true
}

// chimeSound: Windows' own notification sound.
func chimeSound() {
	const sndAlias, sndAsync = 0x00010000, 0x0001
	name, _ := syscall.UTF16PtrFromString("SystemNotification")
	procPlaySound.Call(uintptr(unsafe.Pointer(name)), 0, sndAlias|sndAsync)
}

// shellOpen opens a file, shortcut or link the way double-clicking would.
func shellOpen(target string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(target)
	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}

// onlyOne: false when Desk Buddy is already running.
func onlyOne() bool {
	name, _ := syscall.UTF16PtrFromString(`Local\DeskBuddyWindows`)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h) // don't keep the other copy's mutex alive
		return false
	}
	return true
}

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func startsWithWindows() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue("Desk Buddy")
	return err == nil
}

func setStartWithWindows(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		k.DeleteValue("Desk Buddy")
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue("Desk Buddy", `"`+exe+`"`)
}

const serverExe = "desk-buddy-server.exe"

// hideWindow: the server is a console program; no black window, please.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}

// ejectDrive ejects a removable drive (`E:\`) the way Explorer's Eject does: lock it, take
// it offline, allow removal, eject (the steps Microsoft documents for ejecting media).
// (0.6 ran PowerShell for this, and Defender's guesser flagged the app as a trojan.)
func ejectDrive(d string) bool {
	if len(d) < 2 {
		return false
	}
	path, err := windows.UTF16PtrFromString(`\\.\` + d[:2])
	if err != nil {
		return false
	}
	h, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	const (
		fsctlLockVolume        = 0x00090018
		fsctlDismountVolume    = 0x00090020
		ioctlStorageRemoval    = 0x002D4804
		ioctlStorageEjectMedia = 0x002D4808
	)
	var n uint32
	io := func(code uint32, in *byte, size uint32) error {
		return windows.DeviceIoControl(h, code, in, size, nil, 0, &n, nil)
	}
	locked := false
	for i := 0; i < 10 && !locked; i++ { // something may still have a file open for a moment
		locked = io(fsctlLockVolume, nil, 0) == nil
		if !locked {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if !locked || io(fsctlDismountVolume, nil, 0) != nil {
		return false
	}
	allow := byte(0) // PREVENT_MEDIA_REMOVAL { PreventMediaRemoval: false }
	io(ioctlStorageRemoval, &allow, 1)
	return io(ioctlStorageEjectMedia, nil, 0) == nil
}
