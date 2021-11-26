package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/wader/ffcat/internal/goffmpeg"
	"github.com/wader/ffcat/internal/iterm2"

	"image/draw"
	_ "image/png"
)

type cut struct {
	offset   float64
	duration float64
	delta    float64
}

func (c *cut) String() string {
	return fmt.Sprintf("%f,%f,%f", c.offset, c.delta, c.duration)
}

// TODO:
// 123
// 1:2
// 1:2:3
// 1:2:3.1
// -10ms,0.1
func (c *cut) Set(s string) error {
	timeDeltaParts := strings.Split(s, ",")
	timeParts := strings.Split(timeDeltaParts[0], ":")
	if len(timeParts) == 1 {
		c.offset, _ = strconv.ParseFloat(timeParts[0], 64)
	} else {
		h, m, s := float64(0), float64(0), float64(0)
		s, _ = strconv.ParseFloat(timeParts[len(timeParts)-1], 64)
		if len(timeParts) > 1 {
			m, _ = strconv.ParseFloat(timeParts[len(timeParts)-2], 64)
		}
		if len(timeParts) > 2 {
			h, _ = strconv.ParseFloat(timeParts[len(timeParts)-3], 64)
		}
		c.offset = (h * 60 * 60) + (m * 60) + s
	}
	if len(timeDeltaParts) > 1 {
		c.delta, _ = strconv.ParseFloat(timeDeltaParts[1], 64)
	}
	if len(timeDeltaParts) > 2 {
		c.duration, _ = strconv.ParseFloat(timeDeltaParts[2], 64)
	}

	return nil
}

func isImageCodec(s string) bool {
	switch s {
	case "png", "jpeg":
		return true
	}
	return false
}

func streamPreviews(path string, pr goffmpeg.FFProbeResult, r iterm2.Resolution, c cut) ([]image.Image, error) {
	b := &bytes.Buffer{}

	frames := int(c.duration / c.delta)

	i := &goffmpeg.Input{
		Flags: []string{
			"-ss", fmt.Sprintf("%f", c.offset),
			"-t", fmt.Sprintf("%f", c.duration),
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
		if h > maxStreamHeight {
			maxStreamHeight = h
		}
		if w > maxStreamWidth {
			maxStreamWidth = w
		}
	}
	if maxStreamHeight != 0 && maxStreamWidth != 0 {
		tileWidth = r.Width / frames
		tileHeight = int(float32(maxStreamHeight) / (float32(maxStreamWidth) / float32(tileWidth)))
	}

	// align sizes to cursor cell size
	tileWidth -= tileWidth % r.WidthAlign
	tileHeight -= tileHeight % r.HeightAlign
	audioChannelHeight -= audioChannelHeight % r.HeightAlign

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

	vSelectExpr := fmt.Sprintf(`if(between(t,0,%f), if(isnan(prev_selected_t), 1, gte(t-prev_selected_t,%f)))`, c.duration, c.delta)
	aSelectExpr := fmt.Sprintf(`between(t,0,%f)`, c.duration)

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
				height := int(s.DisplayWidth())
				if width > charAlignedWidth {
					width = charAlignedWidth
					height = int(float32(s.Height) / (float32(width) / float32(charAlignedWidth)))
					height += height % 2
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
				File:   b,
			},
		},
		Flags: []string{
			// "-v", "debug",
			// "-copyts",
		},
	}

	if *debugFlag {
		f.Stderr = os.Stderr
	}

	debugf("%s\n", strings.Join(f.Args(), " "))

	err := f.Run()
	if err != nil {
		return nil, err
	}

	m, _, err := image.Decode(b)
	if err != nil {
		return nil, err
	}

	var ms []image.Image
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
					height = int(float32(s.Height) / (float32(width) / float32(charAlignedWidth)))
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
			ms = append(ms, ci)
			dy += r.Max.Y
		}
	}

	return ms, nil
}

var rangeFlag = cut{
	offset:   0,
	duration: 5,
	delta:    1,
}

var debugFlag = flag.Bool("d", false, "Debug")
var verboseFlag = flag.Bool("v", false, "Verbose")
var clearFlag = flag.Bool("c", false, "Clear")

func verbosef(s string, args ...interface{}) {
	if *verboseFlag {
		fmt.Printf(s, args...)
	}
}

func debugf(s string, args ...interface{}) {
	if *debugFlag {
		fmt.Printf(s, args...)
	}
}

func init() {
	flag.Var(&rangeFlag, "r", "Range [[hh:]mm:]ss[,delta[,duration]]")
}

func previewFile(r iterm2.Resolution, path string, clear bool) error {
	fp := goffmpeg.FFProbeCmd{Input: goffmpeg.Input{File: path}}
	if err := fp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return nil
	}

	cf := rangeFlag
	if cf.offset < 0 {
		cf.offset = fp.ProbeResult.Duration().Seconds() + cf.offset
	}

	ms, err := streamPreviews(path, fp.ProbeResult, r, cf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return nil
	}

	pr := fp.ProbeResult

	// Input #0, mov,mp4,m4a,3gp,3g2,mj2, from '/Users/wader/Downloads/video.mp4':
	if *verboseFlag {
		verbosef("%s: %s: %ds\n", pr.FormatName(), path, pr.Duration()/time.Second)
	}

	log.Printf("fp.ProbeResult.Raw: %#+v\n", fp.ProbeResult.Raw)

	for i, m := range ms {
		s := fp.ProbeResult.Streams[i]

		if clear {
			if err := iterm2.ClearScrollback(os.Stderr); err != nil {
				return err
			}
			clear = false
		}

		verbosef("%d: %s %s %sb/s ", s.Index, s.CodecName, s.CodecType, s.BitRate)

		if s.CodecType == "audio" {
			// Stream #0:1(und): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, mono, fltp, 72 kb/s (default)
			verbosef("%s Hz %d ch %d bit", s.SampleRate, s.Channels, s.BitsPerSample)
		} else if s.CodecType == "video" {
			// Stream #0:0(und): Video: h264 (Constrained Baseline) (avc1 / 0x31637661), yuv420p(tv, bt709), 320x240 [SAR 1:1 DAR 4:3], 80 kb/s, 25 fps, 25 tbr, 12800 tbn, 50 tbc (default)
			verbosef("%dx%d (%d)", s.Width, s.Height, s.Rotation())
		} else if s.CodecType == "subtitle" {
			verbosef("%s", s.Tags.Language)
		}

		verbosef("\n")

		if err := iterm2.Image(os.Stdout, m); err != nil {
			return err
		}

		fmt.Println()
	}

	return nil
}

func main() {
	flag.Parse()

	shouldClear := *clearFlag

	if err := func() error {
		if !iterm2.IsCompatible() {
			fmt.Fprintln(os.Stdin, "not iterm2 terminal")
		}

		r, err := iterm2.PixelResolution(os.Stderr)
		if err != nil {
			return err
		}

		files := flag.Args()
		if len(files) == 0 {
			f, err := os.CreateTemp("", "ffcat")
			if err != nil {
				return err
			}
			defer os.Remove(f.Name())
			if _, err := io.Copy(f, os.Stdin); err != nil {
				return err
			}
			f.Close()
			files = append(files, f.Name())
		}

		for _, a := range files {
			if err := previewFile(r, a, shouldClear); err != nil {
				return err
			}
			shouldClear = false
		}
		return nil
	}(); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}
