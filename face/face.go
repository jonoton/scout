package face

import (
	"image"

	log "github.com/sirupsen/logrus"

	"github.com/jonoton/go-cuda"
	"github.com/jonoton/go-videosource"
	"github.com/jonoton/scout/runtime"
	"gocv.io/x/gocv"
)

const fileLocation = "data/face"

// Face detects faces within images
type Face struct {
	Name                    string
	Skip                    bool
	forceCpu                bool
	padding                 int
	modelFile               string
	configFile              string
	backend                 gocv.NetBackendType
	target                  gocv.NetTargetType
	scaleWidth              int
	minConfidencePercentage int
	maxPercentage           int
	minOverlapPercentage    int
	highlightColor          string
	highlightThickness      int
}

// NewFace creates a new Face
func NewFace(name string) *Face {
	// check cuda available
	backend := gocv.NetBackendDefault
	target := gocv.NetTargetCPU
	if cuda.HasCudaInstalled() {
		backend = gocv.NetBackendCUDA
		target = gocv.NetTargetCUDA
	}
	f := &Face{
		Name:                    name,
		forceCpu:                false,
		padding:                 0,
		modelFile:               "res10_300x300_ssd_iter_140000.caffemodel",
		configFile:              "deploy.prototxt",
		backend:                 backend,
		target:                  target,
		scaleWidth:              320,
		minConfidencePercentage: 50,
		maxPercentage:           50,
		minOverlapPercentage:    75,
		highlightColor:          "green",
		highlightThickness:      3,
	}
	return f
}

// SetConfig on face
func (f *Face) SetConfig(config *Config) {
	if config != nil {
		f.Skip = config.Skip
		f.forceCpu = config.ForceCpu
		if f.forceCpu {
			f.backend = gocv.NetBackendDefault
			f.target = gocv.NetTargetCPU
		}
		if config.Padding != 0 {
			f.padding = config.Padding
		}
		if config.ModelFile != "" {
			f.modelFile = config.ModelFile
		}
		if config.ConfigFile != "" {
			f.configFile = config.ConfigFile
		}
		if config.ScaleWidth < 0 || 0 < config.ScaleWidth {
			f.scaleWidth = config.ScaleWidth
		}
		if config.MinConfidencePercentage != 0 {
			f.minConfidencePercentage = config.MinConfidencePercentage
		}
		if config.MaxPercentage != 0 {
			f.maxPercentage = config.MaxPercentage
		}
		if config.MinOverlapPercentage != 0 {
			f.minOverlapPercentage = config.MinOverlapPercentage
		}
		if config.HighlightColor != "" {
			f.highlightColor = config.HighlightColor
		}
		if config.HighlightThickness != 0 {
			f.highlightThickness = config.HighlightThickness
		}
	}
}

// Run starts the face detection process
func (f *Face) Run(input <-chan videosource.ProcessedImage) <-chan videosource.ProcessedImage {
	r := make(chan videosource.ProcessedImage)
	go func() {
		defer func() {
			// recover from panic if one occured
			if recover() != nil {
				log.Errorln("Recovered from panic in face for", f.Name)
			}
		}()
		defer close(r)
		modelFile := runtime.GetRuntimeDirectory(fileLocation) + f.modelFile
		configFile := runtime.GetRuntimeDirectory(fileLocation) + f.configFile
		net := gocv.ReadNet(modelFile, configFile)
		if net.Empty() {
			log.Printf("Error reading network model from : %v %v for %s\n", modelFile, configFile, f.Name)
			return
		}

		targetName := "Unknown"
		if f.target == gocv.NetTargetCUDA {
			targetName = "CUDA"
		} else if f.target == gocv.NetTargetCPU {
			targetName = "CPU"
		}
		if err := net.SetPreferableBackend(gocv.NetBackendType(f.backend)); err != nil {
			net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendDefault))
			net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))
			targetName = "CPU"
		}
		if err := net.SetPreferableTarget(gocv.NetTargetType(f.target)); err != nil {
			net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendDefault))
			net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))
			targetName = "CPU"
		}

		log.Infof("Face %s using %s and %s for %s\n", targetName, modelFile, configFile, f.Name)

		var ratio float64
		var mean gocv.Scalar
		var swapRGB bool

		ratio = 1.0
		mean = gocv.NewScalar(104, 177, 123, 0)
		swapRGB = false

		for cur := range input {
			result := cur
			if f.Skip || !cur.HasObject() {
				r <- result
				continue
			}
			origWidth := cur.Original.Width()
			scaleWidth := f.scaleWidth
			if f.scaleWidth <= 0 {
				scaleWidth = origWidth
			}
			scaleRatio := float64(origWidth) / float64(scaleWidth)
			scaledImg := cur.Original.ScaleToWidth(scaleWidth)
			tmpMat := scaledImg.SharedMat.Mat
			matType := tmpMat.Type()
			// need to convert for blob usage
			tmpMat.ConvertTo(&tmpMat, gocv.MatTypeCV32F)
			// convert image Mat to 300x300 blob that the object detector can analyze
			blob := gocv.BlobFromImage(tmpMat, ratio, image.Pt(300, 300), mean, swapRGB, false)
			// feed the blob into the detector
			net.SetInput(blob, "")
			// run a forward pass thru the network
			prob := net.Forward("")
			tmpMat.ConvertTo(&tmpMat, matType)

			minConfidence := float32(f.minConfidencePercentage) / float32(100)
			maximumArea := cur.Original.Height() * cur.Original.Width() * f.maxPercentage / 100
			// the results from the detector network,
			// which produces an output blob with a shape 1x1xNx7
			// where N is the number of detections, and each detection
			// is a vector of float values
			// [batchId, classId, confidence, left, top, right, bottom]
			for i := 0; i < prob.Total(); i += 7 {
				confidence := prob.GetFloatAt(0, i+2)
				if confidence > minConfidence {
					left := int(prob.GetFloatAt(0, i+3) * float32(tmpMat.Cols()))
					top := int(prob.GetFloatAt(0, i+4) * float32(tmpMat.Rows()))
					right := int(prob.GetFloatAt(0, i+5) * float32(tmpMat.Cols()))
					bottom := int(prob.GetFloatAt(0, i+6) * float32(tmpMat.Rows()))
					rect := image.Rect(left, top, right, bottom)
					scaledRect := videosource.RectScale(cur.Original, rect, scaleRatio)
					rectArea := scaledRect.Dx() * scaledRect.Dy()
					if rectArea > maximumArea {
						continue
					}
					paddedRect := videosource.RectPadded(cur.Original, scaledRect, f.padding)
					finalRect := videosource.RectSquare(cur.Original, paddedRect)
					withinObj := false
					for _, curObj := range cur.Objects {
						if fPercent, _ := videosource.RectOverlap(finalRect, curObj.Rect); fPercent >= f.minOverlapPercentage {
							withinObj = true
							break
						}
					}
					if !withinObj {
						continue
					}
					newFace := true
					confidencePercent := int(confidence * 100)
					for fIndex, curFace := range result.Faces {
						faceRect := curFace.Rect
						if finalRect.Overlaps(faceRect) {
							newFace = false
							if result.Faces[fIndex].Percentage < confidencePercent {
								// replace face with better percentage
								result.Faces[fIndex].Cleanup()
								faceInfo := videosource.NewFaceInfo(finalRect, *videosource.NewColorThickness(f.highlightColor, f.highlightThickness))
								faceInfo.Percentage = confidencePercent
								result.Faces[fIndex] = *faceInfo
								break
							}
						}
					}
					if !newFace {
						continue
					}
					faceInfo := videosource.NewFaceInfo(finalRect, *videosource.NewColorThickness(f.highlightColor, f.highlightThickness))
					faceInfo.Percentage = confidencePercent
					result.Faces = append(result.Faces, *faceInfo)
				}
			}
			scaledImg.Cleanup()
			prob.Close()
			blob.Close()

			r <- result
		}
		net.Close()
	}()
	return r
}
