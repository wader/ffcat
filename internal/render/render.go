package render

import (
	"image"
)

type Resolution struct {
	Width       int
	Height      int
	WidthAlign  int
	HeightAlign int
}

type Range struct {
	Offset   float64
	Duration float64
	Delta    float64
}

type Render interface {
	CanHandle(bs []byte) bool
	Output(path string, rRes Resolution, rRange Range) (Output, error)
}

type Image interface {
	String() string
	Image() image.Image
}

type Output interface {
	String() string
	Images() []Image
}
