package transfer

import (
	"bytes"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("testTable")
	want := "[testTable] "
	got := logger.Prefix()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf(`Logger.NewLogger("testTable") diff (-want +got):\n%s`, diff)
	}
}

func TestErrorf(t *testing.T) {
	// Test that "Error: " is prefixed to the format string.
	var buf bytes.Buffer
	logger := NewLogger("testTable")
	want := "[testTable] Error: Sample error\n"

	logger.SetOutput(&buf)
	logger.SetFlags(log.Lmsgprefix)
	logger.Errorf("Sample error")
	got := buf.String()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Logger.Errorf() diff (-want +got):\n%s", diff)
	}
}
