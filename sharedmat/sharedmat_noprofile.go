// +build !profile

package sharedmat

import (
	"sync"

	"gocv.io/x/gocv"
)

func newSharedMat(mat gocv.Mat) *SharedMat {
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

func (s *SharedMat) ref() *SharedMat {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.refs++
	return s
}

func (s *SharedMat) clone() *SharedMat {
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

func (s *SharedMat) cleanup() bool {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	closedMat := false
	s.refs--
	if s.refs <= 0 && Valid(&s.Mat) {
		s.Mat.Close()
		closedMat = true
	}
	return closedMat
}
