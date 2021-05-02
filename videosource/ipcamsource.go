package videosource

import (
	log "github.com/sirupsen/logrus"

	"gocv.io/x/gocv"
)

// IPCamSource is a ipcam source
type IPCamSource struct {
	BaseVideo
	url              string
	gocvVideoCapture *gocv.VideoCapture
}

// NewIPCamSource creates a new IPCamSource
func NewIPCamSource(name string, url string) VideoSource {
	i := &IPCamSource{
		BaseVideo:        *NewBaseVideo(name),
		url:              url,
		gocvVideoCapture: nil,
	}
	return i
}

// Initialize implements interface
func (i *IPCamSource) Initialize() (ok bool) {
	gocvVideoCapture, err := gocv.VideoCaptureFile(i.url)
	if err != nil {
		log.Warnf("Could not open video capture url: %s\n", i.url)
		return
	}
	i.gocvVideoCapture = gocvVideoCapture
	ok = true
	return
}

// Cleanup implements interface
func (i *IPCamSource) Cleanup() {
	if i.gocvVideoCapture != nil {
		i.gocvVideoCapture.Close()
	}
}

// ReadImage implements interface
func (i *IPCamSource) ReadImage() (done bool, image Image) {
	if i.gocvVideoCapture == nil {
		done = true
		return
	}
	mat := gocv.NewMat()
	done = !i.gocvVideoCapture.Read(&mat)
	image = *NewImage(mat)
	return
}
