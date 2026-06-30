package motion

import (
	"image"

	log "github.com/sirupsen/logrus"

	"github.com/jonoton/go-videosource"
	"gocv.io/x/gocv"
)

// Motion detects motion within images
type Motion struct {
	Name                string
	Skip                bool
	padding             int
	scaleWidth          int
	minimumPercentage   int
	maximumPercentage   int
	maxMotions          int
	overloadPercent     int
	thresholdPercent    int
	noiseReduction      int
	highlightColor      string
	highlightThickness  int
	backgroundHistory   int
	backgroundThreshold int
	detectShadows       bool
	closingSize         int
	minMotionFrames     int
	mergeOverlapPercent int
	gridSize            int
	mergeDistance       int
}

// NewMotion creates a new Motion
func NewMotion(name string) *Motion {
	m := &Motion{
		Name:                name,
		padding:             0,
		scaleWidth:          320,
		minimumPercentage:   2,
		maximumPercentage:   75,
		maxMotions:          20,
		overloadPercent:     90,
		thresholdPercent:    40,
		noiseReduction:      11, // odd number
		highlightColor:      "purple",
		highlightThickness:  3,
		backgroundHistory:   500,
		backgroundThreshold: 16,
		detectShadows:       true,
		closingSize:         3,
		minMotionFrames:     1,
		mergeOverlapPercent: 20,
		gridSize:            16,
		mergeDistance:       10,
	}
	return m
}

// SetConfig on motion
func (m *Motion) SetConfig(config *Config) {
	if config != nil {
		m.Skip = config.Skip
		if config.Padding > 0 {
			m.padding = config.Padding
		}
		if config.ScaleWidth < 0 || 0 < config.ScaleWidth {
			m.scaleWidth = config.ScaleWidth
		}
		if config.MinimumPercentage >= 0 {
			m.minimumPercentage = config.MinimumPercentage
		}
		if config.MaximumPercentage > 0 {
			m.maximumPercentage = config.MaximumPercentage
		}
		if config.MaxMotions > 0 {
			m.maxMotions = config.MaxMotions
		}
		if config.OverloadPercent > 0 {
			m.overloadPercent = config.OverloadPercent
		}
		if config.ThresholdPercent > 0 {
			m.thresholdPercent = config.ThresholdPercent
		}
		if config.NoiseReduction > 0 {
			m.noiseReduction = config.NoiseReduction
			if m.noiseReduction%2 == 0 {
				m.noiseReduction++
			}
		}
		if config.HighlightColor != "" {
			m.highlightColor = config.HighlightColor
		}
		if config.HighlightThickness > 0 {
			m.highlightThickness = config.HighlightThickness
		}
		if config.BackgroundHistory > 0 {
			m.backgroundHistory = config.BackgroundHistory
		}
		if config.BackgroundThreshold > 0 {
			m.backgroundThreshold = config.BackgroundThreshold
		}
		if config.DetectShadows != nil {
			m.detectShadows = *config.DetectShadows
		}
		if config.ClosingSize > 0 {
			m.closingSize = config.ClosingSize
		}
		if config.MinMotionFrames > 0 {
			m.minMotionFrames = config.MinMotionFrames
		}
		if config.MergeOverlapPercent >= 0 {
			m.mergeOverlapPercent = config.MergeOverlapPercent
		}
		if config.GridSize > 0 {
			m.gridSize = config.GridSize
		}
		if config.MergeDistance >= 0 {
			m.mergeDistance = config.MergeDistance
		}
	}
}

// Run starts the motion detection process
func (m *Motion) Run(input <-chan videosource.Image) <-chan videosource.ProcessedImage {
	r := make(chan videosource.ProcessedImage)
	go func() {
		defer close(r)
		defer func() {
			// recover from panic if one occurred
			if recover() != nil {
				log.Errorln("Recovered from panic in motion for", m.Name)
			}
		}()
		mog2 := gocv.NewBackgroundSubtractorMOG2WithParams(m.backgroundHistory, float64(m.backgroundThreshold), m.detectShadows)
		defer mog2.Close()

		// Local state for temporal hysteresis: map of grid cell -> consecutive frames active
		cellFrames := make(map[image.Point]int)
		gridSize := m.gridSize
		if gridSize <= 0 {
			gridSize = 16
		}

		for cur := range input {
			result := *videosource.NewProcessedImage(cur)
			if m.Skip {
				r <- result
				continue
			}

			origWidth := cur.Width()
			scaleWidth := m.scaleWidth
			if m.scaleWidth <= 0 {
				scaleWidth = origWidth
			}
			scaleRatio := float64(origWidth) / float64(scaleWidth)
			scaledImg := cur.ScaleToWidth(scaleWidth)
			blurMat := gocv.NewMat()

			gocv.GaussianBlur(scaledImg.SharedMat.Mat, &blurMat, image.Pt(m.noiseReduction, m.noiseReduction), 0, 0, gocv.BorderDefault)
			matDelta := gocv.NewMat()
			matThresh := gocv.NewMat()
			// obtain foreground only
			mog2.Apply(blurMat, &matDelta)
			// threshold range is 0-255, lower is more sensitive
			threshold := 255 * m.thresholdPercent / 100
			gocv.Threshold(matDelta, &matThresh, float32(threshold), 255, gocv.ThresholdBinary)
			matDelta.Close()

			// morph close
			kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(m.closingSize, m.closingSize))
			gocv.MorphologyEx(matThresh, &matThresh, gocv.MorphClose, kernel)
			kernel.Close()

			// find contours
			contours := gocv.FindContours(matThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
			matThresh.Close()
			imgWidth := scaledImg.Width()
			imgHeight := scaledImg.Height()
			imageArea := imgHeight * imgWidth
			overloadArea := float64(imageArea * m.overloadPercent / 100)
			minimumArea := float64(imageArea * m.minimumPercentage / 100)
			maximumArea := float64(imageArea * m.maximumPercentage / 100)
			minimumWidth := imgWidth * m.minimumPercentage / 100
			maximumWidth := imgWidth * m.maximumPercentage / 100
			minimumHeight := imgHeight * m.minimumPercentage / 100
			maximumHeight := imgHeight * m.maximumPercentage / 100

			// 1. Check for overload and collect initial bounding rects
			overload := false
			var rects []image.Rectangle
			for index := 0; index < contours.Size(); index++ {
				c := contours.At(index)
				area := gocv.ContourArea(c)
				if area >= overloadArea {
					overload = true
					break
				}
				rect := videosource.CorrectRectangle(scaledImg, gocv.BoundingRect(c))
				rects = append(rects, rect)
			}
			contours.Close()

			if overload {
				// Clear all motions as we exceeded overload area
				for _, motion := range result.Motions {
					motion.Cleanup()
				}
				result.Motions = make([]videosource.MotionInfo, 0)
				scaledImg.Cleanup()
				blurMat.Close()
				r <- result
				continue
			}

			// 2. Merge overlapping or nearby bounding boxes
			if m.mergeOverlapPercent > 0 || m.mergeDistance > 0 {
				rects = mergeRectangles(rects, m.mergeOverlapPercent, m.mergeDistance)
			}

			// 3. Filter merged rects by area and dimensions to get candidates
			var candidates []image.Rectangle
			for _, rect := range rects {
				rectArea := float64(rect.Dx() * rect.Dy())
				if rectArea < minimumArea || rectArea > maximumArea {
					continue
				}
				rectWidth := rect.Dx()
				rectHeight := rect.Dy()
				if rectWidth < minimumWidth || rectWidth > maximumWidth {
					continue
				}
				if rectHeight < minimumHeight || rectHeight > maximumHeight {
					continue
				}
				candidates = append(candidates, rect)
			}

			// 4. Apply temporal hysteresis/stabilization
			var finalCandidates []image.Rectangle
			if m.minMotionFrames > 1 {
				currentCells := make(map[image.Point]bool)
				for _, rect := range candidates {
					for x := rect.Min.X / gridSize; x <= rect.Max.X / gridSize; x++ {
						for y := rect.Min.Y / gridSize; y <= rect.Max.Y / gridSize; y++ {
							currentCells[image.Pt(x, y)] = true
						}
					}
				}

				nextCellFrames := make(map[image.Point]int)
				for cell := range currentCells {
					nextCellFrames[cell] = cellFrames[cell] + 1
				}
				cellFrames = nextCellFrames

				for _, rect := range candidates {
					keep := false
					for x := rect.Min.X / gridSize; x <= rect.Max.X / gridSize; x++ {
						for y := rect.Min.Y / gridSize; y <= rect.Max.Y / gridSize; y++ {
							if cellFrames[image.Pt(x, y)] >= m.minMotionFrames {
								keep = true
								break
							}
						}
						if keep {
							break
						}
					}
					if keep {
						finalCandidates = append(finalCandidates, rect)
					}
				}
			} else {
				finalCandidates = candidates
				// Reset tracking map to free memory if minMotionFrames is disabled
				if len(cellFrames) > 0 {
					cellFrames = make(map[image.Point]int)
				}
			}

			// 5. Create final MotionInfo objects up to maxMotions limit
			numMotions := 0
			for _, rect := range finalCandidates {
				if numMotions >= m.maxMotions {
					// Clear all motions as we exceeded maxMotions
					for _, motion := range result.Motions {
						motion.Cleanup()
					}
					result.Motions = make([]videosource.MotionInfo, 0)
					break
				}
				scaledRect := videosource.RectScale(cur, rect, scaleRatio)
				finalRect := videosource.RectPadded(cur, scaledRect, m.padding)
				motionInfo := videosource.NewMotionInfo(finalRect, *videosource.NewColorThickness(m.highlightColor, m.highlightThickness))
				result.Motions = append(result.Motions, *motionInfo)
				numMotions++
			}

			scaledImg.Cleanup()
			blurMat.Close()

			r <- result
		}

	}()
	return r
}

// mergeRectangles merges rectangles that overlap or are nearby
func mergeRectangles(rects []image.Rectangle, minOverlapPercent int, mergeDistance int) []image.Rectangle {
	if (minOverlapPercent <= 0 && mergeDistance <= 0) || len(rects) <= 1 {
		return rects
	}
	mergedAny := true
	for mergedAny {
		mergedAny = false
		for i := 0; i < len(rects); i++ {
			for j := i + 1; j < len(rects); j++ {
				r1 := rects[i]
				r2 := rects[j]
				var qualifies bool
				inter := r1.Intersect(r2)
				if !inter.Empty() {
					if minOverlapPercent > 0 {
						interArea := inter.Dx() * inter.Dy()
						area1 := r1.Dx() * r1.Dy()
						area2 := r2.Dx() * r2.Dy()
						minArea := area1
						if area2 < minArea {
							minArea = area2
						}
						if minArea > 0 && (interArea*100)/minArea >= minOverlapPercent {
							qualifies = true
						}
					} else {
						qualifies = true
					}
				}

				if !qualifies && mergeDistance > 0 {
					var hDist int
					if r1.Max.X < r2.Min.X {
						hDist = r2.Min.X - r1.Max.X
					} else if r2.Max.X < r1.Min.X {
						hDist = r1.Min.X - r2.Max.X
					}

					var vDist int
					if r1.Max.Y < r2.Min.Y {
						vDist = r2.Min.Y - r1.Max.Y
					} else if r2.Max.Y < r1.Min.Y {
						vDist = r1.Min.Y - r2.Max.Y
					}

					if hDist <= mergeDistance && vDist <= mergeDistance {
						qualifies = true
					}
				}

				if qualifies {
					rects[i] = r1.Union(r2)
					// remove rects[j]
					rects = append(rects[:j], rects[j+1:]...)
					mergedAny = true
					break
				}
			}
			if mergedAny {
				break
			}
		}
	}
	return rects
}
