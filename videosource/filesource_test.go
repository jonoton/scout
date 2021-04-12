// +build config

package videosource

import (
	"testing"
	"time"

	"gocv.io/x/gocv"
)

func TestFileSource(t *testing.T) {
	f := NewVideoReader(NewFileSource("test1", "C:\\video\\2529-video.mp4"), 30, 2)
	images := f.Start()
	defer f.Stop()

	go func() {
		tick := time.NewTicker(35 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				f.Stop()
				return
			}
		}
	}()

	window := gocv.NewWindow("Test Window")
	defer window.Close()
	for img := range images {
		mat := img.SharedMat.Mat
		window.IMShow(mat)
		window.WaitKey(5)
	}
	f.Wait() // should return immediately
	window.WaitKey(5000)
}
