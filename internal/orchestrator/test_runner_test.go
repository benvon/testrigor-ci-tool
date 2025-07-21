package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLogger implements the Logger interface for testing.
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Printf(format string, args ...interface{}) {
	m.logs = append(m.logs, format)
}

func (m *MockLogger) Println(args ...interface{}) {
	m.logs = append(m.logs, "println")
}

// MockTestRigorClient is a mock implementation for testing the orchestrator.
type MockTestRigorClient struct {
	mock.Mock
}

func (m *MockTestRigorClient) StartTestRun(ctx context.Context, opts types.TestRunOptions) (*types.TestRunResult, error) {
	args := m.Called(ctx, opts)
	if result := args.Get(0); result != nil {
		return result.(*types.TestRunResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTestRigorClient) GetTestStatus(ctx context.Context, branchName string, labels []string) (*types.TestStatus, error) {
	args := m.Called(ctx, branchName, labels)
	if status := args.Get(0); status != nil {
		return status.(*types.TestStatus), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTestRigorClient) GetJUnitReport(ctx context.Context, taskID string) ([]byte, error) {
	args := m.Called(ctx, taskID)
	if data := args.Get(0); data != nil {
		return data.([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestNewTestRunner(t *testing.T) {
	cfg := &config.Config{
		TestRigor: config.TestRigorConfig{
			AuthToken: "test-token",
			AppID:     "test-app",
			APIURL:    "https://api.testrigor.com/api/v1",
		},
	}

	httpClient := client.NewDefaultHTTPClient()
	logger := &MockLogger{}

	runner := NewTestRunner(cfg, httpClient, logger)

	assert.NotNil(t, runner)
	assert.Equal(t, cfg, runner.config)
	assert.Equal(t, logger, runner.logger)
	assert.NotNil(t, runner.apiClient)
}

func TestNewTestRunner_WithNilLogger(t *testing.T) {
	cfg := &config.Config{}
	httpClient := client.NewDefaultHTTPClient()

	runner := NewTestRunner(cfg, httpClient, nil)

	assert.NotNil(t, runner)
	assert.NotNil(t, runner.logger)
	assert.IsType(t, DefaultLogger{}, runner.logger)
}

func TestTestRunner_ExecuteTestRun_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	// Create a mock client and inject it
	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	runConfig := TestRunConfig{
		Options: types.TestRunOptions{
			BranchName: "test-branch",
			Labels:     []string{"smoke"},
		},
		PollInterval: 100 * time.Millisecond,
		Timeout:      1 * time.Second,
		FetchReport:  false,
		DebugMode:    false,
	}

	startResult := &types.TestRunResult{
		TaskID:     "task-123",
		BranchName: "test-branch",
	}

	finalStatus := &types.TestStatus{
		Status: types.StatusCompleted,
		Results: types.TestResults{
			Total:  5,
			Passed: 5,
			Failed: 0,
			Crash:  0,
		},
	}

	// Set up mock expectations
	mockClient.On("StartTestRun", mock.Anything, runConfig.Options).Return(startResult, nil)
	mockClient.On("GetTestStatus", mock.Anything, "test-branch", []string{"smoke"}).Return(finalStatus, nil)

	// Execute
	ctx := context.Background()
	result, err := runner.ExecuteTestRun(ctx, runConfig)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "task-123", result.TaskID)
	assert.Equal(t, "test-branch", result.BranchName)
	assert.True(t, result.Success)
	assert.Equal(t, finalStatus, result.Status)

	mockClient.AssertExpectations(t)
	assert.NotEmpty(t, logger.logs)
}

func TestTestRunner_ExecuteTestRun_StartError(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	runConfig := TestRunConfig{
		Options: types.TestRunOptions{
			BranchName: "test-branch",
		},
		PollInterval: 100 * time.Millisecond,
		Timeout:      1 * time.Second,
	}

	expectedError := errors.New("failed to start test")
	mockClient.On("StartTestRun", mock.Anything, runConfig.Options).Return(nil, expectedError)

	// Execute
	ctx := context.Background()
	result, err := runner.ExecuteTestRun(ctx, runConfig)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to start test run")

	mockClient.AssertExpectations(t)
}

func TestTestRunner_ExecuteTestRun_WithReport(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	runConfig := TestRunConfig{
		Options: types.TestRunOptions{
			BranchName: "test-branch",
		},
		PollInterval: 100 * time.Millisecond,
		Timeout:      1 * time.Second,
		FetchReport:  true,
		DebugMode:    false,
	}

	startResult := &types.TestRunResult{
		TaskID:     "task-123",
		BranchName: "test-branch",
	}

	finalStatus := &types.TestStatus{
		Status: types.StatusCompleted,
		Results: types.TestResults{
			Total:  1,
			Passed: 1,
			Failed: 0,
			Crash:  0,
		},
	}

	reportData := []byte(`<?xml version="1.0"?><testsuite></testsuite>`)

	// Set up mock expectations
	mockClient.On("StartTestRun", mock.Anything, runConfig.Options).Return(startResult, nil)
	mockClient.On("GetTestStatus", mock.Anything, "test-branch", mock.Anything).Return(finalStatus, nil)
	mockClient.On("GetJUnitReport", mock.Anything, "task-123").Return(reportData, nil)

	// Execute
	ctx := context.Background()
	result, err := runner.ExecuteTestRun(ctx, runConfig)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ReportPath)

	mockClient.AssertExpectations(t)
}

func TestTestRunner_monitorTestExecution_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	runConfig := TestRunConfig{
		Options: types.TestRunOptions{
			Labels: []string{"smoke"},
		},
		PollInterval: 50 * time.Millisecond,
		Timeout:      500 * time.Millisecond,
		DebugMode:    false,
	}

	completedStatus := &types.TestStatus{
		Status: types.StatusCompleted,
		Results: types.TestResults{
			Total:  1,
			Passed: 1,
		},
	}

	mockClient.On("GetTestStatus", mock.Anything, "test-branch", []string{"smoke"}).Return(completedStatus, nil)

	// Execute
	ctx := context.Background()
	status, err := runner.monitorTestExecution(ctx, "test-branch", runConfig)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, completedStatus, status)
	mockClient.AssertExpectations(t)
}

func TestTestRunner_monitorTestExecution_Timeout(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	runConfig := TestRunConfig{
		Options:      types.TestRunOptions{},
		PollInterval: 50 * time.Millisecond,
		Timeout:      100 * time.Millisecond, // Very short timeout
		DebugMode:    false,
	}

	inProgressStatus := &types.TestStatus{
		Status: types.StatusInProgress,
		Results: types.TestResults{
			Total:      1,
			InProgress: 1,
		},
	}

	mockClient.On("GetTestStatus", mock.Anything, "test-branch", mock.Anything).Return(inProgressStatus, nil).Maybe()

	// Execute
	ctx := context.Background()
	status, err := runner.monitorTestExecution(ctx, "test-branch", runConfig)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Contains(t, err.Error(), "timeout waiting for test completion")
}

func TestTestRunner_monitorTestExecution_Crash(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	runConfig := TestRunConfig{
		Options:      types.TestRunOptions{},
		PollInterval: 50 * time.Millisecond,
		Timeout:      500 * time.Millisecond,
		DebugMode:    false,
	}

	crashedStatus := &types.TestStatus{
		Status: types.StatusFailed,
		Results: types.TestResults{
			Total: 1,
			Crash: 1,
		},
	}

	mockClient.On("GetTestStatus", mock.Anything, "test-branch", mock.Anything).Return(crashedStatus, nil)

	// Execute
	ctx := context.Background()
	status, err := runner.monitorTestExecution(ctx, "test-branch", runConfig)

	// Verify
	assert.Error(t, err)
	assert.Equal(t, crashedStatus, status)
	assert.Contains(t, err.Error(), "test crashed")
	mockClient.AssertExpectations(t)
}

func TestTestRunner_downloadReport_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	reportData := []byte(`<?xml version="1.0"?><testsuite></testsuite>`)
	mockClient.On("GetJUnitReport", mock.Anything, "task-123").Return(reportData, nil)

	// Execute
	ctx := context.Background()
	reportPath, err := runner.downloadReport(ctx, "task-123", false)

	// Verify
	assert.NoError(t, err)
	assert.NotEmpty(t, reportPath)
	mockClient.AssertExpectations(t)
}

func TestTestRunner_downloadReport_RetryLogic(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	logger := &MockLogger{}
	runner := &TestRunner{
		config: cfg,
		logger: logger,
	}

	mockClient := &MockTestRigorClient{}
	runner.apiClient = mockClient

	// First call fails with "not ready", second succeeds
	reportData := []byte(`<?xml version="1.0"?><testsuite></testsuite>`)
	mockClient.On("GetJUnitReport", mock.Anything, "task-123").Return(nil, errors.New("report still being generated")).Once()
	mockClient.On("GetJUnitReport", mock.Anything, "task-123").Return(reportData, nil).Once()

	// Execute
	ctx := context.Background()
	reportPath, err := runner.downloadReport(ctx, "task-123", true) // Debug mode

	// Verify
	assert.NoError(t, err)
	assert.NotEmpty(t, reportPath)
	mockClient.AssertExpectations(t)
}

func TestTestRunner_isTestRunSuccessful(t *testing.T) {
	runner := &TestRunner{}

	tests := []struct {
		name     string
		status   *types.TestStatus
		expected bool
	}{
		{
			name:     "nil status",
			status:   nil,
			expected: false,
		},
		{
			name: "successful completion",
			status: &types.TestStatus{
				Status: types.StatusCompleted,
				Results: types.TestResults{
					Total:  5,
					Passed: 5,
					Failed: 0,
					Crash:  0,
				},
			},
			expected: true,
		},
		{
			name: "completion with failures",
			status: &types.TestStatus{
				Status: types.StatusCompleted,
				Results: types.TestResults{
					Total:  5,
					Passed: 3,
					Failed: 2,
					Crash:  0,
				},
			},
			expected: false,
		},
		{
			name: "completion with crashes",
			status: &types.TestStatus{
				Status: types.StatusCompleted,
				Results: types.TestResults{
					Total:  5,
					Passed: 4,
					Failed: 0,
					Crash:  1,
				},
			},
			expected: false,
		},
		{
			name: "not completed",
			status: &types.TestStatus{
				Status: types.StatusInProgress,
				Results: types.TestResults{
					Total:      5,
					Passed:     2,
					InProgress: 3,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.isTestRunSuccessful(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTestRunner_logRunParameters(t *testing.T) {
	logger := &MockLogger{}
	runner := &TestRunner{logger: logger}

	runConfig := TestRunConfig{
		Options: types.TestRunOptions{
			BranchName:                 "test-branch",
			CommitHash:                 "abc123",
			URL:                        "https://example.com",
			Labels:                     []string{"smoke", "regression"},
			ExcludedLabels:             []string{"slow"},
			CustomName:                 "Custom Test Run",
			TestCaseUUIDs:              []string{"uuid-1", "uuid-2"},
			ForceCancelPreviousTesting: true,
		},
	}

	runner.logRunParameters(runConfig)

	assert.NotEmpty(t, logger.logs)
	// Check that log entries were made (MockLogger captures format strings, not formatted output)
	assert.True(t, len(logger.logs) > 5, "Expected multiple log entries")
}

func TestTestRunner_printStatusUpdate(t *testing.T) {
	logger := &MockLogger{}
	runner := &TestRunner{logger: logger}

	status := &types.TestStatus{
		Status:         types.StatusInProgress,
		HTTPStatusCode: 200,
		Results: types.TestResults{
			Total:      10,
			Passed:     3,
			Failed:     1,
			InProgress: 4,
			InQueue:    2,
			Canceled:   0,
		},
	}

	runner.printStatusUpdate(status)

	assert.NotEmpty(t, logger.logs)
	// Verify that status information was logged
	foundStatusLog := false
	for _, log := range logger.logs {
		if strings.Contains(log, "Test Status:") {
			foundStatusLog = true
			break
		}
	}
	assert.True(t, foundStatusLog)
}

func TestTestRunner_printFinalResults(t *testing.T) {
	logger := &MockLogger{}
	runner := &TestRunner{logger: logger}

	status := &types.TestStatus{
		Status:     types.StatusCompleted,
		DetailsURL: "https://testrigor.com/details/123",
		Results: types.TestResults{
			Total:      10,
			Passed:     8,
			Failed:     1,
			InProgress: 0,
			InQueue:    0,
			NotStarted: 0,
			Canceled:   1,
			Crash:      0,
		},
		Errors: []types.TestError{
			{
				Category:    "BLOCKER",
				Error:       "Test failed due to timeout",
				Severity:    "HIGH",
				Occurrences: 1,
				DetailsURL:  "https://testrigor.com/error/456",
			},
		},
	}

	duration := 5 * time.Minute
	runner.printFinalResults(status, duration)

	assert.NotEmpty(t, logger.logs)
}

func TestDefaultLogger(t *testing.T) {
	logger := DefaultLogger{}

	// Test that these don't panic
	assert.NotPanics(t, func() {
		logger.Printf("Test %s", "message")
	})

	assert.NotPanics(t, func() {
		logger.Println("Test message")
	})
}
