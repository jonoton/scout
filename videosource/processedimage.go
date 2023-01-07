package videosource

import (
	"image"
	"time"

	"gocv.io/x/gocv"
)

// MotionInfo contains the motion information
type MotionInfo struct {
	Rect          image.Rectangle
	HighlightInfo ColorThickness
}

// NewMotionInfo creates a new MotionInfo
func NewMotionInfo(rect image.Rectangle, colorThickness ColorThickness) *MotionInfo {
	m := &MotionInfo{
		Rect:          rect,
		HighlightInfo: colorThickness,
	}
	return m
}

// Ref will reference the MotionInfo and underlying SharedMat
func (m *MotionInfo) Ref() *MotionInfo {
	return m
}

// Clone will clone the MotionInfo
func (m *MotionInfo) Clone() *MotionInfo {
	c := &MotionInfo{
		Rect:          m.Rect,
		HighlightInfo: m.HighlightInfo,
	}
	return c
}

// Cleanup will cleanup the MotionInfo
func (m *MotionInfo) Cleanup() {
	m.Rect = image.Rectangle{}
	m.HighlightInfo = ColorThickness{}
}

// ObjectInfo contains the object information
type ObjectInfo struct {
	Rect          image.Rectangle
	Description   string
	Percentage    int
	HighlightInfo ColorThickness
}

// NewObjectInfo creates a new ObjectInfo
func NewObjectInfo(rect image.Rectangle, colorThickness ColorThickness) *ObjectInfo {
	o := &ObjectInfo{
		Rect:          rect,
		Description:   "",
		Percentage:    0,
		HighlightInfo: colorThickness,
	}
	return o
}

// Ref will reference the ObjectInfo and underlying SharedMat
func (o *ObjectInfo) Ref() *ObjectInfo {
	return o
}

// Clone will clone the ObjectInfo
func (o *ObjectInfo) Clone() *ObjectInfo {
	c := &ObjectInfo{
		Rect:          o.Rect,
		Description:   o.Description,
		Percentage:    o.Percentage,
		HighlightInfo: o.HighlightInfo,
	}
	return c
}

// Cleanup will cleanup the ObjectInfo
func (o *ObjectInfo) Cleanup() {
	o.Rect = image.Rectangle{}
	o.Description = ""
	o.Percentage = 0
	o.HighlightInfo = ColorThickness{}
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
	Rect          image.Rectangle
	Percentage    int
	HighlightInfo ColorThickness
}

// NewFaceInfo creates a new FaceInfo
func NewFaceInfo(rect image.Rectangle, colorThickness ColorThickness) *FaceInfo {
	f := &FaceInfo{
		Rect:          rect,
		Percentage:    0,
		HighlightInfo: colorThickness,
	}
	return f
}

// Ref will reference the FaceInfo and underlying SharedMat
func (f *FaceInfo) Ref() *FaceInfo {
	return f
}

// Clone will clone the FaceInfo
func (f *FaceInfo) Clone() *FaceInfo {
	c := &FaceInfo{
		Rect:          f.Rect,
		Percentage:    f.Percentage,
		HighlightInfo: f.HighlightInfo,
	}
	return c
}

// Cleanup will cleanup the FaceInfo
func (f *FaceInfo) Cleanup() {
	f.Rect = image.Rectangle{}
	f.Percentage = 0
	f.HighlightInfo = ColorThickness{}
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
	Original Image
	Motions  []MotionInfo
	Objects  []ObjectInfo
	Faces    []FaceInfo
}

// NewProcessedImage creates a new ProcessedImage
func NewProcessedImage(original Image) *ProcessedImage {
	p := &ProcessedImage{
		Original: original,
		Motions:  make([]MotionInfo, 0),
		Objects:  make([]ObjectInfo, 0),
		Faces:    make([]FaceInfo, 0),
	}
	return p
}

func (p *ProcessedImage) HasMotion() bool {
	return len(p.Motions) > 0
}
func (p *ProcessedImage) HasObject() bool {
	return len(p.Objects) > 0
}
func (p *ProcessedImage) HasFace() bool {
	return len(p.Faces) > 0
}

func (p *ProcessedImage) HighlightedMotion() *Image {
	highlightedImage := p.Original.Clone()
	highlightedMat := highlightedImage.SharedMat.Mat
	for _, cur := range p.Motions {
		gocv.Rectangle(&highlightedMat, cur.Rect, cur.HighlightInfo.Color.GetRGBA(), cur.HighlightInfo.Thickness)
	}
	return highlightedImage
}
func (p *ProcessedImage) HighlightedObject() *Image {
	highlightedImage := p.Original.Clone()
	highlightedMat := highlightedImage.SharedMat.Mat
	for _, cur := range p.Objects {
		gocv.Rectangle(&highlightedMat, cur.Rect, cur.HighlightInfo.Color.GetRGBA(), cur.HighlightInfo.Thickness)
	}
	return highlightedImage
}
func (p *ProcessedImage) HighlightedFace() *Image {
	highlightedImage := p.Original.Clone()
	highlightedMat := highlightedImage.SharedMat.Mat
	for _, cur := range p.Faces {
		gocv.Rectangle(&highlightedMat, cur.Rect, cur.HighlightInfo.Color.GetRGBA(), cur.HighlightInfo.Thickness)
	}
	return highlightedImage
}
func (p *ProcessedImage) HighlightedAll() *Image {
	highlightedImage := p.Original.Clone()
	highlightedMat := highlightedImage.SharedMat.Mat
	for _, cur := range p.Motions {
		gocv.Rectangle(&highlightedMat, cur.Rect, cur.HighlightInfo.Color.GetRGBA(), cur.HighlightInfo.Thickness)
	}
	for _, cur := range p.Objects {
		gocv.Rectangle(&highlightedMat, cur.Rect, cur.HighlightInfo.Color.GetRGBA(), cur.HighlightInfo.Thickness)
	}
	for _, cur := range p.Faces {
		gocv.Rectangle(&highlightedMat, cur.Rect, cur.HighlightInfo.Color.GetRGBA(), cur.HighlightInfo.Thickness)
	}
	return highlightedImage
}

func (p ProcessedImage) Motion(index int) *Image {
	if index >= 0 && index < len(p.Motions) {
		r := p.Original.GetRegion(p.Motions[index].Rect)
		return &r
	}
	return &Image{}
}
func (p ProcessedImage) Object(index int) *Image {
	if index >= 0 && index < len(p.Objects) {
		r := p.Original.GetRegion(p.Objects[index].Rect)
		return &r
	}
	return &Image{}
}
func (p ProcessedImage) Face(index int) *Image {
	if index >= 0 && index < len(p.Faces) {
		r := p.Original.GetRegion(p.Faces[index].Rect)
		return &r
	}
	return &Image{}
}

// Ref will reference the ProcessedImage and underlying SharedMats
func (p *ProcessedImage) Ref() *ProcessedImage {
	p.Original.Ref()
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
		Original: *p.Original.Clone(),
		Motions:  make([]MotionInfo, 0),
		Objects:  make([]ObjectInfo, 0),
		Faces:    make([]FaceInfo, 0),
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
	for _, cur := range p.Motions {
		cur.Cleanup()
	}
	p.Motions = make([]MotionInfo, 0)
	for _, cur := range p.Objects {
		cur.Cleanup()
	}
	p.Objects = make([]ObjectInfo, 0)
	for _, cur := range p.Faces {
		cur.Cleanup()
	}
	p.Faces = make([]FaceInfo, 0)
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
		var curImage *ProcessedImage
		writeTick := time.NewTicker(time.Duration(1000/p.outFps) * time.Millisecond)
	Loop:
		for {
			select {
			case img, ok := <-p.streamChan:
				if !ok {
					img.Cleanup()
					break Loop
				}
				if curImage != nil {
					curImage.Cleanup()
				}
				curImage = &img
			case <-writeTick.C:
				if curImage != nil {
					outChan <- *curImage
					curImage = nil
				}
			}
		}
		writeTick.Stop()
		if curImage != nil {
			curImage.Cleanup()
		}
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
