package benchmark

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// StandardBenchmarkRunner implements BenchmarkRunner
type StandardBenchmarkRunner struct {
	benchmarks   map[string]*Benchmark
	tasks        map[string][]*BenchmarkTask // benchmark ID -> tasks
	runs         map[string]*BenchmarkRun
	provider     LLMProvider
	codeExecutor CodeExecutor
	debateEval   DebateEvaluator
	mu           sync.RWMutex
	logger       *logrus.Logger
}

// NewStandardBenchmarkRunner creates a new benchmark runner
func NewStandardBenchmarkRunner(provider LLMProvider, logger *logrus.Logger) *StandardBenchmarkRunner {
	if logger == nil {
		logger = logrus.New()
	}

	runner := &StandardBenchmarkRunner{
		benchmarks: make(map[string]*Benchmark),
		tasks:      make(map[string][]*BenchmarkTask),
		runs:       make(map[string]*BenchmarkRun),
		provider:   provider,
		logger:     logger,
	}

	// Initialize built-in benchmarks
	runner.initBuiltInBenchmarks()

	return runner
}

func (r *StandardBenchmarkRunner) initBuiltInBenchmarks() {
	// SWE-Bench (simplified)
	sweBench := &Benchmark{
		ID:          "swe-bench-lite",
		Type:        BenchmarkTypeSWEBench,
		Name:        "SWE-Bench Lite",
		Description: "Simplified software engineering benchmark tasks",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}
	r.benchmarks[sweBench.ID] = sweBench
	r.tasks[sweBench.ID] = r.createSWEBenchTasks()
	sweBench.TaskCount = len(r.tasks[sweBench.ID])

	// HumanEval
	humanEval := &Benchmark{
		ID:          "humaneval",
		Type:        BenchmarkTypeHumanEval,
		Name:        "HumanEval",
		Description: "Code generation benchmark from OpenAI",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}
	r.benchmarks[humanEval.ID] = humanEval
	r.tasks[humanEval.ID] = r.createHumanEvalTasks()
	humanEval.TaskCount = len(r.tasks[humanEval.ID])

	// MMLU
	mmlu := &Benchmark{
		ID:          "mmlu-mini",
		Type:        BenchmarkTypeMMLU,
		Name:        "MMLU Mini",
		Description: "Subset of MMLU benchmark for quick evaluation",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}
	r.benchmarks[mmlu.ID] = mmlu
	r.tasks[mmlu.ID] = r.createMMLUTasks()
	mmlu.TaskCount = len(r.tasks[mmlu.ID])

	// GSM8K
	gsm8k := &Benchmark{
		ID:          "gsm8k-mini",
		Type:        BenchmarkTypeGSM8K,
		Name:        "GSM8K Mini",
		Description: "Subset of GSM8K math benchmark",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
	}
	r.benchmarks[gsm8k.ID] = gsm8k
	r.tasks[gsm8k.ID] = r.createGSM8KTasks()
	gsm8k.TaskCount = len(r.tasks[gsm8k.ID])
}

func (r *StandardBenchmarkRunner) createSWEBenchTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{
			ID:          "swe-001",
			Type:        BenchmarkTypeSWEBench,
			Name:        "Fix null pointer exception",
			Description: "Fix the null pointer exception in the user service",
			Difficulty:  DifficultyEasy,
			Prompt: `Fix the bug in this code that causes a null pointer exception:

func GetUserName(user *User) string {
    return user.Name
}

The function should handle the case when user is nil.`,
			Expected: `func GetUserName(user *User) string {
    if user == nil {
        return ""
    }
    return user.Name
}`,
			Tags: []string{"bug-fix", "go", "null-safety"},
		},
		{
			ID:          "swe-002",
			Type:        BenchmarkTypeSWEBench,
			Name:        "Add error handling",
			Description: "Add proper error handling to the file reader",
			Difficulty:  DifficultyMedium,
			Prompt: `Add error handling to this function:

func ReadConfig(path string) Config {
    data, _ := os.ReadFile(path)
    var config Config
    json.Unmarshal(data, &config)
    return config
}

Return an error if reading or parsing fails.`,
			Expected: `func ReadConfig(path string) (Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Config{}, fmt.Errorf("failed to read config: %w", err)
    }
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return Config{}, fmt.Errorf("failed to parse config: %w", err)
    }
    return config, nil
}`,
			Tags: []string{"error-handling", "go", "best-practices"},
		},
		{
			ID:          "swe-003",
			Type:        BenchmarkTypeSWEBench,
			Name:        "Implement retry logic",
			Description: "Add retry logic with exponential backoff",
			Difficulty:  DifficultyHard,
			Prompt: `Implement a retry function with exponential backoff:

// Retry calls fn up to maxRetries times with exponential backoff starting at baseDelay
// Returns the result of fn or the last error if all retries fail
func Retry(fn func() error, maxRetries int, baseDelay time.Duration) error

Requirements:
- Double the delay after each failed attempt
- Add jitter (random 0-100ms) to prevent thundering herd
- Stop retrying if fn returns nil`,
			Tags: []string{"implementation", "go", "resilience"},
		},
	}
}

func (r *StandardBenchmarkRunner) createHumanEvalTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{
			ID:          "he-001",
			Type:        BenchmarkTypeHumanEval,
			Name:        "has_close_elements",
			Description: "Check if any two elements are closer than threshold",
			Difficulty:  DifficultyEasy,
			Prompt: `Write a function that checks if in given list of numbers, any two numbers are closer to each other than given threshold.

def has_close_elements(numbers: List[float], threshold: float) -> bool:
    """
    >>> has_close_elements([1.0, 2.0, 3.0], 0.5)
    False
    >>> has_close_elements([1.0, 2.8, 3.0, 4.0, 5.0, 2.0], 0.3)
    True
    """`,
			TestCases: []*TestCase{
				{ID: "tc1", Input: "[1.0, 2.0, 3.0], 0.5", Expected: "False"},
				{ID: "tc2", Input: "[1.0, 2.8, 3.0, 4.0, 5.0, 2.0], 0.3", Expected: "True"},
				{ID: "tc3", Input: "[1.0, 2.0, 3.9, 4.0, 5.0, 2.2], 0.3", Expected: "True"},
			},
			Tags: []string{"python", "list", "comparison"},
		},
		{
			ID:          "he-002",
			Type:        BenchmarkTypeHumanEval,
			Name:        "separate_paren_groups",
			Description: "Separate balanced parentheses groups",
			Difficulty:  DifficultyMedium,
			Prompt: `Separate parentheses groups. Input string contains balanced parentheses.

def separate_paren_groups(paren_string: str) -> List[str]:
    """
    >>> separate_paren_groups('( ) (( )) (( )( ))')
    ['()', '(())', '(()())']
    """`,
			TestCases: []*TestCase{
				{ID: "tc1", Input: "'( ) (( )) (( )( ))'", Expected: "['()', '(())', '(()())']"},
				{ID: "tc2", Input: "'(()) ((()))'", Expected: "['(())', '((()))']"},
			},
			Tags: []string{"python", "string", "parsing"},
		},
	}
}

func (r *StandardBenchmarkRunner) createMMLUTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{
			ID:          "mmlu-001",
			Type:        BenchmarkTypeMMLU,
			Name:        "Computer Science - Algorithms",
			Description: "Multiple choice question on algorithms",
			Difficulty:  DifficultyMedium,
			Prompt: `What is the time complexity of binary search on a sorted array of n elements?

A) O(1)
B) O(log n)
C) O(n)
D) O(n log n)

Answer with just the letter.`,
			Expected: "B",
			Tags:     []string{"computer-science", "algorithms", "multiple-choice"},
		},
		{
			ID:          "mmlu-002",
			Type:        BenchmarkTypeMMLU,
			Name:        "Mathematics - Calculus",
			Description: "Multiple choice question on calculus",
			Difficulty:  DifficultyMedium,
			Prompt: `What is the derivative of f(x) = x^3 + 2x^2 - 5x + 1?

A) 3x^2 + 4x - 5
B) 3x^2 + 2x - 5
C) x^2 + 4x - 5
D) 3x^3 + 4x^2 - 5x

Answer with just the letter.`,
			Expected: "A",
			Tags:     []string{"mathematics", "calculus", "multiple-choice"},
		},
		{
			ID:          "mmlu-003",
			Type:        BenchmarkTypeMMLU,
			Name:        "Physics - Mechanics",
			Description: "Multiple choice question on mechanics",
			Difficulty:  DifficultyMedium,
			Prompt: `A ball is thrown vertically upward with initial velocity v. What is its velocity at the highest point?

A) v
B) v/2
C) 0
D) -v

Answer with just the letter.`,
			Expected: "C",
			Tags:     []string{"physics", "mechanics", "multiple-choice"},
		},
	}
}

func (r *StandardBenchmarkRunner) createGSM8KTasks() []*BenchmarkTask {
	return []*BenchmarkTask{
		{
			ID:          "gsm8k-001",
			Type:        BenchmarkTypeGSM8K,
			Name:        "Basic arithmetic word problem",
			Description: "Solve a basic arithmetic word problem",
			Difficulty:  DifficultyEasy,
			Prompt: `Janet's ducks lay 16 eggs per day. She eats three for breakfast every morning and bakes muffins for her friends every day with four. She sells the remainder at the farmers' market daily for $2 per fresh duck egg. How much in dollars does she make every day at the farmers' market?

Think step by step and provide the final numerical answer.`,
			Expected: "18",
			Tags:     []string{"math", "arithmetic", "word-problem"},
		},
		{
			ID:          "gsm8k-002",
			Type:        BenchmarkTypeGSM8K,
			Name:        "Multi-step calculation",
			Description: "Solve a multi-step calculation problem",
			Difficulty:  DifficultyMedium,
			Prompt: `A farmer has 3 fields. The first field produces 200 pounds of wheat per acre. The second field produces 250 pounds per acre. The third field produces 300 pounds per acre. If the first field is 10 acres, the second field is 8 acres, and the third field is 6 acres, how many total pounds of wheat does the farmer produce?

Think step by step and provide the final numerical answer.`,
			Expected: "5800",
			Tags:     []string{"math", "multiplication", "addition"},
		},
	}
}

// SetCodeExecutor sets the code executor for running tests
func (r *StandardBenchmarkRunner) SetCodeExecutor(executor CodeExecutor) {
	r.codeExecutor = executor
}

// SetDebateEvaluator sets the debate evaluator
func (r *StandardBenchmarkRunner) SetDebateEvaluator(evaluator DebateEvaluator) {
	r.debateEval = evaluator
}

// ListBenchmarks lists available benchmarks
func (r *StandardBenchmarkRunner) ListBenchmarks(ctx context.Context) ([]*Benchmark, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Benchmark, 0, len(r.benchmarks))
	for _, b := range r.benchmarks {
		result = append(result, b)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetBenchmark gets a benchmark by ID
func (r *StandardBenchmarkRunner) GetBenchmark(ctx context.Context, id string) (*Benchmark, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.benchmarks[id]
	if !ok {
		return nil, fmt.Errorf("benchmark not found: %s", id)
	}

	return b, nil
}

// GetTasks gets tasks for a benchmark
func (r *StandardBenchmarkRunner) GetTasks(ctx context.Context, benchmarkID string, config *BenchmarkConfig) ([]*BenchmarkTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks, ok := r.tasks[benchmarkID]
	if !ok {
		return nil, fmt.Errorf("benchmark not found: %s", benchmarkID)
	}

	// Apply filters
	if config != nil {
		filtered := make([]*BenchmarkTask, 0)
		for _, task := range tasks {
			if len(config.Difficulties) > 0 {
				found := false
				for _, d := range config.Difficulties {
					if task.Difficulty == d {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			if len(config.Tags) > 0 {
				found := false
				for _, filterTag := range config.Tags {
					for _, taskTag := range task.Tags {
						if taskTag == filterTag {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					continue
				}
			}

			filtered = append(filtered, task)
		}
		tasks = filtered

		// Apply limit
		if config.MaxTasks > 0 && len(tasks) > config.MaxTasks {
			tasks = tasks[:config.MaxTasks]
		}
	}

	return tasks, nil
}

// CreateRun creates a new benchmark run
func (r *StandardBenchmarkRunner) CreateRun(ctx context.Context, run *BenchmarkRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	if run.Config == nil {
		run.Config = DefaultBenchmarkConfig()
	}

	run.Status = BenchmarkStatusPending
	run.CreatedAt = time.Now()

	r.runs[run.ID] = run

	r.logger.WithFields(logrus.Fields{
		"run_id":    run.ID,
		"benchmark": run.BenchmarkType,
		"provider":  run.ProviderName,
	}).Info("Benchmark run created")

	return nil
}

// StartRun starts a benchmark run
func (r *StandardBenchmarkRunner) StartRun(ctx context.Context, runID string) error {
	r.mu.Lock()
	run, ok := r.runs[runID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("run not found: %s", runID)
	}

	if run.Status != BenchmarkStatusPending {
		r.mu.Unlock()
		return fmt.Errorf("run already started or completed")
	}

	now := time.Now()
	run.Status = BenchmarkStatusRunning
	run.StartTime = &now

	// Get benchmark ID from type
	var benchmarkID string
	for id, b := range r.benchmarks {
		if b.Type == run.BenchmarkType {
			benchmarkID = id
			break
		}
	}
	r.mu.Unlock()

	// Get tasks
	tasks, err := r.GetTasks(ctx, benchmarkID, run.Config)
	if err != nil {
		return err
	}

	// Run benchmark asynchronously
	go r.executeRun(ctx, run, tasks)

	return nil
}

func (r *StandardBenchmarkRunner) executeRun(ctx context.Context, run *BenchmarkRun, tasks []*BenchmarkTask) {
	results := make([]*BenchmarkResult, 0, len(tasks))

	// Create worker pool for concurrent execution
	concurrency := run.Config.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	taskCh := make(chan *BenchmarkTask, len(tasks))
	resultCh := make(chan *BenchmarkResult, len(tasks))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				result := r.executeTask(ctx, run, task)
				resultCh <- result
			}
		}()
	}

	// Send tasks
	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	for result := range resultCh {
		results = append(results, result)
	}

	// Update run
	r.mu.Lock()
	now := time.Now()
	run.Status = BenchmarkStatusCompleted
	run.EndTime = &now
	run.Results = results
	run.Summary = r.calculateSummary(results, tasks)
	r.mu.Unlock()

	r.logger.WithFields(logrus.Fields{
		"run_id":    run.ID,
		"tasks":     len(tasks),
		"pass_rate": run.Summary.PassRate,
	}).Info("Benchmark run completed")
}

func (r *StandardBenchmarkRunner) executeTask(ctx context.Context, run *BenchmarkRun, task *BenchmarkTask) *BenchmarkResult {
	start := time.Now()

	result := &BenchmarkResult{
		TaskID:       task.ID,
		RunID:        run.ID,
		ProviderName: run.ProviderName,
		ModelName:    run.ModelName,
		CreatedAt:    time.Now(),
	}

	// Apply timeout
	timeout := run.Config.Timeout
	if task.TimeLimit > 0 {
		timeout = task.TimeLimit
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get response from LLM
	if r.provider != nil {
		response, tokens, err := r.provider.Complete(ctx, task.Prompt, run.Config.SystemPrompt)
		if err != nil {
			result.Error = err.Error()
			result.Passed = false
			result.Latency = time.Since(start)
			return result
		}

		result.Response = response
		result.TokensUsed = tokens
	} else {
		result.Response = "no provider available"
	}

	result.Latency = time.Since(start)

	// Evaluate response
	passed, score := r.evaluateResponse(ctx, run, task, result.Response)
	result.Passed = passed
	result.Score = score

	return result
}

func (r *StandardBenchmarkRunner) evaluateResponse(ctx context.Context, run *BenchmarkRun, task *BenchmarkTask, response string) (bool, float64) {
	// Use debate evaluation if enabled
	if run.Config.UseDebateForEval && r.debateEval != nil {
		score, passed, err := r.debateEval.EvaluateResponse(ctx, task, response)
		if err == nil {
			return passed, score
		}
		r.logger.WithError(err).Warn("Debate evaluation failed, using default")
	}

	// Code execution evaluation
	if task.TestCases != nil && len(task.TestCases) > 0 && r.codeExecutor != nil {
		testResults, err := r.codeExecutor.Validate(ctx, response, "python", task.TestCases)
		if err != nil {
			return false, 0
		}

		passed := 0
		for _, tr := range testResults {
			if tr.Passed {
				passed++
			}
		}

		score := float64(passed) / float64(len(task.TestCases))
		return score >= 0.5, score
	}

	// Simple string matching for expected output
	if task.Expected != "" {
		// Normalize strings
		normResponse := strings.TrimSpace(strings.ToLower(response))
		normExpected := strings.TrimSpace(strings.ToLower(task.Expected))

		// Extract answer if present
		if strings.Contains(normResponse, "answer") {
			parts := strings.Split(normResponse, "answer")
			if len(parts) > 1 {
				normResponse = strings.TrimSpace(parts[len(parts)-1])
				// Clean up common patterns
				normResponse = strings.TrimPrefix(normResponse, ":")
				normResponse = strings.TrimPrefix(normResponse, "is")
				normResponse = strings.TrimSpace(normResponse)
			}
		}

		// Check containment
		if strings.Contains(normResponse, normExpected) {
			return true, 1.0
		}

		// For multiple choice, check just the letter
		if len(normExpected) == 1 && isLetter(normExpected[0]) {
			// Look for the letter at the start or standalone
			if len(normResponse) > 0 && (normResponse[0] == normExpected[0] ||
				strings.HasPrefix(normResponse, normExpected)) {
				return true, 1.0
			}
		}

		return false, 0.0
	}

	// Default: consider it passed if response is non-empty
	if len(strings.TrimSpace(response)) > 0 {
		return true, 0.5
	}

	return false, 0.0
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (r *StandardBenchmarkRunner) calculateSummary(results []*BenchmarkResult, tasks []*BenchmarkTask) *BenchmarkSummary {
	summary := &BenchmarkSummary{
		TotalTasks:   len(results),
		ByDifficulty: make(map[DifficultyLevel]*DifficultySummary),
		ByTag:        make(map[string]*TagSummary),
	}

	// Build task map for metadata
	taskMap := make(map[string]*BenchmarkTask)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	var totalScore float64
	var totalLatency time.Duration

	for _, result := range results {
		if result.Passed {
			summary.PassedTasks++
		} else if result.Error != "" {
			summary.ErrorTasks++
		} else {
			summary.FailedTasks++
		}

		totalScore += result.Score
		totalLatency += result.Latency
		summary.TotalTokens += result.TokensUsed

		// By difficulty
		if task, ok := taskMap[result.TaskID]; ok {
			if summary.ByDifficulty[task.Difficulty] == nil {
				summary.ByDifficulty[task.Difficulty] = &DifficultySummary{}
			}
			summary.ByDifficulty[task.Difficulty].Total++
			if result.Passed {
				summary.ByDifficulty[task.Difficulty].Passed++
			}

			// By tag
			for _, tag := range task.Tags {
				if summary.ByTag[tag] == nil {
					summary.ByTag[tag] = &TagSummary{}
				}
				summary.ByTag[tag].Total++
				if result.Passed {
					summary.ByTag[tag].Passed++
				}
			}
		}
	}

	if summary.TotalTasks > 0 {
		summary.PassRate = float64(summary.PassedTasks) / float64(summary.TotalTasks)
		summary.AverageScore = totalScore / float64(summary.TotalTasks)
		summary.AverageLatency = totalLatency / time.Duration(summary.TotalTasks)
	}

	// Calculate pass rates
	for _, ds := range summary.ByDifficulty {
		if ds.Total > 0 {
			ds.PassRate = float64(ds.Passed) / float64(ds.Total)
		}
	}
	for _, ts := range summary.ByTag {
		if ts.Total > 0 {
			ts.PassRate = float64(ts.Passed) / float64(ts.Total)
		}
	}

	return summary
}

// GetRun gets a benchmark run
func (r *StandardBenchmarkRunner) GetRun(ctx context.Context, runID string) (*BenchmarkRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, ok := r.runs[runID]
	if !ok {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	return run, nil
}

// ListRuns lists benchmark runs
func (r *StandardBenchmarkRunner) ListRuns(ctx context.Context, filter *RunFilter) ([]*BenchmarkRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*BenchmarkRun
	for _, run := range r.runs {
		if r.matchesFilter(run, filter) {
			result = append(result, run)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if filter != nil && filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *StandardBenchmarkRunner) matchesFilter(run *BenchmarkRun, filter *RunFilter) bool {
	if filter == nil {
		return true
	}

	if filter.BenchmarkType != "" && run.BenchmarkType != filter.BenchmarkType {
		return false
	}
	if filter.ProviderName != "" && run.ProviderName != filter.ProviderName {
		return false
	}
	if filter.ModelName != "" && run.ModelName != filter.ModelName {
		return false
	}
	if filter.Status != "" && run.Status != filter.Status {
		return false
	}

	return true
}

// CancelRun cancels a benchmark run
func (r *StandardBenchmarkRunner) CancelRun(ctx context.Context, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	run, ok := r.runs[runID]
	if !ok {
		return fmt.Errorf("run not found: %s", runID)
	}

	if run.Status != BenchmarkStatusRunning && run.Status != BenchmarkStatusPending {
		return fmt.Errorf("cannot cancel run in status: %s", run.Status)
	}

	now := time.Now()
	run.Status = BenchmarkStatusCancelled
	run.EndTime = &now

	return nil
}

// CompareRuns compares two benchmark runs
func (r *StandardBenchmarkRunner) CompareRuns(ctx context.Context, runID1, runID2 string) (*RunComparison, error) {
	r.mu.RLock()
	run1, ok1 := r.runs[runID1]
	run2, ok2 := r.runs[runID2]
	r.mu.RUnlock()

	if !ok1 {
		return nil, fmt.Errorf("run not found: %s", runID1)
	}
	if !ok2 {
		return nil, fmt.Errorf("run not found: %s", runID2)
	}

	if run1.Summary == nil || run2.Summary == nil {
		return nil, fmt.Errorf("both runs must be completed")
	}

	comparison := &RunComparison{
		Run1ID:         runID1,
		Run2ID:         runID2,
		PassRateChange: run2.Summary.PassRate - run1.Summary.PassRate,
		ScoreChange:    run2.Summary.AverageScore - run1.Summary.AverageScore,
		LatencyChange:  run2.Summary.AverageLatency - run1.Summary.AverageLatency,
	}

	// Find regressions and improvements
	result1Map := make(map[string]*BenchmarkResult)
	for _, r := range run1.Results {
		result1Map[r.TaskID] = r
	}

	for _, r2 := range run2.Results {
		if r1, ok := result1Map[r2.TaskID]; ok {
			if r1.Passed && !r2.Passed {
				comparison.Regressions = append(comparison.Regressions, r2.TaskID)
			} else if !r1.Passed && r2.Passed {
				comparison.Improvements = append(comparison.Improvements, r2.TaskID)
			}
		}
	}

	// Generate summary
	if len(comparison.Regressions) > len(comparison.Improvements) {
		comparison.Summary = fmt.Sprintf("Run 2 regressed with %d regressions and %d improvements",
			len(comparison.Regressions), len(comparison.Improvements))
	} else if len(comparison.Improvements) > len(comparison.Regressions) {
		comparison.Summary = fmt.Sprintf("Run 2 improved with %d improvements and %d regressions",
			len(comparison.Improvements), len(comparison.Regressions))
	} else {
		comparison.Summary = "No significant difference between runs"
	}

	return comparison, nil
}

// AddBenchmark adds a custom benchmark
func (r *StandardBenchmarkRunner) AddBenchmark(benchmark *Benchmark, tasks []*BenchmarkTask) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.benchmarks[benchmark.ID] = benchmark
	r.tasks[benchmark.ID] = tasks
	benchmark.TaskCount = len(tasks)
}
