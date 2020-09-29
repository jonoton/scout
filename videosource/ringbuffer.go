package videosource

import (
	"sort"
	"sync"
)

// RingBufferImage is a ring buffer for Image
type RingBufferImage struct {
	ring      []Image
	maxLen    int
	guard     sync.RWMutex
	readyChan chan bool
}

// NewRingBufferImage creates a new RingBufferImage with maxLen
func NewRingBufferImage(maxLen int) *RingBufferImage {
	r := &RingBufferImage{
		ring:      make([]Image, 0),
		maxLen:    maxLen,
		guard:     sync.RWMutex{},
		readyChan: make(chan bool, 1),
	}
	return r
}

// Push will push onto the ring the Image and return any popped Image
func (r *RingBufferImage) Push(img Image) (popped Image) {
	r.guard.Lock()
	defer r.guard.Unlock()
	r.ring = append([]Image{img}, r.ring...)
	if len(r.ring) > r.maxLen {
		popped = r.pop()
	}
	select {
	case r.readyChan <- true:
	default:
	}
	return
}

// Pop will pop off the ring an Image. Check for valid image.
func (r *RingBufferImage) Pop() (popped Image) {
	r.guard.Lock()
	defer r.guard.Unlock()
	popped = r.pop()
	return
}

// PopAll will pop off all on the ring. Check for valid image.
func (r *RingBufferImage) PopAll() (popped []Image) {
	r.guard.Lock()
	defer r.guard.Unlock()
	popped = r.ring
	r.ring = make([]Image, 0)
	return
}

func (r *RingBufferImage) pop() (popped Image) {
	len := len(r.ring)
	if len == 0 {
		return
	}
	lastIndex := len - 1
	popped = r.ring[lastIndex]
	r.ring = r.ring[:lastIndex]
	return
}

// Len returns the length of the ring
func (r *RingBufferImage) Len() int {
	r.guard.RLock()
	defer r.guard.RUnlock()
	return len(r.ring)
}

// Ready returns the internal ready chan for checking ready to pop
func (r *RingBufferImage) Ready() <-chan bool {
	return r.readyChan
}

// SortByCreatedTime sorts the ring buffer
func (r *RingBufferImage) SortByCreatedTime() {
	r.guard.Lock()
	defer r.guard.Unlock()
	sort.Sort(ImageByCreatedTime(r.ring))
}

// RingBufferProcessedImage is a ring buffer for ProcessedImage
type RingBufferProcessedImage struct {
	IsSortByContent bool
	ring            []ProcessedImage
	maxLen          int
	guard           sync.RWMutex
	readyChan       chan bool
}

// NewRingBufferProcessedImage creates a new RingBufferProcessedImage
func NewRingBufferProcessedImage(maxLen int) *RingBufferProcessedImage {
	r := &RingBufferProcessedImage{
		IsSortByContent: false,
		ring:            make([]ProcessedImage, 0),
		maxLen:          maxLen,
		guard:           sync.RWMutex{},
		readyChan:       make(chan bool, 1),
	}
	return r
}

// Push will push onto the ring the ProcessedImage and return any popped ProcessedImage
func (r *RingBufferProcessedImage) Push(img ProcessedImage) (popped ProcessedImage) {
	r.guard.Lock()
	defer r.guard.Unlock()
	r.ring = append([]ProcessedImage{img}, r.ring...)
	if r.IsSortByContent {
		r.sortByContent()
	}
	if len(r.ring) > r.maxLen {
		popped = r.pop()
	}
	select {
	case r.readyChan <- true:
	default:
	}
	return
}

// Pop will pop off the ring an ProcessedImage. Check for valid image.
func (r *RingBufferProcessedImage) Pop() (popped ProcessedImage) {
	r.guard.Lock()
	defer r.guard.Unlock()
	popped = r.pop()
	return
}

// PopAll will pop off all on the ring. Check for valid image.
func (r *RingBufferProcessedImage) PopAll() (popped []ProcessedImage) {
	r.guard.Lock()
	defer r.guard.Unlock()
	popped = r.ring
	r.ring = make([]ProcessedImage, 0)
	return
}

func (r *RingBufferProcessedImage) pop() (popped ProcessedImage) {
	len := len(r.ring)
	if len == 0 {
		return
	}
	lastIndex := len - 1
	popped = r.ring[lastIndex]
	r.ring = r.ring[:lastIndex]
	return
}

// Len returns the length of the ring
func (r *RingBufferProcessedImage) Len() int {
	r.guard.RLock()
	defer r.guard.RUnlock()
	return len(r.ring)
}

// Ready returns the internal ready chan for checking ready to pop
func (r *RingBufferProcessedImage) Ready() <-chan bool {
	return r.readyChan
}

// SortByCreatedTime sorts the ring buffer
func (r *RingBufferProcessedImage) SortByCreatedTime() {
	r.guard.Lock()
	defer r.guard.Unlock()
	sort.Sort(ProcessedImageByCreatedTime(r.ring))
}

// SortByContent sorts the ring buffer
func (r *RingBufferProcessedImage) SortByContent() {
	r.guard.Lock()
	defer r.guard.Unlock()
	r.sortByContent()
}

func (r *RingBufferProcessedImage) sortByContent() {
	sort.Sort(ProcessedImageByObjLen(r.ring))
	sort.Sort(ProcessedImageByObjPercent(r.ring))
	sort.Sort(ProcessedImageByFaceLen(r.ring))
	sort.Sort(ProcessedImageByFacePercent(r.ring))
}
