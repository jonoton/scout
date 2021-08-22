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
	scaleWidth         int
	minimumPercentage  int
	maximumPercentage  int
	maxMotions         int
	overloadPercent    int
	thresholdPercent   int
	noiseReduction     int
	highlightColor     string
	highlightThickness int
}

// NewMotion creates a new Motion
func NewMotion() *Motion {
	m := &Motion{
		padding:            0,
		scaleWidth:         320,
		minimumPercentage:  2,
		maximumPercentage:  75,
		maxMotions:         20,
		overloadPercent:    90,
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
			highlightedImage := *cur.Clone()
			highlightedMat := highlightedImage.SharedMat.Mat

			origWidth := cur.Width()
			scaleWidth := m.scaleWidth
			if m.scaleWidth <= 0 {
				scaleWidth = origWidth
			}
			scaleRatio := float64(origWidth) / float64(scaleWidth)
			scaledImg := cur.ScaleToWidth(scaleWidth)
			blurMat := gocv.NewMat()
			// reduce noise - must be odd number
			if m.noiseReduction%2 == 0 {
				m.noiseReduction++
			}
			gocv.GaussianBlur(scaledImg.SharedMat.Mat, &blurMat, image.Pt(m.noiseReduction, m.noiseReduction), 0, 0, gocv.BorderDefault)
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

			numMotions := 0
			for index := 0; index < contours.Size(); index++ {
				c := contours.At(index)
				area := gocv.ContourArea(c)
				if numMotions > m.maxMotions || area >= overloadArea {
					numMotions = 0
					for _, motion := range result.Motions {
						motion.Cleanup()
					}
					result.Motions = make([]videosource.Image, 0)
					result.MotionRects = make([]image.Rectangle, 0)
					break
				}
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
				scaledRect := videosource.RectScale(cur, rect, scaleRatio)
				finalRect := videosource.RectPadded(cur, scaledRect, m.padding)
				region := cur.GetRegion(finalRect)
				rectColor := videosource.StringToColor(m.highlightColor)
				gocv.Rectangle(&highlightedMat, finalRect, rectColor.GetRGBA(), m.highlightThickness)
				result.Motions = append(result.Motions, region)
				result.MotionRects = append(result.MotionRects, finalRect)
				numMotions++
			}
			contours.Close()
			if numMotions > 0 {
				result.HighlightedMotion = *highlightedImage.Ref()
			}
			scaledImg.Cleanup()
			highlightedImage.Cleanup()
			blurMat.Close()

			r <- result
		}
		mog2.Close()
	}()
	return r
}
