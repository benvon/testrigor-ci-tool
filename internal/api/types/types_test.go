package types

import (
	"testing"
)

func TestTestStatus_IsComplete(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"completed", "completed", true},
		{"failed", "failed", true},
		{"error", "error", true},
		{"cancelled", "cancelled", true},
		{"canceled", "canceled", true},
		{"in_progress", "in_progress", false},
		{"other", "other", false},
	}
	for _, c := range cases {
		ts := &TestStatus{Status: c.input}
		if got := ts.IsComplete(); got != c.expect {
			t.Errorf("IsComplete(%q) = %v, want %v", c.input, got, c.expect)
		}
	}
}

func TestTestStatus_IsInProgress(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		httpCode int
		expect   bool
	}{
		{"status in_progress", StatusInProgress, 0, true},
		{"code 227", "", StatusTestInProgress227, true},
		{"code 228", "", StatusTestInProgress228, true},
		{"not in progress", "completed", 0, false},
	}
	for _, c := range cases {
		ts := &TestStatus{Status: c.status, HTTPStatusCode: c.httpCode}
		if got := ts.IsInProgress(); got != c.expect {
			t.Errorf("IsInProgress(%q, %d) = %v, want %v", c.status, c.httpCode, got, c.expect)
		}
	}
}

func TestTestStatus_HasCrashes(t *testing.T) {
	ts := &TestStatus{Results: TestResults{Crash: 1}}
	if !ts.HasCrashes() {
		t.Error("HasCrashes() = false, want true")
	}
	ts.Results.Crash = 0
	if ts.HasCrashes() {
		t.Error("HasCrashes() = true, want false")
	}
}

func TestTestStatus_HasErrors(t *testing.T) {
	ts := &TestStatus{Errors: []TestError{{Error: "foo"}}}
	if !ts.HasErrors() {
		t.Error("HasErrors() = false, want true")
	}
	ts.Errors = nil
	if ts.HasErrors() {
		t.Error("HasErrors() = true, want false")
	}
}

func TestTestStatus_GetCrashErrors(t *testing.T) {
	ts := &TestStatus{
		Errors: []TestError{
			{Category: ErrorCategoryCrash, Error: "test crashed"},
			{Category: ErrorCategoryBlocker, Error: "test failed"},
			{Category: "", Error: "test crashed"},
			{Category: "", Error: "other error"},
		},
	}
	crashErrs := ts.GetCrashErrors()
	if len(crashErrs) != 3 {
		t.Errorf("GetCrashErrors() = %d, want 3", len(crashErrs))
	}
}
