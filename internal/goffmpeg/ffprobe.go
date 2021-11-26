package goffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/wader/ffcat/internal/goffmpeg/internal/execextra"
	"github.com/wader/ffcat/internal/goffmpeg/internal/kvargs"
	"github.com/wader/ffcat/internal/goffmpeg/internal/linebuffer"
)

// FFprobePath to ffprobe binary. Will be used as name to cmd.Command.
var FFprobePath = "ffprobe"

// FFProbeResult ffprobe result
type FFProbeResult struct {
	Format  FFProbeFormat          `json:"format"`
	Streams []FFProbeStream        `json:"streams"`
	Raw     map[string]interface{} `json:"raw"`
}

const (
	SideDataDisplayMatrix = "Display Matrix"
)

// TODO: is a union of all types at the moment
// if value if not mapped use FFProbeResult.Raw
type SideData struct {
	SideDataType  string `json:"side_data_type"`
	DisplayMatrix string `json:"displaymatrix"`
	Rotation      int    `json:"rotation"` // counter clockwise rotation
}

// FFProbeStream ffprobe stream result
type FFProbeStream struct {
	Index              uint       `json:"index"`
	CodecName          string     `json:"codec_name"`
	CodecLongName      string     `json:"codec_long_name"`
	CodecType          string     `json:"codec_type"`
	CodecTimeBase      string     `json:"codec_time_base"`
	CodecTagString     string     `json:"codec_tag_string"`
	CodecTag           string     `json:"codec_tag"`
	SampleFmt          string     `json:"sample_fmt"`
	SampleRate         string     `json:"sample_rate"`
	Channels           uint       `json:"channels"`
	ChannelLayout      string     `json:"channel_layout"`
	BitsPerSample      uint       `json:"bits_per_sample"`
	RFrameRate         string     `json:"r_frame_rate"`
	AvgFrameRate       string     `json:"avg_frame_rate"`
	TimeBase           string     `json:"time_base"`
	StartPts           int64      `json:"start_pts"`
	StartTime          string     `json:"start_time"`
	DurationTs         uint64     `json:"duration_ts"`
	Duration           string     `json:"duration"`
	BitRate            string     `json:"bit_rate"`
	MaxBitRate         string     `json:"max_bit_rate"`
	Profile            string     `json:"profile"`
	NbFrames           string     `json:"nb_frames"`
	Width              uint       `json:"width"`
	Height             uint       `json:"height"`
	CodedWidth         uint       `json:"coded_width"`
	CodedHeight        uint       `json:"codec_height"`
	HasBFrames         uint       `json:"has_b_frames"`
	SampleAspectRatio  string     `json:"sample_aspect_ratio"`
	DisplayAspectRatio string     `json:"display_aspect_ratio"`
	PixFmt             string     `json:"pix_fmt"`
	Level              int        `json:"level"`
	ChromaLocation     string     `json:"chroma_location"`
	Refs               uint       `json:"refs"`
	IsAvc              string     `json:"is_avc"`
	NalLengthSize      string     `json:"nal_length_size"`
	Tags               Metadata   `json:"tags"`
	SideDataList       []SideData `json:"side_data_list"`
}

func (fps FFProbeStream) Rotation() int {
	for _, s := range fps.SideDataList {
		if s.SideDataType == SideDataDisplayMatrix {
			return s.Rotation
		}
	}
	return 0
}

func (fps FFProbeStream) DisplayWidth() uint {
	switch fps.Rotation() {
	case -90, 90:
		return fps.Height
	}
	return fps.Width
}

func (fps FFProbeStream) DisplayHeight() uint {
	switch fps.Rotation() {
	case -90, 90:
		return fps.Width
	}
	return fps.Height
}

// FFProbeFormat ffprobe format result
type FFProbeFormat struct {
	Filename       string   `json:"filename"`
	FormatName     string   `json:"format_name"`
	FormatLongName string   `json:"format_long_name"`
	StartTime      string   `json:"start_time"`
	Duration       string   `json:"duration"`
	Size           string   `json:"size"`
	BitRate        string   `json:"bit_rate"`
	ProbeScore     uint     `json:"probe_score"`
	Tags           Metadata `json:"tags"`
}

// UnmarshalJSON unmarshal from ffprobe JSON output
func (fpr *FFProbeResult) UnmarshalJSON(text []byte) error {
	type probeInfo FFProbeResult
	var piDummy probeInfo
	err := json.Unmarshal(text, &piDummy)
	// unmarshal a second time in raw form
	json.Unmarshal(text, &piDummy.Raw)
	*fpr = FFProbeResult(piDummy)
	return err
}

// FindFirstStreamCodecType find first stream with codec type
func (fpr FFProbeResult) FindFirstStreamCodecType(codecType string) (FFProbeStream, bool) {
	for _, s := range fpr.Streams {
		if s.CodecType == codecType {
			return s, true
		}
	}
	return FFProbeStream{}, false
}

// FirstVideoStream find first video stream
func (fpr FFProbeResult) FirstVideoStream() string {
	if s, ok := fpr.FindFirstStreamCodecType("video"); ok {
		return s.CodecName
	}
	return ""
}

// FirstAudioStream find first video stream
func (fpr FFProbeResult) FirstAudioStream() string {
	if s, ok := fpr.FindFirstStreamCodecType("audio"); ok {
		return s.CodecName
	}
	return ""
}

// FormatName probed format (first value if comma separated)
func (fpr FFProbeResult) FormatName() string {
	return strings.Split(fpr.Format.FormatName, ",")[0]
}

// Duration probed duration
func (fpr FFProbeResult) Duration() time.Duration {
	v, _ := strconv.ParseFloat(fpr.Format.Duration, 64)
	return time.Second * time.Duration(v)
}

func (fpr FFProbeResult) String() string {
	var codecs []string
	for _, s := range fpr.Streams {
		codecs = append(codecs, s.CodecName)
	}
	return fmt.Sprintf("%s:%s", fpr.FormatName(), strings.Join(codecs, ":"))
}

// FFProbeCmd is a ffprobe command
type FFProbeCmd struct {
	Flags []string
	Input Input

	ProbeResult FFProbeResult `json:"-"`

	Context             context.Context `json:"-"`
	StderrBufferNrLines int             `json:"-"`
	Stderr              io.Writer       `json:"-"`
	DebugLog            Printer         `json:"-"`

	cmd             *execextra.Cmd
	waitCh          chan error
	stderrLastLines *linebuffer.LastLines
}

// Start ffprobe cmd
func (fp *FFProbeCmd) Start() error {
	if fp.Context != nil {
		fp.cmd = execextra.CommandContext(fp.Context, FFprobePath)
	} else {
		fp.cmd = execextra.Command(FFprobePath)
	}
	fp.cmd.Args = append(fp.cmd.Args,
		"-hide_banner",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	)
	fp.cmd.Args = append(fp.cmd.Args, fp.Flags...)
	fp.cmd.Args = append(fp.cmd.Args, kvargs.MapToSortedArgs(fp.Input.Options, kvargs.OptionArg(""))...)
	fp.cmd.Args = append(fp.cmd.Args, fp.Input.Flags...)
	if fp.Input.Format != "" {
		fp.cmd.Args = append(fp.cmd.Args, "-f", fp.Input.Format)
	}
	switch file := fp.Input.File.(type) {
	case io.Reader:
		fp.cmd.Stdin = file
		fp.cmd.Args = append(fp.cmd.Args, "pipe:0")
	case string:
		fp.cmd.Args = append(fp.cmd.Args, file)
	default:
		panic(fmt.Sprintf("unknown input type %#v", fp.Input))
	}

	var stderrws []io.Writer
	nrLines := fp.StderrBufferNrLines
	if nrLines == 0 {
		nrLines = 100
	}
	fp.stderrLastLines = linebuffer.NewLastLines(nrLines)
	stderrws = append(stderrws, fp.stderrLastLines)
	if fp.Stderr != nil {
		stderrws = append(stderrws, fp.Stderr)
	}
	fp.cmd.Stderr = io.MultiWriter(stderrws...)

	stdout, err := fp.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := fp.cmd.Start(); err != nil {
		return err
	}

	fp.waitCh = make(chan error)
	go func() {
		jsonErr := json.NewDecoder(stdout).Decode(&fp.ProbeResult)
		waitErr := fp.cmd.Wait()
		if fp.stderrLastLines != nil {
			fp.stderrLastLines.Close()
		}

		if waitErr != nil {
			fp.waitCh <- waitErr
			return
		}
		fp.waitCh <- jsonErr
	}()

	return nil
}

// Wait for ffprobe cmd to finish
// Note that the error message might include command details that are sensitive
func (fp *FFProbeCmd) Wait() error {
	err := <-fp.waitCh
	if err != nil {
		return fmt.Errorf("%w: %s", err, fp.stderrLastLines.String())
	}

	return nil
}

// Run starts and waits for ffprobe to finish
// Note that the error message might include command details that are sensitive
func (fp *FFProbeCmd) Run() error {
	if err := fp.Start(); err != nil {
		return err
	}
	return fp.Wait()
}

// Result start and wait for ffprobe to finish and return info
// Note that the error message might include command details that are sensitive
func (fp *FFProbeCmd) Result() (FFProbeResult, error) {
	if err := fp.Run(); err != nil {
		return FFProbeResult{}, err
	}
	return fp.ProbeResult, nil
}
