// +build profile

package sharedmat

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

// SharedMatProfile is a pprof.Profile that tracks the sharedmat New/Ref/Clone/Cleanup
// Get the count:
//
//	sharedmat.SharedMatProfile.Count()
//
// Display the current entries:
//
// 	var b bytes.Buffer
//	sharedmat.SharedMatProfile.WriteTo(&b, 1)
//	fmt.Print(b.String())
//
// Build, Run, or Test add to command:
// -tags profile
//
// Profile
//   Terminal
//     Tab1 Run `go run -tags profile github.com/jonoton/scout`
//     Tab2 Run `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/github.com/jonoton/scout/sharedmat.counts`
//
var SharedMatProfile *pprof.Profile
var Counter = int64(0)
var StkSkip = 2
var StkSkpAdd = StkSkip + 2
var StkSkpRemove = StkSkip + 1
var Tracker = make(map[*SharedMat]map[string]*trackSharedMat, 0)
var GuardProfile = sync.RWMutex{}

func init() {
	profName := "github.com/jonoton/scout/sharedmat.counts"
	SharedMatProfile = pprof.Lookup(profName)
	if SharedMatProfile == nil {
		SharedMatProfile = pprof.NewProfile(profName)
	}
}

type trackSharedMat struct {
	s      *SharedMat
	id     int64
	stk    []uintptr
	stkStr string
}

func newTrackSharedMat(s *SharedMat) *trackSharedMat {
	Counter++
	stk := getStack(StkSkpAdd)
	t := &trackSharedMat{
		s:      s,
		id:     Counter,
		stk:    stk,
		stkStr: getStackStr(stk, 0),
	}
	return t
}

func (s *SharedMat) addTracked() {
	GuardProfile.Lock()
	defer GuardProfile.Unlock()
	t := newTrackSharedMat(s)
	if tmap, sfound := Tracker[s]; sfound {
		tmap[t.stkStr] = t
	} else {
		tmap := make(map[string]*trackSharedMat, 0)
		tmap[t.stkStr] = t
		Tracker[s] = tmap
	}
	defer func() {
		if r := recover(); r != nil {
			log.Warnln("Ignored Recover: ADD")
		}
	}()
	SharedMatProfile.Add(t, StkSkip)
}

func (s *SharedMat) removeTracked() {
	GuardProfile.Lock()
	defer GuardProfile.Unlock()
	defer func() {
		if r := recover(); r != nil {
			log.Warnln("Ignored Recover: REMOVE")
		}
	}()
	removedProfile := false
	if tmap, sfound := Tracker[s]; sfound {
		if len(tmap) == 0 {
			// already cleaned up
			return
		}
		curStk := getStack(StkSkpRemove)
		for i := 0; i < len(curStk); i++ {
			stkStr := getStackStr(curStk, i)
			for tkey, t := range tmap {
				tfound := strings.Contains(tkey, stkStr)
				if tfound {
					SharedMatProfile.Remove(t)
					delete(tmap, tkey)
					removedProfile = true
					break
				}
			}
		}
		if !removedProfile {
			log.Warningln("Could not remove!")
		}
	} else {
		log.Warningln("Not found!")
	}
}

func newSharedMat(mat gocv.Mat) *SharedMat {
	s := &SharedMat{
		Mat:   mat,
		refs:  1,
		Guard: sync.RWMutex{},
	}
	s.addTracked()
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
	clone.addTracked()
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
	if s.refs < 0 {
		log.Warnf("Negative Refs at %d\n", s.refs)
	}
	s.removeTracked()
	return closedMat
}

func getStack(stkSkip int) []uintptr {
	stk := make([]uintptr, 32)
	n := runtime.Callers(stkSkip+1, stk[:])
	stk = stk[:n]
	if len(stk) == 0 {
		// The value for skip is too large, and there's no stack trace to record.
		stk = []uintptr{}
	}
	return stk
}

func getStackStr(stk []uintptr, offset int) string {
	var buf bytes.Buffer
	substk := stk[offset:]
	frames := runtime.CallersFrames(substk)
	i := 0
	for {
		frame, more := frames.Next()
		name := frame.Function
		if name == "" {
			if i > 0 {
				fmt.Fprintf(&buf, " ")
			}
			fmt.Fprintf(&buf, "%#x", frame.PC)
		} else if name != "runtime.goexit" && !strings.HasPrefix(name, "runtime.") {
			// Hide runtime.goexit and any runtime functions at the beginning.
			// This is useful mainly for allocation traces.
			if i > 0 {
				fmt.Fprintf(&buf, " ")
			}
			trimmed := strings.ReplaceAll(name, "github.com/jonoton/scout/", "")
			fmt.Fprintf(&buf, "%s", trimmed)
		}
		if !more {
			break
		}
		i++
	}

	return buf.String()
}
