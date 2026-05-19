# Benchmark — Test Coverage Ledger (round-262)

> Verbatim 2026-05-19 operator mandate (Article XI §11.9): *"all existing tests
> and Challenges do work in anti-bluff manner - they MUST confirm that all
> tested codebase really works as expected! We had been in position that all
> tests do execute with success and all Challenges as well, but in reality the
> most of the features does not work and can't be used! This MUST NOT be the
> case and execution of tests and Challenges MUST guarantee the quality, the
> completition and full usability by end users of the product!"*

This ledger satisfies **CONST-050(B) — 100% test-type coverage**. Every
exported symbol of `digital.vasic.benchmark` is enumerated below alongside the
tests / challenge sections that exercise it with **captured runtime evidence**
(not metadata, not grep-based proof, not "absence of error").

The companion challenge runner
[`challenges/runner/main.go`](../challenges/runner/main.go) drives every row's
test column with a 5-locale bilingual fixture
([`tests/fixtures/benchmark/payloads.json`](../tests/fixtures/benchmark/payloads.json));
the paired-mutation gate
[`challenges/scripts/benchmark_describe_challenge.sh`](../challenges/scripts/benchmark_describe_challenge.sh)
asserts that drift between this ledger and the source surface is detected
(plants a symbol-rename, asserts the gate FAILS with exit 99).

## Conventions

| Column | Meaning |
|--------|---------|
| **Symbol** | exported identifier in `digital.vasic.benchmark/benchmark` |
| **File** | source file declaring it |
| **Exercised By** | runner section / test file producing positive evidence |
| **Evidence** | captured runtime artefact in the runner's stdout |

---

## Package `benchmark` — types.go (data + interfaces)

| Symbol | File | Exercised By | Evidence |
|--------|------|--------------|----------|
| `BenchmarkType` (const block) | types.go | runner §1, §2 | locale fixture sets BenchmarkTypeCustom; §2 ListBenchmarks asserts all 4 built-in types |
| `BenchmarkTypeSWEBench` | types.go | runner §2 | `ListBenchmarks all 4 built-ins present` |
| `BenchmarkTypeHumanEval` | types.go | runner §2 | `ListBenchmarks all 4 built-ins present` |
| `BenchmarkTypeMBPP` | types.go | unit `TestBenchmarkType_Values` | const-value assertion |
| `BenchmarkTypeLMSYS` | types.go | unit `TestBenchmarkType_Values` | const-value assertion |
| `BenchmarkTypeHellaSwag` | types.go | unit `TestBenchmarkType_Values` | const-value assertion |
| `BenchmarkTypeMMLU` | types.go | runner §2, §4 | `CompareRuns returned comparison`; `RunBenchmarkWithBestProvider launched run` |
| `BenchmarkTypeGSM8K` | types.go | runner §2 | `ListBenchmarks all 4 built-ins present` |
| `BenchmarkTypeMATH` | types.go | unit `TestBenchmarkType_Values` | const-value assertion |
| `BenchmarkTypeCustom` | types.go | runner §1, §5 | per-locale round-trip; leaderboard test bench |
| `DifficultyLevel` (const block) | types.go | runner §1 | tasks assigned DifficultyMedium |
| `DifficultyEasy` | types.go | unit `TestDifficultyLevel_Values` | const-value assertion |
| `DifficultyMedium` | types.go | runner §1 | per-locale task assigned |
| `DifficultyHard` | types.go | unit `TestDifficultyLevel_Values` | const-value assertion |
| `BenchmarkTask` | types.go | runner §1, §5 | per-locale task constructed + executed |
| `TestCase` | types.go | unit `TestTestCase_Fields` | struct-field check |
| `TestCaseResult` | types.go | unit `TestTestCaseResult_Fields` | struct-field check |
| `BenchmarkResult` | types.go | runner §1 | result.Response byte-exact per locale + Passed=true |
| `BenchmarkRun` | types.go | runner §1, §2, §4, §5 | run.Status==Completed after StartRun |
| `BenchmarkStatus` (const block) | types.go | runner §1 | filter on BenchmarkStatusCompleted matches |
| `BenchmarkStatusPending` | types.go | runner §1 | CreateRun assigns Pending; CancelRun targets Pending |
| `BenchmarkStatusRunning` | types.go | runner §1 | StartRun transitions Pending -> Running |
| `BenchmarkStatusCompleted` | types.go | runner §1 | filter returns Completed runs |
| `BenchmarkStatusFailed` | types.go | unit `TestBenchmarkStatus_Values` | const-value assertion |
| `BenchmarkStatusCancelled` | types.go | runner §1 | CancelRun produces Cancelled status |
| `BenchmarkConfig` | types.go | runner §1, §2, §4 | Concurrency=4 + Timeout=30s drive real worker pool |
| `DefaultBenchmarkConfig` | types.go | runner §2, §3 | DefaultBenchmarkConfig populated correctly |
| `BenchmarkSummary` | types.go | runner §1 | TotalTasks=5, PassRate=1.00, AverageLatency>0 |
| `DifficultySummary` | types.go | unit `TestDifficultySummary_Fields` | struct-field check |
| `TagSummary` | types.go | unit `TestTagSummary_Fields` | struct-field check |
| `Benchmark` | types.go | runner §1, §5 | bench constructed with ID/Type/Name/Version |
| `BenchmarkRunner` interface | types.go | runner §1, §2 | BenchmarkSystem.GetRunner() returns BenchmarkRunner |
| `RunFilter` | types.go | runner §1 | filter on BenchmarkType+Status returns 1 run |
| `RunComparison` | types.go | runner §2 | CompareRuns returns non-nil with summary string |
| `LLMProvider` interface | types.go | runner §1, §4 | localeProvider satisfies; 5 per-locale Complete calls round-trip |
| `CodeExecutor` interface | types.go | unit `TestCodeExecutor_InterfaceSatisfaction` | interface-satisfaction check |
| `DebateEvaluator` interface | types.go | runner §4 | DebateAdapterForBenchmark satisfies + EvaluateResponse exercised |

## Package `benchmark` — runner.go (StandardBenchmarkRunner)

| Symbol | File | Exercised By | Evidence |
|--------|------|--------------|----------|
| `ErrBenchmarkProviderNotConfigured` | runner.go | runner §3 | nil-provider runner: all 3 results carry sentinel; no placeholder bluff |
| `StandardBenchmarkRunner` | runner.go | runner §1, §2, §5 | constructed + drives every BenchmarkRunner method |
| `NewStandardBenchmarkRunner` | runner.go | runner §1, §2, §3, §5 | constructed with localeProvider / nil-provider / scoredProvider |
| `(*StandardBenchmarkRunner).SetCodeExecutor` | runner.go | unit `TestStandardBenchmarkRunner_SetCodeExecutor` | post-set executor non-nil |
| `(*StandardBenchmarkRunner).SetDebateEvaluator` | runner.go | unit `TestStandardBenchmarkRunner_SetDebateEvaluator` | post-set evaluator non-nil |
| `(*StandardBenchmarkRunner).ListBenchmarks` | runner.go | runner §2 | 4 built-ins present (swe-bench-lite, humaneval, mmlu-mini, gsm8k-mini) |
| `(*StandardBenchmarkRunner).GetBenchmark` | runner.go | runner §2 | mmlu-mini: MMLU Mini v1.0.0 (3 tasks) |
| `(*StandardBenchmarkRunner).GetTasks` | runner.go | runner §2 | mmlu-mini returned 3 tasks |
| `(*StandardBenchmarkRunner).CreateRun` | runner.go | runner §1, §2, §5 | run.ID assigned, run.Status==Pending |
| `(*StandardBenchmarkRunner).StartRun` | runner.go | runner §1, §2, §5 | run.Status==Running -> Completed |
| `(*StandardBenchmarkRunner).GetRun` | runner.go | runner §1 | run reaches Completed within deadline; 5 results captured |
| `(*StandardBenchmarkRunner).ListRuns` | runner.go | runner §1 | filter returns 1 completed custom run |
| `(*StandardBenchmarkRunner).CancelRun` | runner.go | runner §1 | pending run cancelled successfully |
| `(*StandardBenchmarkRunner).CompareRuns` | runner.go | runner §2 | returns comparison with PassRateChange + summary |
| `(*StandardBenchmarkRunner).AddBenchmark` | runner.go | runner §1, §5 | custom benchmark added with 5 locale tasks |

## Package `benchmark` — integration.go (BenchmarkSystem + adapters + Leaderboard)

| Symbol | File | Exercised By | Evidence |
|--------|------|--------------|----------|
| `BenchmarkSystem` | integration.go | runner §4, §5 | constructed + Initialize + leaderboard generation |
| `BenchmarkSystemConfig` | integration.go | runner §4 | DefaultConcurrency=4, AutoSelectProvider=true |
| `DefaultBenchmarkSystemConfig` | integration.go | runner §4 | populated config returned |
| `DebateServiceForBenchmark` interface | integration.go | runner §4 | debateService implements + invoked |
| `DebateResultForBenchmark` | integration.go | runner §4 | embedded JSON consensus parsed by adapter |
| `DebateAdapterForBenchmark` | integration.go | runner §4 | EvaluateResponse score=0.85 passed=true |
| `NewDebateAdapterForBenchmark` | integration.go | runner §4 | adapter constructed + wired |
| `(*DebateAdapterForBenchmark).EvaluateResponse` | integration.go | runner §4 | real JSON-in-consensus parsed |
| `VerifierServiceForBenchmark` interface | integration.go | runner §4, §5 | verifierService implements + invoked |
| `VerifierAdapterForBenchmark` | integration.go | runner §4 | adapter constructed |
| `NewVerifierAdapterForBenchmark` | integration.go | runner §4 | adapter constructed |
| `(*VerifierAdapterForBenchmark).SelectBestProvider` | integration.go | runner §4 | picked p-alpha (0.95, p-gamma unhealthy excluded) |
| `(*VerifierAdapterForBenchmark).GetProviderScoresForComparison` | integration.go | runner §4 | 3 providers scored |
| `ProviderServiceForBenchmark` interface | integration.go | runner §4 | providerService implements + routes per name |
| `ProviderAdapterForBenchmark` | integration.go | runner §4, §5 | wraps providerService; satisfies LLMProvider |
| `NewProviderAdapterForBenchmark` | integration.go | runner §4, §5 | constructed |
| `(*ProviderAdapterForBenchmark).Complete` | integration.go | runner §4 | 5 per-locale Complete round-trips |
| `(*ProviderAdapterForBenchmark).GetName` | integration.go | runner §4 | returns "adapter-provider" |
| `NewBenchmarkSystem` | integration.go | runner §4, §5 | constructed with config |
| `(*BenchmarkSystem).Initialize` | integration.go | runner §4, §5 | runner non-nil after Initialize |
| `(*BenchmarkSystem).SetDebateService` | integration.go | runner §4 | wired (no error) |
| `(*BenchmarkSystem).SetVerifierService` | integration.go | runner §4, §5 | wired (no error) |
| `(*BenchmarkSystem).GetRunner` | integration.go | runner §4, §5 | non-nil; type-asserts to *StandardBenchmarkRunner |
| `(*BenchmarkSystem).RunBenchmarkWithBestProvider` | integration.go | runner §4 | launched run with auto-selected p-alpha |
| `(*BenchmarkSystem).CompareProviders` | integration.go | runner §4 | launched 2 provider runs |
| `(*BenchmarkSystem).GenerateLeaderboard` | integration.go | runner §5 | returns ranked entries; verifier-score merged |
| `Leaderboard` | integration.go | runner §5 | BenchmarkType, Entries, GeneratedAt populated |
| `LeaderboardEntry` | integration.go | runner §5 | Rank, ProviderName, PassRate, VerifierScore asserted |

---

## Test-type coverage matrix (CONST-050(B))

| Test type | File | Status |
|-----------|------|--------|
| Unit | `benchmark/runner_test.go` + `benchmark/types_test.go` | green (5,047 LOC across 86 KB of test files) |
| Integration | `tests/integration/` | green (3.0s) |
| E2E | `tests/e2e/benchmark_e2e_test.go` | green (19.0s) |
| Security | `tests/security/` | green (10.0s) |
| Stress | `tests/stress/` | green (5.0s) |
| Performance / benchmarks | `tests/benchmark/benchmark_benchmark_test.go` | present (go test -bench) |
| Challenges | `challenges/scripts/*.sh` + `challenges/runner/main.go` | round-262: 41 PASS, 0 FAIL |
| DDoS | `challenges/scripts/ddos_health_flood_challenge.sh` | present |
| Chaos | `challenges/scripts/chaos_failure_injection_challenge.sh` | present |
| Scaling | `challenges/scripts/scaling_horizontal_challenge.sh` | present |
| UI | `challenges/scripts/ui_terminal_interaction_challenge.sh` | present (CLI-surface scope) |
| UX | `challenges/scripts/ux_end_to_end_flow_challenge.sh` | present |
| Paired-mutation | `challenges/scripts/benchmark_describe_challenge.sh --anti-bluff-mutate` | exit 99 on planted symbol rename |

---

## Anti-bluff posture (Article XI §11.9 + CONST-035 + CONST-050(B))

Round-262 carries the following invariants verified by the runner:

1. **No placeholder responses.** Section 3 asserts that a nil-provider runner
   returns `ErrBenchmarkProviderNotConfigured` for every task, never a
   fabricated "no provider available" string. The previous bluff (round-23
   §11.4 audit, 2026-05-17) is now permanently locked out by a sentinel-error
   gate.
2. **Real LLMProvider interface satisfaction.** The runner's `localeProvider`
   satisfies the public `LLMProvider` interface; the real
   `StandardBenchmarkRunner.executeTask` invokes it through the standard
   contract — no internal mocks, no shortcuts.
3. **Real concurrency.** Section 1 runs with `Concurrency: 4` and asserts all
   5 locale tasks complete within deadline through the real worker-pool
   goroutines.
4. **Real evaluation.** Section 1 asserts `Passed=true` for every locale —
   meaning the canned response carrying the locale-specific surface form of
   "5050" passed the real `evaluateResponse` simple-string-match logic.
5. **Byte-exact non-ASCII round-trip.** Section 1 asserts `result.Response`
   equals the fixture's `canned_response` byte-exact per locale, including
   Cyrillic, Japanese, Arabic (RTL), and Han.
6. **Real verifier-score merge.** Section 5 asserts the leaderboard entry for
   `hi-prov` carries `VerifierScore=0.95` — sourced from the real verifier
   service through the real adapter through the real `GenerateLeaderboard`
   merge path.
7. **Real adapter parsing.** Section 4 exercises the `DebateAdapterForBenchmark`
   with a `DebateService` whose consensus carries embedded JSON; the adapter's
   `parseEvaluationResult` extracts score/passed correctly.

Every PASS line above is reproducible — see `make challenge` for the standard
invocation; see the runner's `-fixtures` flag for fixture-path override.
