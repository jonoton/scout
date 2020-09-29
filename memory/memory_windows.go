// +build !linux

package memory

import (
	"syscall"
	"unsafe"
)

type processMemoryCounters struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uint64
	WorkingSetSize             uint64
	QuotaPeakPagedPoolUsage    uint64
	QuotaPagedPoolUsage        uint64
	QuotaPeakNonPagedPoolUsage uint64
	QuotaNonPagedPoolUsage     uint64
	PagefileUsage              uint64
	PeakPagefileUsage          uint64
}

// GetRAMAppBytes returns the app ram usage in bytes
func GetRAMAppBytes() uint64 {
	curProcess, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0
	}
	psapi := syscall.NewLazyDLL("psapi.dll")
	var pmc processMemoryCounters
	pmc.cb = uint32(unsafe.Sizeof((pmc)))
	proc := psapi.NewProc("GetProcessMemoryInfo")
	proc.Call(uintptr(curProcess), uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.cb))
	return pmc.PagefileUsage
}

// GetRAMSystemBytes returns the system total ram in bytes
func GetRAMSystemBytes() uint64 {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel.NewProc("GetPhysicallyInstalledSystemMemory")
	var sysMem uint64
	proc.Call(uintptr(unsafe.Pointer(&sysMem)))
	sysMem = sysMem * 1000 // convert to bytes
	return sysMem
}
