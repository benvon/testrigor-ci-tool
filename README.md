# TestRigor CI Tool

A command-line tool for running TestRigor tests in CI/CD pipelines.

## Installation

```bash
go install github.com/benvon/testrigor-ci-tool@latest
```

## Configuration

The tool can be configured using environment variables or a config file at `$HOME/.testrigor.yaml`.

### Environment Variables

- `TESTRIGOR_AUTH_TOKEN`: Your TestRigor authentication token
- `TESTRIGOR_APP_ID`: Your TestRigor application ID
- `TESTRIGOR_API_URL`: (Optional) TestRigor API URL (default: https://api.testrigor.com/api/v1)
- `TR_CI_ERROR_ON_TEST_FAILURE`: (Optional) Set to "true" to exit with code 1 on test failures (default: false)

### Config File

Create a file at `$HOME/.testrigor.yaml` with the following content:

```yaml
testrigor:
  authtoken: "your-auth-token"
  appid: "your-app-id"
  apiurl: "https://api.testrigor.com/api/v1"  # Optional
  errorontestfailure: false  # Optional
```

## Usage

### Run Tests and Wait for Completion

```bash
testrigor run-and-wait [flags]
```

#### Flags

- `--labels`: Labels to filter tests (e.g., "Smoke", "Regression")
- `--excluded-labels`: Labels to exclude from test run
- `--branch`: Branch name for tracking the test run (e.g., ci-123, pr-456, manual-smoke)
- `--commit`: Commit hash for test run
- `--url`: URL for test run
- `--test-case`: Test case UUID to run
- `--name`: Custom name for test run
- `--poll-interval`: Polling interval in seconds (default: 10)
- `--debug`: Enable debug output
- `--force-cancel`: Force cancel previous testing
- `--fetch-report`: Download JUnit report after test completion
- `--make-xray-reports`: Enable Xray Cloud reporting (disabled by default)

#### Examples

Run smoke tests:
```bash
testrigor run-and-wait --labels Smoke --url "https://example.com"
```

Run specific test case:
```bash
testrigor run-and-wait --test-case "test-case-uuid" --url "https://example.com"
```

Run tests with custom branch name:
```bash
testrigor run-and-wait --branch "pr-123" --url "https://example.com"
```

Run tests and exit with code 1 on failure:
```bash
TR_CI_ERROR_ON_TEST_FAILURE=true testrigor run-and-wait --labels Smoke
```

## Exit Codes

- `0`: Success (or test failure if `TR_CI_ERROR_ON_TEST_FAILURE` is not set to "true")
- `1`: Error (or test failure if `TR_CI_ERROR_ON_TEST_FAILURE` is set to "true")

## License

MIT