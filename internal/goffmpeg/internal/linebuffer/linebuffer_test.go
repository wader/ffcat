package linebuffer_test

import (
	"strconv"
	"strings"
	"testing"

	"ff/internal/goffmpeg/internal/linebuffer"
)

func Test(t *testing.T) {
	testCases := []struct {
		writes   string
		expected string
	}{
		{writes: "", expected: ""},
		{writes: "a\n,b\n", expected: "a\nb\n"},
		{writes: "a\r", expected: "a\r"},
		{writes: "a\n,b\n,c\n,d\n", expected: "b\nc\nd\n"},
		{writes: "a\n,b\n,c\n,1\n,2\n,3\n", expected: "1\n2\n3\n"},
	}
	for i, tC := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			lb := linebuffer.NewLastLines(3)
			for _, w := range strings.Split(tC.writes, ",") {
				lb.Write([]byte(w))
			}
			lb.Close()

			if tC.expected != lb.String() {
				t.Errorf("expected %q, got %q", tC.expected, lb.String())
			}
		})
	}
}
