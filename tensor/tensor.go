package tensor

import (
	"bufio"
	"image"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"gocv.io/x/gocv"

	"github.com/jonoton/go-cuda"
	"github.com/jonoton/go-runtime"
	"github.com/jonoton/go-videosource"
)

const fileLocationData = "data/tensor"
const fileLocationDotData = ".data/tensor"

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func getModelPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	pathData := filepath.Join(runtime.GetRuntimeDirectory(fileLocationData), filename)
	if fileExists(pathData) {
		return pathData
	}
	pathDotData := filepath.Join(runtime.GetRuntimeDirectory(fileLocationDotData), filename)
	if fileExists(pathDotData) {
		return pathDotData
	}
	// Fallback to old behavior if neither exists, though it will likely fail
	return pathData
}

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
	priorityList            []PriorityItem
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
		priorityList:            []PriorityItem{{Description: "person", MinConfidencePercentage: 50}},
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
		if config.ScaleWidth > 0 {
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
		if len(config.PriorityList) > 0 {
			t.priorityList = config.PriorityList
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
		defer close(r)
		defer func() {
			// recover from panic if one occurred
			if recover() != nil {
				log.Errorln("Recovered from panic in tensor for", t.Name)
			}
		}()
		modelFile := getModelPath(t.modelFile)
		configFile := getModelPath(t.configFile)
		descFile := ""
		if t.descFile != "" {
			descFile = getModelPath(t.descFile)
		}
		net := gocv.ReadNet(modelFile, configFile)
		defer net.Close()
		if net.Empty() {
			log.Errorf("Error reading network model from : %v %v for %s", modelFile, configFile, t.Name)
			return
		}

		targetName := "Unknown"
		switch t.target {
		case gocv.NetTargetCUDA:
			targetName = "CUDA"
		case gocv.NetTargetCPU:
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
				log.Errorf("Error reading descriptions file: %v for %s", t.descFile, t.Name)
				return
			}
			descriptions = descs
		}

		log.Infof("Tensor %s using %s and %s with %s for %s", targetName, modelFile, configFile, descFile, t.Name)

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
					for _, allowed := range t.allowedList {
						if desc == allowed {
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
						if fPercent, oPercent := videosource.RectOverlap(finalRect, objRect); (strings.EqualFold(desc, curObj.Description) && (fPercent >= t.sameOverlapPercentage || oPercent >= t.sameOverlapPercentage)) ||
						(!strings.EqualFold(desc, curObj.Description) && fPercent >= t.sameOverlapPercentage && oPercent >= t.sameOverlapPercentage) {
							newObj = false
							descPri, descMinConf := t.priorityInfo(desc)
							curPri, _ := t.priorityInfo(curObj.Description)
							if (curObj.Percentage < confidencePercent) ||
								(descPri >= 0 && (curPri < 0 || descPri < curPri) && confidencePercent >= descMinConf) {
								// replace object with better
								curObj.Cleanup()
								objectInfo := videosource.NewObjectInfo(finalRect, *videosource.NewColorThickness(t.highlightColor, t.highlightThickness))
								objectInfo.Description = toTitle(desc)
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
					objectInfo.Description = toTitle(desc)
					objectInfo.Percentage = confidencePercent
					result.Objects = append(result.Objects, *objectInfo)
				}
			}
			scaledImg.Cleanup()
			prob.Close()
			blob.Close()

			r <- result
		}

	}()
	return r
}

// priorityInfo returns the index of the description in the priority list
// (lower index = higher priority) and its minimum confidence threshold.
// Returns -1, 0 if not found.
func (t *Tensor) priorityInfo(desc string) (index int, minConfidence int) {
	for i, p := range t.priorityList {
		if strings.EqualFold(desc, p.Description) {
			return i, p.MinConfidencePercentage
		}
	}
	return -1, 0
}

// toTitle converts a string to title case (first letter of each word capitalized).
// Replaces the deprecated strings.Title.
func toTitle(s string) string {
	s = strings.ToLower(s)
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// readDescriptions reads the descriptions from a file
// and returns a slice of its lines.
func readDescriptions(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
