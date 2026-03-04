package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	benchpkg "digital.vasic.benchmark/benchmark"
)

// --- mock implementations for benchmarks ---

type benchLLMProvider struct{}

func (p *benchLLMProvider) Complete(_ context.Context, _, _ string) (string, int, error) {
	return "B", 10, nil
}

func (p *benchLLMProvider) GetName() string {
	return "bench-provider"
}

// --- Benchmarks ---

func BenchmarkRunner_ListBenchmarks(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.ListBenchmarks(ctx)
	}
}

func BenchmarkRunner_GetBenchmark(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.GetBenchmark(ctx, "mmlu-mini")
	}
}

func BenchmarkRunner_GetTasks(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.GetTasks(ctx, "swe-bench-lite", nil)
	}
}

func BenchmarkRunner_GetTasks_WithFilter(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	config := &benchpkg.BenchmarkConfig{
		Difficulties: []benchpkg.DifficultyLevel{benchpkg.DifficultyEasy},
		Tags:         []string{"go"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.GetTasks(ctx, "swe-bench-lite", config)
	}
}

func BenchmarkRunner_CreateRun(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		run := &benchpkg.BenchmarkRun{
			BenchmarkType: benchpkg.BenchmarkTypeMMLU,
			ProviderName:  fmt.Sprintf("provider-%d", i),
			Config:        benchpkg.DefaultBenchmarkConfig(),
		}
		_ = runner.CreateRun(ctx, run)
	}
}

func BenchmarkRunner_ListRuns(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	// Pre-create runs
	for i := 0; i < 100; i++ {
		run := &benchpkg.BenchmarkRun{
			BenchmarkType: benchpkg.BenchmarkTypeMMLU,
			ProviderName:  fmt.Sprintf("provider-%d", i),
		}
		_ = runner.CreateRun(ctx, run)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.ListRuns(ctx, nil)
	}
}

func BenchmarkRunner_ListRuns_WithFilter(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		bType := benchpkg.BenchmarkTypeMMLU
		if i%2 == 0 {
			bType = benchpkg.BenchmarkTypeHumanEval
		}
		run := &benchpkg.BenchmarkRun{
			BenchmarkType: bType,
			ProviderName:  fmt.Sprintf("provider-%d", i),
		}
		_ = runner.CreateRun(ctx, run)
	}

	filter := &benchpkg.RunFilter{
		BenchmarkType: benchpkg.BenchmarkTypeMMLU,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.ListRuns(ctx, filter)
	}
}

func BenchmarkRunner_AddBenchmark(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bench := &benchpkg.Benchmark{
			ID:        fmt.Sprintf("bench-%d", i),
			Type:      benchpkg.BenchmarkTypeCustom,
			Name:      fmt.Sprintf("Bench %d", i),
			Version:   "1.0.0",
			CreatedAt: time.Now(),
		}
		tasks := []*benchpkg.BenchmarkTask{
			{
				ID:     fmt.Sprintf("task-%d", i),
				Type:   benchpkg.BenchmarkTypeCustom,
				Prompt: "test",
			},
		}
		runner.AddBenchmark(bench, tasks)
	}
}

func BenchmarkBenchmarkSystem_Initialize(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := benchpkg.DefaultBenchmarkSystemConfig()
		system := benchpkg.NewBenchmarkSystem(config, nil)
		_ = system.Initialize(nil)
	}
}

func BenchmarkDefaultBenchmarkConfig(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchpkg.DefaultBenchmarkConfig()
	}
}

func BenchmarkDefaultBenchmarkSystemConfig(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchpkg.DefaultBenchmarkSystemConfig()
	}
}

func BenchmarkRunner_CancelRun(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	runner := benchpkg.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	// Pre-create runs
	runIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		run := &benchpkg.BenchmarkRun{
			BenchmarkType: benchpkg.BenchmarkTypeMMLU,
			ProviderName:  fmt.Sprintf("provider-%d", i),
		}
		_ = runner.CreateRun(ctx, run)
		runIDs[i] = run.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runner.CancelRun(ctx, runIDs[i])
	}
}

func BenchmarkVerifierAdapter_SelectBestProvider(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	service := &benchVerifierService{
		scores:   map[string]float64{"openai": 9.0, "anthropic": 8.5},
		healthy:  map[string]bool{"openai": true, "anthropic": true},
		topProvs: []string{"openai", "anthropic"},
	}
	adapter := benchpkg.NewVerifierAdapterForBenchmark(service, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.SelectBestProvider()
	}
}

type benchVerifierService struct {
	scores   map[string]float64
	healthy  map[string]bool
	topProvs []string
}

func (s *benchVerifierService) GetProviderScore(name string) float64 {
	if sc, ok := s.scores[name]; ok {
		return sc
	}
	return 0
}

func (s *benchVerifierService) IsProviderHealthy(name string) bool {
	if h, ok := s.healthy[name]; ok {
		return h
	}
	return false
}

func (s *benchVerifierService) GetTopProviders(count int) []string {
	if count >= len(s.topProvs) {
		return s.topProvs
	}
	return s.topProvs[:count]
}
