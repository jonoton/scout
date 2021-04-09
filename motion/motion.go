package motion

import (
	"image"

	"github.com/jonoton/scout/videosource"
	"gocv.io/x/gocv"
)

// Motion detects motion within images
type Motion struct {
	Skip               bool
	padding            int
	minimumPercentage  int
	maximumPercentage  int
	maxMotions         int
	thresholdPercent   int
	noiseReduction     int
	highlightColor     string
	highlightThickness int
}

// NewMotion creates a new Motion
func NewMotion() *Motion {
	m := &Motion{
		padding:            0,
		minimumPercentage:  2,
		maximumPercentage:  75,
		maxMotions:         20,
		thresholdPercent:   40,
		noiseReduction:     10,
		highlightColor:     "purple",
		highlightThickness: 3,
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
		if config.MinimumPercentage > 0 {
			m.minimumPercentage = config.MinimumPercentage
		}
		if config.MaximumPercentage > 0 {
			m.maximumPercentage = config.MaximumPercentage
		}
		if config.MaxMotions > 0 {
			m.maxMotions = config.MaxMotions
		}
		if config.ThresholdPercent > 0 {
			m.thresholdPercent = config.ThresholdPercent
		}
		if config.NoiseReduction > 0 {
			m.noiseReduction = config.NoiseReduction
		}
		if config.HighlightColor != "" {
			m.highlightColor = config.HighlightColor
		}
		if config.HighlightThickness > 0 {
			m.highlightThickness = config.HighlightThickness
		}
	}
}

// Run starts the motion detection process
func (m *Motion) Run(input <-chan videosource.Image) <-chan videosource.ProcessedImage {
	r := make(chan videosource.ProcessedImage)
	go func() {
		defer close(r)

		mog2 := gocv.NewBackgroundSubtractorMOG2()

		for cur := range input {
			result := *videosource.NewProcessedImage(cur)
			if m.Skip {
				r <- result
				continue
			}
			highlightedMat := result.Original.Mat.Clone()
			blurMat := gocv.NewMat()
			// reduce noise - must be odd number
			if m.noiseReduction%2 == 0 {
				m.noiseReduction++
			}
			gocv.GaussianBlur(result.Original.Mat, &blurMat, image.Pt(m.noiseReduction, m.noiseReduction), 0, 0, gocv.BorderDefault)
			matDelta := gocv.NewMat()
			matThresh := gocv.NewMat()
			// obtain foreground only
			mog2.Apply(blurMat, &matDelta)
			// threshold range is 0-255, lower is more sensitive
			threshold := 255 * m.thresholdPercent / 100
			gocv.Threshold(matDelta, &matThresh, float32(threshold), 255, gocv.ThresholdBinary)
			matDelta.Close()
			// dilate
			kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
			gocv.Dilate(matThresh, &matThresh, kernel)
			kernel.Close()
			// find contours
			contours := gocv.FindContours(matThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
			matThresh.Close()
			minimumArea := float64(cur.Height() * cur.Width() * m.minimumPercentage / 100)
			maximumArea := float64(cur.Height() * cur.Width() * m.maximumPercentage / 100)
			minimumWidth := cur.Width() * m.minimumPercentage / 100
			maximumWidth := cur.Width() * m.maximumPercentage / 100
			minimumHeight := cur.Height() * m.minimumPercentage / 100
			maximumHeight := cur.Height() * m.maximumPercentage / 100

			numMotions := 0
			for _, c := range contours {
				if numMotions > m.maxMotions {
					numMotions = 0
					for _, motion := range result.Motions {
						motion.Cleanup()
					}
					result.Motions = make([]videosource.Image, 0)
					result.MotionRects = make([]image.Rectangle, 0)
					break
				}
				area := gocv.ContourArea(c)
				if area < minimumArea || area > maximumArea {
					continue
				}
				rect := videosource.CorrectRectangle(cur, gocv.BoundingRect(c))
				rectWidth := rect.Dx()
				rectHeight := rect.Dy()
				if rectWidth < minimumWidth || rectWidth > maximumWidth {
					continue
				}
				if rectHeight < minimumHeight || rectHeight > maximumHeight {
					continue
				}
				finalRect := videosource.RectPadded(cur, rect, m.padding)
				region := cur.GetRegion(finalRect)
				rectColor := videosource.StringToColor(m.highlightColor)
				gocv.Rectangle(&highlightedMat, finalRect, rectColor.GetRGBA(), m.highlightThickness)
				result.Motions = append(result.Motions, region)
				result.MotionRects = append(result.MotionRects, finalRect)
				numMotions++
			}
			if numMotions > 0 {
				result.HighlightedMotion = *videosource.NewImage(highlightedMat.Clone())
			}
			blurMat.Close()
			highlightedMat.Close()
			r <- result
		}
		mog2.Close()
	}()
	return r
}
