//go:build windows

package main

// Performance (the Performance page): CPU time and memory of this app and, on "On this PC", of
// its server. CPU is averaged over the last minute from samples every 10 s.

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	psapi                    = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
)

// usage: CPU seconds so far and memory in MB for a process (0 = this one).
func usage(pid int) (cpu float64, mem float64, ok bool) {
	h := windows.CurrentProcess()
	if pid != 0 {
		var err error
		h, err = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
		if err != nil {
			return 0, 0, false
		}
		defer windows.CloseHandle(h)
	}
	var c, e, k, u windows.Filetime
	if windows.GetProcessTimes(h, &c, &e, &k, &u) != nil {
		return 0, 0, false
	}
	ft := func(f windows.Filetime) float64 {
		return float64(uint64(f.HighDateTime)<<32|uint64(f.LowDateTime)) / 1e7
	}
	cpu = ft(k) + ft(u)
	var pmc struct {
		cb                                                                uint32
		PageFaultCount                                                    uint32
		PeakWorkingSetSize, WorkingSetSize                                uintptr
		QuotaPeakPagedPoolUsage, QuotaPagedPoolUsage                      uintptr
		QuotaPeakNonPagedPoolUsage, QuotaNonPagedPoolUsage, PagefileUsage uintptr
		PeakPagefileUsage                                                 uintptr
	}
	pmc.cb = uint32(unsafe.Sizeof(pmc))
	if r, _, _ := procGetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.cb)); r != 0 {
		mem = float64(pmc.WorkingSetSize) / (1 << 20)
	}
	return cpu, mem, true
}
