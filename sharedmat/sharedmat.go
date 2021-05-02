package sharedmat

import (
	"sync"

	"gocv.io/x/gocv"
)

// SharedMat is a share-able gocv.Mat
type SharedMat struct {
	Mat   gocv.Mat
	refs  int
	Guard sync.RWMutex
}

// NewSharedMat creates a new SharedMat
func NewSharedMat(mat gocv.Mat) *SharedMat {
	return newSharedMat(mat)
}

// Ref returns a SharedMat pointer and increments refs
func (s *SharedMat) Ref() *SharedMat {
	return s.ref()
}

// Clone will clone the SharedMat and attempt to clone the gocv.Mat
func (s *SharedMat) Clone() *SharedMat {
	return s.clone()
}

// Cleanup will decrement refs and attempt to cleanup the SharedMat
func (s *SharedMat) Cleanup() bool {
	return s.cleanup()
}

// NumRefs returns the number of references
func (s *SharedMat) NumRefs() int {
	s.Guard.RLock()
	defer s.Guard.RUnlock()
	return s.refs
}

// Valid is a helper to check gocv.Mat not nil
func Valid(mat *gocv.Mat) bool {
	return mat.Ptr() != nil
}

// Filled is a helper to check gocv.Mat not empty
func Filled(mat *gocv.Mat) bool {
	return mat.Ptr() != nil && !mat.Empty()
}
