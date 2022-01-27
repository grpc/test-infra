package xds

import (
	"log"
)

// Logger implements the Logger interface required.
type Logger struct {
}

// Debugf print out debug information.
func (logger Logger) Debugf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// Infof print out useful information.
func (logger Logger) Infof(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// Warnf print out warnings.
func (logger Logger) Warnf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// Errorf print out the error message and stop the process.
func (logger Logger) Errorf(format string, args ...interface{}) {
	log.Fatalf(format+"\n", args...)
}
