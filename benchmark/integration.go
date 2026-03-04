package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BenchmarkSystem is the main orchestrator for benchmarking
type BenchmarkSystem struct {
	runner          BenchmarkRunner
	debateAdapter   *DebateAdapterForBenchmark
	verifierAdapter *VerifierAdapterForBenchmark
	providerAdapter *ProviderAdapterForBenchmark
	config          *BenchmarkSystemConfig
	logger          *logrus.Logger
	mu              sync.RWMutex
}

// BenchmarkSystemConfig configuration for benchmark system
type BenchmarkSystemConfig struct {
	EnableDebateEvaluation bool `json:"enable_debate_evaluation"`
	UseVerifierScores      bool `json:"use_verifier_scores"`
	AutoSelectProvider     bool `json:"auto_select_provider"`
	DefaultConcurrency     int  `json:"default_concurrency"`
}

// DefaultBenchmarkSystemConfig returns default configuration
func DefaultBenchmarkSystemConfig() *BenchmarkSystemConfig {
	return &BenchmarkSystemConfig{
		EnableDebateEvaluation: true,
		UseVerifierScores:      true,
		AutoSelectProvider:     true,
		DefaultConcurrency:     4,
	}
}

// DebateService interface for debate integration
type DebateServiceForBenchmark interface {
	RunDebate(ctx context.Context, topic string) (*DebateResultForBenchmark, error)
}

// DebateResultForBenchmark represents debate result
type DebateResultForBenchmark struct {
	ID         string             `json:"id"`
	Consensus  string             `json:"consensus"`
	Confidence float64            `json:"confidence"`
	Votes      map[string]float64 `json:"votes"`
}

// DebateAdapterForBenchmark adapts debate service for benchmark evaluation
type DebateAdapterForBenchmark struct {
	service DebateServiceForBenchmark
	logger  *logrus.Logger
}

// NewDebateAdapterForBenchmark creates a new debate adapter
func NewDebateAdapterForBenchmark(service DebateServiceForBenchmark, logger *logrus.Logger) *DebateAdapterForBenchmark {
	return &DebateAdapterForBenchmark{
		service: service,
		logger:  logger,
	}
}

// EvaluateResponse implements DebateEvaluator
func (a *DebateAdapterForBenchmark) EvaluateResponse(ctx context.Context, task *BenchmarkTask, response string) (float64, bool, error) {
	if a.service == nil {
		return 0, false, fmt.Errorf("debate service not available")
	}

	topic := fmt.Sprintf(`Evaluate this AI response for the given task.

Task: %s
Description: %s
Expected (if available): %s

AI Response: %s

Rate the response quality from 0.0 to 1.0 and determine if it should PASS or FAIL.
Respond with JSON: {"score": X.X, "passed": true/false, "reasoning": "..."}`,
		task.Name, task.Description, task.Expected, response)

	result, err := a.service.RunDebate(ctx, topic)
	if err != nil {
		return 0, false, err
	}

	// Parse result
	score, passed := a.parseEvaluationResult(result.Consensus, result.Confidence)
	return score, passed, nil
}

func (a *DebateAdapterForBenchmark) parseEvaluationResult(consensus string, confidence float64) (float64, bool) {
	var parsed struct {
		Score  float64 `json:"score"`
		Passed bool    `json:"passed"`
	}

	// Try to extract JSON
	start := strings.Index(consensus, "{")
	end := strings.LastIndex(consensus, "}")
	if start >= 0 && end > start {
		jsonStr := consensus[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
			return parsed.Score, parsed.Passed
		}
	}

	// Fall back to confidence
	return confidence, confidence >= 0.7
}

// VerifierServiceForBenchmark interface for verifier integration
type VerifierServiceForBenchmark interface {
	GetProviderScore(name string) float64
	IsProviderHealthy(name string) bool
	GetTopProviders(count int) []string
}

// VerifierAdapterForBenchmark adapts verifier for benchmark
type VerifierAdapterForBenchmark struct {
	service VerifierServiceForBenchmark
	logger  *logrus.Logger
}

// NewVerifierAdapterForBenchmark creates a new verifier adapter
func NewVerifierAdapterForBenchmark(service VerifierServiceForBenchmark, logger *logrus.Logger) *VerifierAdapterForBenchmark {
	return &VerifierAdapterForBenchmark{
		service: service,
		logger:  logger,
	}
}

// SelectBestProvider selects the best provider for benchmarking
func (a *VerifierAdapterForBenchmark) SelectBestProvider() (string, float64) {
	if a.service == nil {
		return "", 0
	}

	providers := a.service.GetTopProviders(5)
	if len(providers) == 0 {
		return "", 0
	}

	var bestProvider string
	var bestScore float64

	for _, p := range providers {
		if !a.service.IsProviderHealthy(p) {
			continue
		}
		score := a.service.GetProviderScore(p)
		if score > bestScore {
			bestScore = score
			bestProvider = p
		}
	}

	return bestProvider, bestScore
}

// GetProviderScoresForComparison gets provider scores for comparison
func (a *VerifierAdapterForBenchmark) GetProviderScoresForComparison() map[string]float64 {
	if a.service == nil {
		return nil
	}

	providers := a.service.GetTopProviders(10)
	scores := make(map[string]float64)
	for _, p := range providers {
		scores[p] = a.service.GetProviderScore(p)
	}
	return scores
}

// ProviderServiceForBenchmark interface for provider integration
type ProviderServiceForBenchmark interface {
	Complete(ctx context.Context, provider, model, prompt, systemPrompt string) (string, int, error)
	GetProvider(name string) LLMProvider
}

// ProviderAdapterForBenchmark adapts provider service for benchmark
type ProviderAdapterForBenchmark struct {
	service      ProviderServiceForBenchmark
	providerName string
	modelName    string
	logger       *logrus.Logger
}

// NewProviderAdapterForBenchmark creates a new provider adapter
func NewProviderAdapterForBenchmark(service ProviderServiceForBenchmark, providerName, modelName string, logger *logrus.Logger) *ProviderAdapterForBenchmark {
	return &ProviderAdapterForBenchmark{
		service:      service,
		providerName: providerName,
		modelName:    modelName,
		logger:       logger,
	}
}

// Complete implements LLMProvider
func (a *ProviderAdapterForBenchmark) Complete(ctx context.Context, prompt, systemPrompt string) (string, int, error) {
	if a.service == nil {
		return "", 0, fmt.Errorf("provider service not available")
	}
	return a.service.Complete(ctx, a.providerName, a.modelName, prompt, systemPrompt)
}

// GetName implements LLMProvider
func (a *ProviderAdapterForBenchmark) GetName() string {
	return a.providerName
}

// NewBenchmarkSystem creates the main benchmark system
func NewBenchmarkSystem(config *BenchmarkSystemConfig, logger *logrus.Logger) *BenchmarkSystem {
	if config == nil {
		config = DefaultBenchmarkSystemConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	return &BenchmarkSystem{
		config: config,
		logger: logger,
	}
}

// Initialize sets up the benchmark system
func (bs *BenchmarkSystem) Initialize(providerAdapter *ProviderAdapterForBenchmark) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.providerAdapter = providerAdapter

	// Create runner with provider
	var provider LLMProvider
	if providerAdapter != nil {
		provider = providerAdapter
	}
	bs.runner = NewStandardBenchmarkRunner(provider, bs.logger)

	// Set debate evaluator if available and enabled
	if bs.config.EnableDebateEvaluation && bs.debateAdapter != nil {
		if runner, ok := bs.runner.(*StandardBenchmarkRunner); ok {
			runner.SetDebateEvaluator(bs.debateAdapter)
		}
	}

	bs.logger.Info("Benchmark system initialized")
	return nil
}

// SetDebateService sets the debate service
func (bs *BenchmarkSystem) SetDebateService(service DebateServiceForBenchmark) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.debateAdapter = NewDebateAdapterForBenchmark(service, bs.logger)

	// Update runner if already initialized
	if runner, ok := bs.runner.(*StandardBenchmarkRunner); ok && bs.config.EnableDebateEvaluation {
		runner.SetDebateEvaluator(bs.debateAdapter)
	}
}

// SetVerifierService sets the verifier service
func (bs *BenchmarkSystem) SetVerifierService(service VerifierServiceForBenchmark) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.verifierAdapter = NewVerifierAdapterForBenchmark(service, bs.logger)
}

// GetRunner returns the benchmark runner
func (bs *BenchmarkSystem) GetRunner() BenchmarkRunner {
	return bs.runner
}

// RunBenchmarkWithBestProvider runs benchmark with the best available provider
func (bs *BenchmarkSystem) RunBenchmarkWithBestProvider(ctx context.Context, benchmarkType BenchmarkType, config *BenchmarkConfig) (*BenchmarkRun, error) {
	if config == nil {
		config = DefaultBenchmarkConfig()
	}

	// Select best provider
	var providerName string
	var score float64

	if bs.config.AutoSelectProvider && bs.verifierAdapter != nil {
		providerName, score = bs.verifierAdapter.SelectBestProvider()
		bs.logger.WithFields(logrus.Fields{
			"provider": providerName,
			"score":    score,
		}).Info("Auto-selected provider for benchmark")
	}

	if providerName == "" {
		providerName = "default"
	}

	// Create run
	run := &BenchmarkRun{
		ID:            uuid.New().String(),
		Name:          fmt.Sprintf("%s benchmark", benchmarkType),
		BenchmarkType: benchmarkType,
		ProviderName:  providerName,
		Config:        config,
	}

	if err := bs.runner.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	if err := bs.runner.StartRun(ctx, run.ID); err != nil {
		return nil, err
	}

	return run, nil
}

// CompareProviders runs benchmark across multiple providers for comparison
func (bs *BenchmarkSystem) CompareProviders(ctx context.Context, benchmarkType BenchmarkType, providers []string, config *BenchmarkConfig) ([]*BenchmarkRun, error) {
	if config == nil {
		config = DefaultBenchmarkConfig()
	}

	var runs []*BenchmarkRun
	for _, provider := range providers {
		run := &BenchmarkRun{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("%s benchmark - %s", benchmarkType, provider),
			BenchmarkType: benchmarkType,
			ProviderName:  provider,
			Config:        config,
		}

		if err := bs.runner.CreateRun(ctx, run); err != nil {
			bs.logger.WithError(err).WithField("provider", provider).Warn("Failed to create run")
			continue
		}

		if err := bs.runner.StartRun(ctx, run.ID); err != nil {
			bs.logger.WithError(err).WithField("provider", provider).Warn("Failed to start run")
			continue
		}

		runs = append(runs, run)
	}

	return runs, nil
}

// GenerateLeaderboard generates a leaderboard from multiple runs
func (bs *BenchmarkSystem) GenerateLeaderboard(ctx context.Context, benchmarkType BenchmarkType) (*Leaderboard, error) {
	runs, err := bs.runner.ListRuns(ctx, &RunFilter{
		BenchmarkType: benchmarkType,
		Status:        BenchmarkStatusCompleted,
	})
	if err != nil {
		return nil, err
	}

	leaderboard := &Leaderboard{
		BenchmarkType: benchmarkType,
		Entries:       make([]*LeaderboardEntry, 0),
		GeneratedAt:   time.Now(),
	}

	// Group by provider and take best run per provider
	bestByProvider := make(map[string]*BenchmarkRun)
	for _, run := range runs {
		if run.Summary == nil {
			continue
		}
		if best, ok := bestByProvider[run.ProviderName]; !ok || run.Summary.PassRate > best.Summary.PassRate {
			bestByProvider[run.ProviderName] = run
		}
	}

	// Create entries
	for provider, run := range bestByProvider {
		entry := &LeaderboardEntry{
			Rank:           0, // Will be set after sorting
			ProviderName:   provider,
			ModelName:      run.ModelName,
			PassRate:       run.Summary.PassRate,
			AverageScore:   run.Summary.AverageScore,
			AverageLatency: run.Summary.AverageLatency,
			TotalTasks:     run.Summary.TotalTasks,
			RunID:          run.ID,
			RunDate:        run.CreatedAt,
		}

		// Add verifier score if available
		if bs.verifierAdapter != nil && bs.verifierAdapter.service != nil {
			entry.VerifierScore = bs.verifierAdapter.service.GetProviderScore(provider)
		}

		leaderboard.Entries = append(leaderboard.Entries, entry)
	}

	// Sort by pass rate (descending)
	for i := 0; i < len(leaderboard.Entries); i++ {
		for j := i + 1; j < len(leaderboard.Entries); j++ {
			if leaderboard.Entries[j].PassRate > leaderboard.Entries[i].PassRate {
				leaderboard.Entries[i], leaderboard.Entries[j] = leaderboard.Entries[j], leaderboard.Entries[i]
			}
		}
	}

	// Assign ranks
	for i, entry := range leaderboard.Entries {
		entry.Rank = i + 1
	}

	return leaderboard, nil
}

// Leaderboard represents benchmark leaderboard
type Leaderboard struct {
	BenchmarkType BenchmarkType       `json:"benchmark_type"`
	Entries       []*LeaderboardEntry `json:"entries"`
	GeneratedAt   time.Time           `json:"generated_at"`
}

// LeaderboardEntry represents a leaderboard entry
type LeaderboardEntry struct {
	Rank           int           `json:"rank"`
	ProviderName   string        `json:"provider_name"`
	ModelName      string        `json:"model_name"`
	PassRate       float64       `json:"pass_rate"`
	AverageScore   float64       `json:"average_score"`
	AverageLatency time.Duration `json:"average_latency"`
	TotalTasks     int           `json:"total_tasks"`
	VerifierScore  float64       `json:"verifier_score,omitempty"`
	RunID          string        `json:"run_id"`
	RunDate        time.Time     `json:"run_date"`
}
