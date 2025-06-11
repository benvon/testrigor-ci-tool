package types

// TestRunOptions represents the options for starting a test run
type TestRunOptions struct {
	TestCaseUUIDs              []string
	Labels                     []string
	ExcludedLabels             []string
	URL                        string
	BranchName                 string
	CommitHash                 string
	CustomName                 string
	ForceCancelPreviousTesting bool
	MakeXrayReports            bool
}

// TestRunResult represents the result of starting a test run
type TestRunResult struct {
	TaskID     string
	BranchName string
}

// TestError represents an error that occurred during a test run
type TestError struct {
	Category    string
	Error       string
	Occurrences int
	Severity    string
	DetailsURL  string
}

// TestResults represents the overall results of a test run
type TestResults struct {
	Total      int
	InQueue    int
	InProgress int
	Failed     int
	Passed     int
	Canceled   int
	NotStarted int
	Crash      int
}

// TestStatus represents the current status of a test run
type TestStatus struct {
	Status         string
	DetailsURL     string
	TaskID         string
	Errors         []TestError
	Results        TestResults
	HTTPStatusCode int
}
