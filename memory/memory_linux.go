// +build linux

package memory

// #include <unistd.h>
import "C"

import "syscall"

// GetRAMAppBytes returns the app ram usage in bytes
func GetRAMAppBytes() uint64 {
	var rmem syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rmem)
	return uint64(rmem.Maxrss) * 1000 // convert to bytes
}

// GetRAMSystemBytes returns the system total ram in bytes
func GetRAMSystemBytes() uint64 {
	return uint64(C.sysconf(C._SC_PHYS_PAGES) * C.sysconf(C._SC_PAGE_SIZE))
}
