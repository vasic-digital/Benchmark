package benchmark

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeTranslator is a unit-test Translator (CONST-050(A): mocks permitted in
// *_test.go). It records every key requested and returns a locale-prefixed
// rendering so a test can prove the production code routed through tr().
type fakeTranslator struct {
	locale string
	seen   []string
}

func (f *fakeTranslator) Translate(key string, args map[string]interface{}) string {
	f.seen = append(f.seen, key)
	out := f.locale + ":" + key
	if n, ok := args["regressions"]; ok {
		out += ":reg=" + itoa(n.(int))
	}
	if n, ok := args["improvements"]; ok {
		out += ":imp=" + itoa(n.(int))
	}
	return out
}

func (f *fakeTranslator) sawKey(key string) bool {
	for _, k := range f.seen {
		if k == key {
			return true
		}
	}
	return false
}

// TestNoopTranslator_BuiltinBundle verifies the English fallback resolves every
// declared key to a non-empty, stable string.
func TestNoopTranslator_BuiltinBundle(t *testing.T) {
	n := NoopTranslator{}
	for key := range enBundle {
		got := n.Translate(key, nil)
		if got == "" || got == key {
			t.Fatalf("NoopTranslator returned no fallback for key %q: %q", key, got)
		}
	}
	// Unknown key falls back to the key itself (stable, non-empty).
	if got := n.Translate("benchmark.does_not_exist", nil); got != "benchmark.does_not_exist" {
		t.Fatalf("unknown key should fall back to itself, got %q", got)
	}
}

// TestExpand_Substitution proves the {name} substitution path in the English
// fallback renders count arguments.
func TestExpand_Substitution(t *testing.T) {
	got := expand("a {x} b {y}", map[string]interface{}{"x": 3, "y": "z"})
	if got != "a 3 b z" {
		t.Fatalf("expand mismatch: %q", got)
	}
}

// TestSetTranslator_RoutesBenchmarkMetadata is the PAIRED-MUTATION test for the
// CONST-046 migration of built-in benchmark Name/Description. With a fake
// translator installed, the runner's built-in benchmarks MUST carry the
// translator output — proving the literals were removed and routed through tr().
// If a future change reintroduces a hardcoded "SWE-Bench Lite" literal, this
// test fails (the metadata would not carry the locale prefix).
func TestSetTranslator_RoutesBenchmarkMetadata(t *testing.T) {
	ft := &fakeTranslator{locale: "xx"}
	SetTranslator(ft)
	defer SetTranslator(nil)

	runner := NewStandardBenchmarkRunner(nil, nil)
	benchmarks, err := runner.ListBenchmarks(context.Background())
	if err != nil {
		t.Fatalf("ListBenchmarks: %v", err)
	}

	wantKeys := map[string]string{
		"swe-bench-lite": "benchmark.swe_bench_lite",
		"humaneval":      "benchmark.humaneval",
		"mmlu-mini":      "benchmark.mmlu_mini",
		"gsm8k-mini":     "benchmark.gsm8k_mini",
	}
	for _, b := range benchmarks {
		prefix, ok := wantKeys[b.ID]
		if !ok {
			continue
		}
		if !strings.HasPrefix(b.Name, "xx:") {
			t.Fatalf("benchmark %s Name not routed through translator: %q", b.ID, b.Name)
		}
		if !strings.HasPrefix(b.Description, "xx:") {
			t.Fatalf("benchmark %s Description not routed through translator: %q", b.ID, b.Description)
		}
		if !ft.sawKey(prefix + ".name") {
			t.Fatalf("translator never asked for key %s.name", prefix)
		}
		if !ft.sawKey(prefix + ".desc") {
			t.Fatalf("translator never asked for key %s.desc", prefix)
		}
	}
}

// TestSetTranslator_RoutesCompareSummary is the PAIRED-MUTATION test for the
// CONST-046 migration of RunComparison.Summary. It drives CompareRuns down each
// of the three summary branches and asserts every branch carries the
// translator-routed key (with substituted counts) — not a hardcoded English
// sentence.
func TestSetTranslator_RoutesCompareSummary(t *testing.T) {
	ft := &fakeTranslator{locale: "xx"}
	SetTranslator(ft)
	defer SetTranslator(nil)

	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	mkRun := func(name string, results []*BenchmarkResult) string {
		run := &BenchmarkRun{Name: name, BenchmarkType: BenchmarkTypeCustom}
		if err := r.CreateRun(ctx, run); err != nil {
			t.Fatalf("CreateRun: %v", err)
		}
		run.Results = results
		run.Summary = &BenchmarkSummary{PassRate: 0.5, TotalTasks: len(results)}
		r.mu.Lock()
		r.runs[run.ID] = run
		r.mu.Unlock()
		return run.ID
	}

	res := func(taskID string, passed bool) *BenchmarkResult {
		return &BenchmarkResult{TaskID: taskID, Passed: passed, CreatedAt: time.Now()}
	}

	// Regression branch: t1 passed in run1, failed in run2.
	r1 := mkRun("r1", []*BenchmarkResult{res("t1", true), res("t2", true)})
	r2 := mkRun("r2", []*BenchmarkResult{res("t1", false), res("t2", true)})
	cmp, err := r.CompareRuns(ctx, r1, r2)
	if err != nil {
		t.Fatalf("CompareRuns regression: %v", err)
	}
	if !strings.HasPrefix(cmp.Summary, "xx:compare.run2_regressed") {
		t.Fatalf("regression summary not translator-routed: %q", cmp.Summary)
	}
	if !strings.Contains(cmp.Summary, "reg=1") {
		t.Fatalf("regression count not substituted: %q", cmp.Summary)
	}

	// Improvement branch.
	r3 := mkRun("r3", []*BenchmarkResult{res("t1", false), res("t2", false)})
	r4 := mkRun("r4", []*BenchmarkResult{res("t1", true), res("t2", true)})
	cmp, err = r.CompareRuns(ctx, r3, r4)
	if err != nil {
		t.Fatalf("CompareRuns improvement: %v", err)
	}
	if !strings.HasPrefix(cmp.Summary, "xx:compare.run2_improved") {
		t.Fatalf("improvement summary not translator-routed: %q", cmp.Summary)
	}

	// No-difference branch.
	r5 := mkRun("r5", []*BenchmarkResult{res("t1", true)})
	r6 := mkRun("r6", []*BenchmarkResult{res("t1", true)})
	cmp, err = r.CompareRuns(ctx, r5, r6)
	if err != nil {
		t.Fatalf("CompareRuns no-diff: %v", err)
	}
	if cmp.Summary != "xx:compare.no_difference" {
		t.Fatalf("no-difference summary not translator-routed: %q", cmp.Summary)
	}
}

// TestSetTranslator_NilRestoresNoop verifies SetTranslator(nil) restores the
// English fallback so the library stays usable without an i18n backend.
func TestSetTranslator_NilRestoresNoop(t *testing.T) {
	SetTranslator(&fakeTranslator{locale: "yy"})
	SetTranslator(nil)
	if got := tr("benchmark.humaneval.name", nil); got != "HumanEval" {
		t.Fatalf("SetTranslator(nil) did not restore NoopTranslator: %q", got)
	}
}
