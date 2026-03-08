# Benchmark - Getting Started

**Module:** `digital.vasic.benchmark`

## Installation

```go
import "digital.vasic.benchmark/benchmark"
```

## Quick Start: Run a Benchmark

### 1. Create a Benchmark Runner

The runner requires an `LLMProvider` implementation to execute prompts
against a model, and an optional logger:

```go
package main

import (
    "context"
    "fmt"

    "digital.vasic.benchmark/benchmark"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()

    // Implement or provide an LLMProvider
    provider := myLLMProvider{}
    runner := benchmark.NewStandardBenchmarkRunner(provider, logger)
```

### 2. List Available Benchmarks

The runner ships with built-in benchmark suites:

```go
    benchmarks := runner.ListBenchmarks(context.Background())
    for _, b := range benchmarks {
        fmt.Printf("%-20s %s (%d tasks)\n", b.ID, b.Name, b.TaskCount)
    }
```

Built-in suites:

| Suite ID | Name | Type | Description |
|----------|------|------|-------------|
| `swe-bench-lite` | SWE-Bench Lite | SWE-bench | Software engineering tasks |
| `humaneval` | HumanEval | HumanEval | Code generation from OpenAI |
| `mmlu-mini` | MMLU Mini | MMLU | Multitask language understanding |
| `gsm8k-mini` | GSM8K Mini | GSM8K | Grade school math problems |

### 3. Run a Benchmark

```go
    run, err := runner.RunBenchmark(context.Background(), "humaneval", benchmark.RunConfig{
        ProviderName: "openai",
        ModelName:    "gpt-4",
        MaxTasks:     10,
        Timeout:      5 * time.Minute,
        Concurrency:  2,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Run ID: %s\n", run.ID)
    fmt.Printf("Score:  %.2f\n", run.AggregateScore)
    fmt.Printf("Passed: %d/%d\n", run.PassCount, run.TotalTasks)
}
```

## Interpreting Results

### BenchmarkResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `Score` | `float64` | 0.0 to 1.0 overall correctness score |
| `Passed` | `bool` | Whether the task passed evaluation |
| `Latency` | `time.Duration` | Time to generate the response |
| `TokensUsed` | `int` | Total tokens consumed |
| `TestResults` | `[]*TestCaseResult` | Per-test-case pass/fail (code benchmarks) |

### Provider Comparison

Compare multiple providers on the same benchmark:

```go
comparison, err := runner.CompareProviders(ctx, "humaneval", []benchmark.ProviderConfig{
    {Name: "openai", Model: "gpt-4"},
    {Name: "deepseek", Model: "deepseek-coder"},
    {Name: "anthropic", Model: "claude-3-sonnet"},
})
```

### Leaderboard

Retrieve the ranked leaderboard across all runs:

```go
leaderboard := runner.GetLeaderboard(ctx, "humaneval")
for i, entry := range leaderboard.Entries {
    fmt.Printf("#%d %s/%s: %.2f\n", i+1, entry.Provider, entry.Model, entry.Score)
}
```

## Custom Benchmarks

Create domain-specific evaluation suites:

```go
customBench := &benchmark.Benchmark{
    ID:          "my-domain-eval",
    Type:        benchmark.BenchmarkTypeCustom,
    Name:        "Domain Evaluation",
    Description: "Custom domain-specific tasks",
}

tasks := []*benchmark.BenchmarkTask{
    {
        ID:       "task-1",
        Name:     "Summarize Contract",
        Prompt:   "Summarize the following legal contract...",
        Expected: "The contract establishes...",
    },
}

runner.RegisterBenchmark(ctx, customBench, tasks)
```

## Difficulty Levels

Tasks are tagged with difficulty for granular analysis:

| Level | Constant | Use Case |
|-------|----------|----------|
| Easy | `DifficultyEasy` | Basic comprehension, simple code |
| Medium | `DifficultyMedium` | Multi-step reasoning, moderate code |
| Hard | `DifficultyHard` | Complex reasoning, system design |

## Integration with HelixAgent

The Benchmark module connects to HelixAgent through:

1. **Adapter** -- `internal/adapters/benchmark/adapter.go` bridges
   HelixAgent's provider registry to the benchmark runner
2. **Debate Bridge** -- `internal/debate/benchmark/` evaluates debate
   outputs against standard benchmarks using static code analysis

## Next Steps

- See [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- See [API_REFERENCE.md](API_REFERENCE.md) for full type definitions
