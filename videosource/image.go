package videosource

import (
	"image"
	"math"
	"time"

	"gocv.io/x/gocv"
)

// Image contains an image
type Image struct {
	Mat         gocv.Mat
	CreatedTime time.Time
}

// NewImage creates a new Image
func NewImage(mat gocv.Mat) *Image {
	i := &Image{
		Mat:         gocv.Mat{},
		CreatedTime: time.Now(),
	}
	if mat.Ptr() != nil && !mat.Empty() {
		i.Mat = mat
	}
	return i
}

// IsValid checks the underlying image for validity
func (i *Image) IsValid() bool {
	return i.Mat.Ptr() != nil && !i.Mat.Empty()
}

// Height returns the Image height or -1
func (i *Image) Height() int {
	if !i.IsValid() {
		return -1
	}
	return i.Mat.Rows()
}

// Width returns the Image width or -1
func (i *Image) Width() int {
	if !i.IsValid() {
		return -1
	}
	return i.Mat.Cols()
}

// Clone will clone the Image
func (i *Image) Clone() *Image {
	clone := &Image{
		CreatedTime: i.CreatedTime,
	}
	if i.IsValid() {
		clone.Mat = i.Mat.Clone()
	}
	return clone
}

// Cleanup will cleanup the Image
func (i *Image) Cleanup() {
	if i.IsValid() {
		i.Mat.Close()
	}
}

// GetRegion will return a new Image per rectangle parameter
func (i *Image) GetRegion(rect image.Rectangle) (region Image) {
	if !i.IsValid() {
		return
	}
	corrRect := CorrectRectangle(*i, rect)
	if !corrRect.Empty() {
		matRegion := i.Mat.Region(corrRect)
		region = *NewImage(matRegion.Clone())
		matRegion.Close()
	}
	return
}

// ChangeQuality will change the Image quality to percent param
func (i *Image) ChangeQuality(percent int) {
	if !i.IsValid() {
		return
	}
	jpgParams := []int{gocv.IMWriteJpegQuality, percent}
	encoded, err := gocv.IMEncodeWithParams(gocv.JPEGFileExt, i.Mat, jpgParams)
	if err != nil {
		return
	}
	newMat, err := gocv.IMDecode(encoded, gocv.IMReadUnchanged)
	if err != nil {
		return
	}
	i.Mat.Close()
	i.Mat = newMat
}

// ScaleToWidth will change the Image to scale to width
func (i *Image) ScaleToWidth(width int) {
	if !i.IsValid() || width <= 0 {
		return
	}
	scaleWidth := float64(width) / float64(i.Width())
	scaleHeight := float64(width) / float64(i.Height())
	scaleEvenly := math.Min(scaleWidth, scaleHeight)
	dstMat := gocv.NewMat()
	gocv.Resize(i.Mat, &dstMat, image.Point{}, scaleEvenly, scaleEvenly, gocv.InterpolationArea)
	i.Mat.Close()
	i.Mat = dstMat
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
