package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
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

const (
	reset = "\x1b[0m"
	bold  = "\x1b[1m"

	colorError = "\x1b[31m"
	colorWarn  = "\x1b[33m"
	colorInfo  = "\x1b[32m"
	colorDebug = "\x1b[35m"

	pfxError = "[ERROR]"
	pfxWarn  = "[WARN] "
	pfxInfo  = "[INFO] "
	pfxDebug = "[DEBUG]"
)

type AsyncLogger struct {
	level      Level
	w          io.Writer
	logChan    chan string
	done       chan struct{}
	wg         sync.WaitGroup
	bufferSize int

	// Prefixes
	pfxError string
	pfxWarn  string
	pfxInfo  string
	pfxDebug string
}

func NewAsyncLogger(opts ...func(*AsyncLogger)) *AsyncLogger {
	l := &AsyncLogger{
		level:      Info,
		w:          os.Stdout,
		bufferSize: 64, // Default buffer size
	}

	for _, opt := range opts {
		opt(l)
	}

	l.logChan = make(chan string, l.bufferSize)

	if l.w == os.Stdout {
		l.pfxError = bold + colorError + pfxError + reset
		l.pfxWarn = bold + colorWarn + pfxWarn + reset
		l.pfxInfo = bold + colorInfo + pfxInfo + reset
		l.pfxDebug = bold + colorDebug + pfxDebug + reset
	} else {
		l.pfxError, l.pfxWarn, l.pfxInfo, l.pfxDebug = pfxError, pfxWarn, pfxInfo, pfxDebug
	}

	l.wg.Go(l.run)

	return l
}

// I/O worker
func (l *AsyncLogger) run() {
	for msg := range l.logChan {
		fmt.Fprint(l.w, msg)
	}
}

// Close stops the logger and waits for all queued logs to be written
func (l *AsyncLogger) Close() {
	close(l.logChan)
	l.wg.Wait()
}

func (l *AsyncLogger) log(prefix string, format string, a ...any) {
	msg := fmt.Sprintf("%s %s %s\n", prefix, time.Now().Format(time.RFC3339), fmt.Sprintf(format, a...))

	select {
	case l.logChan <- msg:
	default:
		// Buffer overflow
	}
}

func (l *AsyncLogger) Errorf(format string, a ...any) {
	if l.level >= Error {
		l.log(l.pfxError, format, a...)
	}
}

func (l *AsyncLogger) Warnf(format string, a ...any) {
	if l.level >= Warn {
		l.log(l.pfxWarn, format, a...)
	}
}

func (l *AsyncLogger) Infof(format string, a ...any) {
	if l.level >= Info {
		l.log(l.pfxInfo, format, a...)
	}
}

func (l *AsyncLogger) Debugf(format string, a ...any) {
	if l.level >= Debug {
		l.log(l.pfxDebug, format, a...)
	}
}

// Options

func WithLevel(level Level) func(*AsyncLogger) {
	return func(l *AsyncLogger) { l.level = level }
}

func WithWriter(w io.Writer) func(*AsyncLogger) {
	return func(l *AsyncLogger) { l.w = w }
}

func WithBufferSize(size int) func(*AsyncLogger) {
	return func(l *AsyncLogger) { l.bufferSize = size }
}
