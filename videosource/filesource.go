package videosource

import (
	log "github.com/sirupsen/logrus"

	"gocv.io/x/gocv"
)

// FileSource is a file source
type FileSource struct {
	BaseVideo
	filename         string
	gocvVideoCapture *gocv.VideoCapture
}

// NewFileSource creates a new FileSource
func NewFileSource(name string, filename string) VideoSource {
	v := &FileSource{
		BaseVideo:        *NewBaseVideo(name),
		filename:         filename,
		gocvVideoCapture: nil,
	}
	return v
}

// Initialize implements interface
func (f *FileSource) Initialize() (ok bool) {
	gocvVideoCapture, err := gocv.VideoCaptureFile(f.filename)
	if err != nil {
		log.Warnf("Could not open video capture file: %s\n", f.filename)
		return
	}
	f.gocvVideoCapture = gocvVideoCapture
	ok = true
	return
}

// Cleanup implements interface
func (f *FileSource) Cleanup() {
	if f.gocvVideoCapture != nil {
		f.gocvVideoCapture.Close()
	}
}

// ReadImage implements interface
func (f *FileSource) ReadImage() (done bool, image Image) {
	if f.gocvVideoCapture == nil {
		done = true
		return
	}
	mat := gocv.NewMat()
	done = !f.gocvVideoCapture.Read(&mat)
	image = *NewImage(mat)
	return
}
