package tensor

import (
	"bufio"
	"image"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"gocv.io/x/gocv"

	"github.com/jonoton/go-cuda"
	"github.com/jonoton/go-videosource"
	"github.com/jonoton/scout/runtime"
)

const fileLocation = "data/tensor"

// Tensor detects objects within images
type Tensor struct {
	Name                    string
	Skip                    bool
	forceCpu                bool
	padding                 int
	modelFile               string
	configFile              string
	descFile                string
	backend                 gocv.NetBackendType
	target                  gocv.NetTargetType
	scaleWidth              int
	minConfidencePercentage int
	minMotionFrames         int
	minPercentage           int
	maxPercentage           int
	minOverlapPercentage    int
	sameOverlapPercentage   int
	allowedList             []string
	highlightColor          string
	highlightThickness      int
}

// NewTensor creates a new Tensor
func NewTensor(name string) *Tensor {
	// check cuda available
	backend := gocv.NetBackendDefault
	target := gocv.NetTargetCPU
	if cuda.HasCudaInstalled() {
		backend = gocv.NetBackendCUDA
		target = gocv.NetTargetCUDA
	}

	t := &Tensor{
		Name:                    name,
		forceCpu:                false,
		padding:                 0,
		modelFile:               "frozen_inference_graph.pb",
		configFile:              "ssd_mobilenet_v1_coco_2017_11_17.pbtxt",
		descFile:                "coco.names",
		backend:                 backend,
		target:                  target,
		scaleWidth:              320,
		minConfidencePercentage: 50,
		minMotionFrames:         1,
		minPercentage:           2,
		maxPercentage:           50,
		minOverlapPercentage:    75,
		sameOverlapPercentage:   85,
		allowedList:             make([]string, 0),
		highlightColor:          "blue",
		highlightThickness:      3,
	}
	return t
}

// SetConfig on tensor
func (t *Tensor) SetConfig(config *Config) {
	if config != nil {
		t.Skip = config.Skip
		t.forceCpu = config.ForceCpu
		if t.forceCpu {
			t.backend = gocv.NetBackendDefault
			t.target = gocv.NetTargetCPU
		}
		if config.Padding > 0 {
			t.padding = config.Padding
		}
		if config.ModelFile != "" {
			t.modelFile = config.ModelFile
		}
		if config.ConfigFile != "" {
			t.configFile = config.ConfigFile
		}
		if config.DescFile != "" {
			t.descFile = config.DescFile
		}
		if config.ScaleWidth < 0 || 0 < config.ScaleWidth {
			t.scaleWidth = config.ScaleWidth
		}
		if config.MinConfidencePercentage > 0 {
			t.minConfidencePercentage = config.MinConfidencePercentage
		}
		if config.MinMotionFrames > 0 {
			t.minMotionFrames = config.MinMotionFrames
		}
		if config.MinPercentage >= 0 {
			t.minPercentage = config.MinPercentage
		}
		if config.MaxPercentage > 0 {
			t.maxPercentage = config.MaxPercentage
		}
		if config.MinOverlapPercentage > 0 {
			t.minOverlapPercentage = config.MinOverlapPercentage
		}
		if config.SameOverlapPercentage > 0 {
			t.sameOverlapPercentage = config.SameOverlapPercentage
		}
		if len(config.AllowedList) > 0 {
			t.allowedList = config.AllowedList
		}
		if config.HighlightColor != "" {
			t.highlightColor = config.HighlightColor
		}
		if config.HighlightThickness > 0 {
			t.highlightThickness = config.HighlightThickness
		}
	}
}

// Run starts the tensor detection process
func (t *Tensor) Run(input <-chan videosource.ProcessedImage) <-chan videosource.ProcessedImage {
	r := make(chan videosource.ProcessedImage)
	go func() {
		defer func() {
			// recover from panic if one occured
			if recover() != nil {
				log.Errorln("Recovered from panic in tensor for", t.Name)
			}
		}()
		defer close(r)
		modelFile := runtime.GetRuntimeDirectory(fileLocation) + t.modelFile
		configFile := runtime.GetRuntimeDirectory(fileLocation) + t.configFile
		descFile := runtime.GetRuntimeDirectory(fileLocation) + t.descFile
		net := gocv.ReadNet(modelFile, configFile)
		if net.Empty() {
			log.Printf("Error reading network model from : %v %v for %s\n", modelFile, configFile, t.Name)
			return
		}

		targetName := "Unknown"
		if t.target == gocv.NetTargetCUDA {
			targetName = "CUDA"
		} else if t.target == gocv.NetTargetCPU {
			targetName = "CPU"
		}
		if err := net.SetPreferableBackend(gocv.NetBackendType(t.backend)); err != nil {
			net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendDefault))
			net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))
			targetName = "CPU"
		}
		if err := net.SetPreferableTarget(gocv.NetTargetType(t.target)); err != nil {
			net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendDefault))
			net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))
			targetName = "CPU"
		}

		var descriptions []string
		if t.descFile != "" {
			descs, err := readDescriptions(descFile)
			if err != nil {
				log.Printf("Error reading descriptions file: %v for %s\n", t.descFile, t.Name)
				return
			}
			descriptions = descs
		}

		log.Infof("Tensor %s using %s and %s with %s for %s\n", targetName, modelFile, configFile, descFile, t.Name)

		var ratio float64
		var mean gocv.Scalar
		var swapRGB bool

		ratio = 1.0 / 127.5
		mean = gocv.NewScalar(127.5, 127.5, 127.5, 0)
		swapRGB = true

		motionFrames := 0
		for cur := range input {
			result := cur
			if t.Skip || !cur.HasMotion() {
				motionFrames = 0
				r <- result
				continue
			}
			motionFrames++
			if motionFrames < t.minMotionFrames {
				r <- result
				continue
			}

			origWidth := cur.Original.Width()
			scaleWidth := t.scaleWidth
			if t.scaleWidth <= 0 {
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

			minConfidence := float32(t.minConfidencePercentage) / float32(100)
			minimumArea := cur.Original.Height() * cur.Original.Width() * t.minPercentage / 100
			maximumArea := cur.Original.Height() * cur.Original.Width() * t.maxPercentage / 100
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
					classID := int(prob.GetFloatAt(0, i+1))
					desc := ""
					if classID > 0 && classID <= len(descriptions) {
						desc = descriptions[classID-1]
					}
					descInclusive := false
					if len(t.allowedList) == 0 {
						descInclusive = true
					}
					for _, cur := range t.allowedList {
						if desc == cur {
							descInclusive = true
							break
						}
					}
					if !descInclusive {
						continue
					}
					rect := image.Rect(left, top, right, bottom)
					scaledRect := videosource.RectScale(cur.Original, rect, scaleRatio)
					rectArea := scaledRect.Dx() * scaledRect.Dy()
					if rectArea < minimumArea || rectArea > maximumArea {
						continue
					}
					finalRect := videosource.RectPadded(cur.Original, scaledRect, t.padding)
					withinMotion := false
					for _, curMotion := range cur.Motions {
						if fPercent, _ := videosource.RectOverlap(finalRect, curMotion.Rect); fPercent >= t.minOverlapPercentage {
							withinMotion = true
							break
						}
					}
					if !withinMotion {
						continue
					}
					newObj := true
					confidencePercent := int(confidence * 100)
					for oIndex, curObj := range result.Objects {
						objRect := curObj.Rect
						if fPercent, oPercent := videosource.RectOverlap(finalRect, objRect); fPercent >= t.sameOverlapPercentage && oPercent >= t.sameOverlapPercentage {
							newObj = false
							if (curObj.Percentage < confidencePercent) ||
								(strings.ToLower(desc) == "person" && strings.ToLower(curObj.Description) != "person") {
								// replace object with better
								curObj.Cleanup()
								objectInfo := videosource.NewObjectInfo(finalRect, *videosource.NewColorThickness(t.highlightColor, t.highlightThickness))
								objectInfo.Description = strings.Title(strings.ToLower(desc))
								objectInfo.Percentage = confidencePercent
								result.Objects[oIndex] = *objectInfo
								break
							}
						}
					}
					if !newObj {
						continue
					}
					objectInfo := videosource.NewObjectInfo(finalRect, *videosource.NewColorThickness(t.highlightColor, t.highlightThickness))
					objectInfo.Description = strings.Title(strings.ToLower(desc))
					objectInfo.Percentage = confidencePercent
					result.Objects = append(result.Objects, *objectInfo)
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

// readDescriptions reads the descriptions from a file
// and returns a slice of its lines.
func readDescriptions(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()
	return lines, scanner.Err()
}
