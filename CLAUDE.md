# CLAUDE.md - Benchmark Module


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# MMLU + SWE-bench subsets + leaderboard generation
cd Benchmark && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestFullBenchmarkWorkflow_MMLU_E2E|TestFullBenchmarkWorkflow_SWEBench_E2E|TestGenerateLeaderboard_E2E' \
  ./tests/e2e/...
```
Expect: three E2E PASS; pass rate computed, leaderboard ranked, verifier score merged.


## Overview

`digital.vasic.benchmark` is the LLM benchmark suite runner: executes tasks from 9 named benchmarks (SWE-Bench, HumanEval, MBPP, LMSYS, HellaSwag, MMLU, GSM8K, MATH, plus a `custom` bucket) against pluggable providers with concurrency control, produces leaderboards comparing providers, and optionally routes evaluation through the debate service or the LLMsVerifier scorer.

**Module:** `digital.vasic.benchmark` (Go 1.24+, ~3,900 LOC across 4 files — `runner.go` carries the built-in fixture data).

## Architecture

```
BenchmarkSystem
    │
    ├── StandardBenchmarkRunner
    │     • CreateRun / StartRun / GetRun / ListRuns / CancelRun
    │     • ExecuteTask / RecordResult
    │     • Concurrency cap (workers); no cancel propagation to mid-flight workers
    │
    ├── Built-in fixtures (hardcoded)
    │     • SWE-Bench Lite
    │     • HumanEval
    │     • MMLU Mini
    │     • GSM8K Mini
    │     Only a handful of tasks each — production use requires loading full datasets
    │
    ├── BenchmarkTask / BenchmarkRun / BenchmarkSummary
    │     Summary aggregates pass rate, latency, token usage by difficulty and tag
    │
    ├── Leaderboard
    │     • Ranks providers by pass rate (bubble-sort, ties undefined)
    │     • GenerateLeaderboard can merge verifier scores
    │
    ├── DebateAdapterForBenchmark   (optional)
    │     • EvaluateResponse via debate consensus
    │     • Parses JSON from consensus string; falls back to confidence ≥ 0.7
    │
    └── VerifierAdapterForBenchmark (optional)
          • SelectBestProvider / GetProviderScoresForComparison
```

## Key types and interfaces

```go
type BenchmarkRunner interface {
    CreateRun, StartRun, GetRun, ListRuns, CancelRun
    ExecuteTask, RecordResult
}

type LLMProvider interface {
    Complete(ctx, prompt, systemPrompt string) (response string, tokens int, err error)
    GetName() string
}

type DebateEvaluator interface {
    EvaluateResponse(ctx, task *BenchmarkTask, response string) (score float64, passed bool, err error)
}

type BenchmarkConfig struct {
    MaxTasks, Concurrency, Retries int
    Timeout      time.Duration
    Temperature  float64
    MaxTokens    int
    SaveResponses, UseDebateForEval bool
}
```

## Integration Seams

- **Upstream (imports):** none.
- **Downstream (sibling consumer):** root HelixAgent via `internal/handlers/benchmark_handler.go` (REST endpoints under `/v1/benchmark/*`).
- **Sibling complements:** `LLMProvider` (implementation injected as the provider under test), `DebateOrchestrator` (optional evaluator), `LLMsVerifier` (optional scoring for leaderboard).

## Gotchas

1. **Built-in tasks are tiny** — fixture-scale, not benchmark-scale. Do not compare numbers across projects without loading the real datasets. Production usage requires implementing a task loader.
2. **No cancel propagation to in-flight workers** — cancelling a run stops *new* task dispatch but workers complete their current task. Budget your timeouts accordingly.
3. **Debate JSON parsing is lenient** — if consensus text isn't parseable JSON, falls back to confidence threshold (≥ 0.7 = pass). This hides evaluator errors as "pass".
4. **`GetProviderScoresForComparison`** calls `GetTopProviders(10)` but does not use the result — dead code path, be aware if you extend.
5. **In-memory only** — no persistence of runs or results. Process restart loses everything.
6. **Leaderboard sort is O(n²) bubble-sort** — fine for ~50 providers; noticeable for more.

## Acceptance demo

```bash
GOMAXPROCS=2 nice -n 19 go test -race -v \
  -run TestBenchmarkWithMockedProviders_E2E ./tests/e2e/benchmark_e2e_test.go -count=1

GOMAXPROCS=2 nice -n 19 go test -race -v \
  -run TestDebateEvaluationForBenchmark_E2E ./tests/e2e/benchmark_e2e_test.go -count=1

GOMAXPROCS=2 nice -n 19 go test -race -v \
  -run TestGenerateLeaderboard_E2E ./tests/e2e/benchmark_e2e_test.go -count=1

# Expected:
#   PASS: Run SWE-Bench Lite, pass rate computed
#   PASS: Debate adapter invoked, response scored
#   PASS: Multi-provider leaderboard ranked by pass rate, verifier score merged
```

A real demo against live providers is the next step: wire at least one real LLM provider, run a non-trivial benchmark subset, record the leaderboard, and commit it.
