package log

import (
	"fmt"
	"os"
)

// Logger used by socks5 server
type Logger interface {
	Errorf(format string, a ...any)
	Warnf(format string, a ...any)
	Infof(format string, a ...any)
	Debugf(format string, a ...any)
}

type DiscardLogger struct{}

func (d *DiscardLogger) Errorf(format string, a ...any) {}
func (d *DiscardLogger) Warnf(format string, a ...any)  {}
func (d *DiscardLogger) Infof(format string, a ...any)  {}
func (d *DiscardLogger) Debugf(format string, a ...any) {}

type Level int

const (
	None Level = iota
	Error
	Warn
	Info
	Debug
)

type StdoutLogger struct {
	level Level
}

func (s *StdoutLogger) Errorf(format string, a ...any) {
	if s.level >= Error {
		fmt.Fprintf(os.Stdout, "[ERROR] "+format+"\n", a...)
	}
}

func (s *StdoutLogger) Warnf(format string, a ...any) {
	if s.level >= Warn {
		fmt.Fprintf(os.Stdout, "[WARN] "+format+"\n", a...)
	}
}

func (s *StdoutLogger) Infof(format string, a ...any) {
	if s.level >= Info {
		fmt.Fprintf(os.Stdout, "[INFO] "+format+"\n", a...)
	}
}

func (s *StdoutLogger) Debugf(format string, a ...any) {
	if s.level >= Debug {
		fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", a...)
	}
}

func NewStdoutLogger(opts ...func(*StdoutLogger)) *StdoutLogger {
	l := &StdoutLogger{
		level: Info, // Default to info level
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithLevel sets the log level for StdoutLogger
func WithLevel(level Level) func(*StdoutLogger) {
	return func(l *StdoutLogger) {
		l.level = level
	}
}
