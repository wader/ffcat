package all

import (
	"github.com/wader/ffcat/internal/render"
	"github.com/wader/ffcat/internal/render/dot"
	"github.com/wader/ffcat/internal/render/ffmpeg"
	"github.com/wader/ffcat/internal/render/inkscape"
	"github.com/wader/ffcat/internal/render/rsvg"
)

var Renderers = []render.Render{
	inkscape.Render{},
	rsvg.Render{},
	dot.Render{},
	ffmpeg.Render{},
}
