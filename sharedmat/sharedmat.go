package sharedmat

import (
	"sync"

	log "github.com/sirupsen/logrus"
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
	s := &SharedMat{
		Mat:   gocv.Mat{},
		refs:  1,
		Guard: sync.RWMutex{},
	}
	if Valid(&mat) {
		s.Mat = mat
	}
	return s
}

// NumRefs returns the number of references
func (s *SharedMat) NumRefs() int {
	s.Guard.RLock()
	defer s.Guard.RUnlock()
	return s.refs
}

// Ref returns a SharedMat pointer and increments refs
func (s *SharedMat) Ref() *SharedMat {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.refs++
	return s
}

// Clone will clone the SharedMat and attempt to clone the gocv.Mat
func (s *SharedMat) Clone() *SharedMat {
	s.Guard.RLock()
	defer s.Guard.RUnlock()
	clone := &SharedMat{
		Mat:   gocv.Mat{},
		refs:  1,
		Guard: sync.RWMutex{},
	}
	if Valid(&s.Mat) {
		clone.Mat = s.Mat.Clone()
	}
	return clone
}

// Cleanup will decrement refs and attempt to cleanup the SharedMat
func (s *SharedMat) Cleanup() {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.refs--
	if s.refs <= 0 && Valid(&s.Mat) {
		s.Mat.Close()
		log.Debugln("Mat Closed")
	}
	if s.refs < 0 {
		log.Debugf("Refs=%d\n", s.refs)
	}
}

// Valid is a helper to check gocv.Mat validity
func Valid(mat *gocv.Mat) bool {
	return mat.Ptr() != nil && !mat.Empty()
}
