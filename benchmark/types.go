package benchmark

import (
	"context"
	"time"
)

// BenchmarkType represents types of benchmarks
type BenchmarkType string

const (
	BenchmarkTypeSWEBench  BenchmarkType = "swe-bench"
	BenchmarkTypeHumanEval BenchmarkType = "humaneval"
	BenchmarkTypeMBPP      BenchmarkType = "mbpp"
	BenchmarkTypeLMSYS     BenchmarkType = "lmsys"
	BenchmarkTypeHellaSwag BenchmarkType = "hellaswag"
	BenchmarkTypeMMLU      BenchmarkType = "mmlu"
	BenchmarkTypeGSM8K     BenchmarkType = "gsm8k"
	BenchmarkTypeMATH      BenchmarkType = "math"
	BenchmarkTypeCustom    BenchmarkType = "custom"
)

// DifficultyLevel represents task difficulty
type DifficultyLevel string

const (
	DifficultyEasy   DifficultyLevel = "easy"
	DifficultyMedium DifficultyLevel = "medium"
	DifficultyHard   DifficultyLevel = "hard"
)

// BenchmarkTask represents a single benchmark task
type BenchmarkTask struct {
	ID          string                 `json:"id"`
	BenchmarkID string                 `json:"benchmark_id"`
	Type        BenchmarkType          `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Prompt      string                 `json:"prompt"`
	Context     string                 `json:"context,omitempty"`    // Additional context (code, docs)
	Expected    string                 `json:"expected,omitempty"`   // Expected output/solution
	TestCases   []*TestCase            `json:"test_cases,omitempty"` // For code tasks
	Difficulty  DifficultyLevel        `json:"difficulty,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	TimeLimit   time.Duration          `json:"time_limit,omitempty"`
}

// TestCase represents a test case for code benchmarks
type TestCase struct {
	ID       string `json:"id"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Hidden   bool   `json:"hidden"`
}

// BenchmarkResult represents the result of running a benchmark task
type BenchmarkResult struct {
	TaskID       string                 `json:"task_id"`
	RunID        string                 `json:"run_id"`
	ProviderName string                 `json:"provider_name"`
	ModelName    string                 `json:"model_name"`
	Response     string                 `json:"response"`
	Passed       bool                   `json:"passed"`
	Score        float64                `json:"score"` // 0.0 to 1.0
	Latency      time.Duration          `json:"latency"`
	TokensUsed   int                    `json:"tokens_used"`
	TestResults  []*TestCaseResult      `json:"test_results,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// TestCaseResult represents result of a single test case
type TestCaseResult struct {
	TestCaseID string `json:"test_case_id"`
	Passed     bool   `json:"passed"`
	Actual     string `json:"actual"`
	Expected   string `json:"expected"`
	Error      string `json:"error,omitempty"`
}

// BenchmarkRun represents a complete benchmark run
type BenchmarkRun struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	BenchmarkType BenchmarkType      `json:"benchmark_type"`
	ProviderName  string             `json:"provider_name"`
	ModelName     string             `json:"model_name"`
	Status        BenchmarkStatus    `json:"status"`
	Config        *BenchmarkConfig   `json:"config"`
	Results       []*BenchmarkResult `json:"results,omitempty"`
	Summary       *BenchmarkSummary  `json:"summary,omitempty"`
	StartTime     *time.Time         `json:"start_time,omitempty"`
	EndTime       *time.Time         `json:"end_time,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

// BenchmarkStatus represents run status
type BenchmarkStatus string

const (
	BenchmarkStatusPending   BenchmarkStatus = "pending"
	BenchmarkStatusRunning   BenchmarkStatus = "running"
	BenchmarkStatusCompleted BenchmarkStatus = "completed"
	BenchmarkStatusFailed    BenchmarkStatus = "failed"
	BenchmarkStatusCancelled BenchmarkStatus = "cancelled"
)

// BenchmarkConfig configuration for benchmark runs
type BenchmarkConfig struct {
	MaxTasks         int               `json:"max_tasks,omitempty"`   // Limit number of tasks
	Timeout          time.Duration     `json:"timeout,omitempty"`     // Per-task timeout
	Concurrency      int               `json:"concurrency,omitempty"` // Parallel execution
	Retries          int               `json:"retries,omitempty"`     // Retry failed tasks
	Temperature      float64           `json:"temperature,omitempty"` // Model temperature
	MaxTokens        int               `json:"max_tokens,omitempty"`
	SystemPrompt     string            `json:"system_prompt,omitempty"`
	Difficulties     []DifficultyLevel `json:"difficulties,omitempty"` // Filter by difficulty
	Tags             []string          `json:"tags,omitempty"`         // Filter by tags
	SaveResponses    bool              `json:"save_responses"`         // Save full responses
	UseDebateForEval bool              `json:"use_debate_for_eval"`    // Use AI debate for evaluation
}

// DefaultBenchmarkConfig returns default configuration
func DefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		MaxTasks:      0, // No limit
		Timeout:       5 * time.Minute,
		Concurrency:   4,
		Retries:       1,
		Temperature:   0.0,
		MaxTokens:     4096,
		SaveResponses: true,
	}
}

// BenchmarkSummary summarizes benchmark results
type BenchmarkSummary struct {
	TotalTasks     int                                    `json:"total_tasks"`
	PassedTasks    int                                    `json:"passed_tasks"`
	FailedTasks    int                                    `json:"failed_tasks"`
	ErrorTasks     int                                    `json:"error_tasks"`
	PassRate       float64                                `json:"pass_rate"`
	AverageScore   float64                                `json:"average_score"`
	AverageLatency time.Duration                          `json:"average_latency"`
	TotalTokens    int                                    `json:"total_tokens"`
	ByDifficulty   map[DifficultyLevel]*DifficultySummary `json:"by_difficulty,omitempty"`
	ByTag          map[string]*TagSummary                 `json:"by_tag,omitempty"`
}

// DifficultySummary summary by difficulty level
type DifficultySummary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
}

// TagSummary summary by tag
type TagSummary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
}

// Benchmark represents a complete benchmark suite
type Benchmark struct {
	ID          string                 `json:"id"`
	Type        BenchmarkType          `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Version     string                 `json:"version"`
	TaskCount   int                    `json:"task_count"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// BenchmarkRunner runs benchmarks
type BenchmarkRunner interface {
	// ListBenchmarks lists available benchmarks
	ListBenchmarks(ctx context.Context) ([]*Benchmark, error)

	// GetBenchmark gets a benchmark by ID
	GetBenchmark(ctx context.Context, id string) (*Benchmark, error)

	// GetTasks gets tasks for a benchmark
	GetTasks(ctx context.Context, benchmarkID string, config *BenchmarkConfig) ([]*BenchmarkTask, error)

	// CreateRun creates a new benchmark run
	CreateRun(ctx context.Context, run *BenchmarkRun) error

	// StartRun starts a benchmark run
	StartRun(ctx context.Context, runID string) error

	// GetRun gets a benchmark run
	GetRun(ctx context.Context, runID string) (*BenchmarkRun, error)

	// ListRuns lists benchmark runs
	ListRuns(ctx context.Context, filter *RunFilter) ([]*BenchmarkRun, error)

	// CancelRun cancels a benchmark run
	CancelRun(ctx context.Context, runID string) error

	// CompareRuns compares two benchmark runs
	CompareRuns(ctx context.Context, runID1, runID2 string) (*RunComparison, error)
}

// RunFilter for filtering benchmark runs
type RunFilter struct {
	BenchmarkType BenchmarkType   `json:"benchmark_type,omitempty"`
	ProviderName  string          `json:"provider_name,omitempty"`
	ModelName     string          `json:"model_name,omitempty"`
	Status        BenchmarkStatus `json:"status,omitempty"`
	StartTime     *time.Time      `json:"start_time,omitempty"`
	EndTime       *time.Time      `json:"end_time,omitempty"`
	Limit         int             `json:"limit,omitempty"`
}

// RunComparison compares two benchmark runs
type RunComparison struct {
	Run1ID         string        `json:"run1_id"`
	Run2ID         string        `json:"run2_id"`
	PassRateChange float64       `json:"pass_rate_change"`
	ScoreChange    float64       `json:"score_change"`
	LatencyChange  time.Duration `json:"latency_change"`
	Regressions    []string      `json:"regressions,omitempty"`  // Tasks that regressed
	Improvements   []string      `json:"improvements,omitempty"` // Tasks that improved
	Summary        string        `json:"summary"`
}

// LLMProvider interface for benchmark execution
type LLMProvider interface {
	Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error) // Returns response, tokens, error
	GetName() string
}

// CodeExecutor interface for code execution
type CodeExecutor interface {
	Execute(ctx context.Context, code, language string, testInput string) (string, error)
	Validate(ctx context.Context, code, language string, testCases []*TestCase) ([]*TestCaseResult, error)
}

// DebateEvaluator interface for debate-based evaluation
type DebateEvaluator interface {
	EvaluateResponse(ctx context.Context, task *BenchmarkTask, response string) (float64, bool, error)
}
