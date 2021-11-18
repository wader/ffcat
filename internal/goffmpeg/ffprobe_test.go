package goffmpeg_test

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"github.com/wader/ffcat/internal/goffmpeg"
)

func TestProbe(t *testing.T) {
	defer leakChecks(t)()

	testData := generateTestData(t, "wav", "pcm_s16le", "", 1*time.Second, nil)
	p := goffmpeg.FFProbeCmd{Context: context.Background(), Input: goffmpeg.Input{File: bytes.NewBuffer(testData)}}
	if err := p.Run(); err != nil {
		t.Fatal(err)
	}

	log.Printf("pi: %#+v\n", p.ProbeResult)
}
