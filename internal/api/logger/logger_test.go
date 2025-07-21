package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_Info_Warning_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewWithWriter(buf, false)

	l.Info("info message: %d", 1)
	l.Warning("warn message: %s", "foo")
	l.Error("error message: %v", 123)

	out := buf.String()
	if !strings.Contains(out, "INFO: info message: 1") {
		t.Errorf("Info() output missing: %q", out)
	}
	if !strings.Contains(out, "WARNING: warn message: foo") {
		t.Errorf("Warning() output missing: %q", out)
	}
	if !strings.Contains(out, "ERROR: error message: 123") {
		t.Errorf("Error() output missing: %q", out)
	}
}

func TestLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewWithWriter(buf, false)
	l.Debug("should not print")
	if strings.Contains(buf.String(), "DEBUG") {
		t.Error("Debug() should not print when debug=false")
	}

	buf.Reset()
	l = NewWithWriter(buf, true)
	l.Debug("debug message: %s", "bar")
	if !strings.Contains(buf.String(), "DEBUG: debug message: bar") {
		t.Error("Debug() output missing when debug=true")
	}
}

func TestLogger_Printf_Println(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewWithWriter(buf, false)
	l.Printf("printf %d", 42)
	l.Println("foo", 123)
	out := buf.String()
	if !strings.Contains(out, "INFO: printf 42") {
		t.Errorf("Printf() output missing: %q", out)
	}
	if !strings.Contains(out, "INFO: foo 123") {
		t.Errorf("Println() output missing: %q", out)
	}
}
