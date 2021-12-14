package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/wader/ffcat/internal/iterm2"
	"github.com/wader/ffcat/internal/render"
	"github.com/wader/ffcat/internal/render/all"

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

func previewFile(termRes iterm2.Resolution, path string, clear bool) error {

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	probeBs := make([]byte, 512)
	if n, err := io.ReadFull(f, probeBs[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			probeBs = probeBs[0:n]
		} else {
			return err
		}
	}

	var r render.Render
	for _, r = range all.Renderers {
		if r.CanHandle(probeBs) {
			break
		}
	}
	if r == nil {
		return fmt.Errorf("failed to probe format")
	}

	o, err := r.Output(path, render.Resolution{
		Width:       termRes.Width,
		Height:      termRes.Height,
		WidthAlign:  termRes.WidthAlign,
		HeightAlign: termRes.HeightAlign,
	}, render.Range{
		Offset:   rangeFlag.offset,
		Duration: rangeFlag.duration,
		Delta:    rangeFlag.delta,
	})
	if err != nil {
		return err
	}

	if *verboseFlag {
		verbosef("%s: %s:\n", path, o)
	}

	for _, im := range o.Images() {
		if clear {
			if err := iterm2.ClearScrollback(os.Stderr); err != nil {
				return err
			}
			clear = false
		}

		verbosef("%s\n", im)

		if err := iterm2.Image(os.Stdout, im.Image()); err != nil {
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
