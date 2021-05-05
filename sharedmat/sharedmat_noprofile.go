// +build !profile

package sharedmat

import (
	"sync"

	"gocv.io/x/gocv"
)

func newSharedMat(mat gocv.Mat) *SharedMat {
	s := &SharedMat{
		Mat:   mat,
		refs:  1,
		Guard: sync.RWMutex{},
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
	var clone *SharedMat
	if Valid(&s.Mat) {
		clone = &SharedMat{
			Mat:   s.Mat.Clone(),
			refs:  1,
			Guard: sync.RWMutex{},
		}
	} else {
		clone = &SharedMat{
			Mat:   gocv.NewMat(),
			refs:  1,
			Guard: sync.RWMutex{},
		}
	}
	return clone
}

func (s *SharedMat) cleanup() (filled bool, closed bool) {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.refs--
	filled = Filled(&s.Mat)
	if s.refs <= 0 && Valid(&s.Mat) {
		s.Mat.Close()
		closed = true
	}
	return
}
