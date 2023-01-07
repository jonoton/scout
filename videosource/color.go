package videosource

import (
	"image/color"
	"strings"
)

type ColorThickness struct {
	Color     Color
	Thickness int
}

func NewColorThickness(color string, thickness int) *ColorThickness {
	c := &ColorThickness{
		Color:     StringToColor(color),
		Thickness: thickness,
	}
	return c
}

// Color represents a color
type Color int

// Color Constants
const (
	Blue Color = iota
	Purple
	Green
	Red
	White
)

// StringToColor returns a Color
func StringToColor(name string) Color {
	s := strings.ToLower(name)
	switch s {
	case "blue":
		return Blue
	case "purple":
		return Purple
	case "green":
		return Green
	case "red":
		return Red
	case "white":
		return White
	}
	return White
}

func (c Color) String() string {
	switch c {
	case Blue:
		return "blue"
	case Purple:
		return "purple"
	case Green:
		return "green"
	case Red:
		return "red"
	case White:
		return "white"
	}
	return ""
}

// GetRGBA returns the color rgba
func (c Color) GetRGBA() color.RGBA {
	switch c {
	case Blue:
		return color.RGBA{0, 0, 255, 0}
	case Purple:
		return color.RGBA{255, 0, 255, 0}
	case Green:
		return color.RGBA{0, 255, 0, 0}
	case Red:
		return color.RGBA{255, 0, 0, 0}
	case White:
		return color.RGBA{0, 0, 0, 0}
	}
	return color.RGBA{0, 0, 0, 0}
}
