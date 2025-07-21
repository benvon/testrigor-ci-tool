package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Logger provides a structured logging interface
type Logger struct {
	output io.Writer
	debug  bool
}

// New creates a new logger instance
func New(debug bool) *Logger {
	return &Logger{
		output: os.Stdout,
		debug:  debug,
	}
}

// NewWithWriter creates a new logger with a custom writer
func NewWithWriter(writer io.Writer, debug bool) *Logger {
	return &Logger{
		output: writer,
		debug:  debug,
	}
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

// Debug logs a debug message (only if debug mode is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.log("DEBUG", format, args...)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	l.log("WARNING", format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

// log is the internal logging method
func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf(format, args...)
	if _, err := fmt.Fprintf(l.output, "[%s] %s: %s\n", timestamp, level, message); err != nil {
		// Log to stderr if we can't write to the output
		fmt.Fprintf(os.Stderr, "Failed to write to logger output: %v\n", err)
	}
}

// Printf provides backward compatibility for existing code
func (l *Logger) Printf(format string, args ...interface{}) {
	l.Info(format, args...)
}

// Println provides backward compatibility for existing code
func (l *Logger) Println(args ...interface{}) {
	l.Info("%s", strings.TrimRight(fmt.Sprintln(args...), "\n"))
}
