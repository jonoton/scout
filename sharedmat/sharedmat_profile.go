// +build profile

package sharedmat

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"sort"
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
var StkSkip = 3
var StkSkpAdd = 5
var StkSkpRemove = 5
var Tracker = make(map[string]map[string][]*trackSharedMat)
var GuardProfile = sync.RWMutex{}
var LogPrepend = "PROFILE:"

func init() {
	profName := "github.com/jonoton/scout/sharedmat.counts"
	SharedMatProfile = pprof.Lookup(profName)
	if SharedMatProfile == nil {
		SharedMatProfile = pprof.NewProfile(profName)
	}
}

type trackSharedMat struct {
	tKey        string
	id          int64
	stk         []uintptr
	stkStrSlice []string
	stkKey      string
}

func newTrackSharedMat(s *SharedMat) *trackSharedMat {
	Counter++
	stk := getStack(StkSkpAdd)
	stkStrSlice := getStackStrSlice(stk, 0)
	stkKey := getStackStr(stk, 0)
	if len(stkKey) == 0 {
		log.Warningln(LogPrepend, "EMPTY key!!")
	}
	t := &trackSharedMat{
		tKey:        s.getTrackerKey(),
		id:          Counter,
		stk:         stk,
		stkStrSlice: stkStrSlice,
		stkKey:      stkKey,
	}
	return t
}

func (s *SharedMat) getTrackerKey() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%p", s)
	return buf.String()
}

func (s *SharedMat) addTracked() {
	GuardProfile.Lock()
	defer GuardProfile.Unlock()
	t := newTrackSharedMat(s)
	if tmap, tfound := Tracker[t.tKey]; tfound {
		if sslice, sfound := tmap[t.stkKey]; sfound {
			tmap[t.stkKey] = append(sslice, t)
		} else {
			tmap[t.stkKey] = []*trackSharedMat{t}
		}
	} else {
		tmap := make(map[string][]*trackSharedMat)
		tmap[t.stkKey] = []*trackSharedMat{t}
		Tracker[t.tKey] = tmap
	}
	defer func() {
		if r := recover(); r != nil {
			log.Warnln(LogPrepend, "Ignored Recover: ADD")
		}
	}()
	SharedMatProfile.Add(t, StkSkip)
}

func (s *SharedMat) removeTracked() {
	GuardProfile.Lock()
	defer GuardProfile.Unlock()
	defer func() {
		if r := recover(); r != nil {
			log.Warnln(LogPrepend, "Ignored Recover: REMOVE")
		}
	}()
	removedProfile := false
	tkey := s.getTrackerKey()
	if tmap, sfound := Tracker[tkey]; sfound {
		curStk := getStack(StkSkpRemove)
	StkLoop:
		for i := 0; i < len(curStk); i++ {
			stkStr := getStackStr(curStk, i)
			if len(stkStr) == 0 {
				continue
			}
			for stkKey, sslice := range tmap {
				sfound := strings.Contains(stkKey, stkStr)
				if sfound {
					// pop front
					t := sslice[0]
					tmap[stkKey] = sslice[1:]
					SharedMatProfile.Remove(t)
					removedProfile = true

					// check and remove
					if len(tmap[stkKey]) == 0 {
						delete(tmap, stkKey)
					}
					if len(tmap) == 0 {
						delete(Tracker, t.tKey)
					}
					break StkLoop
				}
			}
		}
		if !removedProfile {
			// could not find by stack, pop oldest
			allFound := make(map[int64]*trackSharedMat)
			for _, sslice := range tmap {
				for _, t := range sslice {
					allFound[t.id] = t
				}
			}
			if len(allFound) == 0 {
				log.Warningln(LogPrepend, "Empty tmap")
			} else {
				keys := make([]int64, len(allFound))
				i := 0
				for k := range allFound {
					keys[i] = k
					i++
				}
				sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
				oldestId := keys[0]
				if t, found := allFound[oldestId]; found {
					SharedMatProfile.Remove(t)
					removedProfile = true
				OldestLoop:
					for stkKey, sslice := range tmap {
						for index, t := range sslice {
							if t.id == oldestId {
								// remove index
								tmap[stkKey] = append(sslice[:index], sslice[index+1:]...)

								// check and remove
								if len(tmap[stkKey]) == 0 {
									delete(tmap, stkKey)
								}
								if len(tmap) == 0 {
									delete(Tracker, t.tKey)
								}
								break OldestLoop
							}
						}
					}
				} else {
					log.Warningln(LogPrepend, "Could not remove oldest!")
				}
			}
		}
		if !removedProfile {
			log.Warningln(LogPrepend, "Could not remove!")
		}
	} else {
		log.Warningln(LogPrepend, "Not found!")
	}
}

func newSharedMat(mat gocv.Mat) *SharedMat {
	s := &SharedMat{
		Mat:   mat,
		refs:  1,
		Guard: sync.RWMutex{},
	}
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.addTracked()
	return s
}

func (s *SharedMat) ref() *SharedMat {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.refs++
	s.addTracked()
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

func (s *SharedMat) cleanup() (filled bool, closed bool) {
	s.Guard.Lock()
	defer s.Guard.Unlock()
	s.refs--
	filled = Filled(&s.Mat)
	if s.refs <= 0 && Valid(&s.Mat) {
		s.Mat.Close()
		closed = true
	}
	if s.refs < 0 {
		log.Warnf(LogPrepend, "Negative Refs at %d\n", s.refs)
	}
	if s.refs >= 0 {
		s.removeTracked()
	}
	return
}

func getStack(stkSkip int) []uintptr {
	stk := make([]uintptr, 32)
	n := runtime.Callers(stkSkip+1, stk)
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
		name := getStackNameStr(frame)
		if name != "runtime.goexit" && !strings.HasPrefix(name, "runtime.") {
			if i > 0 {
				fmt.Fprintf(&buf, " %s", name)
			} else {
				fmt.Fprintf(&buf, "%s", name)
			}
		}
		if !more {
			break
		}
		i++
	}
	return buf.String()
}

func getStackStrSlice(stk []uintptr, offset int) []string {
	result := make([]string, 0)
	substk := stk[offset:]
	frames := runtime.CallersFrames(substk)
	for {
		frame, more := frames.Next()
		name := getStackNameStr(frame)
		if name != "runtime.goexit" && !strings.HasPrefix(name, "runtime.") {
			result = append(result, name)
		}
		if !more {
			break
		}
	}
	return result
}

func getStackNameStr(frame runtime.Frame) string {
	name := frame.Function
	if name == "" {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "%#x", frame.PC)
		name = buf.String()
	}
	name = strings.ReplaceAll(name, "github.com/jonoton/scout/", "")
	return name
}
