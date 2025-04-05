package videosource

import (
	"image"
	"math"
	"time"

	"github.com/jonoton/go-sharedmat"
	"gocv.io/x/gocv"
)

// Image contains an image
type Image struct {
	SharedMat   *sharedmat.SharedMat
	CreatedTime time.Time
}

// NewImage creates a new Image
func NewImage(mat gocv.Mat) *Image {
	i := &Image{
		SharedMat:   sharedmat.NewSharedMat(mat),
		CreatedTime: time.Now(),
	}
	return i
}

// IsFilled checks the underlying image not empty
func (i *Image) IsFilled() bool {
	if i.SharedMat == nil {
		return false
	}
	i.SharedMat.Guard.RLock()
	defer i.SharedMat.Guard.RUnlock()
	return sharedmat.Filled(&i.SharedMat.Mat)
}

// Height returns the Image height or -1
func (i *Image) Height() int {
	if i.SharedMat == nil {
		return -1
	}
	i.SharedMat.Guard.RLock()
	defer i.SharedMat.Guard.RUnlock()
	result := -1
	if sharedmat.Filled(&i.SharedMat.Mat) {
		result = i.SharedMat.Mat.Rows()
	}
	return result
}

// Width returns the Image width or -1
func (i *Image) Width() int {
	if i.SharedMat == nil {
		return -1
	}
	i.SharedMat.Guard.RLock()
	defer i.SharedMat.Guard.RUnlock()
	result := -1
	if sharedmat.Filled(&i.SharedMat.Mat) {
		result = i.SharedMat.Mat.Cols()
	}
	return result
}

// Ref will reference the Image and underlying SharedMat
func (i *Image) Ref() *Image {
	copy := &Image{
		CreatedTime: i.CreatedTime,
	}
	if i.SharedMat != nil {
		copy.SharedMat = i.SharedMat.Ref()
	}
	return copy
}

// Clone will clone the Image
func (i *Image) Clone() *Image {
	clone := &Image{
		CreatedTime: i.CreatedTime,
	}
	if i.SharedMat != nil {
		clone.SharedMat = i.SharedMat.Clone()
	}
	return clone
}

// Cleanup will cleanup the Image
func (i *Image) Cleanup() (filled bool, closed bool) {
	if i.SharedMat != nil {
		filled, closed = i.SharedMat.Cleanup()
		if closed {
			i.SharedMat = nil
		}
	}
	return
}

// GetRegion will return a new Image per rectangle parameter
func (i *Image) GetRegion(rect image.Rectangle) (region Image) {
	if i.SharedMat == nil {
		return
	}
	i.SharedMat.Guard.RLock()
	defer i.SharedMat.Guard.RUnlock()
	corrRect := CorrectRectangle(*i, rect)
	if !corrRect.Empty() && sharedmat.Filled(&i.SharedMat.Mat) {
		matRegion := i.SharedMat.Mat.Region(corrRect)
		region = *NewImage(matRegion.Clone())
		matRegion.Close()
	}
	return
}

// ChangeQuality will change the Image quality to percent param
func (i *Image) ChangeQuality(percent int) {
	if i.SharedMat == nil {
		return
	}
	i.SharedMat.Guard.RLock()
	if sharedmat.Filled(&i.SharedMat.Mat) {
		jpgParams := []int{gocv.IMWriteJpegQuality, percent}
		encoded, err := gocv.IMEncodeWithParams(gocv.JPEGFileExt, i.SharedMat.Mat, jpgParams)
		i.SharedMat.Guard.RUnlock()
		if err == nil {
			newMat, err := gocv.IMDecode(encoded.GetBytes(), gocv.IMReadUnchanged)
			if err == nil {
				i.SharedMat.Cleanup()
				i.SharedMat = sharedmat.NewSharedMat(newMat)
			}
		}
	} else {
		i.SharedMat.Guard.RUnlock()
	}
}

// EncodedQuality returns a JPEG byte array with the given quality percentage
func (i *Image) EncodedQuality(percent int) []byte {
	imgArray := make([]byte, 0)
	if i.SharedMat == nil {
		return imgArray
	}
	i.SharedMat.Guard.RLock()
	if sharedmat.Filled(&i.SharedMat.Mat) {
		jpgParams := []int{gocv.IMWriteJpegQuality, percent}
		encoded, err := gocv.IMEncodeWithParams(gocv.JPEGFileExt, i.SharedMat.Mat, jpgParams)
		if err == nil {
			imgArray = encoded.GetBytes()
		}
	}
	i.SharedMat.Guard.RUnlock()
	return imgArray
}

// ScaleToWidth will return a copy of the Image to scale given the width
func (i *Image) ScaleToWidth(width int) Image {
	if width <= 0 || width == i.Width() {
		return *i.Ref()
	}
	if i.SharedMat == nil {
		return *i.Ref()
	}
	var scaled Image
	// scale down
	var interpolationFlags = gocv.InterpolationArea
	// scale up
	if width > i.Width() {
		interpolationFlags = gocv.InterpolationCubic
	}
	scaleWidth := float64(width) / float64(i.Width())
	scaleHeight := float64(width) / float64(i.Height())
	scaleEvenly := math.Min(scaleWidth, scaleHeight)
	dstMat := gocv.NewMat()
	i.SharedMat.Guard.RLock()
	if sharedmat.Filled(&i.SharedMat.Mat) {
		gocv.Resize(i.SharedMat.Mat, &dstMat, image.Point{}, scaleEvenly, scaleEvenly, interpolationFlags)
		scaled = *NewImage(dstMat.Clone())
		scaled.CreatedTime = i.CreatedTime
	} else {
		scaled = *NewImage(dstMat.Clone())
	}
	i.SharedMat.Guard.RUnlock()
	dstMat.Close()
	return scaled
}

// ImageByCreatedTime sorting ascending order
type ImageByCreatedTime []Image

func (b ImageByCreatedTime) Len() int           { return len(b) }
func (b ImageByCreatedTime) Less(i, j int) bool { return b[i].CreatedTime.Before(b[j].CreatedTime) }
func (b ImageByCreatedTime) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// ImageList is a list of Images
type ImageList struct {
	list []Image
}

// NewImageList creates a new ImageList
func NewImageList() *ImageList {
	return &ImageList{
		list: make([]Image, 0),
	}
}

// Len returns the length
func (i *ImageList) Len() int {
	return len(i.list)
}

// Push will push onto the list
func (i *ImageList) Push(img Image) {
	i.list = append([]Image{img}, i.list...)
}

// Set sets the list
func (i *ImageList) Set(imgs []Image) {
	i.list = imgs
}

// Pop will pop off the list
func (i *ImageList) Pop() (popped Image) {
	len := len(i.list)
	if len == 0 {
		return
	}
	lastIndex := len - 1
	popped = i.list[lastIndex]
	i.list = i.list[:lastIndex]
	return
}
