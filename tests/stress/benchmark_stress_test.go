package stress

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.benchmark/benchmark"
)

// --- mock implementations ---

type stressLLMProvider struct{}

func (p *stressLLMProvider) Complete(_ context.Context, _, _ string) (string, int, error) {
	return "B", 10, nil
}

func (p *stressLLMProvider) GetName() string {
	return "stress-provider"
}

// --- Stress Tests ---

func TestRunner_ConcurrentListBenchmarks_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 100

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := runner.ListBenchmarks(ctx)
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}
}

func TestRunner_ConcurrentGetBenchmark_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 75

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	ids := []string{"swe-bench-lite", "humaneval", "mmlu-mini", "gsm8k-mini"}
	var wg sync.WaitGroup
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := runner.GetBenchmark(ctx, ids[idx%len(ids)])
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}
}

func TestRunner_ConcurrentCreateRun_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 80

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	createdIDs := make([]string, 0, goroutines)
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			run := &benchmark.BenchmarkRun{
				BenchmarkType: benchmark.BenchmarkTypeMMLU,
				ProviderName:  fmt.Sprintf("provider-%d", idx),
			}
			err := runner.CreateRun(ctx, run)
			errors[idx] = err
			if err == nil {
				mu.Lock()
				createdIDs = append(createdIDs, run.ID)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}
	assert.Equal(t, goroutines, successCount, "all creates should succeed")
	assert.Len(t, createdIDs, goroutines, "all should have unique IDs")
}

func TestRunner_ConcurrentGetTasks_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 60

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make([]error, goroutines)
	taskCounts := make([]int, goroutines)

	benchmarkIDs := []string{"swe-bench-lite", "humaneval", "mmlu-mini", "gsm8k-mini"}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tasks, err := runner.GetTasks(ctx, benchmarkIDs[idx%len(benchmarkIDs)], nil)
			errors[idx] = err
			if err == nil {
				taskCounts[idx] = len(tasks)
			}
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}

	for i, count := range taskCounts {
		assert.Greater(t, count, 0, "goroutine %d should get tasks", i)
	}
}

func TestRunner_ConcurrentStartRun_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 50

	provider := &stressLLMProvider{}
	runner := benchmark.NewStandardBenchmarkRunner(provider, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	completedCount := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			run := &benchmark.BenchmarkRun{
				BenchmarkType: benchmark.BenchmarkTypeMMLU,
				ProviderName:  fmt.Sprintf("stress-provider-%d", idx),
				Config:        benchmark.DefaultBenchmarkConfig(),
			}

			if err := runner.CreateRun(ctx, run); err != nil {
				return
			}
			if err := runner.StartRun(ctx, run.ID); err != nil {
				return
			}

			// Wait for async completion
			time.Sleep(4 * time.Second)

			fetched, err := runner.GetRun(ctx, run.ID)
			if err == nil && fetched.Status == benchmark.BenchmarkStatusCompleted {
				mu.Lock()
				completedCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, goroutines, completedCount, "all runs should complete")
}

func TestRunner_ConcurrentListRuns_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 75

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	// Pre-create some runs
	for i := 0; i < 10; i++ {
		run := &benchmark.BenchmarkRun{
			BenchmarkType: benchmark.BenchmarkTypeMMLU,
			ProviderName:  fmt.Sprintf("provider-%d", i),
		}
		_ = runner.CreateRun(ctx, run)
	}

	var wg sync.WaitGroup
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := runner.ListRuns(ctx, nil)
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}
}

func TestRunner_ConcurrentCancelRun_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 60

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	cancelledCount := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			run := &benchmark.BenchmarkRun{
				BenchmarkType: benchmark.BenchmarkTypeMMLU,
				ProviderName:  fmt.Sprintf("cancel-provider-%d", idx),
			}
			if err := runner.CreateRun(ctx, run); err != nil {
				return
			}
			if err := runner.CancelRun(ctx, run.ID); err == nil {
				mu.Lock()
				cancelledCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, goroutines, cancelledCount, "all cancellations should succeed")
}

func TestRunner_ConcurrentAddBenchmark_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 50

	runner := benchmark.NewStandardBenchmarkRunner(nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b := &benchmark.Benchmark{
				ID:        fmt.Sprintf("stress-bench-%d", idx),
				Type:      benchmark.BenchmarkTypeCustom,
				Name:      fmt.Sprintf("Stress Benchmark %d", idx),
				Version:   "1.0.0",
				CreatedAt: time.Now(),
			}
			tasks := []*benchmark.BenchmarkTask{
				{
					ID:     fmt.Sprintf("stress-task-%d", idx),
					Type:   benchmark.BenchmarkTypeCustom,
					Name:   "Task",
					Prompt: "Test prompt",
				},
			}
			runner.AddBenchmark(b, tasks)
		}(i)
	}

	wg.Wait()

	// Verify all benchmarks were added
	for i := 0; i < goroutines; i++ {
		b, err := runner.GetBenchmark(ctx, fmt.Sprintf("stress-bench-%d", i))
		require.NoError(t, err, "benchmark %d should exist", i)
		assert.Equal(t, 1, b.TaskCount)
	}
}

func TestBenchmarkSystem_ConcurrentInitialize_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 50

	var wg sync.WaitGroup
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			config := benchmark.DefaultBenchmarkSystemConfig()
			system := benchmark.NewBenchmarkSystem(config, nil)
			errors[idx] = system.Initialize(nil)
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}
}
