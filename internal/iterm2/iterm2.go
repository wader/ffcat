package iterm2

import (
	"encoding/base64"
	"image"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// TODO: make struct with fd

// TODO: query somehow
func IsCompatible() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app"
}

func Image(w io.Writer, m image.Image) error {
	if _, err := w.Write([]byte("\x1b]1337;File=inline=1:")); err != nil {
		return err
	}
	if err := png.Encode(base64.NewEncoder(base64.StdEncoding, w), m); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\x07")); err != nil {
		return err
	}
	return nil
}

type CellSize struct {
	Width  float64
	Height float64
	Scale  float64
}

func ReportCellSize(f *os.File) (sz CellSize, err error) {
	os, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return CellSize{}, err
	}
	defer func() { err = term.Restore(int(f.Fd()), os) }()

	if _, err := f.Write([]byte("\x1b]1337;ReportCellSize\x07")); err != nil {
		return CellSize{}, err
	}

	// note order is height:width[:scale]
	// "\x1b]1337;ReportCellSize=14.0;6.0;1.0\x1b\\"
	b := make([]byte, 50)
	if _, err := f.Read(b); err != nil {
		return CellSize{}, err
	}

	// TODO: cleanup
	s := string(b)
	p := "ReportCellSize="
	start := strings.Index(s, p) + len(p)
	stop := strings.Index(s, "\x1b\\")

	whs := s[start:stop]
	parts := strings.Split(whs, ";")
	sz = CellSize{}
	if len(parts) > 0 {
		sz.Height, _ = strconv.ParseFloat(parts[0], 64)
	}
	if len(parts) > 1 {
		sz.Width, _ = strconv.ParseFloat(parts[1], 64)
	}
	sz.Scale = 1
	if len(parts) > 2 {
		sz.Scale, _ = strconv.ParseFloat(parts[2], 64)
	}

	return sz, nil
}

type Resolution struct {
	Width       int
	Height      int
	WidthAlign  int
	HeightAlign int
}

func PixelResolution(f *os.File) (Resolution, error) {
	w, h, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return Resolution{}, err
	}
	sz, err := ReportCellSize(f)
	if err != nil {
		return Resolution{}, err
	}

	return Resolution{
		Width:       w * int(sz.Width*sz.Scale),
		Height:      h * int(sz.Height*sz.Scale),
		WidthAlign:  int(sz.Width * sz.Scale),
		HeightAlign: int(sz.Height * sz.Scale),
	}, nil
}

func ClearScrollback(w io.Writer) error {
	if _, err := w.Write([]byte("\x1b]1337;ClearScrollback\x07")); err != nil {
		return err
	}
	return nil
}
