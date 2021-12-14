package rsvg

import (
	"bytes"
	"image"
	"os/exec"
	"strings"

	"github.com/wader/ffcat/internal/render"
)

type Render struct{}

func (Render) CanHandle(bs []byte) bool {
	// TODO: improve
	return strings.Contains(strings.ToLower(string(bs)), "<svg")
}

func (Render) Output(path string, rRes render.Resolution, rRange render.Range) (render.Output, error) {
	c := exec.Command("rsvg-convert", "-f", "png", path)
	bs, err := c.Output()
	if err != nil {
		return Output{}, err
	}

	m, _, err := image.Decode(bytes.NewBuffer(bs))
	if err != nil {
		return Output{}, err
	}

	return Output{i: m}, nil
}

type Output struct {
	i image.Image
}

func (o Output) String() string         { return "svg" }
func (o Output) Images() []render.Image { return []render.Image{Image{i: o.i}} }

type Image struct{ i image.Image }

func (i Image) String() string     { return "svg" }
func (i Image) Image() image.Image { return i.i }
