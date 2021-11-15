package goffmpeg

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"ff/internal/goffmpeg/internal/execextra"
	"ff/internal/goffmpeg/internal/kvargs"
	"ff/internal/goffmpeg/internal/linebuffer"
)

// FFmpegPath to ffmpeg binary. Will be used as name to cmd.Command.
var FFmpegPath = "ffmpeg"

// TODO:
// DONE include stderr in return err?
// DONE error log? ring buffer?
// graph input interface{}? *Input/string?
// DONE key/value options?
// DONE func arg for input/output strings?
// DONE stream options -thread
// graph
// DONE metadata/optios sort? stable?
// json encode/decode?
// DONE Path variable?
// -y -hide_banner, nostddin?
//

// FFmpegCmd is a ffmpeg command
// ffmpeg
//   Input
//     -i io.Reader/string
//   ...
//   Output
//     Map
//       -map *Input/Specifier
//     ...
//     io.WriteCloser/string
//   ...
type FFmpegCmd struct {
	Flags       []string     `json:"flags"`
	Inputs      []*Input     `json:"inputs"`
	FilterGraph *FilterGraph `json:"filter_graph"`
	Outputs     []*Output    `json:"outputs"`

	Context             context.Context  `json:"-"`
	CloseAfterStart     []io.Closer      `json:"-"`
	CloseAfterWait      []io.Closer      `json:"-"`
	StderrBufferNrLines int              `json:"-"`
	Stderr              io.Writer        `json:"-"`
	DebugLog            Printer          `json:"-"`
	ProgressFn          func(p Progress) `json:"-"`

	cmd                       *execextra.Cmd
	stderrLastLines           *linebuffer.LastLines
	currentProgress           Progress
	currentProgressLineBuffer *linebuffer.Fn
}

// TODO: input codecs? -codec:<stream>?
type Input struct {
	File    interface{}       `json:"file"` // io.Reader/string
	Format  string            `json:"format"`
	Options map[string]string `json:"options"`
	Flags   []string          `json:"flags"`
}

type Output struct {
	File     interface{}       `json:"file"` // io.Writer/string
	Maps     []*Map            `json:"maps"`
	Format   string            `json:"format"`
	Metadata *Metadata         `json:"metadata"`
	Options  map[string]string `json:"options"`
	Flags    []string          `json:"flags"`
}

type Map struct {
	Input     *Input            `json:"input"`
	Specifier string            `json:"specifier"`
	Codec     string            `json:"codec"`
	Options   map[string]string `json:"options"`
	Flags     []string          `json:"flags"`
}

type FilterGraph []FilterChain

type FilterChain []Filter

type Filter struct {
	Name    string            `json:"name"`
	Inputs  []string          `json:"inputs"`
	Outputs []string          `json:"outputs"`
	Options map[string]string `json:"options"`
}

type Progress struct {
	Frame      int64            `json:"frame"`
	FPS        float32          `json:"fps"`
	Outputs    []ProgressOutput `json:"outputs"`
	BitRate    float32          `json:"bitrate"`
	TotalSize  int64            `json:"totalsize"`
	OutTimeUS  int64            `json:"outtime_us"`
	OutTimeMS  int64            `json:"outtime_ms"`
	OutTime    string           `json:"outtime"`
	DupFrames  int64            `json:"dup_frames"`
	DropFrames int64            `json:"drop_frames"`
	Speed      float32          `json:"speed"`
	Progress   string           `json:"progress"`
}

type ProgressOutput struct {
	Streams []ProgressStream `json:"streams"`
}

type ProgressStream struct {
	Q       float32 `json:"q"`
	PSNRY   float32 `json:"psnry"`
	PSNRU   float32 `json:"psnru"`
	PSNRV   float32 `json:"psnrv"`
	PSNRAll float32 `json:"psnr_all"`
}

type inputReaderFn func(index int, r io.Reader) (string, error)
type outputWriterFn func(index int, w io.Writer) (string, error)

// ParseProgress parse a ffmpeg progress line
// Example output:
// frame=241
// fps=79.81
// stream_0_1_q=0.0
// stream_0_1_psnr_y=62.39
// stream_0_1_psnr_u=56.45
// stream_0_1_psnr_v=54.76
// stream_0_1_psnr_all=58.80
// bitrate= 107.1kbits/s
// total_size=116071
// out_time_us=8674000
// out_time_ms=8674000
// out_time=00:00:08.674000
// dup_frames=0
// drop_frames=0
// speed=2.87x
// progress=continue or end
func ParseProgress(p *Progress, line string) bool {
	parts := strings.SplitN(line, "=", 2)
	name, rawValue := parts[0], parts[1]
	value := strings.TrimFunc(rawValue, func(r rune) bool {
		return !unicode.IsDigit(r) && r != '.'
	})
	i64, _ := strconv.ParseInt(value, 10, 64)
	f64, _ := strconv.ParseFloat(value, 64)
	f32 := float32(f64)

	switch name {
	case "frame":
		p.Frame = i64
	case "fps":
		p.FPS = f32
	case "bitrate":
		var scale float32 = 1.0
		if strings.HasSuffix(rawValue, "kbits/s") {
			scale = 1000.0
		}
		p.BitRate = f32 * scale
	case "total_size":
		p.TotalSize = i64
	case "out_time_us":
		p.OutTimeUS = i64
	case "out_time_ms":
		p.OutTimeMS = i64
	case "out_time":
		p.OutTime = rawValue
	case "dup_frames":
		p.DupFrames = i64
	case "drop_frames":
		p.DropFrames = i64
	case "speed":
		p.Speed = f32
	case "progress":
		p.Progress = rawValue
	}

	if strings.HasPrefix(name, "stream") {
		streamParts := strings.SplitN(name, "_", 4)
		outputIndex, _ := strconv.Atoi(streamParts[1])
		streamIndex, _ := strconv.Atoi(streamParts[2])
		streamName := streamParts[3]

		for outputIndex >= len(p.Outputs) {
			p.Outputs = append(p.Outputs, ProgressOutput{})
		}
		for streamIndex >= len(p.Outputs[outputIndex].Streams) {
			p.Outputs[outputIndex].Streams = append(p.Outputs[outputIndex].Streams, ProgressStream{})
		}
		s := &p.Outputs[outputIndex].Streams[streamIndex]
		switch streamName {
		case "q":
			s.Q = f32
		case "psnr_y":
			s.PSNRY = f32
		case "psnr_u":
			s.PSNRU = f32
		case "psnr_v":
			s.PSNRV = f32
		case "psnr_all":
			s.PSNRAll = f32
		}
	}

	return name == "progress"
}

var filterGraphValueEscapeRe = regexp.MustCompile(`[,:]`)

func (fm *FFmpegCmd) buildArgs(inputReaderFn inputReaderFn, outputWriterFn outputWriterFn) ([]string, error) {
	inputToIndex := map[interface{}]int{}

	args := []string{
		"-nostdin",
		"-hide_banner",
	}
	args = append(args, fm.Flags...)

	if fm.ProgressFn != nil {
		fm.currentProgressLineBuffer = linebuffer.NewFn(fm.progressLine)
		progressFile, err := outputWriterFn(0, fm.currentProgressLineBuffer)
		if err != nil {
			return nil, err
		}
		args = append(args, "-progress", progressFile)
	}

	if fm.FilterGraph != nil {
		var argsGraph []string

		for _, chain := range *fm.FilterGraph {
			var argsChain []string

			for _, filter := range chain {
				var argsFilter []string
				for _, input := range filter.Inputs {
					// TODO: Input ref?
					argsFilter = append(argsFilter, "[", strings.ReplaceAll(input, `]`, `\]`), "]")
				}
				argsFilter = append(argsFilter, filter.Name+"=")
				// uses MapToSortedArgs to keep options in stable order
				filterOpts := kvargs.MapToSortedArgs(filter.Options, func(k, v string) []string {
					// TODO: more escape?
					escapedV := filterGraphValueEscapeRe.ReplaceAllString(v, `\$0`)
					return []string{k + "=" + escapedV}
				})
				argsFilter = append(argsFilter, strings.Join(filterOpts, ":"))
				for _, output := range filter.Outputs {
					argsFilter = append(argsFilter, "[", strings.ReplaceAll(output, `]`, `\]`), "]")
				}

				argsChain = append(argsChain, strings.Join(argsFilter, ""))
			}
			argsGraph = append(argsGraph, strings.Join(argsChain, ","))
		}

		args = append(args, "-filter_complex", strings.Join(argsGraph, ";"))
	}

	for inputIndex, input := range fm.Inputs {
		inputToIndex[input] = inputIndex

		args = append(args, kvargs.MapToSortedArgs(input.Options, kvargs.OptionArg(""))...)
		args = append(args, input.Flags...)
		if input.Format != "" {
			args = append(args, "-f", input.Format)
		}
		args = append(args, "-i")
		switch file := input.File.(type) {
		case string:
			args = append(args, file)
		case io.Reader:
			a, err := inputReaderFn(inputIndex, file)
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		default:
			panic(fmt.Sprintf("unknown input file type %#v should be string or io.Reader", file))
		}
	}

	for outputIndex, output := range fm.Outputs {
		for streamIndex, m := range output.Maps {
			args = append(args, "-map")
			var specifier []string
			if m.Input != nil {
				inputIndex, ok := inputToIndex[m.Input]
				if !ok {
					// TODO: maybe leaks pips if no Start()? return "-map ?"?
					return nil, fmt.Errorf("can't find input %#v for map %#v", m.Input, m)
				}
				specifier = append(specifier, strconv.Itoa(inputIndex))
			}
			if m.Specifier != "" {
				specifier = append(specifier, m.Specifier)
			}
			args = append(args, strings.Join(specifier, ":"))

			streamIndexStr := strconv.Itoa(streamIndex)
			if m.Codec != "" {
				args = append(args, "-codec:"+streamIndexStr, m.Codec)
			}
			args = append(args, kvargs.MapToSortedArgs(m.Options, kvargs.OptionArg(":"+streamIndexStr))...)
			args = append(args, m.Flags...)
		}

		if output.Format != "" {
			args = append(args, "-f", output.Format)
		}
		if output.Metadata != nil {
			args = append(args,
				kvargs.MapToSortedArgs(output.Metadata.ToMap(),
					func(k, v string) []string {
						return []string{"-metadata", k + "=" + v}
					},
				)...,
			)
		}
		args = append(args, kvargs.MapToSortedArgs(output.Options, kvargs.OptionArg(""))...)
		args = append(args, output.Flags...)

		if file, ok := output.File.(string); ok {
			args = append(args, file)
		} else if file, ok := output.File.(io.Writer); ok {
			a, err := outputWriterFn(outputIndex, file)
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		} else {
			panic(fmt.Sprintf("unknown output file type %#v should be string or io.Writer", output.File))
		}
	}

	return args, nil

}

func (fm *FFmpegCmd) Args() []string {
	args, err := fm.buildArgs(
		func(inputIndex int, r io.Reader) (string, error) {
			return fmt.Sprintf("pipe-input-index:%d", inputIndex), nil
		},
		func(outputIndex int, w io.Writer) (string, error) {
			return fmt.Sprintf("pipe-output-index:%d", outputIndex), nil
		},
	)
	if err != nil {
		panic(err)
	}
	return args
}

func (fm *FFmpegCmd) progressLine(line string) {
	if ParseProgress(&fm.currentProgress, line) {
		fm.ProgressFn(fm.currentProgress)
		fm.currentProgress = Progress{}
	}
}

func (fm *FFmpegCmd) Start() error {
	if fm.Context != nil {
		fm.cmd = execextra.CommandContext(fm.Context, FFmpegPath)
	} else {
		fm.cmd = execextra.Command(FFmpegPath)
	}
	for _, closer := range fm.CloseAfterStart {
		fm.cmd.CloseAfterStart(closer)
	}
	for _, closer := range fm.CloseAfterWait {
		fm.cmd.CloseAfterWait(closer)
	}

	args, err := fm.buildArgs(
		func(inputIndex int, r io.Reader) (string, error) {
			inChildFD, err := fm.cmd.ExtraIn(r)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("pipe:%d", inChildFD), nil
		},
		func(outputIndex int, w io.Writer) (string, error) {
			outChildFD, err := fm.cmd.ExtraOut(w)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("pipe:%d", outChildFD), nil
		},
	)
	if err != nil {
		return err
	}

	var stderrws []io.Writer
	nrLines := fm.StderrBufferNrLines
	if nrLines == 0 {
		nrLines = 100
	}
	fm.stderrLastLines = linebuffer.NewLastLines(nrLines)
	stderrws = append(stderrws, fm.stderrLastLines)
	if fm.Stderr != nil {
		stderrws = append(stderrws, fm.Stderr)
	}
	fm.cmd.Stderr = io.MultiWriter(stderrws...)
	fm.cmd.Args = append(fm.cmd.Args, args...)

	return fm.cmd.Start()
}

// Wait for cmd to finish
// Note that the error message might include command details that are sensitive
func (fm *FFmpegCmd) Wait() error {
	err := fm.cmd.Wait()
	if fm.currentProgressLineBuffer != nil {
		fm.currentProgressLineBuffer.Close()
	}
	if fm.stderrLastLines != nil {
		fm.stderrLastLines.Close()
	}

	if err != nil {
		return fmt.Errorf("%w: %s", err, fm.stderrLastLines.String())
	}

	return nil
}

// Run starts and waits for ffmpeg to finish
// Note that the error message might include command details that are sensitive
func (fm *FFmpegCmd) Run() error {
	if err := fm.Start(); err != nil {
		return err
	}
	return fm.Wait()
}

// StderrBuffer returns the last stderr lines as a string
// Note that the stderr might include command details that are sensitive
func (fm *FFmpegCmd) StderrBuffer() string {
	return fm.stderrLastLines.String()
}
