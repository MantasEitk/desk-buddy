//go:build windows

package main

// Windows' own "All apps" list (the apps folder, shell:AppsFolder: Start menu programs and
// Store apps alike), each app's icon, and opening an app the way the Start menu does. All on
// the apps thread (com_windows.go); the icon's processing is in icons.go.
//
// The COM interfaces used, by method number (0–2 are IUnknown's QueryInterface, AddRef, Release):
//   IShellItem             {43826D1E-E718-42EE-BC55-A1E261C37BFE}: 3 BindToHandler, 5 GetDisplayName
//   IEnumShellItems        {70629033-E363-4A28-A567-0DB78006E6D7}: 3 Next
//   IShellItemImageFactory {BCC18B79-BA16-442F-80C4-8A59C30C463B}: 3 GetImage

import (
	"errors"
	"image"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

var (
	shell32                         = windows.NewLazySystemDLL("shell32.dll")
	gdi32                           = windows.NewLazySystemDLL("gdi32.dll")
	procSHGetKnownFolderItem        = shell32.NewProc("SHGetKnownFolderItem")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
	procShellExecuteEx              = shell32.NewProc("ShellExecuteExW")
	procGetObject                   = gdi32.NewProc("GetObjectW")
	procGetDIBits                   = gdi32.NewProc("GetDIBits")
	procDeleteObject                = gdi32.NewProc("DeleteObject")
	procGetDC                       = user32.NewProc("GetDC")
	procReleaseDC                   = user32.NewProc("ReleaseDC")

	folderApps        = ole.NewGUID("{1E87508D-89C2-42F0-8A7E-645A0F50CA58}") // FOLDERID_AppsFolder
	bhidEnumItems     = ole.NewGUID("{94F60519-2850-4924-AA5A-D15E84868039}") // BHID_EnumItems
	iidShellItem      = ole.NewGUID("{43826D1E-E718-42EE-BC55-A1E261C37BFE}")
	iidEnumShellItems = ole.NewGUID("{70629033-E363-4A28-A567-0DB78006E6D7}")
	iidImageFactory   = ole.NewGUID("{BCC18B79-BA16-442F-80C4-8A59C30C463B}")
)

const (
	sigdnNormalDisplay         = 0x00000000 // the name the Start menu shows
	sigdnParentRelativeParsing = 0x80018001 // its id in the apps folder (a Store app's AUMID, or a program's)
	siigbfIconOnly             = 0x4        // the icon, never a thumbnail
	iconAsk                    = 256        // px: Windows' biggest icons, made smaller in icons.go
)

// listApps: name → target ("shell:AppsFolder\<id>") for every app in Windows' "All apps".
func listApps() (map[string]string, error) {
	var found [][2]string
	var err error
	if e := onShell(func() { found, err = appsFolder() }); e != nil {
		return nil, e
	}
	if err != nil {
		return nil, err
	}
	apps := tidyApps(found)
	if len(apps) == 0 {
		return nil, errors.New("Windows listed no apps")
	}
	return apps, nil
}

func appsFolder() ([][2]string, error) {
	var folder unsafe.Pointer
	hr, _, _ := procSHGetKnownFolderItem.Call(uintptr(unsafe.Pointer(folderApps)), 0, 0, uintptr(unsafe.Pointer(iidShellItem)), uintptr(unsafe.Pointer(&folder)))
	if int32(hr) < 0 || folder == nil {
		return nil, ole.NewError(hr)
	}
	defer comRelease(folder)
	var list unsafe.Pointer
	if err := comCall(folder, 3, 0, uintptr(unsafe.Pointer(bhidEnumItems)), uintptr(unsafe.Pointer(iidEnumShellItems)), uintptr(unsafe.Pointer(&list))); err != nil { // BindToHandler
		return nil, err
	}
	if list == nil {
		return nil, errors.New("no apps folder")
	}
	defer comRelease(list)
	var out [][2]string
	for len(out) < 3000 {
		var item unsafe.Pointer
		var n uint32
		if comCall(list, 3, 1, uintptr(unsafe.Pointer(&item)), uintptr(unsafe.Pointer(&n))) != nil || n == 0 || item == nil { // Next (S_FALSE, 0 items: the end)
			break
		}
		out = append(out, [2]string{displayName(item, sigdnNormalDisplay), displayName(item, sigdnParentRelativeParsing)})
		comRelease(item)
	}
	return out, nil
}

func displayName(item unsafe.Pointer, kind uint32) string {
	var p *uint16
	if comCall(item, 5, uintptr(kind), uintptr(unsafe.Pointer(&p))) != nil || p == nil { // GetDisplayName
		return ""
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(p)))
	return windowsString(p)
}

// appIcon: the Kindle's PNG for an app (its target from listApps, or a shortcut's path).
func appIcon(target string) []byte {
	var img image.Image
	if onShell(func() { img = shellIcon(target) }) != nil {
		return nil
	}
	return iconPNG(img) // the processing, off the apps thread
}

func shellIcon(target string) image.Image {
	path, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return nil
	}
	var item unsafe.Pointer
	hr, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(iidShellItem)), uintptr(unsafe.Pointer(&item)))
	if int32(hr) < 0 || item == nil {
		return nil
	}
	defer comRelease(item)
	var factory unsafe.Pointer
	if comCall(item, 0, uintptr(unsafe.Pointer(iidImageFactory)), uintptr(unsafe.Pointer(&factory))) != nil || factory == nil {
		return nil
	}
	defer comRelease(factory)
	// GetImage(SIZE, flags, *HBITMAP): the SIZE (two 32-bit numbers) goes by value, in one
	// 64-bit register on x64: cx in the low half, cy in the high.
	size := uintptr(iconAsk)
	size |= size << 32
	var hbm uintptr
	if comCall(factory, 3, size, siigbfIconOnly, uintptr(unsafe.Pointer(&hbm))) != nil || hbm == 0 {
		return nil
	}
	defer procDeleteObject.Call(hbm)
	return bitmapPixels(hbm)
}

// bitmapPixels reads a bitmap's pixels as 32-bit BGRA, top row first.
func bitmapPixels(hbm uintptr) image.Image {
	var bm struct { // BITMAP
		Type, Width, Height, WidthBytes int32
		Planes, BitsPixel               uint16
		Bits                            uintptr
	}
	if r, _, _ := procGetObject.Call(hbm, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 {
		return nil
	}
	w, h := int(bm.Width), int(bm.Height)
	if h < 0 {
		h = -h
	}
	if w <= 0 || h <= 0 || w > 1024 || h > 1024 {
		return nil
	}
	bi := struct { // BITMAPINFOHEADER, and room for a colour table that 32-bit pixels don't use
		Size                     uint32
		Width, Height            int32
		Planes, BitCount         uint16
		Compression, SizeImage   uint32
		XPerMeter, YPerMeter     int32
		ColorsUsed, ColorsNeeded uint32
		Table                    [4]uint32
	}{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32} // minus: top row first
	buf := make([]byte, w*h*4)
	dc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, dc)
	if r, _, _ := procGetDIBits.Call(dc, hbm, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), 0); r == 0 { // DIB_RGB_COLORS
		return nil
	}
	return bitmapImage(buf, w, h)
}

// launch opens an app (its target), a bare link like spotify: or a program like chrome.exe,
// the way the Start menu or Run would. Never with anything after it, never a command.
func launch(target string) error {
	var err error
	if e := onShell(func() { err = shellExecute(target) }); e != nil {
		return e
	}
	return err
}

func shellExecute(target string) error {
	file, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	info := struct { // SHELLEXECUTEINFOW
		Size, Mask          uint32
		Window              uintptr
		Verb, File, Params  *uint16
		Directory           *uint16
		Show                int32
		InstApp, IDList     uintptr
		Class               *uint16
		KeyClass            uintptr
		HotKey              uint32
		IconOrMonitor, Proc uintptr
	}{File: file, Show: windows.SW_SHOWNORMAL}
	info.Size = uint32(unsafe.Sizeof(info))
	info.Mask = 0x100 | 0x400 // SEE_MASK_NOASYNC (done before we return), SEE_MASK_FLAG_NO_UI (no error boxes on an empty desk)
	if r, _, e := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		return e
	}
	return nil
}
