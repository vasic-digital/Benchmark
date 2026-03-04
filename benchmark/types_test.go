package benchmark

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	runtime.GOMAXPROCS(2)
}

// --- Constant / enum value tests ---

func TestBenchmarkType_Values(t *testing.T) {
	tests := []struct {
		name     string
		value    BenchmarkType
		expected string
	}{
		{"SWEBench", BenchmarkTypeSWEBench, "swe-bench"},
		{"HumanEval", BenchmarkTypeHumanEval, "humaneval"},
		{"MBPP", BenchmarkTypeMBPP, "mbpp"},
		{"LMSYS", BenchmarkTypeLMSYS, "lmsys"},
		{"HellaSwag", BenchmarkTypeHellaSwag, "hellaswag"},
		{"MMLU", BenchmarkTypeMMLU, "mmlu"},
		{"GSM8K", BenchmarkTypeGSM8K, "gsm8k"},
		{"MATH", BenchmarkTypeMATH, "math"},
		{"Custom", BenchmarkTypeCustom, "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.value))
		})
	}
}

func TestDifficultyLevel_Values(t *testing.T) {
	tests := []struct {
		name     string
		value    DifficultyLevel
		expected string
	}{
		{"Easy", DifficultyEasy, "easy"},
		{"Medium", DifficultyMedium, "medium"},
		{"Hard", DifficultyHard, "hard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.value))
		})
	}
}

func TestBenchmarkStatus_Values(t *testing.T) {
	tests := []struct {
		name     string
		value    BenchmarkStatus
		expected string
	}{
		{"Pending", BenchmarkStatusPending, "pending"},
		{"Running", BenchmarkStatusRunning, "running"},
		{"Completed", BenchmarkStatusCompleted, "completed"},
		{"Failed", BenchmarkStatusFailed, "failed"},
		{"Cancelled", BenchmarkStatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.value))
		})
	}
}

// --- DefaultBenchmarkConfig tests ---

func TestDefaultBenchmarkConfig_ReturnsNonNil(t *testing.T) {
	cfg := DefaultBenchmarkConfig()
	require.NotNil(t, cfg)
}

func TestDefaultBenchmarkConfig_Values(t *testing.T) {
	cfg := DefaultBenchmarkConfig()

	assert.Equal(t, 0, cfg.MaxTasks, "MaxTasks should default to 0 (no limit)")
	assert.Equal(t, 5*time.Minute, cfg.Timeout, "Timeout should default to 5 minutes")
	assert.Equal(t, 4, cfg.Concurrency, "Concurrency should default to 4")
	assert.Equal(t, 1, cfg.Retries, "Retries should default to 1")
	assert.Equal(t, 0.0, cfg.Temperature, "Temperature should default to 0.0")
	assert.Equal(t, 4096, cfg.MaxTokens, "MaxTokens should default to 4096")
	assert.True(t, cfg.SaveResponses, "SaveResponses should default to true")
	assert.False(t, cfg.UseDebateForEval, "UseDebateForEval should default to false")
	assert.Nil(t, cfg.Difficulties, "Difficulties should be nil by default")
	assert.Nil(t, cfg.Tags, "Tags should be nil by default")
	assert.Empty(t, cfg.SystemPrompt, "SystemPrompt should be empty by default")
}

func TestDefaultBenchmarkConfig_IndependentInstances(t *testing.T) {
	cfg1 := DefaultBenchmarkConfig()
	cfg2 := DefaultBenchmarkConfig()
	cfg1.MaxTasks = 99
	assert.NotEqual(t, cfg1.MaxTasks, cfg2.MaxTasks,
		"modifying one instance should not affect another")
}

// --- Struct construction / zero-value tests ---

func TestBenchmarkTask_ZeroValue(t *testing.T) {
	var task BenchmarkTask
	assert.Empty(t, task.ID)
	assert.Empty(t, task.BenchmarkID)
	assert.Empty(t, string(task.Type))
	assert.Empty(t, task.Name)
	assert.Empty(t, task.Description)
	assert.Empty(t, task.Prompt)
	assert.Empty(t, task.Context)
	assert.Empty(t, task.Expected)
	assert.Nil(t, task.TestCases)
	assert.Empty(t, string(task.Difficulty))
	assert.Nil(t, task.Tags)
	assert.Nil(t, task.Metadata)
	assert.Equal(t, time.Duration(0), task.TimeLimit)
}

func TestBenchmarkTask_FullConstruction(t *testing.T) {
	tc := &TestCase{ID: "tc1", Input: "input", Expected: "output", Hidden: true}
	meta := map[string]interface{}{"key": "val"}
	task := BenchmarkTask{
		ID:          "task-1",
		BenchmarkID: "bench-1",
		Type:        BenchmarkTypeSWEBench,
		Name:        "My Task",
		Description: "A description",
		Prompt:      "Do something",
		Context:     "Some context",
		Expected:    "Expected output",
		TestCases:   []*TestCase{tc},
		Difficulty:  DifficultyHard,
		Tags:        []string{"tag1", "tag2"},
		Metadata:    meta,
		TimeLimit:   30 * time.Second,
	}

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "bench-1", task.BenchmarkID)
	assert.Equal(t, BenchmarkTypeSWEBench, task.Type)
	assert.Equal(t, "My Task", task.Name)
	assert.Equal(t, "A description", task.Description)
	assert.Equal(t, "Do something", task.Prompt)
	assert.Equal(t, "Some context", task.Context)
	assert.Equal(t, "Expected output", task.Expected)
	require.Len(t, task.TestCases, 1)
	assert.True(t, task.TestCases[0].Hidden)
	assert.Equal(t, DifficultyHard, task.Difficulty)
	assert.ElementsMatch(t, []string{"tag1", "tag2"}, task.Tags)
	assert.Equal(t, "val", task.Metadata["key"])
	assert.Equal(t, 30*time.Second, task.TimeLimit)
}

func TestTestCase_Fields(t *testing.T) {
	tc := TestCase{
		ID:       "tc-1",
		Input:    "x=5",
		Expected: "25",
		Hidden:   false,
	}
	assert.Equal(t, "tc-1", tc.ID)
	assert.Equal(t, "x=5", tc.Input)
	assert.Equal(t, "25", tc.Expected)
	assert.False(t, tc.Hidden)
}

func TestBenchmarkResult_ZeroValue(t *testing.T) {
	var r BenchmarkResult
	assert.Empty(t, r.TaskID)
	assert.Empty(t, r.RunID)
	assert.Empty(t, r.ProviderName)
	assert.Empty(t, r.ModelName)
	assert.Empty(t, r.Response)
	assert.False(t, r.Passed)
	assert.Equal(t, 0.0, r.Score)
	assert.Equal(t, time.Duration(0), r.Latency)
	assert.Equal(t, 0, r.TokensUsed)
	assert.Nil(t, r.TestResults)
	assert.Empty(t, r.Error)
	assert.Nil(t, r.Metadata)
	assert.True(t, r.CreatedAt.IsZero())
}

func TestBenchmarkResult_FullConstruction(t *testing.T) {
	now := time.Now()
	r := BenchmarkResult{
		TaskID:       "t1",
		RunID:        "r1",
		ProviderName: "openai",
		ModelName:    "gpt-4",
		Response:     "response text",
		Passed:       true,
		Score:        0.95,
		Latency:      500 * time.Millisecond,
		TokensUsed:   100,
		TestResults: []*TestCaseResult{
			{TestCaseID: "tc1", Passed: true, Actual: "ok", Expected: "ok"},
		},
		Error:     "",
		Metadata:  map[string]interface{}{"attempt": 1},
		CreatedAt: now,
	}

	assert.Equal(t, "t1", r.TaskID)
	assert.True(t, r.Passed)
	assert.Equal(t, 0.95, r.Score)
	assert.Equal(t, 500*time.Millisecond, r.Latency)
	assert.Equal(t, 100, r.TokensUsed)
	require.Len(t, r.TestResults, 1)
	assert.Equal(t, "tc1", r.TestResults[0].TestCaseID)
}

func TestTestCaseResult_Fields(t *testing.T) {
	tcr := TestCaseResult{
		TestCaseID: "tc-1",
		Passed:     false,
		Actual:     "42",
		Expected:   "43",
		Error:      "mismatch",
	}
	assert.Equal(t, "tc-1", tcr.TestCaseID)
	assert.False(t, tcr.Passed)
	assert.Equal(t, "42", tcr.Actual)
	assert.Equal(t, "43", tcr.Expected)
	assert.Equal(t, "mismatch", tcr.Error)
}

func TestBenchmarkRun_ZeroValue(t *testing.T) {
	var run BenchmarkRun
	assert.Empty(t, run.ID)
	assert.Empty(t, run.Name)
	assert.Empty(t, run.Description)
	assert.Empty(t, string(run.BenchmarkType))
	assert.Empty(t, run.ProviderName)
	assert.Empty(t, run.ModelName)
	assert.Empty(t, string(run.Status))
	assert.Nil(t, run.Config)
	assert.Nil(t, run.Results)
	assert.Nil(t, run.Summary)
	assert.Nil(t, run.StartTime)
	assert.Nil(t, run.EndTime)
	assert.True(t, run.CreatedAt.IsZero())
}

func TestBenchmarkRun_FullConstruction(t *testing.T) {
	now := time.Now()
	cfg := DefaultBenchmarkConfig()
	run := BenchmarkRun{
		ID:            "run-1",
		Name:          "My Run",
		Description:   "A test run",
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "openai",
		ModelName:     "gpt-4",
		Status:        BenchmarkStatusPending,
		Config:        cfg,
		Results:       []*BenchmarkResult{},
		Summary:       &BenchmarkSummary{TotalTasks: 0},
		StartTime:     &now,
		EndTime:       nil,
		CreatedAt:     now,
	}

	assert.Equal(t, "run-1", run.ID)
	assert.Equal(t, BenchmarkTypeMMLU, run.BenchmarkType)
	assert.Equal(t, BenchmarkStatusPending, run.Status)
	assert.NotNil(t, run.Config)
	assert.NotNil(t, run.StartTime)
	assert.Nil(t, run.EndTime)
}

func TestBenchmarkSummary_ZeroValue(t *testing.T) {
	var s BenchmarkSummary
	assert.Equal(t, 0, s.TotalTasks)
	assert.Equal(t, 0, s.PassedTasks)
	assert.Equal(t, 0, s.FailedTasks)
	assert.Equal(t, 0, s.ErrorTasks)
	assert.Equal(t, 0.0, s.PassRate)
	assert.Equal(t, 0.0, s.AverageScore)
	assert.Equal(t, time.Duration(0), s.AverageLatency)
	assert.Equal(t, 0, s.TotalTokens)
	assert.Nil(t, s.ByDifficulty)
	assert.Nil(t, s.ByTag)
}

func TestBenchmark_Fields(t *testing.T) {
	now := time.Now()
	b := Benchmark{
		ID:          "bench-1",
		Type:        BenchmarkTypeCustom,
		Name:        "Custom Bench",
		Description: "A custom benchmark",
		Version:     "2.0.0",
		TaskCount:   10,
		Metadata:    map[string]interface{}{"source": "tests"},
		CreatedAt:   now,
	}

	assert.Equal(t, "bench-1", b.ID)
	assert.Equal(t, BenchmarkTypeCustom, b.Type)
	assert.Equal(t, "Custom Bench", b.Name)
	assert.Equal(t, "A custom benchmark", b.Description)
	assert.Equal(t, "2.0.0", b.Version)
	assert.Equal(t, 10, b.TaskCount)
	assert.Equal(t, "tests", b.Metadata["source"])
	assert.False(t, b.CreatedAt.IsZero())
}

func TestRunFilter_ZeroValue(t *testing.T) {
	var f RunFilter
	assert.Empty(t, string(f.BenchmarkType))
	assert.Empty(t, f.ProviderName)
	assert.Empty(t, f.ModelName)
	assert.Empty(t, string(f.Status))
	assert.Nil(t, f.StartTime)
	assert.Nil(t, f.EndTime)
	assert.Equal(t, 0, f.Limit)
}

func TestRunFilter_FullConstruction(t *testing.T) {
	now := time.Now()
	f := RunFilter{
		BenchmarkType: BenchmarkTypeMMLU,
		ProviderName:  "openai",
		ModelName:     "gpt-4",
		Status:        BenchmarkStatusCompleted,
		StartTime:     &now,
		EndTime:       &now,
		Limit:         5,
	}
	assert.Equal(t, BenchmarkTypeMMLU, f.BenchmarkType)
	assert.Equal(t, "openai", f.ProviderName)
	assert.Equal(t, "gpt-4", f.ModelName)
	assert.Equal(t, BenchmarkStatusCompleted, f.Status)
	assert.NotNil(t, f.StartTime)
	assert.NotNil(t, f.EndTime)
	assert.Equal(t, 5, f.Limit)
}

func TestRunComparison_Fields(t *testing.T) {
	rc := RunComparison{
		Run1ID:         "r1",
		Run2ID:         "r2",
		PassRateChange: 0.1,
		ScoreChange:    0.05,
		LatencyChange:  -100 * time.Millisecond,
		Regressions:    []string{"task-1"},
		Improvements:   []string{"task-2", "task-3"},
		Summary:        "Run 2 improved",
	}

	assert.Equal(t, "r1", rc.Run1ID)
	assert.Equal(t, "r2", rc.Run2ID)
	assert.Equal(t, 0.1, rc.PassRateChange)
	assert.Equal(t, 0.05, rc.ScoreChange)
	assert.Equal(t, -100*time.Millisecond, rc.LatencyChange)
	assert.ElementsMatch(t, []string{"task-1"}, rc.Regressions)
	assert.ElementsMatch(t, []string{"task-2", "task-3"}, rc.Improvements)
	assert.Equal(t, "Run 2 improved", rc.Summary)
}

func TestDifficultySummary_Fields(t *testing.T) {
	ds := DifficultySummary{Total: 10, Passed: 8, PassRate: 0.8}
	assert.Equal(t, 10, ds.Total)
	assert.Equal(t, 8, ds.Passed)
	assert.Equal(t, 0.8, ds.PassRate)
}

func TestTagSummary_Fields(t *testing.T) {
	ts := TagSummary{Total: 5, Passed: 3, PassRate: 0.6}
	assert.Equal(t, 5, ts.Total)
	assert.Equal(t, 3, ts.Passed)
	assert.Equal(t, 0.6, ts.PassRate)
}

// --- Interface satisfaction compile-time checks ---

// Verify that LLMProvider, CodeExecutor, DebateEvaluator are valid interfaces
// by asserting mock implementations satisfy them at compile time.

type testLLMProvider struct{}

func (p *testLLMProvider) Complete(_ context.Context, _, _ string) (string, int, error) {
	return "", 0, nil
}
func (p *testLLMProvider) GetName() string { return "test" }

type testCodeExecutor struct{}

func (e *testCodeExecutor) Execute(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (e *testCodeExecutor) Validate(_ context.Context, _, _ string, _ []*TestCase) ([]*TestCaseResult, error) {
	return nil, nil
}

type testDebateEvaluator struct{}

func (d *testDebateEvaluator) EvaluateResponse(_ context.Context, _ *BenchmarkTask, _ string) (float64, bool, error) {
	return 0, false, nil
}

var _ LLMProvider = (*testLLMProvider)(nil)
var _ CodeExecutor = (*testCodeExecutor)(nil)
var _ DebateEvaluator = (*testDebateEvaluator)(nil)

func TestLLMProvider_InterfaceSatisfaction(t *testing.T) {
	var p LLMProvider = &testLLMProvider{}
	assert.NotNil(t, p)
	assert.Equal(t, "test", p.GetName())
}

func TestCodeExecutor_InterfaceSatisfaction(t *testing.T) {
	var e CodeExecutor = &testCodeExecutor{}
	assert.NotNil(t, e)
}

func TestDebateEvaluator_InterfaceSatisfaction(t *testing.T) {
	var d DebateEvaluator = &testDebateEvaluator{}
	assert.NotNil(t, d)
}

// --- BenchmarkRunner interface compile-time check ---

func TestBenchmarkRunner_InterfaceSatisfaction(t *testing.T) {
	runner := NewStandardBenchmarkRunner(nil, nil)
	var _ BenchmarkRunner = runner
	assert.NotNil(t, runner)
}
