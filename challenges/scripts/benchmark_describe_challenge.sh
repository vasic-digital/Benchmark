#!/usr/bin/env bash
# benchmark_describe_challenge.sh
#
# Round-262 paired-mutation deep-doc challenge for digital.vasic.benchmark.
#
# Validates that:
#   1. The deep-doc ledger (docs/test-coverage.md) lists every exported
#      structural symbol from types.go, runner.go, and integration.go.
#   2. The multi-locale fixture
#      (tests/fixtures/benchmark/payloads.json) parses and contains
#      at least 3 locales.
#   3. The multi-locale runner (challenges/runner/main.go) builds and
#      runs, byte-preserving non-ASCII LLM canned responses through the
#      real StandardBenchmarkRunner + BenchmarkSystem + LeaderboardEntry
#      machinery for all 5 locales.
#   4. The README enumerates the round-262 anti-bluff guarantees.
#
# Paired-mutation invariant (CONST-035 + CONST-050(B)):
#   With --anti-bluff-mutate the script plants a deliberate symbol-rename
#   mutation in the ledger (in a tmp copy: StandardBenchmarkRunner ->
#   StandardBenchmarkRunner_MUTATED), reruns validation, and asserts the
#   gate FAILS with exit 99. This proves the gate actually catches
#   ledger-vs-source drift instead of rubber-stamping it.
#
# Exit codes:
#   0  — gate PASS on clean tree
#   1  — gate FAIL on clean tree (real failure to fix)
#   99 — paired-mutation correctly detected (good — proves anti-bluff)
#   2  — usage / environment error

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MUTATE=0
for arg in "$@"; do
    case "$arg" in
        --anti-bluff-mutate) MUTATE=1 ;;
        --help|-h)
            sed -n '1,32p' "$0"
            exit 0
            ;;
        *)
            echo "unknown argument: $arg" >&2
            exit 2
            ;;
    esac
done

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1"; }

LEDGER="${MODULE_DIR}/docs/test-coverage.md"
FIXTURE="${MODULE_DIR}/tests/fixtures/benchmark/payloads.json"
RUNNER="${MODULE_DIR}/challenges/runner/main.go"
README="${MODULE_DIR}/README.md"

LEDGER_WORK="${LEDGER}"
TMP_LEDGER=""
if [ "${MUTATE}" -eq 1 ]; then
    TMP_LEDGER="$(mktemp)"
    cp "${LEDGER}" "${TMP_LEDGER}"
    # Plant a rename so the symbol no longer matches what the source declares.
    sed -i 's/StandardBenchmarkRunner/StandardBenchmarkRunner_MUTATED/g' "${TMP_LEDGER}"
    LEDGER_WORK="${TMP_LEDGER}"
    echo "=== Benchmark Describe Challenge (anti-bluff-mutate mode) ==="
else
    echo "=== Benchmark Describe Challenge (clean mode) ==="
fi
echo ""

# Section 1: ledger presence and freshness
echo "Section 1: docs/test-coverage.md ledger"
if [ ! -f "${LEDGER_WORK}" ]; then
    fail "ledger missing at ${LEDGER_WORK}"
else
    pass "ledger present"
    if grep -q "round-262" "${LEDGER_WORK}"; then
        pass "ledger marked round-262"
    else
        fail "ledger missing round-262 marker"
    fi
    if grep -q "execution of tests and Challenges MUST guarantee" "${LEDGER_WORK}"; then
        pass "ledger carries Article XI §11.9 mandate"
    else
        fail "ledger missing Article XI §11.9 mandate"
    fi
fi

# Section 2: every exported package symbol appears in ledger.
echo ""
echo "Section 2: structural symbol cross-reference"

EXPECTED_SYMBOLS=(
    # types.go — types + interfaces
    "BenchmarkType" "BenchmarkTypeSWEBench" "BenchmarkTypeHumanEval"
    "BenchmarkTypeMMLU" "BenchmarkTypeGSM8K" "BenchmarkTypeCustom"
    "DifficultyLevel" "DifficultyEasy" "DifficultyMedium" "DifficultyHard"
    "BenchmarkTask" "TestCase" "TestCaseResult"
    "BenchmarkResult" "BenchmarkRun" "BenchmarkStatus"
    "BenchmarkStatusPending" "BenchmarkStatusRunning"
    "BenchmarkStatusCompleted" "BenchmarkStatusFailed" "BenchmarkStatusCancelled"
    "BenchmarkConfig" "DefaultBenchmarkConfig"
    "BenchmarkSummary" "DifficultySummary" "TagSummary" "Benchmark"
    "BenchmarkRunner" "RunFilter" "RunComparison"
    "LLMProvider" "CodeExecutor" "DebateEvaluator"
    # runner.go — runner surface
    "ErrBenchmarkProviderNotConfigured"
    "StandardBenchmarkRunner" "NewStandardBenchmarkRunner"
    # integration.go — system + adapters + leaderboard
    "BenchmarkSystem" "BenchmarkSystemConfig" "DefaultBenchmarkSystemConfig"
    "DebateServiceForBenchmark" "DebateResultForBenchmark"
    "DebateAdapterForBenchmark" "NewDebateAdapterForBenchmark"
    "VerifierServiceForBenchmark" "VerifierAdapterForBenchmark"
    "NewVerifierAdapterForBenchmark"
    "ProviderServiceForBenchmark" "ProviderAdapterForBenchmark"
    "NewProviderAdapterForBenchmark"
    "NewBenchmarkSystem"
    "Leaderboard" "LeaderboardEntry"
)

CHECKED=0
MISSING=0
for sym in "${EXPECTED_SYMBOLS[@]}"; do
    CHECKED=$((CHECKED + 1))
    if grep -qE "\\b${sym}\\b" "${LEDGER_WORK}"; then
        : # found
    else
        fail "ledger missing symbol ${sym}"
        MISSING=$((MISSING + 1))
    fi
done
if [ "${MISSING}" -eq 0 ]; then
    pass "all ${CHECKED} structural symbols cross-referenced in ledger"
fi

# Section 3: multi-locale fixture sanity
echo ""
echo "Section 3: multi-locale fixture"
if [ ! -f "${FIXTURE}" ]; then
    fail "fixture missing at ${FIXTURE}"
else
    pass "fixture present"
    LOCALE_COUNT=$(grep -oE '"locale":\s*"[^"]+"' "${FIXTURE}" | sort -u | wc -l)
    if [ "${LOCALE_COUNT}" -ge 3 ]; then
        pass "fixture covers ${LOCALE_COUNT} locales (>=3)"
    else
        fail "fixture covers only ${LOCALE_COUNT} locales (<3)"
    fi
fi

# Section 4: runner builds + runs against every section
echo ""
echo "Section 4: multi-locale runner build + run (real runner + system + leaderboard)"
if [ ! -f "${RUNNER}" ]; then
    fail "runner missing at ${RUNNER}"
else
    pass "runner source present"
    cd "${MODULE_DIR}"
    if go build -o /tmp/benchmark_round262_runner ./challenges/runner/ 2>/tmp/benchmark_build.log; then
        pass "runner builds"
        if /tmp/benchmark_round262_runner -fixtures "${FIXTURE}" > /tmp/benchmark_run.log 2>&1; then
            pass "runner exit 0 across every section + locale"
            if grep -q "PASS: \[Section1\]\[round-trip\]\[sr\]" /tmp/benchmark_run.log; then
                pass "Section 1 Cyrillic (sr) response round-trip"
            else
                fail "Section 1 Cyrillic (sr) round-trip missing"
            fi
            if grep -q "PASS: \[Section1\]\[round-trip\]\[ja\]" /tmp/benchmark_run.log; then
                pass "Section 1 Japanese (ja) response round-trip"
            else
                fail "Section 1 Japanese (ja) round-trip missing"
            fi
            if grep -q "PASS: \[Section1\]\[round-trip\]\[ar\]" /tmp/benchmark_run.log; then
                pass "Section 1 Arabic (ar) response round-trip"
            else
                fail "Section 1 Arabic (ar) round-trip missing"
            fi
            if grep -q "PASS: \[Section1\]\[round-trip\]\[zh-CN\]" /tmp/benchmark_run.log; then
                pass "Section 1 Han (zh-CN) response round-trip"
            else
                fail "Section 1 Han (zh-CN) round-trip missing"
            fi
            if grep -q "PASS: \[Section1\]\[Summary.PassRate\] 1.00" /tmp/benchmark_run.log; then
                pass "Section 1 100% pass-rate across all 5 locales"
            else
                fail "Section 1 pass-rate not 1.00"
            fi
            if grep -q "PASS: \[Section2\]\[ListBenchmarks\] all 4 built-ins" /tmp/benchmark_run.log; then
                pass "Section 2 all 4 built-in benchmarks present"
            else
                fail "Section 2 built-ins missing"
            fi
            if grep -q "PASS: \[Section2\]\[CompareRuns\]" /tmp/benchmark_run.log; then
                pass "Section 2 CompareRuns exercised"
            else
                fail "Section 2 CompareRuns missing"
            fi
            if grep -q "PASS: \[Section3\]\[nil-provider\] all" /tmp/benchmark_run.log; then
                pass "Section 3 nil-provider sentinel enforced (round-23 §11.4 audit)"
            else
                fail "Section 3 nil-provider sentinel not enforced"
            fi
            if grep -q "PASS: \[Section4\]\[DebateAdapter.EvaluateResponse\]" /tmp/benchmark_run.log; then
                pass "Section 4 DebateAdapter real JSON-in-consensus parsed"
            else
                fail "Section 4 DebateAdapter missing"
            fi
            if grep -q "PASS: \[Section4\]\[VerifierAdapter.SelectBestProvider\]" /tmp/benchmark_run.log; then
                pass "Section 4 VerifierAdapter health-filtered SelectBestProvider"
            else
                fail "Section 4 VerifierAdapter missing"
            fi
            if grep -q "PASS: \[Section4\]\[BenchmarkSystem.RunBenchmarkWithBestProvider\]" /tmp/benchmark_run.log; then
                pass "Section 4 BenchmarkSystem.RunBenchmarkWithBestProvider exercised"
            else
                fail "Section 4 RunBenchmarkWithBestProvider missing"
            fi
            if grep -q "PASS: \[Section5\]\[GenerateLeaderboard\]" /tmp/benchmark_run.log; then
                pass "Section 5 GenerateLeaderboard exercised"
            else
                fail "Section 5 GenerateLeaderboard missing"
            fi
            if grep -q "PASS: \[Section5\]\[Leaderboard.VerifierScore\]" /tmp/benchmark_run.log; then
                pass "Section 5 leaderboard verifier-score merge asserted"
            else
                fail "Section 5 verifier-score merge missing"
            fi
            if grep -q "PASS: \[Section5\]\[Leaderboard.Rank\]" /tmp/benchmark_run.log; then
                pass "Section 5 leaderboard rank assignment asserted"
            else
                fail "Section 5 rank assignment missing"
            fi
        else
            fail "runner exit non-zero — see /tmp/benchmark_run.log"
            sed -n '1,80p' /tmp/benchmark_run.log
        fi
    else
        fail "runner build failed — see /tmp/benchmark_build.log"
        sed -n '1,40p' /tmp/benchmark_build.log
    fi
    rm -f /tmp/benchmark_round262_runner
fi

# Section 5: README round-262 anti-bluff section
echo ""
echo "Section 5: README round-262 anti-bluff section"
if grep -q "Anti-bluff guarantees" "${README}"; then
    pass "README declares Anti-bluff guarantees"
else
    fail "README missing Anti-bluff guarantees section"
fi
if grep -q "round-262" "${README}"; then
    pass "README marked round-262"
else
    fail "README missing round-262 marker"
fi

# Cleanup mutated ledger if any
if [ -n "${TMP_LEDGER}" ]; then
    rm -f "${TMP_LEDGER}"
fi

echo ""
echo "=== Summary: ${PASS}/${TOTAL} PASS, ${FAIL} FAIL ==="

if [ "${MUTATE}" -eq 1 ]; then
    if [ "${FAIL}" -gt 0 ]; then
        echo "anti-bluff-mutate: gate correctly detected planted mutation (exit 99)"
        exit 99
    else
        echo "anti-bluff-mutate: gate FAILED to detect planted mutation — bluff!"
        exit 1
    fi
fi

if [ "${FAIL}" -gt 0 ]; then
    exit 1
fi
exit 0
