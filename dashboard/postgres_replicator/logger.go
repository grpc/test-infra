package transfer

import (
	"fmt"
	"log"
	"os"
)

// Logger proves per-table logging.
type Logger struct {
	*log.Logger
}

// NewLogger returns a new Logger.
func NewLogger(tableName string) *Logger {
	prefix := fmt.Sprintf("[%s] ", tableName)
	prefixedLogger := log.New(os.Stderr, prefix, log.Ldate|log.Ltime|log.Lmsgprefix)
	return &Logger{prefixedLogger}
}

// Errorf adds an "[ERROR]" tag to the log.
func (tl *Logger) Errorf(format string, v ...interface{}) {
	prefix := tl.Prefix()
	tl.SetPrefix("[ERROR]" + prefix)
	tl.Printf(format, v...)
	tl.SetPrefix(prefix)
}
