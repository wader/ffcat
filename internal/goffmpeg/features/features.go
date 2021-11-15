package features

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Features struct {
	Version              VersionParts `json:"version"`
	Codecs               []Codec      `json:"codecs"`
	Encoders             []Coder      `json:"encoders"`
	Decoders             []Coder      `json:"decoders"`
	Muxers               []Format     `json:"muxers"`
	Demuxers             []Format     `json:"demuxers"`
	Filters              []Filter     `json:"filters"`
	PixelFmts            []PixelFmt   `json:"pixel_fmts"`
	SampleFmts           []SampleFmt  `json:"sample_fmts"`
	FormatContextOptions []AVOption   `json:"format_context_options"`
	CodecContextOptions  []AVOption   `json:"codec_context_options"`
	FilterOptions        []AVOption   `json:"filter_options"`
}

type VersionParts struct {
	Full    string `json:"full"`
	Release string `json:"release"`
	Major   uint   `json:"major"`
	Minor   uint   `json:"minor"`
	Patch   uint   `json:"patch"`
}

type MediaType uint

func (mt MediaType) String() string {
	if s, ok := MediaTypeToString[mt]; ok {
		return s
	}
	return fmt.Sprintf("unknown (%d)", mt)
}

func (mt MediaType) MarshalJSON() ([]byte, error) {
	s, ok := MediaTypeToString[mt]
	if !ok {
		return nil, fmt.Errorf("unknown media type %d", mt)
	}

	return json.Marshal(s)
}

func (mt *MediaType) UnmarshalJSON(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var v MediaType
	var ok bool
	if v, ok = MediaTypeFromString[s]; !ok {
		return fmt.Errorf("unknown media type %s", s)
	}
	*mt = v
	return nil
}

func (MediaType) JSONSchemaEnums() []interface{} {
	var enums []interface{}
	for s := range MediaTypeFromString {
		enums = append(enums, s)
	}
	return enums
}

const (
	MediaTypeAudio MediaType = iota
	MediaTypeVideo
	MediaTypeSubtitle
	MediaTypeData
)

var MediaTypeToString = map[MediaType]string{
	MediaTypeAudio:    "audio",
	MediaTypeVideo:    "video",
	MediaTypeSubtitle: "subtitle",
	MediaTypeData:     "data",
}

var MediaTypeFromString = map[string]MediaType{
	"audio":    MediaTypeAudio,
	"video":    MediaTypeVideo,
	"subtitle": MediaTypeSubtitle,
	"data":     MediaTypeData,
}

// Format is a muxer or demuxer
type Format struct {
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Extensions        []string   `json:"extensions"`
	MIMEType          string     `json:"mime_type"`
	DefaultAudioCodec string     `json:"default_audio_codec"`
	DefaultVideoCodec string     `json:"default_video_codec"`
	Options           []AVOption `json:"options"`
}

// Coder is a encoder or decoder
type Coder struct {
	Name                  string     `json:"name"`
	Description           string     `json:"description"`
	MediaType             MediaType  `json:"media_type"`
	Codec                 string     `json:"codec"`
	GeneralCapabilities   []string   `json:"general_capabilities"`
	ThreadingCapabilities []string   `json:"threading_capabilities"`
	FrameRates            []string   `json:"frame_rates"`
	PixelFormats          []string   `json:"pixel_formats"`
	SampleRates           []string   `json:"sample_rates"`
	SampleFormats         []string   `json:"sample_formats"`
	ChannelLayouts        []string   `json:"channel_layouts"`
	Options               []AVOption `json:"options"`
}

// Filter
type Filter struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	TimelineSupport bool                `json:"timeline_support"`
	SliceThreading  bool                `json:"slice_threading"`
	CommandSupport  bool                `json:"command_support"`
	InputsDynamic   bool                `json:"inputs_dynamic"`
	OutputsDynamic  bool                `json:"outputs_dynamic"`
	Inputs          []FilterInputOutput `json:"inputs"`
	Outputs         []FilterInputOutput `json:"outputs"`
	Options         []AVOption          `json:"options"`
}

type FilterInputOutput struct {
	Name      string    `json:"name"`
	MediaType MediaType `json:"media_type"`
}

type Codec struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MediaType   MediaType `json:"media_type"`
	Decoders    []string  `json:"decoders"`
	Encoders    []string  `json:"encoders"`
}

type PixelFmt struct {
	Name         string `json:"name"`
	Input        bool   `json:"input"`
	Output       bool   `json:"output"`
	NbComponents uint   `json:"nb_components"`
	BitsPerPixel uint   `json:"bits_per_pixel"`
}

type SampleFmt struct {
	Name  string `json:"name"`
	Depth uint   `json:"depth"`
}

type OptionType uint

func (ot OptionType) String() string {
	if s, ok := OptionTypeToString[ot]; ok {
		return s
	}
	return "unknown"
}

func (ot OptionType) MarshalJSON() ([]byte, error) {
	if s, ok := OptionTypeToString[ot]; ok {
		return json.Marshal(s)
	}
	return nil, fmt.Errorf("unknown option type %d", ot)
}

func (ot *OptionType) UnmarshalJSON(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var v OptionType
	var ok bool
	if v, ok = OptionTypeFromString[s]; !ok {
		return fmt.Errorf("unknown option type %s", s)
	}
	*ot = v
	return nil
}

func (OptionType) JSONSchemaEnums() []interface{} {
	var enums []interface{}
	for s := range OptionTypeFromString {
		enums = append(enums, s)
	}
	return enums
}

const (
	OptionTypeFlags OptionType = iota
	OptionTypeInt
	OptionTypeInt64
	OptionTypeDouble
	OptionTypeFloat
	OptionTypeString
	OptionTypeRational
	OptionTypeBinary
	OptionTypeDictionary
	OptionTypeUInt64
	OptionTypeConst
	OptionTypeImageSize
	OptionTypePixFmt
	OptionTypeSampleFmt
	OptionTypeVideoRate
	OptionTypeDuration
	OptionTypeColor
	OptionTypeChannelLayout
	OptionTypeBoolean
)

var OptionTypeToString = map[OptionType]string{
	OptionTypeFlags:         "flags",
	OptionTypeInt:           "int",
	OptionTypeInt64:         "int64",
	OptionTypeDouble:        "double",
	OptionTypeFloat:         "float",
	OptionTypeString:        "string",
	OptionTypeRational:      "rational",
	OptionTypeBinary:        "binary",
	OptionTypeDictionary:    "dictionary",
	OptionTypeUInt64:        "uint64",
	OptionTypeConst:         "const",
	OptionTypeImageSize:     "image_size",
	OptionTypePixFmt:        "pix_fmt",
	OptionTypeSampleFmt:     "sample_fmt",
	OptionTypeVideoRate:     "video_rate",
	OptionTypeDuration:      "duration",
	OptionTypeColor:         "color",
	OptionTypeChannelLayout: "channel_layout",
	OptionTypeBoolean:       "boolean",
}

var OptionTypeFromString = map[string]OptionType{
	"flags":          OptionTypeFlags,
	"int":            OptionTypeInt,
	"int64":          OptionTypeInt64,
	"double":         OptionTypeDouble,
	"float":          OptionTypeFloat,
	"string":         OptionTypeString,
	"rational":       OptionTypeRational,
	"binary":         OptionTypeBinary,
	"dictionary":     OptionTypeDictionary,
	"uint64":         OptionTypeUInt64,
	"const":          OptionTypeConst,
	"image_size":     OptionTypeImageSize,
	"pix_fmt":        OptionTypePixFmt,
	"sample_fmt":     OptionTypeSampleFmt,
	"video_rate":     OptionTypeVideoRate,
	"duration":       OptionTypeDuration,
	"color":          OptionTypeColor,
	"channel_layout": OptionTypeChannelLayout,
	"boolean":        OptionTypeBoolean,
}

type AVOption struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Encoding        bool            `json:"encoding"`
	Decoding        bool            `json:"decoding"`
	Filtering       bool            `json:"filtering"`
	Audio           bool            `json:"audio"`
	Video           bool            `json:"video"`
	Subtitle        bool            `json:"subtitle"`
	Export          bool            `json:"export"`
	Readonly        bool            `json:"readonly"`
	BitstreamFilter bool            `json:"bitstream_filter"`
	RuntimeParam    bool            `json:"runtime_param"`
	Type            OptionType      `json:"type"`
	Min             string          `json:"min"` // TODO: int etc?
	Max             string          `json:"max"`
	Default         string          `json:"default"`
	Constants       []AVOptionConst `json:"constants"` // used for enum and flags

}

// AVOptionConst used for enum and flags
type AVOptionConst struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Encoding    bool   `json:"encoding"`
	Decoding    bool   `json:"decoding"`
	Audio       bool   `json:"audio"`
	Video       bool   `json:"video"`
	Value       int    `json:"value"`
}

func reMatchNamedGroups(re *regexp.Regexp, s string) map[string]string {
	match := re.FindStringSubmatch(s)
	if match == nil {
		return nil
	}

	result := map[string]string{}
	for i, name := range re.SubexpNames() {
		if i != 0 {
			result[name] = match[i]
		}
	}

	return result
}

type helpMatch struct {
	skipStartLinesCount int
	headerLineRe        *regexp.Regexp
	headerEndSuffix     string
	lineRe              *regexp.Regexp
}

func reMatchNamedGroupsCommandOutput(cmd *exec.Cmd, hm helpMatch) (headerMatches []map[string]string, optsMatches []map[string]string, err error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if startErr := cmd.Start(); startErr != nil {
		return nil, nil, startErr
	}

	lineScanner := bufio.NewScanner(stdout)

	for i := 0; i < hm.skipStartLinesCount; i++ {
		if line := lineScanner.Scan(); !line {
			return nil, nil, errors.New("no start line to skip")
		}
	}

	// collect header lines
	headerMatches = []map[string]string{}
	for lineScanner.Scan() {
		line := lineScanner.Text()
		if strings.HasSuffix(line, hm.headerEndSuffix) {
			break
		}

		if hm.headerLineRe == nil {
			continue
		}

		headerMatch := reMatchNamedGroups(hm.headerLineRe, line)
		if headerMatch == nil {
			return nil, nil, errors.New("failed to parse header line: '" + line + "'")
		}

		headerMatches = append(headerMatches, headerMatch)
	}

	ignoreRest := false
	optsMatches = []map[string]string{}
	for lineScanner.Scan() {
		if ignoreRest {
			continue
		}

		line := lineScanner.Text()
		// some AVOptions will end with blank like (-help encoder= etc)
		if line == "" {
			ignoreRest = true
			continue
		}

		optMatch := reMatchNamedGroups(hm.lineRe, line)
		if optMatch == nil {
			return nil, nil, errors.New("failed to parse opt line: '" + line + "'")
		}

		optsMatches = append(optsMatches, optMatch)
	}

	if lineScanner.Err() != nil {
		return nil, nil, lineScanner.Err()
	}

	waitErr := cmd.Wait()
	if waitErr != nil {
		return nil, nil, waitErr
	}

	return headerMatches, optsMatches, nil
}

func splitRune(s string, sep rune) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == sep })
}

func LoadFeatures(ffmpegPath string) (Features, error) {
	var f Features
	var err error

	if f.Version, err = Version(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.Codecs, err = Codecs(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.Encoders, err = Encoders(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.Decoders, err = Decoders(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.Muxers, err = Muxers(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.Demuxers, err = Demuxers(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.Filters, err = Filters(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.PixelFmts, err = PixelFmts(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.SampleFmts, err = SampleFmts(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.FormatContextOptions, err = FormatContextOptions(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.CodecContextOptions, err = CodecContextOptions(ffmpegPath); err != nil {
		return Features{}, err
	}
	if f.FilterOptions, err = FilterOptions(ffmpegPath); err != nil {
		return Features{}, err
	}

	return f, nil
}

/*
ffmpeg version n4.0 Copyright (c) 2000-2018 the FFmpeg developers
ffmpeg version 4.2 Copyright (c) 2000-2019 the FFmpeg developers
*/
var versionLineRe = regexp.MustCompile(`` +
	`^ffmpeg version ` +
	`(?P<release>` +
	`(?:\w*(?P<major>\d+))` +
	`(?:\.(?P<minor>\d+))` +
	`(?:\.(?P<patch>\d+))?` +
	`)` +
	` Copyright.*$` +
	``)

func Version(ffmpegPath string) (VersionParts, error) {
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-version")
	versionBytes, cmdErr := cmd.Output()
	if cmdErr != nil {
		return VersionParts{}, cmdErr
	}

	full := string(versionBytes)

	versionLines := strings.Split(full, "\n")
	if len(versionLines) == 0 {
		return VersionParts{}, fmt.Errorf("no version lines")
	}

	versionMatch := reMatchNamedGroups(versionLineRe, versionLines[0])

	release := versionMatch["release"]
	major, _ := strconv.Atoi(versionMatch["major"])
	minor, _ := strconv.Atoi(versionMatch["minor"])
	patch := 0
	if versionMatch["patch"] != "" {
		patch, _ = strconv.Atoi(versionMatch["patch"])
	}

	v := VersionParts{
		Full:    full,
		Release: release,
		Major:   uint(major),
		Minor:   uint(minor),
		Patch:   uint(patch),
	}

	return v, nil
}

/*
ffmpeg 4.1?:
  -test              <boolean>    ED.VA... test (default false)
  -layer             <string>     .D.V.... Set the decoding layer (default "")
  -gamma             <float>      .D.V.... Set the float gamma value when decoding (from 0.001 to FLT_MAX) (default 1)
  -ps                <int>        E..V.... RTP payload size in bytes (from INT_MIN to INT_MAX) (default 0)
  -apply_trc         <int>        .D.V.... color transfer characteristics to apply to EXR linear input (from 1 to 18) (default gamma)
     bt709                        .D.V.... BT.709
	 gamma                        .D.V.... gamma
  -test              <sample_fmt>  ED.VA... test
ffmpeg 4.3:
  -aq-mode           <int>        E..V...... AQ method (from -1 to INT_MAX) (default -1)
     none            0            E..V......
     variance        1            E..V...... Variance AQ (complexity mask)
     autovariance    2            E..V...... Auto-variance AQ
     autovariance-biased 3            E..V...... Auto-variance AQ with bias to dark scenes
*/
var avoptionLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<name>\S+)` +
	`\s+` +
	`(?:<(?P<type>\S+)>|(?P<value>\d+))?` + // type or enum const value (enum line)
	`\s+` +
	`(?P<encoding>.)` +
	`(?P<decoding>.)` +
	`(?P<filtering>.)` +
	`(?P<video>.)` +
	`(?P<audio>.)` +
	`(?P<subtitle>.)` +
	`(?P<export>.)` +
	`(?P<readonly>.)` +
	`(?P<bitstream_filter>.)` +
	`(?P<runtime_param>.)` +
	`\s*` +
	`(?P<description>.*?)` + // non-greedy to allow optional match below
	`\s*` +
	`(?:\(from (?P<from>\S+) to (?P<to>\S+)\))?` +
	`\s*` +
	`(?:\(default (?P<default>.*?)\))?` +
	`\s*` +
	`$` +
	``)

func parseAVOptions(optMatches []map[string]string) ([]AVOption, error) {
	lineOptions := []interface{}{}
	for i := 0; i < len(optMatches); i++ {
		lineMatch := optMatches[i]

		t := lineMatch["type"]
		if t != "" {
			var optionType OptionType
			var optionTypeOk bool
			if optionType, optionTypeOk = OptionTypeFromString[t]; !optionTypeOk {
				return nil, errors.New("unknown option type: " + t)
			}

			defaultValue := lineMatch["default"]
			if optionType == OptionTypeString && len(defaultValue) >= 2 {
				// remove "" around default string value
				defaultValue = defaultValue[1 : len(defaultValue)-1]
			}

			lineOptions = append(lineOptions, AVOption{
				Type:            optionType,
				Name:            lineMatch["name"],
				Description:     lineMatch["description"],
				Encoding:        lineMatch["encoding"] == "E",
				Decoding:        lineMatch["decoding"] == "D",
				Filtering:       lineMatch["filtering"] == "F",
				Audio:           lineMatch["audio"] == "A",
				Video:           lineMatch["video"] == "V",
				Subtitle:        lineMatch["subtitle"] == "S",
				Export:          lineMatch["export"] == "X",
				Readonly:        lineMatch["readonly"] == "R",
				BitstreamFilter: lineMatch["bitstream_filter"] == "B",
				RuntimeParam:    lineMatch["runtime_param"] == "T",
				Min:             lineMatch["from"],
				Max:             lineMatch["to"],
				Default:         defaultValue,
			})
		} else {
			value, _ := strconv.Atoi(lineMatch["value"])
			lineOptions = append(lineOptions, AVOptionConst{
				Name:        lineMatch["name"],
				Description: lineMatch["description"],
				Encoding:    lineMatch["encoding"] == "E",
				Decoding:    lineMatch["decoding"] == "D",
				Audio:       lineMatch["audio"] == "A",
				Video:       lineMatch["video"] == "V",
				Value:       value,
			})
		}
	}

	options := []AVOption{}
	for i := 0; i < len(lineOptions); i++ {
		option := lineOptions[i].(AVOption)

		// look for following related enum/flag option lines
		for j := i + 1; j < len(lineOptions); j++ {
			optionConst, ok := lineOptions[j].(AVOptionConst)
			if !ok {
				break
			}
			i++

			option.Constants = append(option.Constants, optionConst)
		}

		options = append(options, option)
	}

	return options, nil
}

/*
Codecs:
D..... = Decoding supported
.E.... = Encoding supported
..V... = Video codec
..A... = Audio codec
..S... = Subtitle codec
...I.. = Intra frame-only codec
....L. = Lossy compression
.....S = Lossless compression
-------
 DEVILS test                 Test (decoders: test libtest ) (encoders: test libtest )
*/
var codecLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<decoding>.)` +
	`(?P<encoding>.)` +
	`(?P<codectype>.)` +
	`(?P<intraframe>.)` +
	`(?P<lossy>.)` +
	`(?P<lossless>.)` +
	`\s+` +
	`(?P<codecname>\S+)` +
	`\s*` +
	`(?P<description>.*?)` + // non-greedy to allow optional match below
	`\s*` +
	`(?:\(decoders: (?P<decoders>.*?)\))?` +
	`\s*` +
	`(?:\(encoders: (?P<encoders>.*?)\))?` +
	`\s*` +
	`$` +
	``)

func Codecs(ffmpegPath string) ([]Codec, error) {
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", "-codecs")

	_, optsMatches, matchErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		skipStartLinesCount: 1,
		headerEndSuffix:     "----",
		lineRe:              codecLineRe,
	})
	if matchErr != nil {
		return nil, matchErr
	}

	codecs := []Codec{}
	for _, opt := range optsMatches {
		codecType := opt["codectype"]
		var mediaType MediaType
		switch codecType {
		case "A":
			mediaType = MediaTypeAudio
		case "V":
			mediaType = MediaTypeVideo
		case "S":
			mediaType = MediaTypeSubtitle
		case "D":
			mediaType = MediaTypeData
		default:
			return nil, errors.New("unknown media type: " + codecType)
		}

		codecName := opt["codecname"]

		codecs = append(codecs, Codec{
			Name:        codecName,
			Description: opt["description"],
			MediaType:   mediaType,
			Decoders:    strings.Fields(opt["decoders"]),
			Encoders:    strings.Fields(opt["encoders"]),
		})
	}

	return codecs, nil
}

/*
Filter buffersink
  Buffer video frames, and make them available to the end of the filter graph.
    Inputs:
       #0: default (video)
    Outputs:
        none (sink filter)
buffersink AVOptions:
*/
var filterHeaderLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?:` +
	`(?P<inputouput>(?:Inputs|Outputs)):` +
	`|` +
	`#(?P<number>\d+):\s+(?P<name>\S+)\s+\((?P<mediatype>\S+)\)` +
	`|` +
	`(?P<dynamic>dynamic.*)` +
	`|` +
	`(?P<ignored>.*?)` +
	`)` +
	`\s*` +
	`$` +
	``)

/*
Filters:
  T.. = Timeline support
  .S. = Slice threading
  ..C = Command support
  A = Audio input/output
  V = Video input/output
  N = Dynamic number and/or type of input/output
  | = Source or sink filter
 ... abench            A->A       Benchmark part of a filtergraph.
*/
var filtersLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<timelinesupport>.)` +
	`(?P<slicethreading>.)` +
	`(?P<commandsupport>.)` +
	`\s+` +
	`(?P<filtername>\S+)` +
	`\s*` +
	`(?P<input>\S+)` +
	`->` +
	`(?P<output>\S+)` +
	`\s*` +
	`(?P<description>.*)` +
	`\s*` +
	`$` +
	``)

func Filters(ffmpegPath string) ([]Filter, error) {
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", "-filters")

	// TODO: filters output has no real header
	_, filtersMatches, filtersMatchErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		skipStartLinesCount: 1,
		headerEndSuffix:     "Source or sink filter",
		lineRe:              filtersLineRe,
	})
	if filtersMatchErr != nil {
		return nil, filtersMatchErr
	}

	filters := []Filter{}
	for _, filtersMatch := range filtersMatches {
		filterName := filtersMatch["filtername"]

		helpCmd := exec.CommandContext(context.Background(),
			ffmpegPath,
			"-hide_banner", "-help", "filter="+filterName,
		)

		ioMatches, avoptionsMatches, helpMatchErr := reMatchNamedGroupsCommandOutput(helpCmd, helpMatch{
			skipStartLinesCount: 2,
			headerLineRe:        filterHeaderLineRe,
			headerEndSuffix:     "AVOptions:",
			lineRe:              avoptionLineRe,
		})
		if helpMatchErr != nil {
			return nil, helpMatchErr
		}

		inputsDynamic := false
		outputsDynamic := false
		intputs := []FilterInputOutput{}
		outputs := []FilterInputOutput{}
		currentInputOutput := ""

		for _, ioMatch := range ioMatches {
			if ioMatch["ignored"] != "" {
				continue
			}

			if inputOutput := ioMatch["inputouput"]; inputOutput != "" {
				if inputOutput == "Outputs" {
					currentInputOutput = "outputs"
				} else if inputOutput == "Inputs" {
					currentInputOutput = "inputs"
				}
				continue
			}

			if ioMatch["dynamic"] != "" {
				if currentInputOutput == "inputs" {
					inputsDynamic = true
				} else if currentInputOutput == "outputs" {
					outputsDynamic = true
				}
				continue
			}

			mediaType := ioMatch["mediatype"]
			mt, ok := MediaTypeFromString[mediaType]
			if !ok {
				return nil, fmt.Errorf("unknown filter io media type %s", mediaType)
			}
			filterInputOutput := FilterInputOutput{
				Name:      ioMatch["name"],
				MediaType: mt,
			}

			if currentInputOutput == "inputs" {
				intputs = append(intputs, filterInputOutput)
			} else if currentInputOutput == "outputs" {
				outputs = append(outputs, filterInputOutput)
			}
		}

		options, optionsErr := parseAVOptions(avoptionsMatches)
		if optionsErr != nil {
			return nil, optionsErr
		}

		filters = append(filters, Filter{
			Name:            filterName,
			Description:     filtersMatch["description"],
			TimelineSupport: filtersMatch["timelinesupport"] == "T",
			SliceThreading:  filtersMatch["slicethreading"] == "S",
			CommandSupport:  filtersMatch["commandsupport"] == "C",
			InputsDynamic:   inputsDynamic,
			OutputsDynamic:  outputsDynamic,
			Inputs:          intputs,
			Outputs:         outputs,
			Options:         options,
		})
	}

	return filters, nil
}

/*
Encoder libfdk_aac [Fraunhofer FDK AAC]:
    General capabilities: delay small
    Threading capabilities: none
    Supported sample rates: 96000 88200 64000 48000 44100 32000 24000 22050 16000 12000 11025 8000
    Supported sample formats: s16
    Supported channel layouts: mono stereo 3.0 4.0 5.0 5.1 7.1(wide) 7.1
libfdk_aac AVOptions:
*/
var coderHeaderLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<key>\S.*):` +
	`\s*` +
	`(?P<value>.*)` +
	`$` +
	``)

/*
Encoders:
or
Decoders:
 V..... = Video
 A..... = Audio
 S..... = Subtitle
 .F.... = Frame-level multithreading
 ..S... = Slice-level multithreading
 ...X.. = Codec is experimental
 ....B. = Supports draw_horiz_band
 .....D = Supports direct rendering method 1
 ------
 VFSXBD a64multi             Multicolor charset for Commodore 64 (codec a64_multi)
*/
var codersLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<codectype>.)` +
	`(?P<framelevel>.)` +
	`(?P<slicelevel>.)` +
	`(?P<experimental>.)` +
	`(?P<drawhorizband>.)` +
	`(?P<directrenderingmethod1>.)` +
	`\s+` +
	`(?P<codername>\S+)` +
	`\s*` +
	`(?P<description>.*?)` + // non-greedy to allow optional match below
	`\s*` +
	`(?:\(codec (?P<codec>.*?)\))?` +
	`\s*` +
	`$` +
	``)

func coders(ffmpegPath string, arg string, helpArg string, headerEndSuffix string) ([]Coder, error) {
	// arg is -encoders/-decoders
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", arg)

	_, codersMatches, codersMatchErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		headerEndSuffix: headerEndSuffix,
		lineRe:          codersLineRe,
	})
	if codersMatchErr != nil {
		return nil, codersMatchErr
	}

	coders := []Coder{}
	for _, codersMatch := range codersMatches {
		codecType := codersMatch["codectype"]
		var mediaType MediaType
		switch codecType {
		case "A":
			mediaType = MediaTypeAudio
		case "V":
			mediaType = MediaTypeVideo
		case "S":
			mediaType = MediaTypeSubtitle
		default:
			return nil, errors.New("unknown media type: " + codecType)
		}

		coderName := codersMatch["codername"]

		helpCmd := exec.CommandContext(context.Background(),
			ffmpegPath,
			"-hide_banner", "-help", helpArg+"="+coderName,
		)

		headerMatches, avoptionsMatches, helpMatchErr := reMatchNamedGroupsCommandOutput(helpCmd, helpMatch{
			skipStartLinesCount: 1,
			headerLineRe:        coderHeaderLineRe,
			headerEndSuffix:     "AVOptions:",
			lineRe:              avoptionLineRe,
		})
		if helpMatchErr != nil {
			return nil, helpMatchErr
		}

		headerMap := map[string]string{}
		for _, headerMatch := range headerMatches {
			headerMap[headerMatch["key"]] = headerMatch["value"]
		}

		options, optionsErr := parseAVOptions(avoptionsMatches)
		if optionsErr != nil {
			return nil, optionsErr
		}

		coders = append(coders, Coder{
			Name:                  coderName,
			Description:           codersMatch["description"],
			MediaType:             mediaType,
			Codec:                 codersMatch["codec"],
			GeneralCapabilities:   strings.Fields(headerMap["General capabilities"]),
			ThreadingCapabilities: strings.Fields(headerMap["Threading capabilities"]),
			FrameRates:            strings.Fields(headerMap["Suppoted frame rates"]),
			PixelFormats:          strings.Fields(headerMap["Supported pixel formats"]),
			SampleRates:           strings.Fields(headerMap["Supported sample rates"]),
			SampleFormats:         strings.Fields(headerMap["Supported sample formats"]),
			ChannelLayouts:        strings.Fields(headerMap["Supported channel layouts"]),
			Options:               options,
		})
	}

	return coders, nil
}

func Encoders(ffmpegPath string) ([]Coder, error) {
	return coders(ffmpegPath, "-encoders", "encoder", "-----")
}

func Decoders(ffmpegPath string) ([]Coder, error) {
	return coders(ffmpegPath, "-decoders", "decoder", "-----")
}

/*
Muxer mp3 [MP3 (MPEG audio layer 3)]:
    Common extensions: mp3.
    Mime type: audio/mpeg.
    Default video codec: png.
    Default audio codec: mp3.
MP3 muxer AVOptions:
*/
var formatHeaderLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<key>\S.*):` +
	`\s*` +
	`(?P<value>.*)` +
	`\.` +
	`$` +
	``)

/*
File formats:
 D. = Demuxing supported
 .E = Muxing supported
 --
 D  3dostr          3DO STR
*/
var formatsLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<demuxing>.)` +
	`(?P<muxing>.)` +
	`\s+` +
	`(?P<formatname>\S+)` +
	`\s*` +
	`(?P<description>.*)` +
	`\s*` +
	`$` +
	``)

func formats(ffmpegPath string, arg string, helpArg string, headerEndSuffix string) ([]Format, error) {
	// arg -muxers/-demuxers helpArg muxer/demuxer
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", arg)

	_, formatsMatches, formatsMatchErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		headerEndSuffix: headerEndSuffix,
		lineRe:          formatsLineRe,
	})
	if formatsMatchErr != nil {
		return nil, formatsMatchErr
	}

	formats := []Format{}
	for _, formatsMatch := range formatsMatches {
		formatName := formatsMatch["formatname"]
		formatName = strings.Split(formatName, ",")[0]

		helpCmd := exec.CommandContext(context.Background(),
			ffmpegPath,
			"-hide_banner", "-help", helpArg+"="+formatName,
		)

		headerMatches, avoptionsMatches, helpMatchErr := reMatchNamedGroupsCommandOutput(helpCmd, helpMatch{
			skipStartLinesCount: 1,
			headerLineRe:        formatHeaderLineRe,
			headerEndSuffix:     "AVOptions:",
			lineRe:              avoptionLineRe,
		})
		if helpMatchErr != nil {
			return nil, helpMatchErr
		}

		headerMap := map[string]string{}
		for _, headerMatch := range headerMatches {
			headerMap[headerMatch["key"]] = headerMatch["value"]
		}

		options, optionsErr := parseAVOptions(avoptionsMatches)
		if optionsErr != nil {
			return nil, optionsErr
		}

		formats = append(formats, Format{
			Name:              formatName,
			Description:       formatsMatch["description"],
			Extensions:        splitRune(headerMap["Common extensions"], ','),
			MIMEType:          headerMap["Mime type"],
			DefaultAudioCodec: headerMap["Default audio codec"],
			DefaultVideoCodec: headerMap["Default video codec"],
			Options:           options,
		})
	}

	return formats, nil
}

func Muxers(ffmpegPath string) ([]Format, error) {
	return formats(ffmpegPath, "-muxers", "muxer", "--")
}

func Demuxers(ffmpegPath string) ([]Format, error) {
	return formats(ffmpegPath, "-demuxers", "demuxer", "--")
}

// TODO: cache full output?
func contextOptions(ffmpegPath string, headerEndSuffix string) ([]AVOption, error) {
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", "-help", "full")

	_, optMatches, optMatchesErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		headerEndSuffix: headerEndSuffix,
		lineRe:          avoptionLineRe,
	})
	if optMatchesErr != nil {
		return nil, optMatchesErr
	}

	return parseAVOptions(optMatches)
}

func CodecContextOptions(ffmpegPath string) ([]AVOption, error) {
	return contextOptions(ffmpegPath, "AVCodecContext AVOptions:")
}

func FormatContextOptions(ffmpegPath string) ([]AVOption, error) {
	return contextOptions(ffmpegPath, "AVFormatContext AVOptions:")
}

func FilterOptions(ffmpegPath string) ([]AVOption, error) {
	return contextOptions(ffmpegPath, "AVFilter AVOptions:")
}

/*
Pixel formats:
I.... = Supported Input  format for conversion
.O... = Supported Output format for conversion
..H.. = Hardware accelerated format
...P. = Paletted format
....B = Bitstream format
FLAGS NAME            NB_COMPONENTS BITS_PER_PIXEL
-----
IO... yuv420p                3            12
*/
var pixelFmtLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<input>.)` +
	`(?P<output>.)` +
	`(?P<hw>.)` +
	`(?P<paletted>.)` +
	`(?P<bitstream>.)` +
	`\s+` +
	`(?P<name>\S+)` +
	`\s+` +
	`(?P<nbcomponents>\d+)` +
	`\s+` +
	`(?P<bitsperpixel>\d+)` +
	`\s*` +
	`$` +
	``)

func PixelFmts(ffmpegPath string) ([]PixelFmt, error) {
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", "-pix_fmts")

	_, optMatches, optMatchesErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		headerEndSuffix: "----",
		lineRe:          pixelFmtLineRe,
	})
	if optMatchesErr != nil {
		return nil, optMatchesErr
	}

	pixelFmts := []PixelFmt{}
	for _, lineMatch := range optMatches {
		name := lineMatch["name"]

		nbComponents, nbComponentsErr := strconv.Atoi(lineMatch["nbcomponents"])
		if nbComponentsErr != nil {
			return nil, errors.New("failed to parse nbcomponents")
		}
		bitsPerPixel, bitsPerPixelErr := strconv.Atoi(lineMatch["bitsperpixel"])
		if bitsPerPixelErr != nil {
			return nil, errors.New("failed to parse bitsperpixel")
		}

		pixelFmts = append(pixelFmts, PixelFmt{
			Name:         name,
			Input:        lineMatch["input"] == "I",
			Output:       lineMatch["output"] == "O",
			NbComponents: uint(nbComponents),
			BitsPerPixel: uint(bitsPerPixel),
		})
	}

	return pixelFmts, nil
}

/*
name   depth
u8        8
*/
var sampleFmtLineRe = regexp.MustCompile(`` +
	`^` +
	`\s*` +
	`(?P<name>\S+)` +
	`\s+` +
	`(?P<depth>\d+)` +
	`\s*` +
	`$` +
	``)

func SampleFmts(ffmpegPath string) ([]SampleFmt, error) {
	cmd := exec.CommandContext(context.Background(), ffmpegPath, "-hide_banner", "-sample_fmts")

	_, optMatches, optMatchesErr := reMatchNamedGroupsCommandOutput(cmd, helpMatch{
		skipStartLinesCount: 0,
		lineRe:              sampleFmtLineRe,
	})
	if optMatchesErr != nil {
		return nil, optMatchesErr
	}

	sampleFmts := []SampleFmt{}
	for _, lineMatch := range optMatches {
		name := lineMatch["name"]

		depth, depthErr := strconv.Atoi(lineMatch["depth"])
		if depthErr != nil {
			return nil, errors.New("failed to parse depth")
		}

		sampleFmts = append(sampleFmts, SampleFmt{
			Name:  name,
			Depth: uint(depth),
		})
	}

	return sampleFmts, nil
}
