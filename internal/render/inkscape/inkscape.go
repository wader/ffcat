package inkscape

import (
	"bytes"
	"image"
	"os/exec"
	"strings"

	"github.com/wader/ffcat/internal/render"
)

var Paths = []string{
	"inkscape",
	"/Applications/Inkscape.app/Contents/MacOS/inkscape",
}

func findPath(ss []string) (string, bool) {
	for _, s := range ss {
		if p, err := exec.LookPath(s); err == nil {
			return p, true
		}
	}
	return "", false
}

type Render struct{}

func (Render) CanHandle(bs []byte) bool {
	if _, ok := findPath(Paths); !ok {
		return false
	}
	// TODO: improve
	return strings.Contains(strings.ToLower(string(bs)), "<svg")
}

func (Render) Output(path string, rRes render.Resolution, rRange render.Range) (render.Output, error) {
	p, _ := findPath(Paths)

	c := exec.Command(p, "--pipe", "--export-type=png", "-o", "-", path)
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
