# Benchmark

`digital.vasic.benchmark` -- LLM benchmarking with SWE-bench, HumanEval, MMLU, GSM8K, and custom benchmarks; leaderboard generation and provider comparison.

## Overview

Benchmark is a Go module that provides a complete LLM evaluation framework for running standardized benchmarks against any language model provider. It ships with built-in benchmark suites (SWE-Bench Lite, HumanEval, MMLU Mini, GSM8K Mini) and supports custom benchmarks with arbitrary task definitions, test cases, and scoring criteria.

The module supports concurrent task execution with configurable worker pools, multiple evaluation strategies (string matching, code execution, AI debate-based evaluation), and comprehensive result summarization broken down by difficulty level and tags. Benchmark runs can be compared to identify regressions and improvements across provider versions or model updates.

A leaderboard system aggregates results across multiple runs, ranking providers by pass rate while incorporating LLMsVerifier trust scores. The `BenchmarkSystem` orchestrator ties everything together with auto-provider selection, multi-provider comparison, and integration hooks for HelixAgent's debate service and verification pipeline.

## Architecture

```
+-----------------------------+
|     BenchmarkSystem         |
|  (Main Orchestrator)        |
+------+----------+-----------+
       |          |
  +----v----+ +---v-----------+
  | Runner  | | Leaderboard   |
  | (Exec)  | | (Ranking)     |
  +----+----+ +---------------+
       |
  +----v-----------+----------+----------+
  | Task Execution | Evaluation          |
  | (Worker Pool)  | (String/Code/Debate)|
  +----------------+---------------------+

Integration Adapters:
  - DebateAdapterForBenchmark  (debate-based eval)
  - VerifierAdapterForBenchmark (provider selection)
  - ProviderAdapterForBenchmark (LLM execution)
```

## Package Structure

| Package | Purpose |
|---------|---------|
| `benchmark` | Core module: types, runner, integration adapters, leaderboard |

### Source Files

| File | Description |
|------|-------------|
| `types.go` | All type definitions, interfaces, benchmark/task/result/config structs |
| `runner.go` | `StandardBenchmarkRunner` -- built-in benchmarks, task execution, evaluation, summaries |
| `integration.go` | `BenchmarkSystem` orchestrator, debate/verifier/provider adapters, leaderboard generation |

## API Reference

### Types

**Benchmark types**: `BenchmarkTypeSWEBench`, `BenchmarkTypeHumanEval`, `BenchmarkTypeMBPP`, `BenchmarkTypeLMSYS`, `BenchmarkTypeHellaSwag`, `BenchmarkTypeMMLU`, `BenchmarkTypeGSM8K`, `BenchmarkTypeMATH`, `BenchmarkTypeCustom`

**Difficulty levels**: `DifficultyEasy`, `DifficultyMedium`, `DifficultyHard`

**Run statuses**: `BenchmarkStatusPending`, `BenchmarkStatusRunning`, `BenchmarkStatusCompleted`, `BenchmarkStatusFailed`, `BenchmarkStatusCancelled`

### Core Interfaces

```go
// BenchmarkRunner runs benchmarks
type BenchmarkRunner interface {
    ListBenchmarks(ctx context.Context) ([]*Benchmark, error)
    GetBenchmark(ctx context.Context, id string) (*Benchmark, error)
    GetTasks(ctx context.Context, benchmarkID string, config *BenchmarkConfig) ([]*BenchmarkTask, error)
    CreateRun(ctx context.Context, run *BenchmarkRun) error
    StartRun(ctx context.Context, runID string) error
    GetRun(ctx context.Context, runID string) (*BenchmarkRun, error)
    ListRuns(ctx context.Context, filter *RunFilter) ([]*BenchmarkRun, error)
    CancelRun(ctx context.Context, runID string) error
    CompareRuns(ctx context.Context, runID1, runID2 string) (*RunComparison, error)
}

// LLMProvider interface for benchmark execution
type LLMProvider interface {
    Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error)
    GetName() string
}

// CodeExecutor for running test cases against generated code
type CodeExecutor interface {
    Execute(ctx context.Context, code, language string, testInput string) (string, error)
    Validate(ctx context.Context, code, language string, testCases []*TestCase) ([]*TestCaseResult, error)
}

// DebateEvaluator for AI debate-based response evaluation
type DebateEvaluator interface {
    EvaluateResponse(ctx context.Context, task *BenchmarkTask, response string) (float64, bool, error)
}
```

### BenchmarkSystem Methods

```go
func NewBenchmarkSystem(config *BenchmarkSystemConfig, logger *logrus.Logger) *BenchmarkSystem
func (bs *BenchmarkSystem) Initialize(providerAdapter *ProviderAdapterForBenchmark) error
func (bs *BenchmarkSystem) SetDebateService(service DebateServiceForBenchmark)
func (bs *BenchmarkSystem) SetVerifierService(service VerifierServiceForBenchmark)
func (bs *BenchmarkSystem) GetRunner() BenchmarkRunner
func (bs *BenchmarkSystem) RunBenchmarkWithBestProvider(ctx, benchmarkType, config) (*BenchmarkRun, error)
func (bs *BenchmarkSystem) CompareProviders(ctx, benchmarkType, providers, config) ([]*BenchmarkRun, error)
func (bs *BenchmarkSystem) GenerateLeaderboard(ctx, benchmarkType) (*Leaderboard, error)
```

### StandardBenchmarkRunner Methods

```go
func NewStandardBenchmarkRunner(provider LLMProvider, logger *logrus.Logger) *StandardBenchmarkRunner
func (r *StandardBenchmarkRunner) SetCodeExecutor(executor CodeExecutor)
func (r *StandardBenchmarkRunner) SetDebateEvaluator(evaluator DebateEvaluator)
func (r *StandardBenchmarkRunner) AddBenchmark(benchmark *Benchmark, tasks []*BenchmarkTask)
```

## Usage Examples

### Run a built-in benchmark

```go
runner := benchmark.NewStandardBenchmarkRunner(llmProvider, logger)

// Create a benchmark run
run := &benchmark.BenchmarkRun{
    Name:          "MMLU evaluation",
    BenchmarkType: benchmark.BenchmarkTypeMMLU,
    ProviderName:  "claude",
    ModelName:     "claude-3-sonnet",
    Config:        benchmark.DefaultBenchmarkConfig(),
}

runner.CreateRun(ctx, run)
runner.StartRun(ctx, run.ID)

// Check results
completed, _ := runner.GetRun(ctx, run.ID)
fmt.Printf("Pass rate: %.1f%%\n", completed.Summary.PassRate*100)
fmt.Printf("Average latency: %v\n", completed.Summary.AverageLatency)
```

### Compare providers and generate leaderboard

```go
system := benchmark.NewBenchmarkSystem(benchmark.DefaultBenchmarkSystemConfig(), logger)
system.Initialize(providerAdapter)
system.SetVerifierService(verifier)

// Compare multiple providers on HumanEval
runs, _ := system.CompareProviders(ctx,
    benchmark.BenchmarkTypeHumanEval,
    []string{"claude", "openai", "deepseek"},
    nil,
)

// Generate leaderboard
leaderboard, _ := system.GenerateLeaderboard(ctx, benchmark.BenchmarkTypeHumanEval)
for _, entry := range leaderboard.Entries {
    fmt.Printf("#%d %s: %.1f%% pass rate, %.2f avg score\n",
        entry.Rank, entry.ProviderName, entry.PassRate*100, entry.AverageScore)
}
```

### Add a custom benchmark

```go
runner.AddBenchmark(
    &benchmark.Benchmark{
        ID:          "custom-go",
        Type:        benchmark.BenchmarkTypeCustom,
        Name:        "Go Idioms",
        Description: "Tests knowledge of Go idiomatic patterns",
        Version:     "1.0.0",
    },
    []*benchmark.BenchmarkTask{
        {
            ID:         "go-001",
            Name:       "Error wrapping",
            Prompt:     "Write a Go function that wraps errors with context...",
            Expected:   "fmt.Errorf",
            Difficulty: benchmark.DifficultyMedium,
            Tags:       []string{"go", "errors"},
        },
    },
)
```

## Configuration

```go
type BenchmarkConfig struct {
    MaxTasks         int               // Limit number of tasks (0 = no limit)
    Timeout          time.Duration     // Per-task timeout (default: 5m)
    Concurrency      int               // Parallel workers (default: 4)
    Retries          int               // Retry failed tasks (default: 1)
    Temperature      float64           // Model temperature (default: 0.0)
    MaxTokens        int               // Max tokens per response (default: 4096)
    SystemPrompt     string            // Custom system prompt
    Difficulties     []DifficultyLevel // Filter by difficulty
    Tags             []string          // Filter by tags
    SaveResponses    bool              // Save full responses (default: true)
    UseDebateForEval bool              // Use AI debate for evaluation
}

type BenchmarkSystemConfig struct {
    EnableDebateEvaluation bool // Use debate for evaluation (default: true)
    UseVerifierScores      bool // Incorporate verifier scores (default: true)
    AutoSelectProvider     bool // Auto-select best provider (default: true)
    DefaultConcurrency     int  // Default worker count (default: 4)
}
```

### Built-in Benchmarks

| ID | Type | Tasks | Description |
|----|------|-------|-------------|
| `swe-bench-lite` | SWE-Bench | 3 | Bug fixes, error handling, retry logic |
| `humaneval` | HumanEval | 2 | Python code generation with test cases |
| `mmlu-mini` | MMLU | 3 | CS, math, physics multiple choice |
| `gsm8k-mini` | GSM8K | 2 | Math word problems |

### Result Summary Structure

The `BenchmarkSummary` provides:
- `PassRate` / `AverageScore` -- overall metrics
- `ByDifficulty` -- pass rates broken down by Easy/Medium/Hard
- `ByTag` -- pass rates broken down by task tags
- `AverageLatency` / `TotalTokens` -- performance metrics

## Testing

```bash
go build ./...
go test ./... -count=1 -race
```

## Integration with HelixAgent

Benchmark connects to HelixAgent through adapter types:

- **DebateAdapterForBenchmark**: Wraps the debate service to evaluate benchmark responses through multi-LLM consensus. Parses score/pass/fail from debate consensus JSON.
- **VerifierAdapterForBenchmark**: Uses LLMsVerifier scores to auto-select the best provider for benchmarking and to enrich leaderboard entries with provider trust scores.
- **ProviderAdapterForBenchmark**: Bridges HelixAgent's provider registry to the benchmark's `LLMProvider` interface, routing completions through the specified provider and model.
- **Leaderboard**: The `GenerateLeaderboard` method combines benchmark pass rates with verifier scores to produce a ranked comparison of all evaluated providers.
