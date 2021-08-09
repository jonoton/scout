package videosource

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestCleanup(t *testing.T) {
	newMat, _ := gocv.NewMatFromBytes(1, 1, gocv.MatTypeCV16S, make([]byte, 10))
	image := NewImage(newMat)
	secondImage := image.Ref()
	secondImage.Cleanup()
	if !image.IsFilled() {
		t.Fatalf("sharedMat is not filled, expected to be filled\n")
	}
	if image.SharedMat.NumRefs() != 1 {
		t.Fatalf("sharedMat refs = %d, expected 1\n", image.SharedMat.NumRefs())
	}
	image.Cleanup()
	if image.SharedMat != nil {
		t.Logf("sharedMat should be nil\n")
		if image.SharedMat.NumRefs() != 0 {
			t.Fatalf("sharedMat refs = %d, expected 0\n", image.SharedMat.NumRefs())
		} else {
			t.Logf("sharedMat refs = %d\n", image.SharedMat.NumRefs())
		}
	} else {
		t.Logf("sharedMat is nil\n")
	}
}

func TestScale(t *testing.T) {
	newMat, _ := gocv.NewMatFromBytes(1, 1, gocv.MatTypeCV16S, make([]byte, 10))
	image := NewImage(newMat)
	scaled := image.ScaleToWidth(image.Width())
	scaled.Cleanup()
	scaled = image.ScaleToWidth(0)
	scaled.Cleanup()
	scaled = image.ScaleToWidth(5)
	scaled.Cleanup()
	if !image.IsFilled() {
		t.Fatalf("sharedMat is not filled, expected to be filled\n")
	}
	if image.SharedMat.NumRefs() != 1 {
		t.Fatalf("sharedMat refs = %d, expected 1\n", image.SharedMat.NumRefs())
	}
	image.Cleanup()
	if image.SharedMat != nil {
		t.Logf("sharedMat should be nil\n")
		if image.SharedMat.NumRefs() != 0 {
			t.Fatalf("sharedMat refs = %d, expected 0\n", image.SharedMat.NumRefs())
		} else {
			t.Logf("sharedMat refs = %d\n", image.SharedMat.NumRefs())
		}
	} else {
		t.Logf("sharedMat is nil\n")
	}
}
