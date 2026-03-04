package benchmark

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	runtime.GOMAXPROCS(2)
}

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockLLMProvider struct {
	name      string
	response  string
	tokens    int
	err       error
	mu        sync.Mutex
	callCount int
	delay     time.Duration
}

func (m *mockLLMProvider) Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", 0, ctx.Err()
		}
	}
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return m.response, m.tokens, m.err
}

func (m *mockLLMProvider) GetName() string { return m.name }

func (m *mockLLMProvider) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

type mockCodeExecutor struct {
	results []*TestCaseResult
	err     error
}

func (m *mockCodeExecutor) Execute(_ context.Context, _, _, _ string) (string, error) {
	return "executed", nil
}

func (m *mockCodeExecutor) Validate(_ context.Context, _, _ string, _ []*TestCase) ([]*TestCaseResult, error) {
	return m.results, m.err
}

type mockDebateEvaluator struct {
	score  float64
	passed bool
	err    error
}

func (m *mockDebateEvaluator) EvaluateResponse(_ context.Context, _ *BenchmarkTask, _ string) (float64, bool, error) {
	return m.score, m.passed, m.err
}

// helper: create a runner with a simple mock provider returning the given response
func newTestRunner(response string, tokens int) *StandardBenchmarkRunner {
	p := &mockLLMProvider{name: "test-provider", response: response, tokens: tokens}
	return NewStandardBenchmarkRunner(p, nil)
}

// ---------------------------------------------------------------------------
// NewStandardBenchmarkRunner
// ---------------------------------------------------------------------------

func TestNewStandardBenchmarkRunner_NilLogger(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	require.NotNil(t, r)
	assert.NotNil(t, r.logger, "should create default logger when nil")
}

func TestNewStandardBenchmarkRunner_NilProvider(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	require.NotNil(t, r)
	assert.Nil(t, r.provider)
}

func TestNewStandardBenchmarkRunner_InitializesBuiltInBenchmarks(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	assert.Len(t, r.benchmarks, 4, "should have 4 built-in benchmarks")
	assert.Contains(t, r.benchmarks, "swe-bench-lite")
	assert.Contains(t, r.benchmarks, "humaneval")
	assert.Contains(t, r.benchmarks, "mmlu-mini")
	assert.Contains(t, r.benchmarks, "gsm8k-mini")
}

func TestNewStandardBenchmarkRunner_TaskCounts(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	assert.Equal(t, 3, r.benchmarks["swe-bench-lite"].TaskCount)
	assert.Equal(t, 2, r.benchmarks["humaneval"].TaskCount)
	assert.Equal(t, 3, r.benchmarks["mmlu-mini"].TaskCount)
	assert.Equal(t, 2, r.benchmarks["gsm8k-mini"].TaskCount)
}

// ---------------------------------------------------------------------------
// SetCodeExecutor / SetDebateEvaluator
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_SetCodeExecutor(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	assert.Nil(t, r.codeExecutor)
	exec := &mockCodeExecutor{}
	r.SetCodeExecutor(exec)
	assert.NotNil(t, r.codeExecutor)
}

func TestStandardBenchmarkRunner_SetDebateEvaluator(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	assert.Nil(t, r.debateEval)
	eval := &mockDebateEvaluator{score: 0.9, passed: true}
	r.SetDebateEvaluator(eval)
	assert.NotNil(t, r.debateEval)
}

// ---------------------------------------------------------------------------
// ListBenchmarks
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_ListBenchmarks_ReturnsAll(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	benchmarks, err := r.ListBenchmarks(ctx)
	require.NoError(t, err)
	assert.Len(t, benchmarks, 4)
}

func TestStandardBenchmarkRunner_ListBenchmarks_SortedByName(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	benchmarks, err := r.ListBenchmarks(ctx)
	require.NoError(t, err)
	for i := 1; i < len(benchmarks); i++ {
		assert.LessOrEqual(t, benchmarks[i-1].Name, benchmarks[i].Name,
			"benchmarks should be sorted by Name")
	}
}

func TestStandardBenchmarkRunner_ListBenchmarks_EmptyAfterClear(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	// Clear all built-ins
	r.mu.Lock()
	r.benchmarks = make(map[string]*Benchmark)
	r.mu.Unlock()

	ctx := context.Background()
	benchmarks, err := r.ListBenchmarks(ctx)
	require.NoError(t, err)
	assert.Empty(t, benchmarks)
}

// ---------------------------------------------------------------------------
// GetBenchmark
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_GetBenchmark_Exists(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	ids := []string{"swe-bench-lite", "humaneval", "mmlu-mini", "gsm8k-mini"}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			b, err := r.GetBenchmark(ctx, id)
			require.NoError(t, err)
			require.NotNil(t, b)
			assert.Equal(t, id, b.ID)
		})
	}
}

func TestStandardBenchmarkRunner_GetBenchmark_NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	b, err := r.GetBenchmark(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, b)
	assert.Contains(t, err.Error(), "benchmark not found")
}

func TestStandardBenchmarkRunner_GetBenchmark_EmptyID(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	b, err := r.GetBenchmark(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, b)
}

// ---------------------------------------------------------------------------
// GetTasks
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_GetTasks_AllBenchmarks(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	tests := []struct {
		benchmarkID   string
		expectedCount int
	}{
		{"swe-bench-lite", 3},
		{"humaneval", 2},
		{"mmlu-mini", 3},
		{"gsm8k-mini", 2},
	}
	for _, tt := range tests {
		t.Run(tt.benchmarkID, func(t *testing.T) {
			tasks, err := r.GetTasks(ctx, tt.benchmarkID, nil)
			require.NoError(t, err)
			assert.Len(t, tasks, tt.expectedCount)
		})
	}
}

func TestStandardBenchmarkRunner_GetTasks_NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	tasks, err := r.GetTasks(ctx, "nonexistent", nil)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "benchmark not found")
}

func TestStandardBenchmarkRunner_GetTasks_FilterByDifficulty(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	cfg := &BenchmarkConfig{Difficulties: []DifficultyLevel{DifficultyEasy}}
	tasks, err := r.GetTasks(ctx, "swe-bench-lite", cfg)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, DifficultyEasy, tasks[0].Difficulty)
}

func TestStandardBenchmarkRunner_GetTasks_FilterByDifficulty_NoMatch(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	// gsm8k-mini has Easy and Medium; filter Hard should return 0
	cfg := &BenchmarkConfig{Difficulties: []DifficultyLevel{DifficultyHard}}
	tasks, err := r.GetTasks(ctx, "gsm8k-mini", cfg)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestStandardBenchmarkRunner_GetTasks_FilterByTags(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	cfg := &BenchmarkConfig{Tags: []string{"python"}}
	tasks, err := r.GetTasks(ctx, "humaneval", cfg)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestStandardBenchmarkRunner_GetTasks_FilterByTags_NoMatch(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	cfg := &BenchmarkConfig{Tags: []string{"nonexistent-tag"}}
	tasks, err := r.GetTasks(ctx, "humaneval", cfg)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestStandardBenchmarkRunner_GetTasks_MaxTasksLimit(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	cfg := &BenchmarkConfig{MaxTasks: 1}
	tasks, err := r.GetTasks(ctx, "swe-bench-lite", cfg)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestStandardBenchmarkRunner_GetTasks_MaxTasksZero_NoLimit(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	cfg := &BenchmarkConfig{MaxTasks: 0}
	tasks, err := r.GetTasks(ctx, "swe-bench-lite", cfg)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestStandardBenchmarkRunner_GetTasks_MaxTasksExceedsTotal(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	cfg := &BenchmarkConfig{MaxTasks: 100}
	tasks, err := r.GetTasks(ctx, "swe-bench-lite", cfg)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestStandardBenchmarkRunner_GetTasks_CombinedFilters(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	// SWE-bench: Easy has tags [bug-fix, go, null-safety]
	cfg := &BenchmarkConfig{
		Difficulties: []DifficultyLevel{DifficultyEasy},
		Tags:         []string{"go"},
	}
	tasks, err := r.GetTasks(ctx, "swe-bench-lite", cfg)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "swe-001", tasks[0].ID)
}

func TestStandardBenchmarkRunner_GetTasks_NilConfig(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	tasks, err := r.GetTasks(ctx, "mmlu-mini", nil)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

// ---------------------------------------------------------------------------
// Built-in task content verification
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_SWEBenchTasks_Content(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	tasks, err := r.GetTasks(ctx, "swe-bench-lite", nil)
	require.NoError(t, err)
	require.Len(t, tasks, 3)

	assert.Equal(t, "swe-001", tasks[0].ID)
	assert.Equal(t, DifficultyEasy, tasks[0].Difficulty)
	assert.Contains(t, tasks[0].Tags, "bug-fix")

	assert.Equal(t, "swe-002", tasks[1].ID)
	assert.Equal(t, DifficultyMedium, tasks[1].Difficulty)

	assert.Equal(t, "swe-003", tasks[2].ID)
	assert.Equal(t, DifficultyHard, tasks[2].Difficulty)
}

func TestStandardBenchmarkRunner_HumanEvalTasks_HaveTestCases(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	tasks, err := r.GetTasks(ctx, "humaneval", nil)
	require.NoError(t, err)
	for _, task := range tasks {
		assert.NotEmpty(t, task.TestCases, "HumanEval tasks should have test cases")
	}
	assert.Len(t, tasks[0].TestCases, 3)
	assert.Len(t, tasks[1].TestCases, 2)
}

func TestStandardBenchmarkRunner_MMLUTasks_HaveExpected(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	tasks, err := r.GetTasks(ctx, "mmlu-mini", nil)
	require.NoError(t, err)

	expectedAnswers := []string{"B", "A", "C"}
	for i, task := range tasks {
		assert.Equal(t, expectedAnswers[i], task.Expected)
	}
}

func TestStandardBenchmarkRunner_GSM8KTasks_HaveExpected(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	tasks, err := r.GetTasks(ctx, "gsm8k-mini", nil)
	require.NoError(t, err)

	assert.Equal(t, "18", tasks[0].Expected)
	assert.Equal(t, "5800", tasks[1].Expected)
}

// ---------------------------------------------------------------------------
// CreateRun
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_CreateRun_AssignsID(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Name: "test"}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID, "should auto-assign UUID when ID is empty")
}

func TestStandardBenchmarkRunner_CreateRun_PreservesID(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{ID: "my-run-id", Name: "test"}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.Equal(t, "my-run-id", run.ID)
}

func TestStandardBenchmarkRunner_CreateRun_DefaultConfig(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Name: "test"}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)
	require.NotNil(t, run.Config)
	assert.Equal(t, 4, run.Config.Concurrency)
}

func TestStandardBenchmarkRunner_CreateRun_CustomConfig(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	cfg := &BenchmarkConfig{Concurrency: 8, MaxTokens: 2048}
	run := &BenchmarkRun{Name: "test", Config: cfg}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.Equal(t, 8, run.Config.Concurrency)
}

func TestStandardBenchmarkRunner_CreateRun_SetsPending(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Name: "test", Status: BenchmarkStatusRunning}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.Equal(t, BenchmarkStatusPending, run.Status, "CreateRun should force Pending status")
}

func TestStandardBenchmarkRunner_CreateRun_SetsCreatedAt(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	before := time.Now()
	run := &BenchmarkRun{Name: "test"}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)
	assert.False(t, run.CreatedAt.IsZero())
	assert.True(t, run.CreatedAt.After(before) || run.CreatedAt.Equal(before))
}

func TestStandardBenchmarkRunner_CreateRun_StoresInMap(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{ID: "stored-run", Name: "test"}
	err := r.CreateRun(ctx, run)
	require.NoError(t, err)

	got, err := r.GetRun(ctx, "stored-run")
	require.NoError(t, err)
	assert.Equal(t, "stored-run", got.ID)
}

// ---------------------------------------------------------------------------
// StartRun
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_StartRun_NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	err := r.StartRun(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

func TestStandardBenchmarkRunner_StartRun_AlreadyRunning(t *testing.T) {
	r := newTestRunner("answer", 10)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
	}
	require.NoError(t, r.CreateRun(ctx, run))

	// Force status to running
	r.mu.Lock()
	run.Status = BenchmarkStatusRunning
	r.mu.Unlock()

	err := r.StartRun(ctx, "run-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started or completed")
}

func TestStandardBenchmarkRunner_StartRun_AlreadyCompleted(t *testing.T) {
	r := newTestRunner("answer", 10)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
	}
	require.NoError(t, r.CreateRun(ctx, run))

	r.mu.Lock()
	run.Status = BenchmarkStatusCompleted
	r.mu.Unlock()

	err := r.StartRun(ctx, "run-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started or completed")
}

func TestStandardBenchmarkRunner_StartRun_SetsRunning(t *testing.T) {
	r := newTestRunner("B", 10)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "run-1"))

	// Give the goroutine a moment to start
	time.Sleep(50 * time.Millisecond)

	r.mu.RLock()
	status := run.Status
	startTime := run.StartTime
	r.mu.RUnlock()

	// Status should be running or completed by now
	assert.True(t, status == BenchmarkStatusRunning || status == BenchmarkStatusCompleted)
	assert.NotNil(t, startTime)
}

func TestStandardBenchmarkRunner_StartRun_CompletesWithResults(t *testing.T) {
	r := newTestRunner("B", 10)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
		Config:        &BenchmarkConfig{Concurrency: 2, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "run-1"))

	// Wait for async execution
	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	assert.Equal(t, BenchmarkStatusCompleted, run.Status)
	assert.NotNil(t, run.EndTime)
	assert.NotNil(t, run.Summary)
	assert.NotEmpty(t, run.Results)
	assert.Equal(t, 3, run.Summary.TotalTasks) // MMLU has 3 tasks
}

// ---------------------------------------------------------------------------
// executeRun / executeTask (tested indirectly via StartRun)
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_ExecuteRun_ConcurrentExecution(t *testing.T) {
	provider := &mockLLMProvider{name: "slow", response: "B", tokens: 5, delay: 50 * time.Millisecond}
	r := NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "conc-run",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "slow",
		ModelName:     "slow-model",
		Config:        &BenchmarkConfig{Concurrency: 3, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))

	start := time.Now()
	require.NoError(t, r.StartRun(ctx, "conc-run"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	elapsed := time.Since(start)
	// 3 tasks with 50ms each, concurrency=3: should finish in ~50-100ms, not 150ms
	assert.Less(t, elapsed, 500*time.Millisecond, "concurrent execution should be faster than serial")
	assert.Equal(t, 3, provider.getCallCount())
}

func TestStandardBenchmarkRunner_ExecuteTask_NilProvider(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "nil-prov",
		BenchmarkType: BenchmarkTypeGSM8K,
		ProviderName:  "none",
		ModelName:     "none",
		Config:        &BenchmarkConfig{Concurrency: 1, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "nil-prov"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, result := range run.Results {
		assert.Equal(t, "no provider available", result.Response)
	}
}

func TestStandardBenchmarkRunner_ExecuteTask_ProviderError(t *testing.T) {
	provider := &mockLLMProvider{
		name: "failing",
		err:  errors.New("provider unavailable"),
	}
	r := NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "err-run",
		BenchmarkType: BenchmarkTypeGSM8K,
		ProviderName:  "failing",
		ModelName:     "fail-model",
		Config:        &BenchmarkConfig{Concurrency: 1, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "err-run"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, result := range run.Results {
		assert.False(t, result.Passed)
		assert.Contains(t, result.Error, "provider unavailable")
	}
}

func TestStandardBenchmarkRunner_ExecuteTask_TaskTimeLimit(t *testing.T) {
	// Provider that takes longer than task time limit
	provider := &mockLLMProvider{
		name:     "slow",
		response: "B",
		tokens:   5,
		delay:    2 * time.Second,
	}
	r := NewStandardBenchmarkRunner(provider, nil)

	// Add custom benchmark with a very short time limit
	customBench := &Benchmark{
		ID:        "quick-bench",
		Type:      BenchmarkTypeCustom,
		Name:      "Quick Bench",
		Version:   "1.0.0",
		CreatedAt: time.Now(),
	}
	customTasks := []*BenchmarkTask{
		{
			ID:        "quick-1",
			Type:      BenchmarkTypeCustom,
			Name:      "Quick task",
			Prompt:    "Answer quickly",
			Expected:  "fast",
			TimeLimit: 100 * time.Millisecond,
		},
	}
	r.AddBenchmark(customBench, customTasks)

	ctx := context.Background()
	run := &BenchmarkRun{
		ID:            "timeout-run",
		BenchmarkType: BenchmarkTypeCustom,
		ProviderName:  "slow",
		ModelName:     "slow-model",
		Config:        &BenchmarkConfig{Concurrency: 1, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "timeout-run"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	require.Len(t, run.Results, 1)
	assert.False(t, run.Results[0].Passed)
	assert.NotEmpty(t, run.Results[0].Error, "should have a timeout error")
}

func TestStandardBenchmarkRunner_ExecuteTask_ZeroConcurrency(t *testing.T) {
	r := newTestRunner("B", 10)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "zero-conc",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
		Config:        &BenchmarkConfig{Concurrency: 0, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "zero-conc"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	assert.Equal(t, 3, len(run.Results), "should still complete all tasks with concurrency=0 (defaults to 1)")
}

func TestStandardBenchmarkRunner_ExecuteTask_NegativeConcurrency(t *testing.T) {
	r := newTestRunner("B", 10)
	ctx := context.Background()

	run := &BenchmarkRun{
		ID:            "neg-conc",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
		Config:        &BenchmarkConfig{Concurrency: -1, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run))
	require.NoError(t, r.StartRun(ctx, "neg-conc"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	assert.Equal(t, 3, len(run.Results))
}

// ---------------------------------------------------------------------------
// evaluateResponse
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_Exact(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "B"}

	passed, score := r.evaluateResponse(ctx, run, task, "B")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_CaseInsensitive(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "B"}

	passed, score := r.evaluateResponse(ctx, run, task, "b")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_WithWhitespace(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "18"}

	passed, score := r.evaluateResponse(ctx, run, task, "  18  ")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_Containment(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "18"}

	passed, score := r.evaluateResponse(ctx, run, task, "The answer is 18 dollars")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_AnswerExtraction(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "18"}

	passed, score := r.evaluateResponse(ctx, run, task, "Step by step... The answer: 18")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_AnswerIsExtraction(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "18"}

	passed, score := r.evaluateResponse(ctx, run, task, "The answer is 18")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_StringMatch_Failure(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: "18"}

	passed, score := r.evaluateResponse(ctx, run, task, "42")
	assert.False(t, passed)
	assert.Equal(t, 0.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_MultipleChoice_LetterAtStart(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}

	tests := []struct {
		name     string
		expected string
		response string
		passed   bool
	}{
		{"exact letter", "B", "b", true},
		{"letter with explanation", "B", "B) O(log n)", true},
		{"wrong letter", "B", "C", false},
		{"letter A", "A", "a) 3x^2 + 4x - 5", true},
		{"letter C", "C", "c", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &BenchmarkTask{Expected: tt.expected}
			passed, _ := r.evaluateResponse(ctx, run, task, tt.response)
			assert.Equal(t, tt.passed, passed)
		})
	}
}

func TestStandardBenchmarkRunner_EvaluateResponse_EmptyResponse_NoExpected(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: ""}

	// Empty response with no expected => fail
	passed, score := r.evaluateResponse(ctx, run, task, "")
	assert.False(t, passed)
	assert.Equal(t, 0.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_NonEmptyResponse_NoExpected(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: ""}

	// Non-empty response with no expected => pass with 0.5
	passed, score := r.evaluateResponse(ctx, run, task, "some response")
	assert.True(t, passed)
	assert.Equal(t, 0.5, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_WhitespaceOnlyResponse_NoExpected(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{Expected: ""}

	passed, score := r.evaluateResponse(ctx, run, task, "   \n\t  ")
	assert.False(t, passed)
	assert.Equal(t, 0.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_DebateEval_Success(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	de := &mockDebateEvaluator{score: 0.85, passed: true}
	r.SetDebateEvaluator(de)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{UseDebateForEval: true}}
	task := &BenchmarkTask{Expected: "something"}

	passed, score := r.evaluateResponse(ctx, run, task, "response")
	assert.True(t, passed)
	assert.Equal(t, 0.85, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_DebateEval_Failure_FallsBack(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	de := &mockDebateEvaluator{err: errors.New("debate failed")}
	r.SetDebateEvaluator(de)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{UseDebateForEval: true}}
	task := &BenchmarkTask{Expected: "18"}

	// Should fall back to string matching
	passed, score := r.evaluateResponse(ctx, run, task, "18")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_DebateEval_NilEvaluator(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	// debateEval is nil

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{UseDebateForEval: true}}
	task := &BenchmarkTask{Expected: "18"}

	// Should fall back to string matching since evaluator is nil
	passed, score := r.evaluateResponse(ctx, run, task, "18")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_DebateDisabled(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	de := &mockDebateEvaluator{score: 0.1, passed: false}
	r.SetDebateEvaluator(de)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{UseDebateForEval: false}}
	task := &BenchmarkTask{Expected: "18"}

	// Debate disabled, should use string matching, not debate evaluator
	passed, score := r.evaluateResponse(ctx, run, task, "18")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_CodeExec_AllPass(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ce := &mockCodeExecutor{
		results: []*TestCaseResult{
			{TestCaseID: "tc1", Passed: true},
			{TestCaseID: "tc2", Passed: true},
		},
	}
	r.SetCodeExecutor(ce)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{
			{ID: "tc1"}, {ID: "tc2"},
		},
	}

	passed, score := r.evaluateResponse(ctx, run, task, "def solution(): pass")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_CodeExec_PartialPass(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ce := &mockCodeExecutor{
		results: []*TestCaseResult{
			{TestCaseID: "tc1", Passed: true},
			{TestCaseID: "tc2", Passed: false},
		},
	}
	r.SetCodeExecutor(ce)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{
			{ID: "tc1"}, {ID: "tc2"},
		},
	}

	passed, score := r.evaluateResponse(ctx, run, task, "def solution(): pass")
	assert.True(t, passed) // 1/2 = 0.5 >= 0.5 threshold
	assert.Equal(t, 0.5, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_CodeExec_BelowThreshold(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ce := &mockCodeExecutor{
		results: []*TestCaseResult{
			{TestCaseID: "tc1", Passed: true},
			{TestCaseID: "tc2", Passed: false},
			{TestCaseID: "tc3", Passed: false},
		},
	}
	r.SetCodeExecutor(ce)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{
			{ID: "tc1"}, {ID: "tc2"}, {ID: "tc3"},
		},
	}

	passed, score := r.evaluateResponse(ctx, run, task, "def solution(): pass")
	assert.False(t, passed) // 1/3 ~= 0.33 < 0.5
	assert.InDelta(t, 0.333, score, 0.01)
}

func TestStandardBenchmarkRunner_EvaluateResponse_CodeExec_AllFail(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ce := &mockCodeExecutor{
		results: []*TestCaseResult{
			{TestCaseID: "tc1", Passed: false},
		},
	}
	r.SetCodeExecutor(ce)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{{ID: "tc1"}},
	}

	passed, score := r.evaluateResponse(ctx, run, task, "code")
	assert.False(t, passed) // 0/1 = 0 < 0.5
	assert.Equal(t, 0.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_CodeExec_Error(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ce := &mockCodeExecutor{err: errors.New("execution error")}
	r.SetCodeExecutor(ce)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{{ID: "tc1"}},
	}

	passed, score := r.evaluateResponse(ctx, run, task, "code")
	assert.False(t, passed)
	assert.Equal(t, 0.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_TestCases_NoExecutor(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	// No code executor set — should fall through to string matching / default

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{{ID: "tc1"}},
		Expected:  "expected",
	}

	// Falls through code exec (no executor), then string matching
	passed, score := r.evaluateResponse(ctx, run, task, "expected")
	assert.True(t, passed)
	assert.Equal(t, 1.0, score)
}

func TestStandardBenchmarkRunner_EvaluateResponse_Priority_DebateOverCodeExec(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	de := &mockDebateEvaluator{score: 0.7, passed: true}
	ce := &mockCodeExecutor{
		results: []*TestCaseResult{{TestCaseID: "tc1", Passed: false}},
	}
	r.SetDebateEvaluator(de)
	r.SetCodeExecutor(ce)

	ctx := context.Background()
	run := &BenchmarkRun{Config: &BenchmarkConfig{UseDebateForEval: true}}
	task := &BenchmarkTask{
		TestCases: []*TestCase{{ID: "tc1"}},
		Expected:  "wrong",
	}

	// Debate should take priority
	passed, score := r.evaluateResponse(ctx, run, task, "code")
	assert.True(t, passed)
	assert.Equal(t, 0.7, score)
}

// ---------------------------------------------------------------------------
// isLetter
// ---------------------------------------------------------------------------

func TestIsLetter(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{'a', true}, {'z', true}, {'m', true},
		{'A', true}, {'Z', true}, {'M', true},
		{'0', false}, {'9', false},
		{' ', false}, {'!', false}, {'.', false},
		{'@', false}, {'[', false}, {'{', false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("byte_%d", tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, isLetter(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// calculateSummary
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_CalculateSummary_Basic(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	tasks := []*BenchmarkTask{
		{ID: "t1", Difficulty: DifficultyEasy, Tags: []string{"go"}},
		{ID: "t2", Difficulty: DifficultyMedium, Tags: []string{"go", "python"}},
		{ID: "t3", Difficulty: DifficultyHard, Tags: []string{"python"}},
	}

	results := []*BenchmarkResult{
		{TaskID: "t1", Passed: true, Score: 1.0, Latency: 100 * time.Millisecond, TokensUsed: 50},
		{TaskID: "t2", Passed: false, Score: 0.0, Latency: 200 * time.Millisecond, TokensUsed: 80},
		{TaskID: "t3", Passed: true, Score: 0.8, Latency: 300 * time.Millisecond, TokensUsed: 100},
	}

	summary := r.calculateSummary(results, tasks)

	assert.Equal(t, 3, summary.TotalTasks)
	assert.Equal(t, 2, summary.PassedTasks)
	assert.Equal(t, 1, summary.FailedTasks)
	assert.Equal(t, 0, summary.ErrorTasks)
	assert.InDelta(t, 2.0/3.0, summary.PassRate, 0.01)
	assert.InDelta(t, 1.8/3.0, summary.AverageScore, 0.01)
	assert.Equal(t, 200*time.Millisecond, summary.AverageLatency)
	assert.Equal(t, 230, summary.TotalTokens)
}

func TestStandardBenchmarkRunner_CalculateSummary_ErrorTasks(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	tasks := []*BenchmarkTask{
		{ID: "t1", Difficulty: DifficultyEasy},
	}
	results := []*BenchmarkResult{
		{TaskID: "t1", Passed: false, Score: 0.0, Error: "provider error"},
	}

	summary := r.calculateSummary(results, tasks)
	assert.Equal(t, 1, summary.ErrorTasks)
	assert.Equal(t, 0, summary.FailedTasks)
	assert.Equal(t, 0, summary.PassedTasks)
}

func TestStandardBenchmarkRunner_CalculateSummary_EmptyResults(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	summary := r.calculateSummary([]*BenchmarkResult{}, []*BenchmarkTask{})
	assert.Equal(t, 0, summary.TotalTasks)
	assert.Equal(t, 0.0, summary.PassRate)
	assert.Equal(t, 0.0, summary.AverageScore)
	assert.Equal(t, time.Duration(0), summary.AverageLatency)
}

func TestStandardBenchmarkRunner_CalculateSummary_ByDifficulty(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	tasks := []*BenchmarkTask{
		{ID: "t1", Difficulty: DifficultyEasy},
		{ID: "t2", Difficulty: DifficultyEasy},
		{ID: "t3", Difficulty: DifficultyHard},
	}
	results := []*BenchmarkResult{
		{TaskID: "t1", Passed: true, Score: 1.0},
		{TaskID: "t2", Passed: true, Score: 1.0},
		{TaskID: "t3", Passed: false, Score: 0.0},
	}

	summary := r.calculateSummary(results, tasks)

	require.Contains(t, summary.ByDifficulty, DifficultyEasy)
	require.Contains(t, summary.ByDifficulty, DifficultyHard)

	easy := summary.ByDifficulty[DifficultyEasy]
	assert.Equal(t, 2, easy.Total)
	assert.Equal(t, 2, easy.Passed)
	assert.Equal(t, 1.0, easy.PassRate)

	hard := summary.ByDifficulty[DifficultyHard]
	assert.Equal(t, 1, hard.Total)
	assert.Equal(t, 0, hard.Passed)
	assert.Equal(t, 0.0, hard.PassRate)
}

func TestStandardBenchmarkRunner_CalculateSummary_ByTag(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	tasks := []*BenchmarkTask{
		{ID: "t1", Tags: []string{"go", "testing"}},
		{ID: "t2", Tags: []string{"go"}},
	}
	results := []*BenchmarkResult{
		{TaskID: "t1", Passed: true, Score: 1.0},
		{TaskID: "t2", Passed: false, Score: 0.0},
	}

	summary := r.calculateSummary(results, tasks)

	require.Contains(t, summary.ByTag, "go")
	goTag := summary.ByTag["go"]
	assert.Equal(t, 2, goTag.Total)
	assert.Equal(t, 1, goTag.Passed)
	assert.Equal(t, 0.5, goTag.PassRate)

	require.Contains(t, summary.ByTag, "testing")
	testingTag := summary.ByTag["testing"]
	assert.Equal(t, 1, testingTag.Total)
	assert.Equal(t, 1, testingTag.Passed)
	assert.Equal(t, 1.0, testingTag.PassRate)
}

func TestStandardBenchmarkRunner_CalculateSummary_UnknownTaskID(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	// Result with task ID not in tasks list
	tasks := []*BenchmarkTask{}
	results := []*BenchmarkResult{
		{TaskID: "unknown", Passed: true, Score: 1.0, Latency: 100 * time.Millisecond, TokensUsed: 10},
	}

	summary := r.calculateSummary(results, tasks)
	assert.Equal(t, 1, summary.TotalTasks)
	assert.Equal(t, 1, summary.PassedTasks)
	// No difficulty/tag breakdown for unknown task
	assert.Empty(t, summary.ByDifficulty)
	assert.Empty(t, summary.ByTag)
}

// ---------------------------------------------------------------------------
// GetRun
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_GetRun_Exists(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	run := &BenchmarkRun{ID: "get-run-1", Name: "test"}
	require.NoError(t, r.CreateRun(ctx, run))

	got, err := r.GetRun(ctx, "get-run-1")
	require.NoError(t, err)
	assert.Equal(t, "get-run-1", got.ID)
}

func TestStandardBenchmarkRunner_GetRun_NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	got, err := r.GetRun(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "run not found")
}

// ---------------------------------------------------------------------------
// ListRuns
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_ListRuns_NilFilter(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r1", Name: "run1"}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r2", Name: "run2"}))

	runs, err := r.ListRuns(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, runs, 2)
}

func TestStandardBenchmarkRunner_ListRuns_Empty(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	runs, err := r.ListRuns(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, runs)
}

func TestStandardBenchmarkRunner_ListRuns_FilterByBenchmarkType(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r1", BenchmarkType: BenchmarkTypeMMLU}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r2", BenchmarkType: BenchmarkTypeGSM8K}))

	filter := &RunFilter{BenchmarkType: BenchmarkTypeMMLU}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.Equal(t, "r1", runs[0].ID)
}

func TestStandardBenchmarkRunner_ListRuns_FilterByProvider(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r1", ProviderName: "openai"}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r2", ProviderName: "claude"}))

	filter := &RunFilter{ProviderName: "openai"}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.Equal(t, "r1", runs[0].ID)
}

func TestStandardBenchmarkRunner_ListRuns_FilterByModel(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r1", ModelName: "gpt-4"}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r2", ModelName: "gpt-3.5"}))

	filter := &RunFilter{ModelName: "gpt-4"}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.Equal(t, "r1", runs[0].ID)
}

func TestStandardBenchmarkRunner_ListRuns_FilterByStatus(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r1"}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r2"}))

	// Force one to completed
	r.mu.Lock()
	r.runs["r1"].Status = BenchmarkStatusCompleted
	r.mu.Unlock()

	filter := &RunFilter{Status: BenchmarkStatusCompleted}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.Equal(t, "r1", runs[0].ID)
}

func TestStandardBenchmarkRunner_ListRuns_Limit(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{
			ID:   fmt.Sprintf("r%d", i),
			Name: fmt.Sprintf("run-%d", i),
		}))
	}

	filter := &RunFilter{Limit: 2}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 2)
}

func TestStandardBenchmarkRunner_ListRuns_LimitExceedsTotal(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "r1"}))

	filter := &RunFilter{Limit: 100}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
}

func TestStandardBenchmarkRunner_ListRuns_CombinedFilters(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{
		ID: "r1", BenchmarkType: BenchmarkTypeMMLU, ProviderName: "openai",
	}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{
		ID: "r2", BenchmarkType: BenchmarkTypeMMLU, ProviderName: "claude",
	}))
	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{
		ID: "r3", BenchmarkType: BenchmarkTypeGSM8K, ProviderName: "openai",
	}))

	filter := &RunFilter{BenchmarkType: BenchmarkTypeMMLU, ProviderName: "openai"}
	runs, err := r.ListRuns(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.Equal(t, "r1", runs[0].ID)
}

func TestStandardBenchmarkRunner_ListRuns_SortedByCreatedAtDesc(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	// Create runs with different creation times
	run1 := &BenchmarkRun{ID: "r1"}
	require.NoError(t, r.CreateRun(ctx, run1))

	// Small delay to get different creation times
	time.Sleep(10 * time.Millisecond)
	run2 := &BenchmarkRun{ID: "r2"}
	require.NoError(t, r.CreateRun(ctx, run2))

	runs, err := r.ListRuns(ctx, nil)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	// Newest first
	assert.Equal(t, "r2", runs[0].ID)
	assert.Equal(t, "r1", runs[1].ID)
}

// ---------------------------------------------------------------------------
// CancelRun
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_CancelRun_Pending(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "cancel-1"}))

	err := r.CancelRun(ctx, "cancel-1")
	require.NoError(t, err)

	run, err := r.GetRun(ctx, "cancel-1")
	require.NoError(t, err)
	assert.Equal(t, BenchmarkStatusCancelled, run.Status)
	assert.NotNil(t, run.EndTime)
}

func TestStandardBenchmarkRunner_CancelRun_Running(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "cancel-2"}))
	r.mu.Lock()
	r.runs["cancel-2"].Status = BenchmarkStatusRunning
	r.mu.Unlock()

	err := r.CancelRun(ctx, "cancel-2")
	require.NoError(t, err)

	run, err := r.GetRun(ctx, "cancel-2")
	require.NoError(t, err)
	assert.Equal(t, BenchmarkStatusCancelled, run.Status)
}

func TestStandardBenchmarkRunner_CancelRun_NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()
	err := r.CancelRun(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

func TestStandardBenchmarkRunner_CancelRun_AlreadyCompleted(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "cancel-3"}))
	r.mu.Lock()
	r.runs["cancel-3"].Status = BenchmarkStatusCompleted
	r.mu.Unlock()

	err := r.CancelRun(ctx, "cancel-3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel run in status")
}

func TestStandardBenchmarkRunner_CancelRun_AlreadyCancelled(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "cancel-4"}))
	require.NoError(t, r.CancelRun(ctx, "cancel-4"))

	// Cancel again
	err := r.CancelRun(ctx, "cancel-4")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel run in status")
}

func TestStandardBenchmarkRunner_CancelRun_Failed(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	require.NoError(t, r.CreateRun(ctx, &BenchmarkRun{ID: "cancel-5"}))
	r.mu.Lock()
	r.runs["cancel-5"].Status = BenchmarkStatusFailed
	r.mu.Unlock()

	err := r.CancelRun(ctx, "cancel-5")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel run in status")
}

// ---------------------------------------------------------------------------
// CompareRuns
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_CompareRuns_Improvement(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{
		ID: "r1",
		Summary: &BenchmarkSummary{
			PassRate:       0.5,
			AverageScore:   0.5,
			AverageLatency: 200 * time.Millisecond,
		},
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: true},
			{TaskID: "t2", Passed: false},
		},
	}
	r.runs["r2"] = &BenchmarkRun{
		ID: "r2",
		Summary: &BenchmarkSummary{
			PassRate:       1.0,
			AverageScore:   0.9,
			AverageLatency: 100 * time.Millisecond,
		},
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: true},
			{TaskID: "t2", Passed: true},
		},
	}
	r.mu.Unlock()

	comparison, err := r.CompareRuns(ctx, "r1", "r2")
	require.NoError(t, err)

	assert.Equal(t, "r1", comparison.Run1ID)
	assert.Equal(t, "r2", comparison.Run2ID)
	assert.Equal(t, 0.5, comparison.PassRateChange)
	assert.Equal(t, 0.4, comparison.ScoreChange)
	assert.Equal(t, -100*time.Millisecond, comparison.LatencyChange)
	assert.Contains(t, comparison.Improvements, "t2")
	assert.Empty(t, comparison.Regressions)
	assert.Contains(t, comparison.Summary, "improved")
}

func TestStandardBenchmarkRunner_CompareRuns_Regression(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{
		ID: "r1",
		Summary: &BenchmarkSummary{
			PassRate:       1.0,
			AverageScore:   1.0,
			AverageLatency: 100 * time.Millisecond,
		},
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: true},
			{TaskID: "t2", Passed: true},
		},
	}
	r.runs["r2"] = &BenchmarkRun{
		ID: "r2",
		Summary: &BenchmarkSummary{
			PassRate:       0.0,
			AverageScore:   0.0,
			AverageLatency: 500 * time.Millisecond,
		},
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: false},
			{TaskID: "t2", Passed: false},
		},
	}
	r.mu.Unlock()

	comparison, err := r.CompareRuns(ctx, "r1", "r2")
	require.NoError(t, err)

	assert.Equal(t, -1.0, comparison.PassRateChange)
	assert.Len(t, comparison.Regressions, 2)
	assert.Empty(t, comparison.Improvements)
	assert.Contains(t, comparison.Summary, "regressed")
}

func TestStandardBenchmarkRunner_CompareRuns_NoChange(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	summary := &BenchmarkSummary{
		PassRate:       0.5,
		AverageScore:   0.5,
		AverageLatency: 100 * time.Millisecond,
	}
	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{
		ID:      "r1",
		Summary: summary,
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: true},
			{TaskID: "t2", Passed: false},
		},
	}
	r.runs["r2"] = &BenchmarkRun{
		ID:      "r2",
		Summary: summary,
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: true},
			{TaskID: "t2", Passed: false},
		},
	}
	r.mu.Unlock()

	comparison, err := r.CompareRuns(ctx, "r1", "r2")
	require.NoError(t, err)

	assert.Equal(t, 0.0, comparison.PassRateChange)
	assert.Empty(t, comparison.Regressions)
	assert.Empty(t, comparison.Improvements)
	assert.Contains(t, comparison.Summary, "No significant difference")
}

func TestStandardBenchmarkRunner_CompareRuns_Run1NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r2"] = &BenchmarkRun{ID: "r2", Summary: &BenchmarkSummary{}}
	r.mu.Unlock()

	_, err := r.CompareRuns(ctx, "nonexistent", "r2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

func TestStandardBenchmarkRunner_CompareRuns_Run2NotFound(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{ID: "r1", Summary: &BenchmarkSummary{}}
	r.mu.Unlock()

	_, err := r.CompareRuns(ctx, "r1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

func TestStandardBenchmarkRunner_CompareRuns_NilSummary_Run1(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{ID: "r1", Summary: nil}
	r.runs["r2"] = &BenchmarkRun{ID: "r2", Summary: &BenchmarkSummary{}}
	r.mu.Unlock()

	_, err := r.CompareRuns(ctx, "r1", "r2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both runs must be completed")
}

func TestStandardBenchmarkRunner_CompareRuns_NilSummary_Run2(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{ID: "r1", Summary: &BenchmarkSummary{}}
	r.runs["r2"] = &BenchmarkRun{ID: "r2", Summary: nil}
	r.mu.Unlock()

	_, err := r.CompareRuns(ctx, "r1", "r2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both runs must be completed")
}

func TestStandardBenchmarkRunner_CompareRuns_MixedImprovementsAndRegressions(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	r.mu.Lock()
	r.runs["r1"] = &BenchmarkRun{
		ID:      "r1",
		Summary: &BenchmarkSummary{PassRate: 0.5, AverageScore: 0.5, AverageLatency: 100 * time.Millisecond},
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: true},
			{TaskID: "t2", Passed: false},
			{TaskID: "t3", Passed: true},
		},
	}
	r.runs["r2"] = &BenchmarkRun{
		ID:      "r2",
		Summary: &BenchmarkSummary{PassRate: 0.5, AverageScore: 0.5, AverageLatency: 100 * time.Millisecond},
		Results: []*BenchmarkResult{
			{TaskID: "t1", Passed: false}, // regression
			{TaskID: "t2", Passed: true},  // improvement
			{TaskID: "t3", Passed: true},  // same
		},
	}
	r.mu.Unlock()

	comparison, err := r.CompareRuns(ctx, "r1", "r2")
	require.NoError(t, err)

	assert.Len(t, comparison.Regressions, 1)
	assert.Contains(t, comparison.Regressions, "t1")
	assert.Len(t, comparison.Improvements, 1)
	assert.Contains(t, comparison.Improvements, "t2")
	// Equal count => "No significant difference"
	assert.Contains(t, comparison.Summary, "No significant difference")
}

// ---------------------------------------------------------------------------
// AddBenchmark
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_AddBenchmark_Custom(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	custom := &Benchmark{
		ID:          "custom-1",
		Type:        BenchmarkTypeCustom,
		Name:        "My Custom Benchmark",
		Description: "Testing custom benchmark",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}
	tasks := []*BenchmarkTask{
		{ID: "c1", Name: "Custom Task 1", Prompt: "Do X", Expected: "Y"},
		{ID: "c2", Name: "Custom Task 2", Prompt: "Do A", Expected: "B"},
	}

	r.AddBenchmark(custom, tasks)

	assert.Equal(t, 2, custom.TaskCount)

	// Verify it's accessible
	got, err := r.GetBenchmark(ctx, "custom-1")
	require.NoError(t, err)
	assert.Equal(t, "My Custom Benchmark", got.Name)

	gotTasks, err := r.GetTasks(ctx, "custom-1", nil)
	require.NoError(t, err)
	assert.Len(t, gotTasks, 2)
}

func TestStandardBenchmarkRunner_AddBenchmark_OverwriteExisting(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	bench := &Benchmark{
		ID:      "custom-overwrite",
		Type:    BenchmarkTypeCustom,
		Name:    "Original",
		Version: "1.0.0",
	}
	r.AddBenchmark(bench, []*BenchmarkTask{{ID: "t1"}})

	// Overwrite
	bench2 := &Benchmark{
		ID:      "custom-overwrite",
		Type:    BenchmarkTypeCustom,
		Name:    "Updated",
		Version: "2.0.0",
	}
	r.AddBenchmark(bench2, []*BenchmarkTask{{ID: "t1"}, {ID: "t2"}})

	got, err := r.GetBenchmark(ctx, "custom-overwrite")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, 2, got.TaskCount)
}

func TestStandardBenchmarkRunner_AddBenchmark_EmptyTasks(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	bench := &Benchmark{
		ID:      "empty-bench",
		Type:    BenchmarkTypeCustom,
		Name:    "Empty",
		Version: "1.0.0",
	}
	r.AddBenchmark(bench, []*BenchmarkTask{})

	assert.Equal(t, 0, bench.TaskCount)
}

func TestStandardBenchmarkRunner_AddBenchmark_NilTasks(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)

	bench := &Benchmark{
		ID:      "nil-tasks-bench",
		Type:    BenchmarkTypeCustom,
		Name:    "Nil Tasks",
		Version: "1.0.0",
	}
	r.AddBenchmark(bench, nil)

	assert.Equal(t, 0, bench.TaskCount)
}

func TestStandardBenchmarkRunner_AddBenchmark_AppearsInList(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	bench := &Benchmark{
		ID:      "list-test",
		Type:    BenchmarkTypeCustom,
		Name:    "Listed Benchmark",
		Version: "1.0.0",
	}
	r.AddBenchmark(bench, []*BenchmarkTask{})

	benchmarks, err := r.ListBenchmarks(ctx)
	require.NoError(t, err)
	assert.Len(t, benchmarks, 5) // 4 built-in + 1 custom
}

// ---------------------------------------------------------------------------
// matchesFilter (tested via ListRuns, but ensure edge cases)
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_MatchesFilter_EmptyFilter(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	run := &BenchmarkRun{ID: "r1", BenchmarkType: BenchmarkTypeMMLU, ProviderName: "p", ModelName: "m"}
	assert.True(t, r.matchesFilter(run, &RunFilter{}))
}

func TestStandardBenchmarkRunner_MatchesFilter_NilFilter(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, nil)
	run := &BenchmarkRun{ID: "r1"}
	assert.True(t, r.matchesFilter(run, nil))
}

// ---------------------------------------------------------------------------
// End-to-end: create, start, wait, compare
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_E2E_CreateStartAndCompare(t *testing.T) {
	r := newTestRunner("B", 10)
	ctx := context.Background()

	// Create and start run 1
	run1 := &BenchmarkRun{
		ID:            "e2e-r1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "test",
		ModelName:     "test-model",
		Config:        &BenchmarkConfig{Concurrency: 2, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run1))
	require.NoError(t, r.StartRun(ctx, "e2e-r1"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run1.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	// Create and start run 2 with a "wrong" answer
	provider2 := &mockLLMProvider{name: "wrong", response: "D", tokens: 5}
	r.provider = provider2
	run2 := &BenchmarkRun{
		ID:            "e2e-r2",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "wrong",
		ModelName:     "wrong-model",
		Config:        &BenchmarkConfig{Concurrency: 2, Timeout: 30 * time.Second},
	}
	require.NoError(t, r.CreateRun(ctx, run2))
	require.NoError(t, r.StartRun(ctx, "e2e-r2"))

	require.Eventually(t, func() bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return run2.Status == BenchmarkStatusCompleted
	}, 10*time.Second, 50*time.Millisecond)

	// Compare
	comparison, err := r.CompareRuns(ctx, "e2e-r1", "e2e-r2")
	require.NoError(t, err)

	// Run 1 answers "B" which matches mmlu-001 (expected B), so pass rate > 0
	// Run 2 answers "D" which doesn't match any expected, so pass rate = 0
	assert.Less(t, comparison.PassRateChange, 0.0, "Run 2 should have lower pass rate")
	assert.NotEmpty(t, comparison.Regressions)
}

// ---------------------------------------------------------------------------
// Concurrency safety
// ---------------------------------------------------------------------------

func TestStandardBenchmarkRunner_ConcurrentAccess(t *testing.T) {
	r := newTestRunner("B", 10)
	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// Concurrent ListBenchmarks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.ListBenchmarks(ctx)
			if err != nil {
				errCh <- err
			}
		}()
	}

	// Concurrent CreateRun
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			run := &BenchmarkRun{
				ID:   fmt.Sprintf("conc-%d", idx),
				Name: fmt.Sprintf("concurrent-%d", idx),
			}
			if err := r.CreateRun(ctx, run); err != nil {
				errCh <- err
			}
		}(i)
	}

	// Concurrent GetBenchmark
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.GetBenchmark(ctx, "mmlu-mini")
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent access error: %v", err)
	}
}

// ===========================================================================
// integration.go tests
// ===========================================================================

// ---------------------------------------------------------------------------
// DefaultBenchmarkSystemConfig
// ---------------------------------------------------------------------------

func TestDefaultBenchmarkSystemConfig_ReturnsNonNil(t *testing.T) {
	cfg := DefaultBenchmarkSystemConfig()
	require.NotNil(t, cfg)
}

func TestDefaultBenchmarkSystemConfig_Values(t *testing.T) {
	cfg := DefaultBenchmarkSystemConfig()
	assert.True(t, cfg.EnableDebateEvaluation)
	assert.True(t, cfg.UseVerifierScores)
	assert.True(t, cfg.AutoSelectProvider)
	assert.Equal(t, 4, cfg.DefaultConcurrency)
}

// ---------------------------------------------------------------------------
// Mock implementations for integration tests
// ---------------------------------------------------------------------------

type mockDebateServiceForBenchmark struct {
	result *DebateResultForBenchmark
	err    error
}

func (m *mockDebateServiceForBenchmark) RunDebate(_ context.Context, _ string) (*DebateResultForBenchmark, error) {
	return m.result, m.err
}

type mockVerifierServiceForBenchmark struct {
	scores       map[string]float64
	healthy      map[string]bool
	topProviders []string
}

func (m *mockVerifierServiceForBenchmark) GetProviderScore(name string) float64 {
	if m.scores != nil {
		return m.scores[name]
	}
	return 0
}

func (m *mockVerifierServiceForBenchmark) IsProviderHealthy(name string) bool {
	if m.healthy != nil {
		return m.healthy[name]
	}
	return false
}

func (m *mockVerifierServiceForBenchmark) GetTopProviders(_ int) []string {
	return m.topProviders
}

type mockProviderServiceForBenchmark struct {
	response string
	tokens   int
	err      error
	provider LLMProvider
}

func (m *mockProviderServiceForBenchmark) Complete(_ context.Context, _, _, _, _ string) (string, int, error) {
	return m.response, m.tokens, m.err
}

func (m *mockProviderServiceForBenchmark) GetProvider(_ string) LLMProvider {
	return m.provider
}

// ---------------------------------------------------------------------------
// NewDebateAdapterForBenchmark / EvaluateResponse
// ---------------------------------------------------------------------------

func TestNewDebateAdapterForBenchmark(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{}
	adapter := NewDebateAdapterForBenchmark(ds, nil)
	require.NotNil(t, adapter)
	assert.NotNil(t, adapter.service)
}

func TestDebateAdapterForBenchmark_EvaluateResponse_NilService(t *testing.T) {
	adapter := NewDebateAdapterForBenchmark(nil, nil)
	_, _, err := adapter.EvaluateResponse(context.Background(), &BenchmarkTask{}, "response")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debate service not available")
}

func TestDebateAdapterForBenchmark_EvaluateResponse_Success_JSON(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{
		result: &DebateResultForBenchmark{
			ID:         "d-1",
			Consensus:  `The evaluation is: {"score": 0.85, "passed": true}`,
			Confidence: 0.9,
		},
	}
	adapter := NewDebateAdapterForBenchmark(ds, nil)

	score, passed, err := adapter.EvaluateResponse(context.Background(), &BenchmarkTask{
		Name:        "test task",
		Description: "desc",
		Expected:    "expected",
	}, "response text")
	require.NoError(t, err)
	assert.Equal(t, 0.85, score)
	assert.True(t, passed)
}

func TestDebateAdapterForBenchmark_EvaluateResponse_FallbackToConfidence(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{
		result: &DebateResultForBenchmark{
			ID:         "d-2",
			Consensus:  "no json here at all",
			Confidence: 0.75,
		},
	}
	adapter := NewDebateAdapterForBenchmark(ds, nil)

	score, passed, err := adapter.EvaluateResponse(context.Background(), &BenchmarkTask{}, "response")
	require.NoError(t, err)
	assert.Equal(t, 0.75, score)
	assert.True(t, passed) // 0.75 >= 0.7
}

func TestDebateAdapterForBenchmark_EvaluateResponse_FallbackToConfidence_Below70(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{
		result: &DebateResultForBenchmark{
			Consensus:  "no json",
			Confidence: 0.6,
		},
	}
	adapter := NewDebateAdapterForBenchmark(ds, nil)

	score, passed, err := adapter.EvaluateResponse(context.Background(), &BenchmarkTask{}, "response")
	require.NoError(t, err)
	assert.Equal(t, 0.6, score)
	assert.False(t, passed) // 0.6 < 0.7
}

func TestDebateAdapterForBenchmark_EvaluateResponse_ServiceError(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{err: fmt.Errorf("debate failed")}
	adapter := NewDebateAdapterForBenchmark(ds, nil)

	_, _, err := adapter.EvaluateResponse(context.Background(), &BenchmarkTask{}, "response")
	require.Error(t, err)
}

func TestDebateAdapterForBenchmark_ParseEvaluationResult_ValidJSON(t *testing.T) {
	adapter := NewDebateAdapterForBenchmark(nil, nil)
	score, passed := adapter.parseEvaluationResult(`{"score": 0.95, "passed": true}`, 0.5)
	assert.Equal(t, 0.95, score)
	assert.True(t, passed)
}

func TestDebateAdapterForBenchmark_ParseEvaluationResult_InvalidJSON(t *testing.T) {
	adapter := NewDebateAdapterForBenchmark(nil, nil)
	score, passed := adapter.parseEvaluationResult(`{invalid json}`, 0.8)
	assert.Equal(t, 0.8, score) // Falls back to confidence
	assert.True(t, passed)      // 0.8 >= 0.7
}

func TestDebateAdapterForBenchmark_ParseEvaluationResult_NoJSON(t *testing.T) {
	adapter := NewDebateAdapterForBenchmark(nil, nil)
	score, passed := adapter.parseEvaluationResult("just text no braces", 0.3)
	assert.Equal(t, 0.3, score)
	assert.False(t, passed) // 0.3 < 0.7
}

func TestDebateAdapterForBenchmark_ParseEvaluationResult_WrappedJSON(t *testing.T) {
	adapter := NewDebateAdapterForBenchmark(nil, nil)
	score, passed := adapter.parseEvaluationResult(
		`Here is my evaluation: {"score": 0.72, "passed": false} done`, 0.5)
	assert.Equal(t, 0.72, score)
	assert.False(t, passed)
}

// ---------------------------------------------------------------------------
// NewVerifierAdapterForBenchmark / SelectBestProvider
// ---------------------------------------------------------------------------

func TestNewVerifierAdapterForBenchmark(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{}
	adapter := NewVerifierAdapterForBenchmark(vs, nil)
	require.NotNil(t, adapter)
}

func TestVerifierAdapterForBenchmark_SelectBestProvider_NilService(t *testing.T) {
	adapter := NewVerifierAdapterForBenchmark(nil, nil)
	name, score := adapter.SelectBestProvider()
	assert.Empty(t, name)
	assert.Equal(t, float64(0), score)
}

func TestVerifierAdapterForBenchmark_SelectBestProvider_NoProviders(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{topProviders: []string{}}
	adapter := NewVerifierAdapterForBenchmark(vs, nil)
	name, score := adapter.SelectBestProvider()
	assert.Empty(t, name)
	assert.Equal(t, float64(0), score)
}

func TestVerifierAdapterForBenchmark_SelectBestProvider_SelectsHighestScore(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{
		topProviders: []string{"claude", "openai", "gemini"},
		scores:       map[string]float64{"claude": 0.9, "openai": 0.8, "gemini": 0.85},
		healthy:      map[string]bool{"claude": true, "openai": true, "gemini": true},
	}
	adapter := NewVerifierAdapterForBenchmark(vs, nil)
	name, score := adapter.SelectBestProvider()
	assert.Equal(t, "claude", name)
	assert.Equal(t, 0.9, score)
}

func TestVerifierAdapterForBenchmark_SelectBestProvider_SkipsUnhealthy(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{
		topProviders: []string{"claude", "openai"},
		scores:       map[string]float64{"claude": 0.9, "openai": 0.8},
		healthy:      map[string]bool{"claude": false, "openai": true},
	}
	adapter := NewVerifierAdapterForBenchmark(vs, nil)
	name, score := adapter.SelectBestProvider()
	assert.Equal(t, "openai", name)
	assert.Equal(t, 0.8, score)
}

func TestVerifierAdapterForBenchmark_SelectBestProvider_AllUnhealthy(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{
		topProviders: []string{"claude", "openai"},
		scores:       map[string]float64{"claude": 0.9, "openai": 0.8},
		healthy:      map[string]bool{"claude": false, "openai": false},
	}
	adapter := NewVerifierAdapterForBenchmark(vs, nil)
	name, score := adapter.SelectBestProvider()
	assert.Empty(t, name)
	assert.Equal(t, float64(0), score)
}

func TestVerifierAdapterForBenchmark_GetProviderScoresForComparison_NilService(t *testing.T) {
	adapter := NewVerifierAdapterForBenchmark(nil, nil)
	scores := adapter.GetProviderScoresForComparison()
	assert.Nil(t, scores)
}

func TestVerifierAdapterForBenchmark_GetProviderScoresForComparison_Success(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{
		topProviders: []string{"claude", "openai"},
		scores:       map[string]float64{"claude": 0.9, "openai": 0.8},
	}
	adapter := NewVerifierAdapterForBenchmark(vs, nil)
	scores := adapter.GetProviderScoresForComparison()
	require.NotNil(t, scores)
	assert.Len(t, scores, 2)
	assert.Equal(t, 0.9, scores["claude"])
	assert.Equal(t, 0.8, scores["openai"])
}

// ---------------------------------------------------------------------------
// NewProviderAdapterForBenchmark / Complete / GetName
// ---------------------------------------------------------------------------

func TestNewProviderAdapterForBenchmark(t *testing.T) {
	svc := &mockProviderServiceForBenchmark{}
	adapter := NewProviderAdapterForBenchmark(svc, "claude", "sonnet", nil)
	require.NotNil(t, adapter)
	assert.Equal(t, "claude", adapter.providerName)
	assert.Equal(t, "sonnet", adapter.modelName)
}

func TestProviderAdapterForBenchmark_Complete_NilService(t *testing.T) {
	adapter := NewProviderAdapterForBenchmark(nil, "claude", "sonnet", nil)
	_, _, err := adapter.Complete(context.Background(), "prompt", "system")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider service not available")
}

func TestProviderAdapterForBenchmark_Complete_Success(t *testing.T) {
	svc := &mockProviderServiceForBenchmark{
		response: "test response",
		tokens:   50,
	}
	adapter := NewProviderAdapterForBenchmark(svc, "claude", "sonnet", nil)
	resp, tokens, err := adapter.Complete(context.Background(), "prompt", "system")
	require.NoError(t, err)
	assert.Equal(t, "test response", resp)
	assert.Equal(t, 50, tokens)
}

func TestProviderAdapterForBenchmark_GetName(t *testing.T) {
	adapter := NewProviderAdapterForBenchmark(nil, "my-provider", "model", nil)
	assert.Equal(t, "my-provider", adapter.GetName())
}

// ---------------------------------------------------------------------------
// NewBenchmarkSystem
// ---------------------------------------------------------------------------

func TestNewBenchmarkSystem_NilConfig(t *testing.T) {
	bs := NewBenchmarkSystem(nil, nil)
	require.NotNil(t, bs)
	assert.NotNil(t, bs.config)
	assert.NotNil(t, bs.logger)
}

func TestNewBenchmarkSystem_CustomConfig(t *testing.T) {
	cfg := &BenchmarkSystemConfig{
		EnableDebateEvaluation: false,
		DefaultConcurrency:     8,
	}
	bs := NewBenchmarkSystem(cfg, nil)
	require.NotNil(t, bs)
	assert.False(t, bs.config.EnableDebateEvaluation)
	assert.Equal(t, 8, bs.config.DefaultConcurrency)
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.Initialize
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_Initialize_NilProvider(t *testing.T) {
	bs := NewBenchmarkSystem(nil, nil)
	err := bs.Initialize(nil)
	require.NoError(t, err)
	assert.NotNil(t, bs.runner)
}

func TestBenchmarkSystem_Initialize_WithProvider(t *testing.T) {
	svc := &mockProviderServiceForBenchmark{response: "test"}
	adapter := NewProviderAdapterForBenchmark(svc, "claude", "sonnet", nil)
	bs := NewBenchmarkSystem(nil, nil)
	err := bs.Initialize(adapter)
	require.NoError(t, err)
	assert.NotNil(t, bs.runner)
	assert.NotNil(t, bs.providerAdapter)
}

func TestBenchmarkSystem_Initialize_SetsDebateEvaluator(t *testing.T) {
	debateSvc := &mockDebateServiceForBenchmark{
		result: &DebateResultForBenchmark{Confidence: 0.8},
	}
	bs := NewBenchmarkSystem(nil, nil)
	bs.SetDebateService(debateSvc)
	err := bs.Initialize(nil)
	require.NoError(t, err)
	// The debate adapter should be set on the runner
	runner, ok := bs.runner.(*StandardBenchmarkRunner)
	require.True(t, ok)
	assert.NotNil(t, runner.debateEval)
}

func TestBenchmarkSystem_Initialize_DebateDisabled(t *testing.T) {
	debateSvc := &mockDebateServiceForBenchmark{
		result: &DebateResultForBenchmark{Confidence: 0.8},
	}
	cfg := &BenchmarkSystemConfig{EnableDebateEvaluation: false}
	bs := NewBenchmarkSystem(cfg, nil)
	bs.SetDebateService(debateSvc)
	err := bs.Initialize(nil)
	require.NoError(t, err)
	runner, ok := bs.runner.(*StandardBenchmarkRunner)
	require.True(t, ok)
	assert.Nil(t, runner.debateEval) // Debate disabled
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.SetDebateService
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_SetDebateService_BeforeInit(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{}
	bs := NewBenchmarkSystem(nil, nil)
	bs.SetDebateService(ds)
	assert.NotNil(t, bs.debateAdapter)
}

func TestBenchmarkSystem_SetDebateService_AfterInit(t *testing.T) {
	ds := &mockDebateServiceForBenchmark{}
	bs := NewBenchmarkSystem(nil, nil)
	_ = bs.Initialize(nil)
	bs.SetDebateService(ds)
	assert.NotNil(t, bs.debateAdapter)
	// Runner should have debate evaluator set
	runner, ok := bs.runner.(*StandardBenchmarkRunner)
	require.True(t, ok)
	assert.NotNil(t, runner.debateEval)
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.SetVerifierService
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_SetVerifierService(t *testing.T) {
	vs := &mockVerifierServiceForBenchmark{}
	bs := NewBenchmarkSystem(nil, nil)
	bs.SetVerifierService(vs)
	assert.NotNil(t, bs.verifierAdapter)
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.GetRunner
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_GetRunner_BeforeInit(t *testing.T) {
	bs := NewBenchmarkSystem(nil, nil)
	assert.Nil(t, bs.GetRunner())
}

func TestBenchmarkSystem_GetRunner_AfterInit(t *testing.T) {
	bs := NewBenchmarkSystem(nil, nil)
	_ = bs.Initialize(nil)
	assert.NotNil(t, bs.GetRunner())
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.RunBenchmarkWithBestProvider
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_RunBenchmarkWithBestProvider_NoVerifier(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	bs.runner = NewStandardBenchmarkRunner(provider, nil)

	run, err := bs.RunBenchmarkWithBestProvider(context.Background(), BenchmarkTypeMMLU, nil)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "default", run.ProviderName)
	assert.Equal(t, BenchmarkTypeMMLU, run.BenchmarkType)
}

func TestBenchmarkSystem_RunBenchmarkWithBestProvider_WithVerifier(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	vs := &mockVerifierServiceForBenchmark{
		topProviders: []string{"claude", "openai"},
		scores:       map[string]float64{"claude": 0.9, "openai": 0.8},
		healthy:      map[string]bool{"claude": true, "openai": true},
	}
	bs := NewBenchmarkSystem(nil, nil)
	bs.runner = NewStandardBenchmarkRunner(provider, nil)
	bs.verifierAdapter = NewVerifierAdapterForBenchmark(vs, nil)

	run, err := bs.RunBenchmarkWithBestProvider(context.Background(), BenchmarkTypeMMLU, nil)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "claude", run.ProviderName)
}

func TestBenchmarkSystem_RunBenchmarkWithBestProvider_CustomConfig(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	bs.runner = NewStandardBenchmarkRunner(provider, nil)

	cfg := DefaultBenchmarkConfig()
	cfg.MaxTasks = 1
	run, err := bs.RunBenchmarkWithBestProvider(context.Background(), BenchmarkTypeMMLU, cfg)
	require.NoError(t, err)
	require.NotNil(t, run)
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.CompareProviders
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_CompareProviders(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	bs.runner = NewStandardBenchmarkRunner(provider, nil)

	runs, err := bs.CompareProviders(context.Background(), BenchmarkTypeMMLU,
		[]string{"claude", "openai"}, nil)
	require.NoError(t, err)
	assert.Len(t, runs, 2)
	assert.Equal(t, "claude", runs[0].ProviderName)
	assert.Equal(t, "openai", runs[1].ProviderName)
}

func TestBenchmarkSystem_CompareProviders_NilConfig(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	bs.runner = NewStandardBenchmarkRunner(provider, nil)

	runs, err := bs.CompareProviders(context.Background(), BenchmarkTypeMMLU,
		[]string{"provider1"}, nil)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
}

// ---------------------------------------------------------------------------
// BenchmarkSystem.GenerateLeaderboard
// ---------------------------------------------------------------------------

func TestBenchmarkSystem_GenerateLeaderboard_NoRuns(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	bs.runner = NewStandardBenchmarkRunner(provider, nil)

	lb, err := bs.GenerateLeaderboard(context.Background(), BenchmarkTypeMMLU)
	require.NoError(t, err)
	require.NotNil(t, lb)
	assert.Equal(t, BenchmarkTypeMMLU, lb.BenchmarkType)
	assert.Empty(t, lb.Entries)
}

func TestBenchmarkSystem_GenerateLeaderboard_WithCompletedRuns(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	runner := NewStandardBenchmarkRunner(provider, nil)
	bs.runner = runner

	// Create and manually populate completed runs
	now := time.Now()
	run1 := &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "claude",
		ModelName:     "sonnet",
		Status:        BenchmarkStatusCompleted,
		Summary: &BenchmarkSummary{
			TotalTasks:     10,
			PassedTasks:    8,
			PassRate:       0.8,
			AverageScore:   0.85,
			AverageLatency: 100 * time.Millisecond,
		},
		CreatedAt: now,
	}
	run2 := &BenchmarkRun{
		ID:            "run-2",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "openai",
		ModelName:     "gpt-4",
		Status:        BenchmarkStatusCompleted,
		Summary: &BenchmarkSummary{
			TotalTasks:     10,
			PassedTasks:    9,
			PassRate:       0.9,
			AverageScore:   0.92,
			AverageLatency: 200 * time.Millisecond,
		},
		CreatedAt: now,
	}

	runner.mu.Lock()
	runner.runs["run-1"] = run1
	runner.runs["run-2"] = run2
	runner.mu.Unlock()

	lb, err := bs.GenerateLeaderboard(context.Background(), BenchmarkTypeMMLU)
	require.NoError(t, err)
	require.NotNil(t, lb)
	assert.Len(t, lb.Entries, 2)
	// Sorted by pass rate, openai (0.9) should be first
	assert.Equal(t, 1, lb.Entries[0].Rank)
	assert.Equal(t, "openai", lb.Entries[0].ProviderName)
	assert.Equal(t, 2, lb.Entries[1].Rank)
	assert.Equal(t, "claude", lb.Entries[1].ProviderName)
}

func TestBenchmarkSystem_GenerateLeaderboard_WithVerifierScores(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	vs := &mockVerifierServiceForBenchmark{
		topProviders: []string{"claude"},
		scores:       map[string]float64{"claude": 0.95},
		healthy:      map[string]bool{"claude": true},
	}
	bs := NewBenchmarkSystem(nil, nil)
	runner := NewStandardBenchmarkRunner(provider, nil)
	bs.runner = runner
	bs.verifierAdapter = NewVerifierAdapterForBenchmark(vs, nil)

	now := time.Now()
	runner.mu.Lock()
	runner.runs["run-1"] = &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "claude",
		Status:        BenchmarkStatusCompleted,
		Summary:       &BenchmarkSummary{TotalTasks: 5, PassRate: 0.8},
		CreatedAt:     now,
	}
	runner.mu.Unlock()

	lb, err := bs.GenerateLeaderboard(context.Background(), BenchmarkTypeMMLU)
	require.NoError(t, err)
	require.Len(t, lb.Entries, 1)
	assert.Equal(t, 0.95, lb.Entries[0].VerifierScore)
}

func TestBenchmarkSystem_GenerateLeaderboard_SkipsNilSummary(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	runner := NewStandardBenchmarkRunner(provider, nil)
	bs.runner = runner

	runner.mu.Lock()
	runner.runs["run-1"] = &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "claude",
		Status:        BenchmarkStatusCompleted,
		Summary:       nil, // No summary
		CreatedAt:     time.Now(),
	}
	runner.mu.Unlock()

	lb, err := bs.GenerateLeaderboard(context.Background(), BenchmarkTypeMMLU)
	require.NoError(t, err)
	assert.Empty(t, lb.Entries)
}

func TestBenchmarkSystem_GenerateLeaderboard_BestRunPerProvider(t *testing.T) {
	provider := &mockLLMProvider{name: "test", response: "B", tokens: 10}
	bs := NewBenchmarkSystem(nil, nil)
	runner := NewStandardBenchmarkRunner(provider, nil)
	bs.runner = runner

	now := time.Now()
	runner.mu.Lock()
	runner.runs["run-1"] = &BenchmarkRun{
		ID:            "run-1",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "claude",
		Status:        BenchmarkStatusCompleted,
		Summary:       &BenchmarkSummary{PassRate: 0.7},
		CreatedAt:     now,
	}
	runner.runs["run-2"] = &BenchmarkRun{
		ID:            "run-2",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "claude",
		Status:        BenchmarkStatusCompleted,
		Summary:       &BenchmarkSummary{PassRate: 0.9},
		CreatedAt:     now,
	}
	runner.mu.Unlock()

	lb, err := bs.GenerateLeaderboard(context.Background(), BenchmarkTypeMMLU)
	require.NoError(t, err)
	require.Len(t, lb.Entries, 1)                // Only one entry per provider
	assert.Equal(t, 0.9, lb.Entries[0].PassRate) // Best run
}

// ---------------------------------------------------------------------------
// Leaderboard / LeaderboardEntry structs
// ---------------------------------------------------------------------------

func TestLeaderboard_Fields(t *testing.T) {
	now := time.Now()
	lb := Leaderboard{
		BenchmarkType: BenchmarkTypeMMLU,
		Entries: []*LeaderboardEntry{
			{Rank: 1, ProviderName: "claude", PassRate: 0.9},
		},
		GeneratedAt: now,
	}
	assert.Equal(t, BenchmarkTypeMMLU, lb.BenchmarkType)
	assert.Len(t, lb.Entries, 1)
	assert.Equal(t, now, lb.GeneratedAt)
}

func TestLeaderboardEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := LeaderboardEntry{
		Rank:           1,
		ProviderName:   "claude",
		ModelName:      "sonnet",
		PassRate:       0.9,
		AverageScore:   0.85,
		AverageLatency: 100 * time.Millisecond,
		TotalTasks:     50,
		VerifierScore:  0.95,
		RunID:          "run-1",
		RunDate:        now,
	}
	assert.Equal(t, 1, entry.Rank)
	assert.Equal(t, "claude", entry.ProviderName)
	assert.Equal(t, 0.95, entry.VerifierScore)
}

// ---------------------------------------------------------------------------
// BenchmarkSystemConfig struct
// ---------------------------------------------------------------------------

func TestBenchmarkSystemConfig_ZeroValue(t *testing.T) {
	var cfg BenchmarkSystemConfig
	assert.False(t, cfg.EnableDebateEvaluation)
	assert.False(t, cfg.UseVerifierScores)
	assert.False(t, cfg.AutoSelectProvider)
	assert.Equal(t, 0, cfg.DefaultConcurrency)
}

// ---------------------------------------------------------------------------
// DebateResultForBenchmark struct
// ---------------------------------------------------------------------------

func TestDebateResultForBenchmark_Fields(t *testing.T) {
	dr := DebateResultForBenchmark{
		ID:         "d-1",
		Consensus:  "answer is B",
		Confidence: 0.85,
		Votes:      map[string]float64{"p1": 0.9, "p2": 0.8},
	}
	assert.Equal(t, "d-1", dr.ID)
	assert.Equal(t, "answer is B", dr.Consensus)
	assert.Equal(t, 0.85, dr.Confidence)
	assert.Len(t, dr.Votes, 2)
}
