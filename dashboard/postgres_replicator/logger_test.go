package transfer

import (
	"bytes"
	"fmt"
	"log"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tableName := "foobar"
	logger := NewLogger(tableName)
	expectedPrefix := fmt.Sprintf("[%s] ", tableName)
	actualPrefix := logger.Prefix()
	if actualPrefix != expectedPrefix {
		t.Errorf(`Expected prefix "%s". Actual prefix: "%s"`, tableName, actualPrefix)
	}
}

func TestErrorf(t *testing.T) {
	// Test that prefix is properly set during an error.
	var buf bytes.Buffer
	tableName := "foobar"
	errorMsg := "Sample error"
	expectedLog := fmt.Sprintf("[ERROR][%s] %s\n", tableName, errorMsg)

	logger := NewLogger(tableName)
	logger.SetOutput(&buf)
	logger.SetFlags(log.Lmsgprefix)
	logger.Errorf(errorMsg)
	logString := buf.String()

	if logString != expectedLog {
		t.Errorf(`Expected log "%s". Actual log "%s"`, expectedLog, logString)
	}

	// Test that prefix is restored after an error message.
	expectedPrefix := fmt.Sprintf("[%s] ", tableName)
	if logger.Prefix() != expectedPrefix {
		t.Errorf(`After an error message, expected log "%s". Actual log "%s"`, expectedLog, logString)
	}
}
