package integration

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

type mockLLMProvider struct {
	response string
	tokens   int
}

func (m *mockLLMProvider) Complete(_ context.Context, prompt, _ string) (string, int, error) {
	if m.response != "" {
		return m.response, m.tokens, nil
	}
	return "B", 10, nil
}

func (m *mockLLMProvider) GetName() string {
	return "mock-provider"
}

type mockDebateService struct {
	confidence float64
}

func (m *mockDebateService) RunDebate(_ context.Context, topic string) (*benchmark.DebateResultForBenchmark, error) {
	return &benchmark.DebateResultForBenchmark{
		ID:         "debate-1",
		Consensus:  `{"score": 0.85, "passed": true}`,
		Confidence: m.confidence,
		Votes:      map[string]float64{"provider-a": 0.8, "provider-b": 0.9},
	}, nil
}

type mockVerifierService struct {
	scores   map[string]float64
	healthy  map[string]bool
	topProvs []string
}

func (m *mockVerifierService) GetProviderScore(name string) float64 {
	if s, ok := m.scores[name]; ok {
		return s
	}
	return 0
}

func (m *mockVerifierService) IsProviderHealthy(name string) bool {
	if h, ok := m.healthy[name]; ok {
		return h
	}
	return false
}

func (m *mockVerifierService) GetTopProviders(count int) []string {
	if count >= len(m.topProvs) {
		return m.topProvs
	}
	return m.topProvs[:count]
}

// --- Integration Tests ---

func TestRunner_ListBenchmarks_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	benchmarks, err := runner.ListBenchmarks(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(benchmarks), 4, "should have at least 4 built-in benchmarks")

	names := make(map[string]bool)
	for _, b := range benchmarks {
		names[b.Name] = true
		assert.NotEmpty(t, b.ID)
		assert.NotEmpty(t, b.Type)
		assert.NotEmpty(t, b.Version)
		assert.Greater(t, b.TaskCount, 0)
	}
}

func TestRunner_GetBenchmark_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	b, err := runner.GetBenchmark(ctx, "humaneval")
	require.NoError(t, err)
	assert.Equal(t, "humaneval", b.ID)
	assert.Equal(t, benchmark.BenchmarkTypeHumanEval, b.Type)
	assert.Greater(t, b.TaskCount, 0)
}

func TestRunner_GetBenchmark_NotFound_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	_, err := runner.GetBenchmark(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "benchmark not found")
}

func TestRunner_GetTasks_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	tasks, err := runner.GetTasks(ctx, "swe-bench-lite", nil)
	require.NoError(t, err)
	assert.Greater(t, len(tasks), 0)

	for _, task := range tasks {
		assert.NotEmpty(t, task.ID)
		assert.NotEmpty(t, task.Prompt)
		assert.Equal(t, benchmark.BenchmarkTypeSWEBench, task.Type)
	}
}

func TestRunner_GetTasks_WithDifficultyFilter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	config := &benchmark.BenchmarkConfig{
		Difficulties: []benchmark.DifficultyLevel{benchmark.DifficultyEasy},
	}

	tasks, err := runner.GetTasks(ctx, "swe-bench-lite", config)
	require.NoError(t, err)

	for _, task := range tasks {
		assert.Equal(t, benchmark.DifficultyEasy, task.Difficulty)
	}
}

func TestRunner_GetTasks_WithTagFilter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	config := &benchmark.BenchmarkConfig{
		Tags: []string{"go"},
	}

	tasks, err := runner.GetTasks(ctx, "swe-bench-lite", config)
	require.NoError(t, err)
	assert.Greater(t, len(tasks), 0)

	for _, task := range tasks {
		found := false
		for _, tag := range task.Tags {
			if tag == "go" {
				found = true
				break
			}
		}
		assert.True(t, found, "task %s should have 'go' tag", task.ID)
	}
}

func TestRunner_GetTasks_WithMaxTasks_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	config := &benchmark.BenchmarkConfig{
		MaxTasks: 1,
	}

	tasks, err := runner.GetTasks(ctx, "swe-bench-lite", config)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(tasks), 1)
}

func TestRunner_CreateAndStartRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	provider := &mockLLMProvider{response: "B", tokens: 15}
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	run := &benchmark.BenchmarkRun{
		BenchmarkType: benchmark.BenchmarkTypeMMLU,
		ProviderName:  "mock",
		ModelName:     "mock-model",
		Config:        benchmark.DefaultBenchmarkConfig(),
	}

	err := runner.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID)
	assert.Equal(t, benchmark.BenchmarkStatusPending, run.Status)

	err = runner.StartRun(ctx, run.ID)
	require.NoError(t, err)

	// Wait for async completion
	time.Sleep(2 * time.Second)

	fetched, err := runner.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, benchmark.BenchmarkStatusCompleted, fetched.Status)
	assert.NotNil(t, fetched.Summary)
}

func TestRunner_ListRuns_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		run := &benchmark.BenchmarkRun{
			BenchmarkType: benchmark.BenchmarkTypeMMLU,
			ProviderName:  fmt.Sprintf("provider-%d", i),
		}
		err := runner.CreateRun(ctx, run)
		require.NoError(t, err)
	}

	runs, err := runner.ListRuns(ctx, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(runs), 3)
}

func TestRunner_CancelRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	run := &benchmark.BenchmarkRun{
		BenchmarkType: benchmark.BenchmarkTypeHumanEval,
		ProviderName:  "test",
	}
	err := runner.CreateRun(ctx, run)
	require.NoError(t, err)

	err = runner.CancelRun(ctx, run.ID)
	require.NoError(t, err)

	fetched, err := runner.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, benchmark.BenchmarkStatusCancelled, fetched.Status)
}

func TestRunner_AddCustomBenchmark_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	customBench := &benchmark.Benchmark{
		ID:          "custom-test",
		Type:        benchmark.BenchmarkTypeCustom,
		Name:        "Custom Integration Test",
		Description: "A custom benchmark for testing",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}

	customTasks := []*benchmark.BenchmarkTask{
		{
			ID:         "custom-001",
			Type:       benchmark.BenchmarkTypeCustom,
			Name:       "Custom task",
			Prompt:     "What is 2+2?",
			Expected:   "4",
			Difficulty: benchmark.DifficultyEasy,
			Tags:       []string{"math"},
		},
	}

	runner.AddBenchmark(customBench, customTasks)

	b, err := runner.GetBenchmark(ctx, "custom-test")
	require.NoError(t, err)
	assert.Equal(t, "Custom Integration Test", b.Name)
	assert.Equal(t, 1, b.TaskCount)

	tasks, err := runner.GetTasks(ctx, "custom-test", nil)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestBenchmarkSystem_Initialize_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := benchmark.DefaultBenchmarkSystemConfig()
	system := benchmark.NewBenchmarkSystem(config, nil)
	require.NotNil(t, system)

	err := system.Initialize(nil)
	require.NoError(t, err)

	runner := system.GetRunner()
	require.NotNil(t, runner)

	benchmarks, err := runner.ListBenchmarks(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(benchmarks), 4)
}

func TestBenchmarkSystem_WithDebateService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := benchmark.DefaultBenchmarkSystemConfig()
	config.EnableDebateEvaluation = true
	system := benchmark.NewBenchmarkSystem(config, nil)

	debateService := &mockDebateService{confidence: 0.85}
	system.SetDebateService(debateService)

	err := system.Initialize(nil)
	require.NoError(t, err)
	assert.NotNil(t, system.GetRunner())
}

func TestBenchmarkSystem_WithVerifierService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := benchmark.DefaultBenchmarkSystemConfig()
	system := benchmark.NewBenchmarkSystem(config, nil)

	verifierService := &mockVerifierService{
		scores:   map[string]float64{"openai": 9.0, "anthropic": 8.5},
		healthy:  map[string]bool{"openai": true, "anthropic": true},
		topProvs: []string{"openai", "anthropic"},
	}
	system.SetVerifierService(verifierService)

	err := system.Initialize(nil)
	require.NoError(t, err)
}

func TestVerifierAdapter_SelectBestProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	service := &mockVerifierService{
		scores:   map[string]float64{"openai": 9.0, "anthropic": 8.5, "deepseek": 7.0},
		healthy:  map[string]bool{"openai": true, "anthropic": true, "deepseek": false},
		topProvs: []string{"openai", "anthropic", "deepseek"},
	}

	adapter := benchmark.NewVerifierAdapterForBenchmark(service, nil)
	provider, score := adapter.SelectBestProvider()
	assert.Equal(t, "openai", provider)
	assert.Equal(t, 9.0, score)
}

func TestDebateAdapter_EvaluateResponse_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	debateService := &mockDebateService{confidence: 0.85}
	adapter := benchmark.NewDebateAdapterForBenchmark(debateService, nil)

	task := &benchmark.BenchmarkTask{
		ID:       "test-task",
		Name:     "Test task",
		Prompt:   "What is 2+2?",
		Expected: "4",
	}

	score, passed, err := adapter.EvaluateResponse(context.Background(), task, "4")
	require.NoError(t, err)
	assert.True(t, passed)
	assert.Equal(t, 0.85, score)
}
