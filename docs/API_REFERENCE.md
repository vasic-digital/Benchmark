# Benchmark - API Reference

**Module:** `digital.vasic.benchmark`
**Package:** `benchmark`

## Constructor Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewStandardBenchmarkRunner` | `NewStandardBenchmarkRunner(provider LLMProvider, logger *logrus.Logger) *StandardBenchmarkRunner` | Creates a runner with built-in benchmark suites pre-loaded. |

## Interfaces

### BenchmarkRunner

Primary interface for running benchmarks and collecting results.

| Method | Signature | Description |
|--------|-----------|-------------|
| `RunBenchmark` | `RunBenchmark(ctx context.Context, benchmarkID string, config RunConfig) (*BenchmarkRun, error)` | Executes a benchmark suite against a provider. |
| `ListBenchmarks` | `ListBenchmarks(ctx context.Context) []*Benchmark` | Returns all registered benchmark suites. |
| `GetResults` | `GetResults(ctx context.Context, runID string) ([]*BenchmarkResult, error)` | Returns results for a specific run. |
| `CompareProviders` | `CompareProviders(ctx context.Context, benchmarkID string, providers []ProviderConfig) (*Comparison, error)` | Runs the same suite across multiple providers. |
| `GetLeaderboard` | `GetLeaderboard(ctx context.Context, benchmarkID string) *Leaderboard` | Returns ranked provider scores. |
| `RegisterBenchmark` | `RegisterBenchmark(ctx context.Context, bench *Benchmark, tasks []*BenchmarkTask) error` | Registers a custom benchmark suite. |

### LLMProvider

Provider contract required by the runner to execute prompts.

```go
type LLMProvider interface {
    Complete(ctx context.Context, prompt string, config map[string]interface{}) (string, error)
    Name() string
}
```

### CodeExecutor

Optional interface for evaluating generated code against test cases.

```go
type CodeExecutor interface {
    Execute(ctx context.Context, code string, testCases []*TestCase) ([]*TestCaseResult, error)
}
```

### DebateEvaluator

Optional interface for evaluating debate outputs.

```go
type DebateEvaluator interface {
    Evaluate(ctx context.Context, output string, metrics []string) (map[string]float64, error)
}
```

## Core Types

### Benchmark

Describes a benchmark suite.

```go
type Benchmark struct {
    ID          string        `json:"id"`
    Type        BenchmarkType `json:"type"`
    Name        string        `json:"name"`
    Description string        `json:"description"`
    Version     string        `json:"version"`
    TaskCount   int           `json:"task_count"`
    CreatedAt   time.Time     `json:"created_at"`
}
```

### BenchmarkTask

A single evaluation task within a suite.

```go
type BenchmarkTask struct {
    ID          string                 `json:"id"`
    BenchmarkID string                 `json:"benchmark_id"`
    Type        BenchmarkType          `json:"type"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Prompt      string                 `json:"prompt"`
    Context     string                 `json:"context,omitempty"`
    Expected    string                 `json:"expected,omitempty"`
    TestCases   []*TestCase            `json:"test_cases,omitempty"`
    Difficulty  DifficultyLevel        `json:"difficulty,omitempty"`
    Tags        []string               `json:"tags,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    TimeLimit   time.Duration          `json:"time_limit,omitempty"`
}
```

### BenchmarkResult

Result of evaluating one task against one provider.

```go
type BenchmarkResult struct {
    TaskID       string                 `json:"task_id"`
    RunID        string                 `json:"run_id"`
    ProviderName string                 `json:"provider_name"`
    ModelName    string                 `json:"model_name"`
    Response     string                 `json:"response"`
    Passed       bool                   `json:"passed"`
    Score        float64                `json:"score"`
    Latency      time.Duration          `json:"latency"`
    TokensUsed   int                    `json:"tokens_used"`
    TestResults  []*TestCaseResult      `json:"test_results,omitempty"`
    Error        string                 `json:"error,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt    time.Time              `json:"created_at"`
}
```

### TestCase / TestCaseResult

```go
type TestCase struct {
    ID       string `json:"id"`
    Input    string `json:"input"`
    Expected string `json:"expected"`
    Hidden   bool   `json:"hidden"`
}

type TestCaseResult struct {
    TestCaseID string `json:"test_case_id"`
    Passed     bool   `json:"passed"`
    Actual     string `json:"actual"`
    Expected   string `json:"expected"`
    Error      string `json:"error,omitempty"`
}
```

## Enums

### BenchmarkType

| Constant | Value | Description |
|----------|-------|-------------|
| `BenchmarkTypeSWEBench` | `"swe-bench"` | Software engineering tasks |
| `BenchmarkTypeHumanEval` | `"humaneval"` | Code generation correctness |
| `BenchmarkTypeMBPP` | `"mbpp"` | Mostly Basic Programming Problems |
| `BenchmarkTypeLMSYS` | `"lmsys"` | LMSYS chatbot arena |
| `BenchmarkTypeHellaSwag` | `"hellaswag"` | Commonsense inference |
| `BenchmarkTypeMMLU` | `"mmlu"` | Multitask language understanding |
| `BenchmarkTypeGSM8K` | `"gsm8k"` | Grade school math |
| `BenchmarkTypeMATH` | `"math"` | Advanced mathematics |
| `BenchmarkTypeCustom` | `"custom"` | User-defined evaluation |

### DifficultyLevel

| Constant | Value |
|----------|-------|
| `DifficultyEasy` | `"easy"` |
| `DifficultyMedium` | `"medium"` |
| `DifficultyHard` | `"hard"` |

## Configuration Types

### RunConfig

```go
type RunConfig struct {
    ProviderName string
    ModelName    string
    MaxTasks     int
    Timeout      time.Duration
    Concurrency  int
}
```

### ProviderConfig

```go
type ProviderConfig struct {
    Name  string
    Model string
}
```

## Result Aggregation Types

### BenchmarkRun

Aggregated results for a complete benchmark execution.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique run identifier |
| `BenchmarkID` | `string` | Which benchmark was run |
| `AggregateScore` | `float64` | Average score across all tasks |
| `PassCount` | `int` | Number of tasks that passed |
| `TotalTasks` | `int` | Total tasks executed |
| `Duration` | `time.Duration` | Total execution time |

### Leaderboard

Ranked provider comparison.

| Field | Type | Description |
|-------|------|-------------|
| `BenchmarkID` | `string` | Benchmark being compared |
| `Entries` | `[]LeaderboardEntry` | Ranked list of providers |
| `UpdatedAt` | `time.Time` | Last update timestamp |
