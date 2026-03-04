package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.benchmark/benchmark"
)

// --- mock implementations ---

type e2eLLMProvider struct {
	answers map[string]string
}

func newE2EProvider() *e2eLLMProvider {
	return &e2eLLMProvider{
		answers: map[string]string{
			"mmlu":      "B",
			"gsm8k":     "18",
			"humaneval": "def has_close_elements(numbers, threshold):\n  for i in range(len(numbers)):\n    for j in range(i+1, len(numbers)):\n      if abs(numbers[i] - numbers[j]) < threshold:\n        return True\n  return False",
			"swe":       "func GetUserName(user *User) string {\n    if user == nil {\n        return \"\"\n    }\n    return user.Name\n}",
		},
	}
}

func (p *e2eLLMProvider) Complete(_ context.Context, prompt, _ string) (string, int, error) {
	// Simple response selection based on prompt content
	if len(prompt) > 20 {
		for key, answer := range p.answers {
			if containsSubstring(prompt, key) {
				return answer, 25, nil
			}
		}
	}
	return "B", 10, nil
}

func (p *e2eLLMProvider) GetName() string {
	return "e2e-provider"
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

type e2eDebateService struct{}

func (s *e2eDebateService) RunDebate(_ context.Context, _ string) (*benchmark.DebateResultForBenchmark, error) {
	return &benchmark.DebateResultForBenchmark{
		ID:         "e2e-debate",
		Consensus:  `{"score": 0.9, "passed": true}`,
		Confidence: 0.9,
	}, nil
}

type e2eVerifierService struct{}

func (s *e2eVerifierService) GetProviderScore(name string) float64 {
	scores := map[string]float64{
		"e2e-provider": 9.0,
		"backup":       7.5,
	}
	if sc, ok := scores[name]; ok {
		return sc
	}
	return 0
}

func (s *e2eVerifierService) IsProviderHealthy(name string) bool {
	return name == "e2e-provider"
}

func (s *e2eVerifierService) GetTopProviders(count int) []string {
	return []string{"e2e-provider", "backup"}
}

// --- E2E Tests ---

func TestFullBenchmarkWorkflow_MMLU_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	provider := newE2EProvider()
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	// Step 1: List available benchmarks
	benchmarks, err := runner.ListBenchmarks(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(benchmarks), 4)

	// Step 2: Get MMLU benchmark
	mmlu, err := runner.GetBenchmark(ctx, "mmlu-mini")
	require.NoError(t, err)
	assert.Equal(t, "MMLU Mini", mmlu.Name)

	// Step 3: Get tasks
	tasks, err := runner.GetTasks(ctx, "mmlu-mini", nil)
	require.NoError(t, err)
	assert.Greater(t, len(tasks), 0)

	// Step 4: Create and start a run
	run := &benchmark.BenchmarkRun{
		Name:          "E2E MMLU Test",
		BenchmarkType: benchmark.BenchmarkTypeMMLU,
		ProviderName:  "e2e-provider",
		ModelName:     "e2e-model",
		Config:        benchmark.DefaultBenchmarkConfig(),
	}

	err = runner.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID)

	err = runner.StartRun(ctx, run.ID)
	require.NoError(t, err)

	// Step 5: Wait for completion
	time.Sleep(3 * time.Second)

	completed, err := runner.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, benchmark.BenchmarkStatusCompleted, completed.Status)
	assert.NotNil(t, completed.Summary)
	assert.Greater(t, completed.Summary.TotalTasks, 0)
}

func TestFullBenchmarkWorkflow_SWEBench_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	provider := newE2EProvider()
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	run := &benchmark.BenchmarkRun{
		Name:          "E2E SWE-Bench Test",
		BenchmarkType: benchmark.BenchmarkTypeSWEBench,
		ProviderName:  "e2e-provider",
		ModelName:     "e2e-model",
		Config:        benchmark.DefaultBenchmarkConfig(),
	}

	err := runner.CreateRun(ctx, run)
	require.NoError(t, err)

	err = runner.StartRun(ctx, run.ID)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	completed, err := runner.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, benchmark.BenchmarkStatusCompleted, completed.Status)
	assert.NotNil(t, completed.Summary)
}

func TestFullBenchmarkWorkflow_CompareRuns_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	provider := newE2EProvider()
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	// Run 1
	run1 := &benchmark.BenchmarkRun{
		BenchmarkType: benchmark.BenchmarkTypeMMLU,
		ProviderName:  "provider-a",
		Config:        benchmark.DefaultBenchmarkConfig(),
	}
	err := runner.CreateRun(ctx, run1)
	require.NoError(t, err)
	err = runner.StartRun(ctx, run1.ID)
	require.NoError(t, err)

	// Run 2
	run2 := &benchmark.BenchmarkRun{
		BenchmarkType: benchmark.BenchmarkTypeMMLU,
		ProviderName:  "provider-b",
		Config:        benchmark.DefaultBenchmarkConfig(),
	}
	err = runner.CreateRun(ctx, run2)
	require.NoError(t, err)
	err = runner.StartRun(ctx, run2.ID)
	require.NoError(t, err)

	// Wait for both
	time.Sleep(4 * time.Second)

	comparison, err := runner.CompareRuns(ctx, run1.ID, run2.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, comparison.Summary)
	assert.Equal(t, run1.ID, comparison.Run1ID)
	assert.Equal(t, run2.ID, comparison.Run2ID)
}

func TestFullBenchmarkWorkflow_CustomBenchmark_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	provider := &e2eLLMProvider{
		answers: map[string]string{},
	}
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	// Add custom benchmark
	customBench := &benchmark.Benchmark{
		ID:        "e2e-custom",
		Type:      benchmark.BenchmarkTypeCustom,
		Name:      "E2E Custom Benchmark",
		Version:   "1.0.0",
		CreatedAt: time.Now(),
	}

	customTasks := []*benchmark.BenchmarkTask{
		{
			ID:         "e2e-c1",
			Type:       benchmark.BenchmarkTypeCustom,
			Name:       "Simple question",
			Prompt:     "What is the capital of France?",
			Expected:   "paris",
			Difficulty: benchmark.DifficultyEasy,
			Tags:       []string{"geography"},
		},
		{
			ID:         "e2e-c2",
			Type:       benchmark.BenchmarkTypeCustom,
			Name:       "Math question",
			Prompt:     "What is 10*10?",
			Expected:   "100",
			Difficulty: benchmark.DifficultyEasy,
			Tags:       []string{"math"},
		},
	}

	runner.AddBenchmark(customBench, customTasks)

	tasks, err := runner.GetTasks(ctx, "e2e-custom", nil)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	run := &benchmark.BenchmarkRun{
		BenchmarkType: benchmark.BenchmarkTypeCustom,
		ProviderName:  "e2e-provider",
		Config:        benchmark.DefaultBenchmarkConfig(),
	}

	err = runner.CreateRun(ctx, run)
	require.NoError(t, err)

	err = runner.StartRun(ctx, run.ID)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	completed, err := runner.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, benchmark.BenchmarkStatusCompleted, completed.Status)
}

func TestBenchmarkSystem_FullWorkflow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	config := benchmark.DefaultBenchmarkSystemConfig()
	config.EnableDebateEvaluation = false
	system := benchmark.NewBenchmarkSystem(config, nil)

	provider := benchmark.NewProviderAdapterForBenchmark(nil, "test", "test-model", nil)
	err := system.Initialize(provider)
	require.NoError(t, err)

	runner := system.GetRunner()
	require.NotNil(t, runner)

	benchmarks, err := runner.ListBenchmarks(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(benchmarks), 4)
}

func TestBenchmarkSystem_WithVerifier_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	config := benchmark.DefaultBenchmarkSystemConfig()
	config.AutoSelectProvider = true
	system := benchmark.NewBenchmarkSystem(config, nil)

	system.SetVerifierService(&e2eVerifierService{})
	err := system.Initialize(nil)
	require.NoError(t, err)

	ctx := context.Background()
	run, err := system.RunBenchmarkWithBestProvider(ctx, benchmark.BenchmarkTypeMMLU, nil)
	require.NoError(t, err)
	assert.NotNil(t, run)
	assert.Equal(t, "e2e-provider", run.ProviderName)
}

func TestBenchmarkSystem_GenerateLeaderboard_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	provider := newE2EProvider()
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	// Create multiple runs with different providers
	for i := 0; i < 3; i++ {
		run := &benchmark.BenchmarkRun{
			BenchmarkType: benchmark.BenchmarkTypeMMLU,
			ProviderName:  fmt.Sprintf("provider-%d", i),
			ModelName:     fmt.Sprintf("model-%d", i),
			Config:        benchmark.DefaultBenchmarkConfig(),
		}
		err := runner.CreateRun(ctx, run)
		require.NoError(t, err)
		err = runner.StartRun(ctx, run.ID)
		require.NoError(t, err)
	}

	time.Sleep(5 * time.Second)

	config := benchmark.DefaultBenchmarkSystemConfig()
	system := benchmark.NewBenchmarkSystem(config, nil)
	system.Initialize(nil)

	// Use the runner we already have
	leaderboard, err := system.GenerateLeaderboard(ctx, benchmark.BenchmarkTypeMMLU)
	require.NoError(t, err)
	assert.NotNil(t, leaderboard)
	assert.Equal(t, benchmark.BenchmarkTypeMMLU, leaderboard.BenchmarkType)
}

func TestRunner_ListRuns_WithFilter_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		bType := benchmark.BenchmarkTypeMMLU
		if i%2 == 0 {
			bType = benchmark.BenchmarkTypeHumanEval
		}
		run := &benchmark.BenchmarkRun{
			BenchmarkType: bType,
			ProviderName:  fmt.Sprintf("provider-%d", i),
		}
		err := runner.CreateRun(ctx, run)
		require.NoError(t, err)
	}

	filter := &benchmark.RunFilter{
		BenchmarkType: benchmark.BenchmarkTypeMMLU,
	}

	runs, err := runner.ListRuns(ctx, filter)
	require.NoError(t, err)

	for _, run := range runs {
		assert.Equal(t, benchmark.BenchmarkTypeMMLU, run.BenchmarkType)
	}
}

func TestDefaultBenchmarkConfig_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	config := benchmark.DefaultBenchmarkConfig()
	assert.Greater(t, config.Timeout, time.Duration(0))
	assert.Greater(t, config.Concurrency, 0)
	assert.Greater(t, config.MaxTokens, 0)
	assert.True(t, config.SaveResponses)
}
