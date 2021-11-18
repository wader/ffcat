package goffmpeg_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/wader/ffcat/internal/goffmpeg"
	"github.com/wader/ffcat/internal/goffmpeg/cmdgroup"
)

func TestError(t *testing.T) {
	c := &goffmpeg.FFmpegCmd{
		Context: context.Background(),
		Flags:   []string{"-y", "-v", "trace"},
		Inputs:  []*goffmpeg.Input{{Flags: []string{"-t", "100"}, Format: "lavfi", File: "sine"}},
		Outputs: []*goffmpeg.Output{
			{
				Format: "mp3",
				File:   "output",
			},
		},
	}

	err := c.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = c.Wait()

	args := c.Args()
	log.Printf("err: %#+v\n", err)
	log.Printf("args: %#+v\n", args)
	log.Printf("c.StderrBuffer(): %s\n", c.StderrBuffer())
}

func TestArgs(t *testing.T) {

	i1 := &goffmpeg.Input{File: "test"}
	i2 := &goffmpeg.Input{File: os.Stdin}

	c := &goffmpeg.FFmpegCmd{
		Inputs: []*goffmpeg.Input{i1, i2},
		Outputs: []*goffmpeg.Output{
			{
				Maps: []*goffmpeg.Map{
					{Input: i1, Specifier: "a", Codec: "mp3", Options: map[string]string{"threads": "4"}},
					{Input: i1, Specifier: "a", Codec: "mp3"},
					{Input: i2, Specifier: "a", Codec: "mp3"},
					{Input: i2, Specifier: "a", Codec: "mp3", Flags: []string{"-threads", "4"}},
				},
				Format: "mp4",
				Flags:  []string{"-codec:a", "aac"},
				File:   "output",
				Metadata: &goffmpeg.Metadata{
					Title: "title",
				},
			},
		},
	}

	args := c.Args()

	log.Printf("args: %#+v\n", args)

}

func TestArgs2(t *testing.T) {

	b := &bytes.Buffer{}

	i1 := &goffmpeg.Input{Format: "lavfi", File: "sine"}
	c := &goffmpeg.FFmpegCmd{
		Context: context.Background(),
		Inputs:  []*goffmpeg.Input{i1},
		Outputs: []*goffmpeg.Output{
			{
				Maps: []*goffmpeg.Map{
					{Input: i1, Specifier: "0", Codec: "mp3"},
				},
				Format: "mp3",
				File:   b,
				Metadata: &goffmpeg.Metadata{
					Title:   "title2",
					Comment: "asdsad",
				},
				Flags: []string{"-t", "1"},
			},
		},
	}

	err := c.Start()
	log.Printf("err: %#+v\n", err)
	err = c.Wait()
	log.Printf("err: %#+v\n", err)

	log.Printf("b: %#+v\n", b)

}

func TestPipe(t *testing.T) {

	b := &bytes.Buffer{}

	pr, pw, _ := os.Pipe()
	//bpr, bpw := bufio.NewReader(pr), bufio.NewWriter(pw)

	i1 := &goffmpeg.Input{Format: "lavfi", File: "sine"}
	c1 := &goffmpeg.FFmpegCmd{
		Context: context.Background(),
		Inputs:  []*goffmpeg.Input{i1},
		Outputs: []*goffmpeg.Output{
			{
				Maps: []*goffmpeg.Map{
					{Input: i1, Specifier: "a:0", Codec: "pcm_s16le"},
				},
				Format: "wav",
				File:   pw,
				Flags:  []string{"-t", "1"},
			},
		},
		Stderr: os.Stderr,
	}

	i2 := &goffmpeg.Input{File: pr}
	c2 := &goffmpeg.FFmpegCmd{
		Context: context.Background(),
		Inputs:  []*goffmpeg.Input{i2},
		Outputs: []*goffmpeg.Output{
			{
				Maps: []*goffmpeg.Map{
					{Input: i2, Specifier: "a:0"},
				},
				Format: "wav",
				File:   b,
			},
		},
		Stderr: os.Stderr,
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		err := c1.Start()
		pw.Close()
		// log.Printf("err: %#+v\n", err)
		err = c1.Wait()
		log.Printf("err: %#+v\n", err)
		wg.Done()
	}()

	go func() {
		err := c2.Start()
		log.Printf("err: %#+v\n", err)
		err = c2.Wait()
		log.Printf("err: %#+v\n", err)
		wg.Done()
	}()

	wg.Wait()

	//log.Printf("b: %#+v\n", b)

}

func TestParseProgress(t *testing.T) {
	type call struct {
		line string
		r    bool
	}
	testCases := []struct {
		calls            []call
		expectedProgress goffmpeg.Progress
	}{
		{
			calls: []call{
				{line: "frame=241", r: false},
				{line: "fps=79.81", r: false},
				{line: "stream_0_1_q=10.0", r: false},
				{line: "stream_0_1_psnr_y=62.39", r: false},
				{line: "stream_0_1_psnr_u=56.45", r: false},
				{line: "stream_0_1_psnr_v=54.76", r: false},
				{line: "stream_0_1_psnr_all=58.80", r: false},
				{line: "bitrate= 107.1kbits/s", r: false},
				{line: "total_size=116071", r: false},
				{line: "out_time_us=8674000", r: false},
				{line: "out_time_ms=8674001", r: false},
				{line: "out_time=00:00:08.674000", r: false},
				{line: "dup_frames=1", r: false},
				{line: "drop_frames=2", r: false},
				{line: "speed=2.87x", r: false},
				{line: "progress=continue", r: true},
			},
			expectedProgress: goffmpeg.Progress{
				Frame: 241,
				FPS:   79.81,
				Outputs: []goffmpeg.ProgressOutput{
					{
						Streams: []goffmpeg.ProgressStream{
							{Q: 0, PSNRY: 0, PSNRU: 0, PSNRV: 0, PSNRAll: 0},
							{Q: 10, PSNRY: 62.39, PSNRU: 56.45, PSNRV: 54.76, PSNRAll: 58.8},
						},
					},
				},
				BitRate:    107100,
				TotalSize:  116071,
				OutTimeUS:  8674000,
				OutTimeMS:  8674001,
				OutTime:    "00:00:08.674000",
				DupFrames:  1,
				DropFrames: 2,
				Speed:      2.87,
				Progress:   "continue",
			},
		},
	}
	for i, tC := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			p := goffmpeg.Progress{}
			for _, c := range tC.calls {
				actualR := goffmpeg.ParseProgress(&p, c.line)
				if c.r != actualR {
					t.Errorf("expected %v, got %v", c.r, actualR)
				}
			}
			if !reflect.DeepEqual(tC.expectedProgress, p) {
				t.Errorf("expected %#v, got %#v", tC.expectedProgress, p)
			}
		})
	}
}

func TestFilterGraph(t *testing.T) {
	defer leakChecks(t)()

	generate0R, generate0W, _ := os.Pipe()
	generate1R, generate1W, _ := os.Pipe()
	cg, ctx := cmdgroup.WithContext(context.Background())
	errs := cg.Run(
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			FilterGraph: &goffmpeg.FilterGraph{
				{
					{Name: "sine", Options: map[string]string{"sample_rate": "44100", "frequency": "100"}},
					{Name: "atrim", Options: map[string]string{"duration": "1s"}, Outputs: []string{"atrim0"}},
				},
				{
					{Name: "sine", Options: map[string]string{"sample_rate": "44100", "frequency": "600"}},
					{Name: "atrim", Options: map[string]string{"duration": "1s"}, Outputs: []string{"atrim1"}},
				},
				{
					{Name: "anoisesrc", Options: map[string]string{"sample_rate": "44100", "seed": "1"}},
					{Name: "atrim", Options: map[string]string{"duration": "1s"}, Outputs: []string{"atrim2"}},
				},
				{
					{Name: "anullsrc", Options: map[string]string{"sample_rate": "44100"}},
					{Name: "atrim", Options: map[string]string{"duration": "1s"}, Outputs: []string{"atrim3"}},
				},
				{
					{
						Name:    "amerge",
						Options: map[string]string{"inputs": "2"},
						Inputs:  []string{"atrim0", "atrim1"},
						Outputs: []string{"amerge0"},
					},
				},
				{
					{
						Name:    "amerge",
						Options: map[string]string{"inputs": "2"},
						Inputs:  []string{"atrim2", "atrim3"},
						Outputs: []string{"amerge1"},
					},
				},
				{
					{
						Name:    "concat",
						Options: map[string]string{"n": "2", "a": "1", "v": "0"},
						Inputs:  []string{"amerge0", "amerge1"},
						Outputs: []string{"concat0"},
					},
				},
				{
					{
						Name:    "asplit",
						Options: map[string]string{"outputs": "2"},
						Inputs:  []string{"concat0"},
						Outputs: []string{"out0", "out1"},
					},
				},
			},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "[out0]", Codec: "pcm_s16le"}},
					Format: "wav",
					File:   generate0W,
				},
				{
					Maps:   []*goffmpeg.Map{{Specifier: "[out1]", Codec: "pcm_s16le"}},
					Format: "wav",
					File:   generate1W,
				},
			},
			CloseAfterStart: []io.Closer{generate0W, generate1W},
			CloseAfterWait:  []io.Closer{generate0R, generate1R},
		},
		&goffmpeg.FFmpegCmd{
			Flags:  []string{"-y"},
			Inputs: []*goffmpeg.Input{{File: generate0R}},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "0:0", Options: map[string]string{"b": "128k"}}},
					Format: "ogg",
					File:   "out128.ogg",
				},
			},
		},
		&goffmpeg.FFmpegCmd{
			Flags:  []string{"-y"},
			Inputs: []*goffmpeg.Input{{File: generate1R}},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "0:0", Options: map[string]string{"b": "256k"}}},
					Format: "ogg",
					File:   "out256.ogg",
				},
			},
		},
	)
	log.Printf("errs: %#+v\n", errs)
}

func TestOsPipe(t *testing.T) {
	defer leakChecks(t)()

	generateR, generateW, _ := os.Pipe()
	passthruR, passthruW, _ := os.Pipe()
	probeR, probeW, _ := os.Pipe()
	cg, ctx := cmdgroup.WithContext(context.Background())
	probe := &goffmpeg.FFProbeCmd{Context: ctx, Input: goffmpeg.Input{File: probeR}}
	errs := cg.Run(
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			FilterGraph: &goffmpeg.FilterGraph{
				{
					{Name: "sine"},
					{Name: "atrim", Options: map[string]string{"duration": "1s"}, Outputs: []string{"out0"}},
				},
			},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "[out0]", Codec: "pcm_s16le"}},
					Format: "wav",
					File:   generateW,
				},
			},
			CloseAfterStart: []io.Closer{generateW},
			CloseAfterWait:  []io.Closer{generateR},
		},
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			Inputs:  []*goffmpeg.Input{{File: generateR}},
			Outputs: []*goffmpeg.Output{
				{
					Format: "wav",
					File:   passthruW,
				},
			},
			CloseAfterStart: []io.Closer{passthruW},
			CloseAfterWait:  []io.Closer{passthruR},
		},
		exec.CommandContext(ctx, "true"),
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			Inputs:  []*goffmpeg.Input{{File: passthruR}},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "0:0", Options: map[string]string{"b": "128k"}}},
					Format: "ogg",
					File:   probeW,
				},
			},
			CloseAfterStart: []io.Closer{probeW},
			CloseAfterWait:  []io.Closer{probeW},
		},
		probe,
	)

	log.Printf("probe: %#+v\n", probe)

	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestIoPipe(t *testing.T) {
	defer leakChecks(t)()

	generateR, generateW := io.Pipe()
	passthruR, passthruW := io.Pipe()
	probeR, probeW := io.Pipe()
	cg, ctx := cmdgroup.WithContext(context.Background())
	probe := &goffmpeg.FFProbeCmd{Context: ctx, Input: goffmpeg.Input{File: probeR}}
	errs := cg.Run(
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			FilterGraph: &goffmpeg.FilterGraph{
				{
					{Name: "sine"},
					{Name: "atrim", Options: map[string]string{"duration": "1s"}, Outputs: []string{"out0"}},
				},
			},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "[out0]", Codec: "pcm_s16le"}},
					Format: "wav",
					File:   generateW,
				},
			},
			CloseAfterWait: []io.Closer{generateW},
		},
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			Inputs:  []*goffmpeg.Input{{File: generateR}},
			Outputs: []*goffmpeg.Output{
				{
					Format: "wav",
					File:   passthruW,
				},
			},
			CloseAfterWait: []io.Closer{passthruW},
		},
		&goffmpeg.FFmpegCmd{
			Context: ctx,
			Inputs:  []*goffmpeg.Input{{File: passthruR}},
			Outputs: []*goffmpeg.Output{
				{
					Maps:   []*goffmpeg.Map{{Specifier: "0:0", Options: map[string]string{"b": "128k"}}},
					Format: "ogg",
					File:   probeW,
				},
			},
			CloseAfterWait: []io.Closer{probeW},
		},
		probe,
	)

	log.Printf("probe: %#+v\n", probe)

	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
