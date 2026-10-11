//go:build windows

package main

// One thread for Windows' media and sound calls (COM, the way Windows apps talk to these
// parts of Windows, wants its calls on a thread that's set up for it). The rest of the app hands
// it small jobs and waits for them, never longer than 10 seconds.

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
)

var (
	comJobs  = make(chan func())
	comStart = make(chan struct{}, 1)
	errBusy  = errors.New("Windows didn't answer in time")
)

func init() { comStart <- struct{}{} }

// onCOM runs f on the media thread and waits for it.
func onCOM(f func()) error {
	select {
	case <-comStart:
		go comThread()
	default:
	}
	done := make(chan struct{})
	job := func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil { // the WinRT bindings panic when Windows says no
				report(fmt.Sprintf("music: %v", r))
			}
		}()
		f()
	}
	timeout := time.After(10 * time.Second)
	select {
	case comJobs <- job:
	case <-timeout:
		return errBusy
	}
	select {
	case <-done:
		return nil
	case <-timeout:
		return errBusy
	}
}

func comThread() {
	runtime.LockOSThread()
	ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	for f := range comJobs {
		f()
	}
}

var iidAsyncInfo = ole.NewGUID("{00000036-0000-0000-C000-000000000046}")

// await waits (up to 3 seconds) for one of Windows' "answer later" calls, and returns the
// answer. It looks every 10 ms rather than asking Windows to call back.
func await(op *foundation.IAsyncOperation, err error) (unsafe.Pointer, error) {
	if err != nil {
		return nil, err
	}
	if op == nil {
		return nil, errors.New("no answer from Windows")
	}
	defer op.Release()
	d, err := op.QueryInterface(iidAsyncInfo)
	if err != nil {
		return nil, err
	}
	info := (*foundation.IAsyncInfo)(unsafe.Pointer(d))
	defer info.Release()
	for end := time.Now().Add(3 * time.Second); ; {
		st, err := info.GetStatus()
		if err != nil {
			return nil, err
		}
		switch st {
		case foundation.AsyncStatusCompleted:
			return op.GetResults()
		case foundation.AsyncStatusStarted:
			if time.Now().After(end) {
				info.Cancel()
				return nil, errBusy
			}
			time.Sleep(10 * time.Millisecond)
		default:
			code, _ := info.GetErrorCode()
			return nil, fmt.Errorf("Windows said no (%#x)", uint32(code.Value))
		}
	}
}

// comCall calls method n of a COM object (0–2 are the ones every object has). The first four
// numbers also go in the registers Windows reads decimals from, so a float32's bits work too
// (Go's Windows calls do that on x64).
func comCall(obj unsafe.Pointer, n int, args ...uintptr) error {
	if obj == nil {
		return errors.New("nothing to ask")
	}
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, n*int(unsafe.Sizeof(uintptr(0)))))
	hr, _, _ := syscall.SyscallN(fn, append([]uintptr{uintptr(obj)}, args...)...)
	if int32(hr) < 0 {
		return ole.NewError(hr)
	}
	return nil
}

func comRelease(obj unsafe.Pointer) {
	if obj != nil {
		comCall(obj, 2)
	}
}
