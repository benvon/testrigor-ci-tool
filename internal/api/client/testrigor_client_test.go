package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	resp, _ := args.Get(0).(*http.Response)
	return resp, args.Error(1)
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
		Request:    &http.Request{},
		Close:      true,
	}
}

func TestNewTestRigorClient(t *testing.T) {
	cfg := &config.Config{TestRigor: config.TestRigorConfig{AuthToken: "token", AppID: "app", APIURL: "http://api"}}
	c := NewTestRigorClient(cfg, &mockHTTPClient{})
	assert.NotNil(t, c)
}

func TestStartTestRunSuccess(t *testing.T) {
	cfg := &config.Config{TestRigor: config.TestRigorConfig{AuthToken: "token", AppID: "app", APIURL: "http://api"}}
	mockClient := &mockHTTPClient{}
	respBody := `{"taskId":"tid"}`
	mockClient.On("Do", mock.Anything).Return(newHTTPResponse(200, respBody), nil)
	c := NewTestRigorClient(cfg, mockClient)
	opts := types.TestRunOptions{}
	result, err := c.StartTestRun(context.Background(), opts, false)
	assert.NoError(t, err)
	assert.Equal(t, "tid", result.TaskID)
}

func TestStartTestRunError(t *testing.T) {
	cfg := &config.Config{TestRigor: config.TestRigorConfig{AuthToken: "token", AppID: "app", APIURL: "http://api"}}
	mockClient := &mockHTTPClient{}
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("fail")).Once()
	c := NewTestRigorClient(cfg, mockClient)
	_, err := c.StartTestRun(context.Background(), types.TestRunOptions{}, false)
	assert.Error(t, err)
}

func TestGetTestStatusSuccess(t *testing.T) {
	cfg := &config.Config{TestRigor: config.TestRigorConfig{AuthToken: "token", AppID: "app", APIURL: "http://api"}}
	mockClient := &mockHTTPClient{}
	status := map[string]interface{}{"status": "completed", "taskId": "tid", "overallResults": map[string]interface{}{"Total": 1, "Passed": 1}}
	body, _ := json.Marshal(status)
	mockClient.On("Do", mock.Anything).Return(newHTTPResponse(200, string(body)), nil)
	c := NewTestRigorClient(cfg, mockClient)
	result, err := c.GetTestStatus(context.Background(), "", nil, false)
	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 1, result.Results.Total)
	assert.Equal(t, 1, result.Results.Passed)
}

func TestCancelTestRunSuccess(t *testing.T) {
	cfg := &config.Config{TestRigor: config.TestRigorConfig{AuthToken: "token", AppID: "app", APIURL: "http://api"}}
	mockClient := &mockHTTPClient{}
	mockClient.On("Do", mock.Anything).Return(newHTTPResponse(200, `{}`), nil)
	c := NewTestRigorClient(cfg, mockClient)
	err := c.CancelTestRun(context.Background(), "runid")
	assert.NoError(t, err)
}

func TestGetJUnitReportSuccess(t *testing.T) {
	cfg := &config.Config{TestRigor: config.TestRigorConfig{AuthToken: "token", AppID: "app", APIURL: "http://api"}}
	mockClient := &mockHTTPClient{}
	mockClient.On("Do", mock.Anything).Return(newHTTPResponse(200, `<xml></xml>`), nil)
	c := NewTestRigorClient(cfg, mockClient)
	data, err := c.GetJUnitReport(context.Background(), "tid")
	assert.NoError(t, err)
	assert.Equal(t, []byte(`<xml></xml>`), data)
}

func TestBuildStartTestRunBodyCustomNameOnly(t *testing.T) {
	c := &TestRigorClient{}

	t.Run("customName only", func(t *testing.T) {
		opts := types.TestRunOptions{CustomName: "n"}
		body := c.buildStartTestRunBody(opts)
		if v, ok := body["customName"]; ok {
			assert.Equal(t, "n", v)
		} else {
			assert.Fail(t, "customName should be present in body")
		}
	})

	t.Run("customName and TestCaseUUIDs", func(t *testing.T) {
		opts := types.TestRunOptions{TestCaseUUIDs: []string{"uuid"}, CustomName: "n"}
		body := c.buildStartTestRunBody(opts)
		_, ok := body["customName"]
		assert.False(t, ok, "customName should NOT be present in body when TestCaseUUIDs is set")
	})
}

func TestBuildBranchInfo(t *testing.T) {
	c := &TestRigorClient{}
	opts := types.TestRunOptions{BranchName: "b", CommitHash: "c"}
	info := c.buildBranchInfo(opts)
	assert.Equal(t, "b", info["name"])
	assert.Equal(t, "c", info["commit"])
}

func TestExtractBranchName(t *testing.T) {
	c := &TestRigorClient{}
	opts := types.TestRunOptions{BranchName: "b"}
	body := map[string]interface{}{"branch": map[string]string{"name": "b"}}
	name := c.extractBranchName(opts, body)
	assert.Equal(t, "b", name)
}

func TestBuildStatusURL(t *testing.T) {
	c := &TestRigorClient{config: &config.Config{TestRigor: config.TestRigorConfig{APIURL: "http://api", AppID: "app"}}}
	url := c.buildStatusURL("b", []string{"l1", "l2"})
	assert.Contains(t, url, "branchName=b")
	assert.Contains(t, url, "labels=l1%2Cl2") // URL-encoded comma
}

func TestParseAPIError(t *testing.T) {
	c := &TestRigorClient{}
	err := c.parseAPIError(400, []byte(`{"message":"fail"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail")
}

func TestGetStringAndGetInt(t *testing.T) {
	c := &TestRigorClient{}
	m := map[string]interface{}{"str": "s", "int": 1, "float": 2.0, "strint": "3"}
	assert.Equal(t, "s", c.getString(m, "str"))
	assert.Equal(t, 1, c.getInt(m, "int"))
	assert.Equal(t, 2, c.getInt(m, "float"))
	assert.Equal(t, 3, c.getInt(m, "strint"))
}

func TestGenerateBranchNameAndFakeCommitHash(t *testing.T) {
	c := &TestRigorClient{}
	name := c.generateBranchName([]string{"foo", "bar"})
	assert.Contains(t, name, "foo-bar-")
	hash := c.generateFakeCommitHash()
	assert.Equal(t, 40, len(hash))
}
