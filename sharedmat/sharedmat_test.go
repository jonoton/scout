package sharedmat

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestIncrement(t *testing.T) {
	sharedMat := NewSharedMat(gocv.NewMat())
	t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	sharedMat.Ref()
	if sharedMat.NumRefs() != 2 {
		t.Fatalf("sharedMat refs = %d, expected 2\n", sharedMat.NumRefs())
	} else {
		t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	}
}

func TestDecrement(t *testing.T) {
	sharedMat := NewSharedMat(gocv.NewMat())
	t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	sharedMat.Cleanup()
	if sharedMat.NumRefs() != 0 {
		t.Fatalf("sharedMat refs = %d, expected 0\n", sharedMat.NumRefs())
	} else {
		t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	}
}

func TestCleanup(t *testing.T) {
	newMat, _ := gocv.NewMatFromBytes(1, 1, gocv.MatTypeCV16S, make([]byte, 10))
	sharedMat := NewSharedMat(newMat)
	secondMat := sharedMat.Ref()
	secondMat.Cleanup()
	if !Valid(&sharedMat.Mat) {
		t.Fatalf("sharedMat is not valid, expected to be valid\n")
	}
	if sharedMat.NumRefs() != 1 {
		t.Fatalf("sharedMat refs = %d, expected 1\n", sharedMat.NumRefs())
	}
	sharedMat.Cleanup()
	if sharedMat.NumRefs() != 0 {
		t.Fatalf("sharedMat refs = %d, expected 0\n", sharedMat.NumRefs())
	} else {
		t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	}
}

func TestDoubleCleanup(t *testing.T) {
	newMat, _ := gocv.NewMatFromBytes(1, 1, gocv.MatTypeCV16S, make([]byte, 10))
	sharedMat := NewSharedMat(newMat)
	if !Valid(&sharedMat.Mat) {
		t.Fatalf("sharedMat is not valid, expected to be valid\n")
	}
	if sharedMat.NumRefs() != 1 {
		t.Fatalf("sharedMat refs = %d, expected 1\n", sharedMat.NumRefs())
	}
	sharedMat.Cleanup()
	if sharedMat.NumRefs() != 0 {
		t.Fatalf("sharedMat refs = %d, expected 0\n", sharedMat.NumRefs())
	} else {
		t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	}
	sharedMat.Cleanup()
	if sharedMat.NumRefs() != -1 {
		t.Fatalf("sharedMat refs = %d, expected -1\n", sharedMat.NumRefs())
	} else {
		t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	}
}

func TestClone(t *testing.T) {
	newMat, _ := gocv.NewMatFromBytes(1, 1, gocv.MatTypeCV16S, make([]byte, 10))
	sharedMat := NewSharedMat(newMat)
	secondMat := sharedMat.Clone()
	if sharedMat.NumRefs() != 1 {
		t.Fatalf("sharedMat refs = %d, expected 1\n", sharedMat.NumRefs())
	}
	secondMat.Cleanup()
	if !Valid(&sharedMat.Mat) {
		t.Fatalf("sharedMat is not valid, expected to be valid\n")
	}
	if sharedMat.NumRefs() != 1 {
		t.Fatalf("sharedMat refs = %d, expected 1\n", sharedMat.NumRefs())
	}
	if secondMat.NumRefs() != 0 {
		t.Fatalf("secondMat refs = %d, expected 0\n", secondMat.NumRefs())
	} else {
		t.Logf("secondMat refs = %d\n", secondMat.NumRefs())
	}
	sharedMat.Cleanup()
	if sharedMat.NumRefs() != 0 {
		t.Fatalf("sharedMat refs = %d, expected 0\n", sharedMat.NumRefs())
	} else {
		t.Logf("sharedMat refs = %d\n", sharedMat.NumRefs())
	}
}
