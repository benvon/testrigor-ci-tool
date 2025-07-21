# TestRigor CI Tool

A robust command-line tool for managing TestRigor test suite runs in CI/CD pipelines with enhanced status reporting and error handling.

## Features

- 🚀 **Start and monitor test runs** with real-time status updates
- 📊 **Resilient status reporting** with guaranteed periodic output
- 🔍 **Check test status** independently
- ❌ **Cancel running tests** by run ID
- 📄 **Download JUnit reports** after test completion
- 🐛 **Debug mode** for troubleshooting
- ⚡ **Configurable polling intervals** and timeouts
- 🏷️ **Label-based test filtering** with exclusion support
- 🔗 **Branch and commit tracking** for test runs
- 🎯 **Single test case execution**
- 📈 **Xray Cloud reporting** integration
- 🛡️ **Comprehensive error handling** with retry logic

## Installation

### From Source
```bash
git clone https://github.com/benvon/testrigor-ci-tool.git
cd testrigor-ci-tool
go build -o testrigor-ci-tool .
```

### Using Go Install
```bash
go install github.com/benvon/testrigor-ci-tool@latest
```

## Configuration

The tool can be configured using environment variables, a config file, or command-line flags.

### Environment Variables

- `TESTRIGOR_AUTH_TOKEN`: Your TestRigor authentication token (required)
- `TESTRIGOR_APP_ID`: Your TestRigor application ID (required)
- `TESTRIGOR_API_URL`: TestRigor API URL (default: https://api.testrigor.com/api/v1)
- `TR_CI_ERROR_ON_TEST_FAILURE`: Set to "true" to exit with code 1 on test failures (default: false)

### Config File

Create a file at `$HOME/.testrigor.yaml` with the following content:

```yaml
testrigor:
  authtoken: "your-auth-token"
  appid: "your-app-id"
  apiurl: "https://api.testrigor.com/api/v1"  # Optional
  errorontestfailure: false  # Optional
```

### Command-Line Configuration

Use the `--config` flag to specify a custom config file:

```bash
testrigor --config /path/to/config.yaml run-and-wait
```

## Commands

### `run-and-wait` - Start and Monitor Test Runs

Start a test run and wait for completion with real-time status updates.

```bash
testrigor run-and-wait [flags]
```

#### Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--labels` | string slice | Labels to filter tests (e.g., "Smoke", "Regression") | `[]` |
| `--excluded-labels` | string slice | Labels to exclude from test run | `[]` |
| `--branch` | string | Branch name for tracking (e.g., ci-123, pr-456, manual-smoke) | auto-generated |
| `--commit` | string | Commit hash for test run | auto-generated |
| `--url` | string | URL for test run | - |
| `--test-case` | string | Test case UUID to run | - |
| `--name` | string | Custom name for test run | - |
| `--poll-interval` | int | Polling interval in seconds | `10` |
| `--timeout` | int | Maximum wait time in minutes | `30` |
| `--debug` | bool | Enable debug output | `false` |
| `--force-cancel` | bool | Force cancel previous testing | `false` |
| `--fetch-report` | bool | Download JUnit report after completion | `false` |
| `--make-xray-reports` | bool | Enable Xray Cloud reporting | `false` |

#### Examples

**Run smoke tests:**
```bash
testrigor run-and-wait --labels Smoke --url "https://example.com"
```

**Run specific test case:**
```bash
testrigor run-and-wait --test-case "test-case-uuid" --url "https://example.com"
```

**Run tests with custom branch tracking:**
```bash
testrigor run-and-wait --branch "pr-123" --commit "abc123def" --url "https://example.com"
```

**Run tests with JUnit report download:**
```bash
testrigor run-and-wait --labels Regression --fetch-report --url "https://example.com"
```

**Run tests with debug output:**
```bash
testrigor run-and-wait --labels Smoke --debug --url "https://example.com"
```

**Run tests and exit with code 1 on failure:**
```bash
TR_CI_ERROR_ON_TEST_FAILURE=true testrigor run-and-wait --labels Smoke
```

### `status` - Check Test Status

Check the current status of a test suite run without starting a new one.

```bash
testrigor status [flags]
```

#### Flags

| Flag | Type | Description | Required |
|------|------|-------------|----------|
| `--branch` | string | Branch name to check status for | Yes |
| `--labels` | string | Comma-separated list of labels to filter by | No |

#### Examples

**Check status by branch:**
```bash
testrigor status --branch "pr-123"
```

**Check status by branch and labels:**
```bash
testrigor status --branch "ci-456" --labels "Smoke,Regression"
```

### `cancel` - Cancel Running Tests

Cancel a currently running test suite by its run ID.

```bash
testrigor cancel --run-id <run-id>
```

#### Flags

| Flag | Type | Description | Required |
|------|------|-------------|----------|
| `--run-id` | string | ID of the run to cancel | Yes |

#### Examples

**Cancel a running test:**
```bash
testrigor cancel --run-id "run-abc123def"
```

### `--version` - Version Information

Display version information and exit.

```bash
testrigor --version
```

## Status Reporting

The tool provides resilient status reporting with the following features:

- **Real-time updates**: Status changes are reported immediately
- **Periodic heartbeats**: Guaranteed output every polling period, even when status hasn't changed
- **Progress tracking**: Shows test progress with passed/failed/in-progress counts
- **Error details**: Displays detailed error information when tests fail
- **Duration tracking**: Shows total test run duration
- **URL links**: Provides direct links to test details

### Status Output Example

```
[20:47:24] Test Status: in_progress
  Progress: 5/10 tests | Queue: 2 | Running: 3 | Passed: 4 | Failed: 1 | Canceled: 0 (50.0% complete)

Test run completed with status: completed
Total duration: 2m 15s
Details URL: https://testrigor.com/details/123

Final Results:
  Total: 10
  Passed: 8
  Failed: 1
  In Progress: 0
  In Queue: 0
  Not Started: 0
  Canceled: 1
  Crash: 0
```

## Error Handling

The tool includes comprehensive error handling:

- **Retry logic**: Automatically retries on transient errors
- **Timeout handling**: Configurable timeouts with graceful failure
- **Status code handling**: Proper handling of TestRigor-specific status codes (227, 228, 230)
- **404 handling**: Treats 404 errors as "not ready" rather than fatal errors
- **Consecutive error limits**: Prevents infinite retry loops
- **Graceful degradation**: Continues operation even when some operations fail

## Exit Codes

- `0`: Success (or test failure if `TR_CI_ERROR_ON_TEST_FAILURE` is not set to "true")
- `1`: Error (or test failure if `TR_CI_ERROR_ON_TEST_FAILURE` is set to "true")

## CI/CD Integration

### GitHub Actions Example

```yaml
name: TestRigor Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          
      - name: Install TestRigor CI Tool
        run: go install github.com/benvon/testrigor-ci-tool@latest
        
      - name: Run Smoke Tests
        env:
          TESTRIGOR_AUTH_TOKEN: ${{ secrets.TESTRIGOR_AUTH_TOKEN }}
          TESTRIGOR_APP_ID: ${{ secrets.TESTRIGOR_APP_ID }}
          TR_CI_ERROR_ON_TEST_FAILURE: "true"
        run: |
          testrigor run-and-wait \
            --labels "Smoke" \
            --branch "ci-${{ github.run_id }}" \
            --commit "${{ github.sha }}" \
            --url "https://staging.example.com" \
            --fetch-report
```

### GitLab CI Example

```yaml
testrigor_tests:
  image: golang:1.21
  script:
    - go install github.com/benvon/testrigor-ci-tool@latest
    - testrigor run-and-wait \
        --labels "Regression" \
        --branch "ci-$CI_PIPELINE_ID" \
        --commit "$CI_COMMIT_SHA" \
        --url "https://staging.example.com" \
        --fetch-report
  variables:
    TESTRIGOR_AUTH_TOKEN: $TESTRIGOR_AUTH_TOKEN
    TESTRIGOR_APP_ID: $TESTRIGOR_APP_ID
    TR_CI_ERROR_ON_TEST_FAILURE: "true"
```

## Troubleshooting

### Debug Mode

Enable debug mode to see detailed API requests and responses:

```bash
testrigor run-and-wait --debug --labels Smoke
```

### Common Issues

1. **Authentication errors**: Verify your `TESTRIGOR_AUTH_TOKEN` is correct
2. **App ID errors**: Ensure your `TESTRIGOR_APP_ID` is valid
3. **Network timeouts**: Increase the `--timeout` value for slow networks
4. **Test not starting**: Check that your labels or test case UUIDs are correct

### Logs and Output

The tool provides detailed logging:
- **Status updates**: Real-time test progress
- **Error messages**: Detailed error information
- **API debugging**: Request/response details in debug mode
- **File operations**: JUnit report download status

## Development

### Building from Source

```bash
git clone https://github.com/benvon/testrigor-ci-tool.git
cd testrigor-ci-tool
go build -o testrigor-ci-tool .
```

### Running Tests

```bash
go test ./... -v
```

### Linting

The project uses golangci-lint for code quality. **All PRs must pass linting checks before merging.**

```bash
golangci-lint run
```

The CI/CD pipeline automatically runs linting on all pull requests and will block merging if any linting issues are found.

### CI/CD Pipeline

The project includes two streamlined CI/CD workflows:

#### **PR Quality Checks** (`.github/workflows/pr-quality.yml`)
- **Lint**: Runs golangci-lint to check code quality
- **Test**: Runs all unit tests
- **Triggers**: Pull requests, pushes to main, manual dispatch
- **Purpose**: Ensures code quality before merging

#### **Release Pipeline** (`.github/workflows/release.yml`)
- **Quality Checks**: Runs linting and tests
- **Build**: Verifies the application builds successfully
- **Release**: Automatically creates releases on tags using GoReleaser
- **Triggers**: Tag pushes (v*)
- **Purpose**: Creates production releases with quality guarantees

All workflows ensure code quality is maintained throughout the development lifecycle.

## License

Apache 2.0 License - see [LICENSE](LICENSE) file for details.