package ffmpeg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"os"
	"strings"
	"time"

	"github.com/wader/ffcat/internal/goffmpeg"
	"github.com/wader/ffcat/internal/render"
)

type Render struct{}

func (Render) CanHandle(bs []byte) bool { return true }

func isImageCodec(s string) bool {
	switch s {
	case "png", "jpeg":
		return true
	}
	return false
}

func (Render) Output(path string, rRes render.Resolution, rRange render.Range) (render.Output, error) {
	fp := goffmpeg.FFProbeCmd{Input: goffmpeg.Input{File: path}}
	if err := fp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return nil, err
	}

	cf := rRange
	if cf.Offset < 0 {
		cf.Offset = fp.ProbeResult.Duration().Seconds() + cf.Offset
	}

	pr := fp.ProbeResult

	bb := &bytes.Buffer{}

	frames := int(rRange.Duration / rRange.Delta)

	i := &goffmpeg.Input{
		Flags: []string{
			"-ss", fmt.Sprintf("%f", rRange.Offset),
			"-t", fmt.Sprintf("%f", rRange.Duration),
		},
		File: path,
	}

	maxStreamHeight := uint(0)
	maxStreamWidth := uint(0)
	tileWidth := 320
	tileHeight := 200
	audioChannelHeight := 100

	for _, s := range pr.Streams {
		w := s.DisplayWidth()
		h := s.DisplayHeight()
		if w > maxStreamWidth {
			maxStreamWidth = w
		}
		if h > maxStreamHeight {
			maxStreamHeight = h
		}
	}
	if maxStreamHeight != 0 && maxStreamWidth != 0 {
		tileWidth = rRes.Width / frames
		tileHeight = int(float32(maxStreamHeight) / (float32(maxStreamWidth) / float32(tileWidth)))
	}

	// align sizes to cursor cell size
	tileWidth -= tileWidth % rRes.WidthAlign
	tileHeight -= tileHeight % rRes.HeightAlign
	audioChannelHeight -= audioChannelHeight % rRes.HeightAlign

	charAlignedWidth := tileWidth * frames

	var fg goffmpeg.FilterGraph
	var outs []string

	subtitleStreamCount := 0
	for _, s := range pr.Streams {
		if s.CodecType == "subtitle" {
			subtitleStreamCount++
		}
	}
	subtitleStreamIndex := 0
	if subtitleStreamCount > 0 {
		for _, s := range pr.Streams {
			if s.CodecType == "video" {
				subtitleStreamIndex = int(s.Index)
				break
			}
		}
	}

	vSelectExpr := fmt.Sprintf(`if(between(t,0,%f), if(isnan(prev_selected_t), 1, gte(t-prev_selected_t,%f)))`, rRange.Duration, rRange.Delta)
	aSelectExpr := fmt.Sprintf(`between(t,0,%f)`, rRange.Duration)

	subtitleOutCount := 0

	for _, s := range pr.Streams {
		o := fmt.Sprintf("out%d", len(outs))
		if s.CodecType == "audio" {
			fg = append(fg, goffmpeg.FilterChain{
				{
					Name:   "aselect",
					Inputs: []string{fmt.Sprintf("0:%d", s.Index)},
					Options: map[string]string{
						"expr": aSelectExpr,
					},
				},
				{
					Name: "showwavespic",
					Options: map[string]string{
						"size":           fmt.Sprintf("%dx%d", charAlignedWidth, audioChannelHeight*int(s.Channels)),
						"split_channels": "1",
						"colors":         "white",
					},
				},
				// colorspace filter wants even size
				{
					Name: "pad",
					Options: map[string]string{
						"width":  "iw+mod(iw,2)",
						"height": "ih+mod(ih,2)",
					},
				},
				// make sure all outputs has same colorspace as vstack seems to pick the first
				{
					Name: "colorspace",
					Options: map[string]string{
						"iall": "bt709",
						"all":  "bt709",
						"trc":  "srgb",
					},
					Outputs: []string{o},
				},
			})
			outs = append(outs, o)
		} else if s.CodecType == "video" {
			if isImageCodec(s.CodecName) {
				width := int(s.DisplayWidth())
				height := int(s.DisplayHeight())
				if width > charAlignedWidth {
					height = int(float32(height) / (float32(width) / float32(charAlignedWidth)))
					height += height % 2
					width = charAlignedWidth
				}
				fg = append(fg, goffmpeg.FilterChain{
					{
						Name: "scale",
						Options: map[string]string{
							"width": fmt.Sprintf("%d:%d", width, height),
						},
					},
					{
						Name: "colorspace",
						Options: map[string]string{
							"iall": "bt709",
							"all":  "bt709",
							"trc":  "srgb",
						},
						Outputs: []string{o},
					},
				})
				outs = append(outs, o)
			} else {
				splitOuts := []string{o}
				if s.Index == uint(subtitleStreamIndex) {
					for i := 0; i < subtitleStreamCount; i++ {
						splitOuts = append(splitOuts, fmt.Sprintf("subtitle_video%d", i))
					}
				}

				fg = append(fg, goffmpeg.FilterChain{
					{
						Name:   "select",
						Inputs: []string{fmt.Sprintf("0:%d", s.Index)},
						Options: map[string]string{
							"expr": vSelectExpr,
						},
					},
					{
						Name: "scale",
						Options: map[string]string{
							"width": fmt.Sprintf("%d:%d", tileWidth, tileHeight),
						},
					},
					{
						Name: "tile",
						Options: map[string]string{
							"layout":    fmt.Sprintf("%dx%d", frames, 1),
							"nb_frames": fmt.Sprintf("%d", frames),
						},
					},
					{
						Name: "pad",
						Options: map[string]string{
							"width":  "iw+mod(iw,2)",
							"height": "ih+mod(ih,2)",
						},
					},
					{
						Name: "colorspace",
						Options: map[string]string{
							"iall": "bt709",
							"all":  "bt709",
							"trc":  "srgb",
						},
					},
					{
						Name: "split",
						Options: map[string]string{
							"outputs": fmt.Sprintf("%d", len(splitOuts)),
						},
						Outputs: splitOuts,
					},
				})
				outs = append(outs, o)
			}

		} else if s.CodecType == "subtitle" {
			sbo := fmt.Sprintf("subtitle_main%d", subtitleOutCount)
			fg = append(fg, goffmpeg.FilterChain{
				{
					Inputs: []string{fmt.Sprintf("subtitle_video%d", subtitleOutCount)},
					Name:   "drawbox",
					Options: map[string]string{
						"color":     "#707070",
						"thickness": "fill",
					},
					Outputs: []string{sbo},
				},
			})
			// TODO: have to use subtitles/overlay depending on text/bitmap subtitle
			fg = append(fg, goffmpeg.FilterChain{
				{
					Name:    "overlay",
					Inputs:  []string{sbo, fmt.Sprintf("0:%d", s.Index)},
					Options: map[string]string{},
					Outputs: []string{o},
				},
			})
			subtitleOutCount++
			outs = append(outs, o)
		}
	}

	// vstack require > 1 inputs
	if len(outs) > 1 {
		fg = append(fg, goffmpeg.FilterChain{
			{
				Name:   "vstack",
				Inputs: outs,
				Options: map[string]string{
					"inputs": fmt.Sprintf(`%d`, len(outs)),
				},
				Outputs: []string{"out"},
			},
		})
	} else {
		fg = append(fg, goffmpeg.FilterChain{
			{
				Name:    "copy",
				Inputs:  outs,
				Outputs: []string{"out"},
			},
		})
	}

	f := goffmpeg.FFmpegCmd{
		// DebugLog: log.New(os.Stderr, "debug>", 0),
		// Stderr:      os.Stderr,
		Inputs:      []*goffmpeg.Input{i},
		FilterGraph: &fg,
		Outputs: []*goffmpeg.Output{
			{
				Maps: []*goffmpeg.Map{
					{
						Specifier: "[out]",
						Codec:     "png",
					},
				},
				Flags: []string{
					"-frames", "1",
				},
				Format: "image2",
				File:   bb,
			},
		},
		Flags: []string{
			// "-v", "debug",
			// "-copyts",
		},
	}

	// if *debugFlag {
	// 	f.Stderr = os.Stderr
	// }

	// debugf("%s\n", strings.Join(f.Args(), " "))

	err := f.Run()
	if err != nil {
		return nil, err
	}

	m, _, err := image.Decode(bb)
	if err != nil {
		return nil, err
	}

	var is []render.Image
	dy := 0
	for _, s := range pr.Streams {
		height := 0

		if s.CodecType == "audio" {
			height = audioChannelHeight * int(s.Channels)
		} else if s.CodecType == "video" {
			if isImageCodec(s.CodecName) {
				width := int(s.DisplayWidth())
				height = int(s.DisplayHeight())
				if width > charAlignedWidth {
					height = int(float32(height) / (float32(width) / float32(charAlignedWidth)))
					height += height % 2
				}

			} else {
				height = tileHeight
			}
		} else if s.CodecType == "subtitle" {
			height = tileHeight
		}

		if height != 0 {
			r := image.Rectangle{Max: image.Point{X: charAlignedWidth, Y: height}}
			ci := image.NewNRGBA(r)
			draw.Draw(ci, r, m, image.Point{X: 0, Y: dy}, draw.Over)
			is = append(is, Image{s: s, i: ci})
			dy += r.Max.Y
		}
	}

	return Output{
		pr: pr,
		is: is,
	}, nil
}

type Output struct {
	pr goffmpeg.FFProbeResult
	is []render.Image
}

func (o Output) String() string {
	return fmt.Sprintf("%s: %ds", o.pr.FormatName(), o.pr.Duration()/time.Second)
}

func (o Output) Images() []render.Image { return o.is }

type Image struct {
	s goffmpeg.FFProbeStream
	i image.Image
}

func (i Image) String() string {
	s := i.s

	var ss []string

	ss = append(ss, fmt.Sprintf("%d: %s %s %sb/s ", s.Index, s.CodecName, s.CodecType, s.BitRate))

	if s.CodecType == "audio" {
		// Stream #0:1(und): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, mono, fltp, 72 kb/s (default)
		ss = append(ss, fmt.Sprintf("%s Hz %d ch %d bit", s.SampleRate, s.Channels, s.BitsPerSample))
	} else if s.CodecType == "video" {
		// Stream #0:0(und): Video: h264 (Constrained Baseline) (avc1 / 0x31637661), yuv420p(tv, bt709), 320x240 [SAR 1:1 DAR 4:3], 80 kb/s, 25 fps, 25 tbr, 12800 tbn, 50 tbc (default)
		ss = append(ss, fmt.Sprintf("%dx%d (%d)", s.DisplayWidth(), s.DisplayHeight(), s.Rotation()))
	} else if s.CodecType == "subtitle" {
		ss = append(ss, fmt.Sprintf("%s", s.Tags.Language))
	}

	return strings.Join(ss, "")
}

func (i Image) Image() image.Image { return i.i }
