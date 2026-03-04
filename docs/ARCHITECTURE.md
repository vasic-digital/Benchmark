# Benchmark Module Architecture

**Module:** `digital.vasic.benchmark`

## Overview

LLM benchmarking framework supporting SWE-bench, HumanEval, MMLU, and custom benchmarks with leaderboard tracking and provider comparison.

## Core Components

### Runner (`runner.go`)
Benchmark execution engine. Loads benchmark suites, runs evaluations against configured providers, collects metrics, and generates comparison reports.

**Key features:**
- Parallel provider evaluation
- Timeout-bounded execution
- Result persistence

### Types (`types.go`)
Shared type definitions: BenchmarkSuite, BenchmarkCase, EvaluationResult, ProviderScore, Leaderboard.

**Standard suites:**
- **SWE-bench** — Software engineering task completion
- **HumanEval** — Code generation correctness
- **MMLU** — Massive multitask language understanding
- **Custom** — User-defined evaluation criteria

### Integration (`integration.go`)
Connects benchmark system with HelixAgent's provider registry and debate system. Provides the debate benchmark bridge for evaluating debate outputs.

## Evaluation Flow

```
Select Suite → Load Cases → For each Provider:
                                Execute Case
                                Score Result
                                Record Metrics
                            ↓
                        Aggregate Scores
                            ↓
                        Update Leaderboard
```

## Package Structure

```
benchmark/
├── runner.go        # Benchmark execution engine
├── types.go         # Suite/case/result types
└── integration.go   # HelixAgent integration
```

## Integration

The debate benchmark bridge (`internal/debate/benchmark/`) uses this module for evaluating debate outputs against standard benchmarks.
Adapter: `internal/adapters/benchmark/adapter.go`
