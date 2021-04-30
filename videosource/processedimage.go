package videosource

import (
	"image"
	"time"
)

// ObjectInfo contains the object information
type ObjectInfo struct {
	Object      Image
	Description string
	Percentage  int
}

// NewObjectInfo creates a new ObjectInfo
func NewObjectInfo(img Image) *ObjectInfo {
	o := &ObjectInfo{
		Object:      img,
		Description: "",
		Percentage:  0,
	}
	return o
}

// Ref will reference the ObjectInfo and underlying SharedMat
func (o *ObjectInfo) Ref() *ObjectInfo {
	o.Object.Ref()
	return o
}

// Clone will clone the ObjectInfo
func (o *ObjectInfo) Clone() *ObjectInfo {
	c := &ObjectInfo{
		Object:      *o.Object.Clone(),
		Description: o.Description,
		Percentage:  o.Percentage,
	}
	return c
}

// Cleanup will cleanup the ObjectInfo
func (o *ObjectInfo) Cleanup() {
	o.Object.Cleanup()
	o.Description = ""
	o.Percentage = 0
}

func getHighestObjectPercentage(objs []ObjectInfo) (result int) {
	for _, cur := range objs {
		if result < cur.Percentage {
			result = cur.Percentage
		}
	}
	return
}

// FaceInfo contains the face information
type FaceInfo struct {
	Face       Image
	Percentage int
}

// NewFaceInfo creates a new FaceInfo
func NewFaceInfo(img Image) *FaceInfo {
	f := &FaceInfo{
		Face:       img,
		Percentage: 0,
	}
	return f
}

// Ref will reference the FaceInfo and underlying SharedMat
func (f *FaceInfo) Ref() *FaceInfo {
	f.Face.Ref()
	return f
}

// Clone will clone the FaceInfo
func (f *FaceInfo) Clone() *FaceInfo {
	c := &FaceInfo{
		Face:       *f.Face.Clone(),
		Percentage: f.Percentage,
	}
	return c
}

// Cleanup will cleanup the FaceInfo
func (f *FaceInfo) Cleanup() {
	f.Face.Cleanup()
	f.Percentage = 0
}

func getHighestFacePercentage(faces []FaceInfo) (result int) {
	for _, cur := range faces {
		if result < cur.Percentage {
			result = cur.Percentage
		}
	}
	return
}

// ProcessedImage is the result of running through the processes
type ProcessedImage struct {
	Original          Image
	HighlightedMotion Image
	HighlightedObject Image
	HighlightedFace   Image
	Motions           []Image
	MotionRects       []image.Rectangle
	Objects           []ObjectInfo
	ObjectRects       []image.Rectangle
	Faces             []FaceInfo
	FaceRects         []image.Rectangle
}

// NewProcessedImage creates a new ProcessedImage
func NewProcessedImage(original Image) *ProcessedImage {
	p := &ProcessedImage{
		Original:          original,
		HighlightedMotion: Image{},
		HighlightedObject: Image{},
		HighlightedFace:   Image{},
		Motions:           make([]Image, 0),
		MotionRects:       make([]image.Rectangle, 0),
		Objects:           make([]ObjectInfo, 0),
		ObjectRects:       make([]image.Rectangle, 0),
		Faces:             make([]FaceInfo, 0),
		FaceRects:         make([]image.Rectangle, 0),
	}
	return p
}

// Ref will reference the ProcessedImage and underlying SharedMats
func (p *ProcessedImage) Ref() *ProcessedImage {
	p.Original.Ref()
	p.HighlightedMotion.Ref()
	p.HighlightedObject.Ref()
	p.HighlightedFace.Ref()
	for _, cur := range p.Motions {
		cur.Ref()
	}
	for _, cur := range p.Objects {
		cur.Ref()
	}
	for _, cur := range p.Faces {
		cur.Ref()
	}
	return p
}

// Clone will clone the ProcessedImage
func (p *ProcessedImage) Clone() *ProcessedImage {
	c := &ProcessedImage{
		Original:          *p.Original.Clone(),
		HighlightedMotion: *p.HighlightedMotion.Clone(),
		HighlightedObject: *p.HighlightedObject.Clone(),
		HighlightedFace:   *p.HighlightedFace.Clone(),
		Motions:           make([]Image, 0),
		MotionRects:       p.MotionRects,
		Objects:           make([]ObjectInfo, 0),
		ObjectRects:       p.ObjectRects,
		Faces:             make([]FaceInfo, 0),
		FaceRects:         p.FaceRects,
	}
	for _, cur := range p.Motions {
		c.Motions = append(c.Motions, *cur.Clone())
	}
	for _, cur := range p.Objects {
		c.Objects = append(c.Objects, *cur.Clone())
	}
	for _, cur := range p.Faces {
		c.Faces = append(c.Faces, *cur.Clone())
	}
	return c
}

// Cleanup will cleanup the ProcessedImage
func (p *ProcessedImage) Cleanup() {
	p.Original.Cleanup()
	p.HighlightedMotion.Cleanup()
	p.HighlightedObject.Cleanup()
	p.HighlightedFace.Cleanup()
	for _, cur := range p.Motions {
		cur.Cleanup()
	}
	p.Motions = make([]Image, 0)
	p.MotionRects = make([]image.Rectangle, 0)
	for _, cur := range p.Objects {
		cur.Cleanup()
	}
	p.Objects = make([]ObjectInfo, 0)
	p.ObjectRects = make([]image.Rectangle, 0)
	for _, cur := range p.Faces {
		cur.Cleanup()
	}
	p.Faces = make([]FaceInfo, 0)
	p.FaceRects = make([]image.Rectangle, 0)
}

// ProcessedImageByCreatedTime sorting ascending order
type ProcessedImageByCreatedTime []ProcessedImage

func (b ProcessedImageByCreatedTime) Len() int { return len(b) }
func (b ProcessedImageByCreatedTime) Less(i, j int) bool {
	return b[i].Original.CreatedTime.Before(b[j].Original.CreatedTime)
}
func (b ProcessedImageByCreatedTime) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// ProcessedImageByObjLen sorting descending order
type ProcessedImageByObjLen []ProcessedImage

func (b ProcessedImageByObjLen) Len() int { return len(b) }
func (b ProcessedImageByObjLen) Less(i, j int) bool {
	return len(b[i].Objects) > len(b[j].Objects)
}
func (b ProcessedImageByObjLen) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// ProcessedImageByObjPercent sorting descending order
type ProcessedImageByObjPercent []ProcessedImage

func (b ProcessedImageByObjPercent) Len() int { return len(b) }
func (b ProcessedImageByObjPercent) Less(i, j int) bool {
	return getHighestObjectPercentage(b[i].Objects) > getHighestObjectPercentage(b[j].Objects)
}
func (b ProcessedImageByObjPercent) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// ProcessedImageByFaceLen sorting descending order
type ProcessedImageByFaceLen []ProcessedImage

func (b ProcessedImageByFaceLen) Len() int { return len(b) }
func (b ProcessedImageByFaceLen) Less(i, j int) bool {
	return len(b[i].Faces) > len(b[j].Faces)
}
func (b ProcessedImageByFaceLen) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// ProcessedImageByFacePercent sorting descending order
type ProcessedImageByFacePercent []ProcessedImage

func (b ProcessedImageByFacePercent) Len() int { return len(b) }
func (b ProcessedImageByFacePercent) Less(i, j int) bool {
	return getHighestFacePercentage(b[i].Faces) > getHighestFacePercentage(b[j].Faces)
}
func (b ProcessedImageByFacePercent) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// ProcessedImageFpsChan will notify caller via ProcessedImage channel at given fps
type ProcessedImageFpsChan struct {
	outFps     int
	streamChan chan ProcessedImage
	done       chan bool
}

// NewProcessedImageFpsChan creates a new ProcessedImageFpsChan
func NewProcessedImageFpsChan(outFps int) *ProcessedImageFpsChan {
	p := &ProcessedImageFpsChan{
		outFps:     outFps,
		streamChan: make(chan ProcessedImage),
		done:       make(chan bool),
	}
	return p
}

// Start runs the channel
func (p *ProcessedImageFpsChan) Start() chan ProcessedImage {
	outChan := make(chan ProcessedImage)
	go func() {
		var curImage ProcessedImage
		writeTick := time.NewTicker(time.Duration(1000/p.outFps) * time.Millisecond)
	Loop:
		for {
			select {
			case img, ok := <-p.streamChan:
				if !ok {
					img.Cleanup()
					break Loop
				}
				curImage.Cleanup()
				curImage = img
			case <-writeTick.C:
				outChan <- curImage
				curImage = ProcessedImage{}
			}
		}
		writeTick.Stop()
		curImage.Cleanup()
		close(outChan)
		close(p.done)
	}()
	return outChan
}

// Send ProcessedImage to channel
func (p *ProcessedImageFpsChan) Send(img ProcessedImage) {
	p.streamChan <- img
}

// Close notified by caller that input stream is done/closed
func (p *ProcessedImageFpsChan) Close() {
	close(p.streamChan)
}

// Wait until done
func (p *ProcessedImageFpsChan) Wait() {
	<-p.done
}
