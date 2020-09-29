package memory

import (
	"runtime"
)

// Memory contains system memory information
type Memory struct {
	HeapAllocatedBytes uint64
	HeapTotalBytes     uint64
	RAMAppBytes        uint64
	RAMSystemBytes     uint64
}

// NewMemory creates a new Memory
func NewMemory() *Memory {
	m := &Memory{}
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.HeapAllocatedBytes = memStats.Alloc
	m.HeapTotalBytes = memStats.Sys
	m.RAMAppBytes = GetRAMAppBytes()
	m.RAMSystemBytes = GetRAMSystemBytes()
	return m
}

// BytesToMegaBytes converts Bytes to MegaBytes
func BytesToMegaBytes(in uint64) float64 {
	return float64(in) / 1000 / 1000
}

// BytesToGigaBytes converts Bytes to GigaBytes
func BytesToGigaBytes(in uint64) float64 {
	return float64(in) / 1000 / 1000 / 1000
}
