package goffmpeg_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"ff/internal/goffmpeg"

	"github.com/fortytw2/leaktest"
	"github.com/wader/osleaktest"
)

func leakChecks(t *testing.T) func() {
	leakFn := leaktest.Check(t)
	osLeakFn := osleaktest.Check(t)
	return func() {
		leakFn()
		osLeakFn()
	}
}

func generateTestData(t *testing.T, format string, acodec string, vcodec string, duration time.Duration, metadata *goffmpeg.Metadata) []byte {
	data := &bytes.Buffer{}
	var inputs []*goffmpeg.Input
	var maps []*goffmpeg.Map

	if acodec != "" {
		i := &goffmpeg.Input{Format: "lavfi", File: "sine"}
		inputs = append(inputs, i)
		maps = append(maps, &goffmpeg.Map{Input: i, Specifier: "0", Codec: acodec})
	}
	if vcodec != "" {
		i := &goffmpeg.Input{Format: "lavfi", File: "testsrc"}
		inputs = append(inputs, i)
		maps = append(maps, &goffmpeg.Map{Input: i, Specifier: "0", Codec: vcodec})
	}

	c := &goffmpeg.FFmpegCmd{
		Context: context.Background(),
		Inputs:  inputs,
		Outputs: []*goffmpeg.Output{
			{
				Maps:     maps,
				Format:   format,
				File:     data,
				Metadata: metadata,
				Flags:    []string{"-t", strconv.Itoa(int(duration.Seconds()))},
			},
		},
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}

	return data.Bytes()
}
