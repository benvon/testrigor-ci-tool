# TestRigor CI Tool Architecture

This document describes the architecture of the TestRigor CI Tool after the major refactoring to simplify the codebase and follow Go best practices.

## Design Principles

The refactored architecture follows these key principles:

1. **Separation of Concerns**: Clear boundaries between different responsibilities
2. **Primitive vs Orchestrator Pattern**: Simple, single-purpose primitives coordinated by orchestrators
3. **Testability**: All components are easily testable with clear interfaces
4. **Go Best Practices**: Idiomatic Go code with proper error handling and documentation
5. **Minimal Complexity**: Each function has a single, well-defined purpose

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
├─────────────────────────────────────────────────────────────┤
│  cmd/                                                       │
│  ├── root.go           (Configuration & CLI setup)          │
│  ├── run_and_wait.go   (Test execution command)             │
│  ├── status.go         (Status checking command)            │
│  └── cancel.go         (Test cancellation command)          │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator Layer                       │
├─────────────────────────────────────────────────────────────┤
│  internal/orchestrator/                                     │
│  └── test_runner.go    (Coordinates test execution workflow)│
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                     Primitive Layer                         │
├─────────────────────────────────────────────────────────────┤
│  internal/api/client/                                       │
│  ├── http_client.go       (HTTP operations)                 │
│  ├── testrigor_client.go  (TestRigor API calls)            │
│  └── status_manager.go    (Status monitoring)               │
│                                                             │
│  internal/config/                                           │
│  └── config.go           (Configuration management)         │
│                                                             │
│  internal/api/types/                                        │
│  └── types.go            (Data structures)                  │
└─────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. Primitives (Single Purpose Functions)

Primitives are simple, focused functions that perform one specific task. They have minimal dependencies and are easily testable.

#### HTTP Client (`internal/api/client/http_client.go`)
- **Purpose**: Pure HTTP operations
- **Responsibilities**: 
  - Making HTTP requests
  - Handling request/response serialization
  - Basic error handling
- **Key Functions**:
  - `Execute()`: Performs HTTP request with context
  - `buildHTTPRequest()`: Constructs HTTP requests

#### TestRigor API Client (`internal/api/client/testrigor_client.go`)
- **Purpose**: TestRigor-specific API operations
- **Responsibilities**:
  - TestRigor API endpoint calls
  - Request/response parsing
  - API error interpretation
- **Key Functions**:
  - `StartTestRun()`: Initiates test execution
  - `GetTestStatus()`: Retrieves test status
  - `CancelTestRun()`: Cancels running tests
  - `GetJUnitReport()`: Downloads test reports

#### Configuration (`internal/config/config.go`)
- **Purpose**: Configuration management
- **Responsibilities**:
  - Loading configuration from various sources
  - Validation of required settings
  - Environment variable binding
- **Key Functions**:
  - `LoadConfig()`: Loads and validates configuration
  - `validate()`: Ensures required fields are present

#### Types (`internal/api/types/types.go`)
- **Purpose**: Data structure definitions
- **Responsibilities**:
  - Type definitions for API operations
  - Status constants and enums
  - Helper methods for status checking
- **Key Types**:
  - `TestRunOptions`: Test execution parameters
  - `TestStatus`: Current test state
  - `TestResults`: Test outcome metrics

### 2. Orchestrators (Coordinate Primitives)

Orchestrators combine multiple primitives to implement complex business workflows with minimal internal logic.

#### Test Runner (`internal/orchestrator/test_runner.go`)
- **Purpose**: Orchestrates complete test execution workflow
- **Responsibilities**:
  - Coordinating test start, monitoring, and reporting
  - Managing execution timeouts and retries
  - Handling user output and logging
  - Report download coordination
- **Key Functions**:
  - `ExecuteTestRun()`: Main orchestration function
  - `monitorTestExecution()`: Coordinates status polling
  - `downloadReport()`: Manages report retrieval

### 3. CLI Layer (`cmd/`)

The CLI layer provides user interface and command routing.

#### Root Command (`cmd/root.go`)
- **Purpose**: CLI framework setup and global configuration
- **Responsibilities**:
  - Command registration
  - Global flag handling
  - Version information display

#### Commands
- **`run_and_wait.go`**: Orchestrates test execution workflow
- **`status.go`**: Provides status checking functionality  
- **`cancel.go`**: Handles test cancellation requests

## Data Flow

### Test Execution Flow

```
1. User runs: testrigor run-and-wait --labels Smoke --url https://example.com

2. CLI Layer (cmd/run_and_wait.go):
   ├── Parses command line flags
   ├── Loads configuration
   └── Creates TestRunner orchestrator

3. Orchestrator (orchestrator/test_runner.go):
   ├── Calls primitive: StartTestRun()
   ├── Monitors execution: GetTestStatus() (repeatedly)
   ├── Downloads report: GetJUnitReport() (if requested)
   └── Returns comprehensive result

4. Primitives (api/client/):
   ├── HTTP Client: Makes actual HTTP requests
   ├── TestRigor Client: Handles API-specific logic
   └── Status parsing: Interprets responses

5. CLI Layer: 
   └── Processes result and exits with appropriate code
```

### Status Checking Flow

```
1. User runs: testrigor status --branch test-branch

2. CLI Layer (cmd/status.go):
   ├── Parses command line flags
   ├── Loads configuration
   └── Creates TestRigor client primitive

3. Primitive (client/testrigor_client.go):
   ├── Calls GetTestStatus()
   └── Returns parsed status

4. CLI Layer:
   └── Formats and displays status information
```

## Error Handling Strategy

### 1. Primitive Level
- Return errors immediately with context
- Use `fmt.Errorf()` with `%w` verb for error wrapping
- No logging or user output at primitive level

### 2. Orchestrator Level
- Coordinate error handling across multiple primitives
- Implement retry logic where appropriate
- Convert technical errors to user-friendly messages
- Handle timeouts and cancellation

### 3. CLI Level
- Display user-friendly error messages
- Set appropriate exit codes
- Handle configuration errors gracefully

## Testing Strategy

### 1. Primitive Testing
- Unit tests for all primitive functions
- Mock external dependencies (HTTP client, file system)
- Test error conditions and edge cases
- Use table-driven tests for multiple scenarios

### 2. Orchestrator Testing
- Unit tests with mocked primitives
- Integration tests for complete workflows
- Test timeout and cancellation scenarios
- Verify proper error propagation

### 3. CLI Testing
- Command-line interface testing
- Flag parsing verification
- Exit code validation
- Help output testing

## Interfaces and Contracts

### HTTPClient Interface
```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

### Logger Interface
```go
type Logger interface {
    Printf(format string, args ...interface{})
    Println(args ...interface{})
}
```

## Benefits of This Architecture

### 1. Simplicity
- Each component has a single, clear responsibility
- Functions are small and focused
- Minimal interdependencies

### 2. Testability
- Primitives can be tested in isolation
- Orchestrators can be tested with mocked primitives
- Clear interfaces enable easy mocking

### 3. Maintainability
- Changes are localized to specific components
- New features can be added as new primitives/orchestrators
- Debugging is simplified with clear boundaries

### 4. Extensibility
- New commands can reuse existing primitives
- New primitives can be added without affecting orchestrators
- Alternative implementations can be swapped easily

## Best Practices Followed

### 1. Go Conventions
- Package names are descriptive and lowercase
- Interfaces are small and focused
- Error handling follows Go idioms
- Context is used for cancellation and timeouts

### 2. Documentation
- All public functions have clear documentation
- Package documentation explains purpose
- Examples provided for complex functionality

### 3. Error Handling
- Errors are wrapped with additional context
- Error messages are user-friendly
- Failures are handled gracefully

### 4. Testing
- Comprehensive test coverage (>70%)
- Both unit and integration tests
- Mocking used appropriately
- Table-driven tests for multiple scenarios

## Migration Guide

### For Developers
1. **Adding New Features**: Create new primitives for core functionality, orchestrators for workflows
2. **Modifying Existing Features**: Identify the appropriate layer and make targeted changes
3. **Testing**: Write tests at the appropriate level (primitive, orchestrator, or CLI)

### For Contributors
1. **Code Review**: Focus on single responsibility and proper layer separation
2. **Testing**: Ensure new code includes appropriate tests
3. **Documentation**: Update relevant documentation for significant changes

This architecture provides a solid foundation for maintaining and extending the TestRigor CI Tool while keeping the codebase simple and comprehensible.