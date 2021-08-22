package videosource

import (
	"image"
	"math"
)

// CorrectRectangle will fix a rectangle to fit within the Image i
func CorrectRectangle(i Image, rect image.Rectangle) (result image.Rectangle) {
	if !i.IsFilled() {
		return
	}
	result = rect
	if result.Min.X < 0 {
		result.Min.X = 0
	}
	if result.Min.Y < 0 {
		result.Min.Y = 0
	}
	if result.Max.X > i.Width() {
		result.Max.X = i.Width()
	}
	if result.Max.Y > i.Height() {
		result.Max.Y = i.Height()
	}
	return
}

// RectAddWidth will add width to the rect as evenly as possible
func RectAddWidth(i Image, rect image.Rectangle, width int) (result image.Rectangle) {
	result = CorrectRectangle(i, rect)
	if width <= 0 {
		return
	}
	availMin := result.Min.X
	availMax := i.Width() - result.Max.X
	half := width / 2
	if availMin >= half && availMax >= half {
		result.Min.X -= half
		result.Max.X += half
	} else if availMin > availMax {
		remain := width - availMax
		result.Min.X -= remain
		result.Max.X += availMax
	} else {
		remain := width - availMin
		result.Min.X -= availMin
		result.Max.X += remain
	}
	result = CorrectRectangle(i, result)
	return
}

// RectAddHeight will add height to the rect as evenly as possible
func RectAddHeight(i Image, rect image.Rectangle, height int) (result image.Rectangle) {
	result = CorrectRectangle(i, rect)
	if height <= 0 {
		return
	}
	availMin := result.Min.Y
	availMax := i.Height() - result.Max.Y
	half := height / 2
	if availMin >= half && availMax >= half {
		result.Min.Y -= half
		result.Max.Y += half
	} else if availMin > availMax {
		remain := height - availMax
		result.Min.Y -= remain
		result.Max.Y += availMax
	} else {
		remain := height - availMin
		result.Min.Y -= availMin
		result.Max.Y += remain
	}
	result = CorrectRectangle(i, result)
	return
}

// RectScale will scale the rect as evenly as possible
func RectScale(i Image, rect image.Rectangle, scale float64) (result image.Rectangle) {
	if !i.IsFilled() {
		return
	}
	if scale <= 0.0 {
		return
	}
	scaleInt := int(math.Ceil(scale))
	result = rect
	scaledMin := result.Min.Mul(scaleInt)
	scaledMax := result.Max.Mul(scaleInt)
	result.Min = scaledMin
	result.Max = scaledMax
	result = CorrectRectangle(i, result)
	return
}

// RectPadded returns a padded rectangle
func RectPadded(i Image, rect image.Rectangle, paddingPercent int) (result image.Rectangle) {
	if !i.IsFilled() {
		return
	}
	result = CorrectRectangle(i, rect)
	if paddingPercent <= 0 {
		return
	}
	rectWidth := result.Dx()
	rectHeight := result.Dy()
	addWidth := rectWidth * paddingPercent / 100
	addHeight := rectHeight * paddingPercent / 100
	result = RectAddWidth(i, result, addWidth)
	result = RectAddHeight(i, result, addHeight)
	return
}

// RectSquare will return a square that fits within the Image i
func RectSquare(i Image, rect image.Rectangle) (result image.Rectangle) {
	if !i.IsFilled() {
		return
	}
	result = CorrectRectangle(i, rect)
	width := result.Dx()
	height := result.Dy()
	if width > height {
		delta := width - height
		result = RectAddHeight(i, result, delta)
	} else if height > width {
		delta := height - width
		result = RectAddWidth(i, result, delta)
	}
	return
}

// RectRect will return a rectangle that fits within the Image i
func RectRect(i Image, rect image.Rectangle) (result image.Rectangle) {
	if !i.IsFilled() {
		return
	}
	result = CorrectRectangle(i, rect)
	width := result.Dx()
	height := result.Dy()
	newWidth := height * 16 / 9
	if width < newWidth {
		delta := newWidth - width
		result = RectAddWidth(i, result, delta)
	}
	return
}

// RectRelative returns a relative rectangle given child and parent rectangles
func RectRelative(i Image, child image.Rectangle, parent image.Rectangle) (result image.Rectangle) {
	result = child
	result = result.Add(parent.Min)
	result = CorrectRectangle(i, result)
	return
}

// RectOverlap returns the rectangle's percentage overlapped by the other
func RectOverlap(rect1 image.Rectangle, rect2 image.Rectangle) (percentage1 int, percentage2 int) {
	rect1Area := rect1.Dx() * rect1.Dy()
	rect2Area := rect2.Dx() * rect2.Dy()
	overlapRect := rect1.Intersect(rect2)
	overlapArea := overlapRect.Dx() * overlapRect.Dy()
	if rect1Area > 0 {
		percentage1 = 100 * overlapArea / rect1Area
	}
	if rect2Area > 0 {
		percentage2 = 100 * overlapArea / rect2Area
	}
	return
}
