//go:build windows

package main

// The PC's sound, for the Kindle's music screen: the volume (up / down in steps of 10) and the
// outputs (speakers, headphones, a monitor…), and switching between them. Windows' Core Audio,
// called directly on the media thread (com_windows.go).
//
// Switching the default output uses IPolicyConfig: Windows has no public call for it, so every
// sound switcher (EarTrumpet, SoundSwitch…) uses this one, unchanged since Windows 7. If it's
// ever missing, switching says it couldn't and nothing else is affected.

import (
	"errors"
	"math"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
)

var (
	clsidDeviceEnumerator = ole.NewGUID("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	iidDeviceEnumerator   = ole.NewGUID("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	iidEndpointVolume     = ole.NewGUID("{5CDF2C82-841E-4546-9722-0CF74078229A}")
	clsidPolicyConfig     = ole.NewGUID("{870AF99C-171D-4F9E-AF0D-E63DF40C2BC9}")
	iidPolicyConfig       = ole.NewGUID("{F8679F50-850A-41CF-9C72-430F290290C8}")

	ole32                = syscall.NewLazyDLL("ole32.dll")
	procPropVariantClear = ole32.NewProc("PropVariantClear")
)

// The property keys for a device's name ("Speakers (Realtek Audio)") and form factor.
type propertyKey struct {
	fmtid ole.GUID
	pid   uint32
}

var (
	keyFriendlyName = propertyKey{*ole.NewGUID("{A45C254E-DF1C-4EFD-8020-67D146A850E0}"), 14}
	keyFormFactor   = propertyKey{*ole.NewGUID("{1DA5D803-D492-4EDD-8C23-E0C0FFEE7F0E}"), 0}
)

// propVariant: Windows' "any value" (24 bytes on 64-bit Windows).
type propVariant struct {
	vt uint16
	_  [3]uint16
	p  unsafe.Pointer // a string, or the number itself
	_  uint64
}

const (
	eRender      = 0
	eConsole     = 0
	eMultimedia  = 1
	deviceActive = 1
	clsctxAll    = 0x17
	vtLPWStr     = 31
	vtUI4        = 19
)

var devices unsafe.Pointer // the device enumerator, made once (media thread only)

func enumerator() (unsafe.Pointer, error) {
	if devices != nil {
		return devices, nil
	}
	unk, err := ole.CreateInstance(clsidDeviceEnumerator, iidDeviceEnumerator)
	if err != nil {
		return nil, err
	}
	devices = unsafe.Pointer(unk)
	return devices, nil
}

// defaultOutput: the device music plays on now (release it).
func defaultOutput() (unsafe.Pointer, error) {
	e, err := enumerator()
	if err != nil {
		return nil, err
	}
	var dev unsafe.Pointer
	if err := comCall(e, 4, eRender, eMultimedia, uintptr(unsafe.Pointer(&dev))); err != nil { // GetDefaultAudioEndpoint
		return nil, err
	}
	if dev == nil {
		return nil, errors.New("no sound output")
	}
	return dev, nil
}

func deviceID(dev unsafe.Pointer) string {
	var p *uint16
	if comCall(dev, 5, uintptr(unsafe.Pointer(&p))) != nil || p == nil { // GetId
		return ""
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(p)))
	return windowsString(p)
}

func windowsString(p *uint16) string {
	var s []uint16
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0 && len(s) < 1024; ptr = unsafe.Add(ptr, 2) {
		s = append(s, *(*uint16)(ptr))
	}
	return syscall.UTF16ToString(s)
}

// deviceInfo: a device's name and form factor, from its property store.
func deviceInfo(dev unsafe.Pointer) (name string, form uint32) {
	var store unsafe.Pointer
	if comCall(dev, 4, 0, uintptr(unsafe.Pointer(&store))) != nil || store == nil { // OpenPropertyStore(STGM_READ)
		return "", 0
	}
	defer comRelease(store)
	var v propVariant
	if comCall(store, 5, uintptr(unsafe.Pointer(&keyFriendlyName)), uintptr(unsafe.Pointer(&v))) == nil { // GetValue
		if v.vt == vtLPWStr && v.p != nil {
			name = windowsString((*uint16)(v.p))
		}
		procPropVariantClear.Call(uintptr(unsafe.Pointer(&v)))
	}
	v = propVariant{}
	if comCall(store, 5, uintptr(unsafe.Pointer(&keyFormFactor)), uintptr(unsafe.Pointer(&v))) == nil {
		if v.vt == vtUI4 {
			form = *(*uint32)(unsafe.Pointer(&v.p))
		}
		procPropVariantClear.Call(uintptr(unsafe.Pointer(&v)))
	}
	return name, form
}

// endpointVolume: the volume control of the current output (release it).
func endpointVolume() (unsafe.Pointer, error) {
	dev, err := defaultOutput()
	if err != nil {
		return nil, err
	}
	defer comRelease(dev)
	var vol unsafe.Pointer
	if err := comCall(dev, 3, uintptr(unsafe.Pointer(iidEndpointVolume)), clsctxAll, 0, uintptr(unsafe.Pointer(&vol))); err != nil { // Activate
		return nil, err
	}
	if vol == nil {
		return nil, errors.New("no volume control")
	}
	return vol, nil
}

// readVolume: 0–100, and whether it's muted.
func readVolume(vol unsafe.Pointer) (int, bool, error) {
	var level float32
	if err := comCall(vol, 9, uintptr(unsafe.Pointer(&level))); err != nil { // GetMasterVolumeLevelScalar
		return 0, false, err
	}
	var muted int32
	comCall(vol, 15, uintptr(unsafe.Pointer(&muted))) // GetMute
	return int(math.Round(float64(level) * 100)), muted != 0, nil
}

// soundVolume: the PC's volume, 0–100 (0 when muted, as you'd hear it).
func soundVolume() (v int, ok bool) {
	onCOM(func() {
		vol, err := endpointVolume()
		if err != nil {
			return
		}
		defer comRelease(vol)
		level, muted, err := readVolume(vol)
		if err != nil {
			return
		}
		if muted {
			level = 0
		}
		v, ok = level, true
	})
	return v, ok
}

// changeVolume: up or down by 10 (up also unmutes).
func changeVolume(up bool) (err error) {
	if e := onCOM(func() {
		var vol unsafe.Pointer
		if vol, err = endpointVolume(); err != nil {
			return
		}
		defer comRelease(vol)
		var level int
		var muted bool
		if level, muted, err = readVolume(vol); err != nil {
			return
		}
		if muted {
			if !up {
				return
			}
			comCall(vol, 14, 0, 0) // SetMute(false)
			level = 0
		}
		next := float32(stepVolume(level, up)) / 100
		err = comCall(vol, 7, uintptr(math.Float32bits(next)), 0) // SetMasterVolumeLevelScalar
	}); e != nil {
		return e
	}
	return err
}

// soundOutputs: where the PC's sound can go (the outputs that are plugged in / connected).
func soundOutputs() (outs []output) {
	onCOM(func() {
		e, err := enumerator()
		if err != nil {
			return
		}
		cur := ""
		if dev, err := defaultOutput(); err == nil {
			cur = deviceID(dev)
			comRelease(dev)
		}
		var list unsafe.Pointer
		if comCall(e, 3, eRender, deviceActive, uintptr(unsafe.Pointer(&list))) != nil || list == nil { // EnumAudioEndpoints
			return
		}
		defer comRelease(list)
		var n uint32
		comCall(list, 3, uintptr(unsafe.Pointer(&n))) // GetCount
		for i := uint32(0); i < n && len(outs) < 10; i++ {
			var dev unsafe.Pointer
			if comCall(list, 4, uintptr(i), uintptr(unsafe.Pointer(&dev))) != nil || dev == nil { // Item
				continue
			}
			id := deviceID(dev)
			name, form := deviceInfo(dev)
			comRelease(dev)
			if id == "" || name == "" {
				continue
			}
			outs = append(outs, output{ID: id, Name: name, Kind: outputKind(form, name), Current: id == cur, Connected: true})
		}
	})
	return outs
}

// setOutput makes that output the PC's sound output (for music and everything else, like
// "Set as default device" in Windows' sound settings).
func setOutput(id string) (err error) {
	if e := onCOM(func() {
		unk, e := ole.CreateInstance(clsidPolicyConfig, iidPolicyConfig)
		if e != nil {
			err = errors.New("this PC can't switch outputs from here")
			return
		}
		pc := unsafe.Pointer(unk)
		defer comRelease(pc)
		wid, e := syscall.UTF16PtrFromString(id)
		if e != nil {
			err = e
			return
		}
		for _, role := range []uintptr{eConsole, eMultimedia} {
			if err = comCall(pc, 13, uintptr(unsafe.Pointer(wid)), role); err != nil { // SetDefaultEndpoint
				return
			}
		}
	}); e != nil {
		return e
	}
	return err
}
