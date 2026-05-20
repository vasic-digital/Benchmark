package benchmark

import "sync"

// Translator resolves a message key (and optional named arguments) into a
// locale-appropriate string. It is the CONST-046 seam for digital.vasic.benchmark:
// every user-facing label, description, or report sentence the library emits is
// routed through tr() so a consuming application can render it in the end user's
// language instead of the built-in English fallback.
//
// CONST-051(B) decoupling: the benchmark module is project-not-aware. It ships
// only NoopTranslator (English fallback, key-stable). A consuming project
// injects its own i18n backend via SetTranslator — the module never reaches
// into a parent's resource tree.
type Translator interface {
	// Translate returns the localized string for key. args holds named
	// substitution values (e.g. {"count": 3}); a backend that does not support
	// substitution may ignore them. An unknown key MUST fall back to a stable,
	// non-empty string (NoopTranslator returns the built-in English default).
	Translate(key string, args map[string]interface{}) string
}

// NoopTranslator is the default Translator. It returns the built-in English
// default for every known key and the key itself for any unknown key, so the
// library is fully usable with no i18n backend wired. It performs no locale
// negotiation — that is the consuming project's responsibility.
type NoopTranslator struct{}

// Translate implements Translator using the built-in English bundle.
func (NoopTranslator) Translate(key string, args map[string]interface{}) string {
	if msg, ok := enBundle[key]; ok {
		return expand(msg, args)
	}
	return key
}

var (
	translatorMu      sync.RWMutex
	activeTranslator  Translator = NoopTranslator{}
)

// SetTranslator installs a Translator for the process. Passing nil restores the
// built-in NoopTranslator. Safe for concurrent use; a consuming application
// typically calls this once at startup after negotiating the user's locale.
func SetTranslator(t Translator) {
	translatorMu.Lock()
	defer translatorMu.Unlock()
	if t == nil {
		activeTranslator = NoopTranslator{}
		return
	}
	activeTranslator = t
}

// tr resolves key through the active Translator. It is the single internal
// entry point every user-facing string in this module funnels through.
func tr(key string, args map[string]interface{}) string {
	translatorMu.RLock()
	t := activeTranslator
	translatorMu.RUnlock()
	return t.Translate(key, args)
}

// expand performs minimal {name}-style substitution on a bundle template so the
// built-in English fallback can render counts and identifiers without pulling a
// templating dependency into the decoupled module.
func expand(msg string, args map[string]interface{}) string {
	if len(args) == 0 {
		return msg
	}
	out := msg
	for k, v := range args {
		out = replaceAll(out, "{"+k+"}", toStr(v))
	}
	return out
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	var b []byte
	for {
		i := indexOf(s, old)
		if i < 0 {
			b = append(b, s...)
			break
		}
		b = append(b, s[:i]...)
		b = append(b, new...)
		s = s[i+len(old):]
	}
	return string(b)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toStr(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return itoa(x)
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// enBundle is the built-in English message bundle. A consuming project's
// Translator backend supplies locale-specific overrides keyed by the same IDs;
// keys absent from a locale bundle fall back here. Keys are stable contract
// surface — renaming one is a breaking change for downstream locale bundles.
var enBundle = map[string]string{
	// Built-in benchmark metadata (user-facing labels shown in benchmark lists).
	"benchmark.swe_bench_lite.name":  "SWE-Bench Lite",
	"benchmark.swe_bench_lite.desc":  "Simplified software engineering benchmark tasks",
	"benchmark.humaneval.name":       "HumanEval",
	"benchmark.humaneval.desc":       "Code generation benchmark from OpenAI",
	"benchmark.mmlu_mini.name":       "MMLU Mini",
	"benchmark.mmlu_mini.desc":       "Subset of MMLU benchmark for quick evaluation",
	"benchmark.gsm8k_mini.name":      "GSM8K Mini",
	"benchmark.gsm8k_mini.desc":      "Subset of GSM8K math benchmark",
	// Run-comparison report sentences (returned in RunComparison.Summary,
	// surfaced directly to end users of a benchmark report).
	"compare.run2_regressed":  "Run 2 regressed with {regressions} regressions and {improvements} improvements",
	"compare.run2_improved":   "Run 2 improved with {improvements} improvements and {regressions} regressions",
	"compare.no_difference":   "No significant difference between runs",
	// Built-in benchmark task names + descriptions (user-facing labels shown
	// when a benchmark's task list is rendered to an end user; the Prompt /
	// Expected payloads are evaluation fixtures, NOT routed through tr()).
	"task.swe_001.name": "Fix null pointer exception",
	"task.swe_001.desc": "Fix the null pointer exception in the user service",
	"task.swe_002.name": "Add error handling",
	"task.swe_002.desc": "Add proper error handling to the file reader",
	"task.swe_003.name": "Implement retry logic",
	"task.swe_003.desc": "Add retry logic with exponential backoff",
	"task.he_001.desc":  "Check if any two elements are closer than threshold",
	"task.he_002.desc":  "Separate balanced parentheses groups",
	"task.mmlu_001.name": "Computer Science - Algorithms",
	"task.mmlu_001.desc": "Multiple choice question on algorithms",
	"task.mmlu_002.name": "Mathematics - Calculus",
	"task.mmlu_002.desc": "Multiple choice question on calculus",
	"task.mmlu_003.name": "Physics - Mechanics",
	"task.mmlu_003.desc": "Multiple choice question on mechanics",
	"task.gsm8k_001.name": "Basic arithmetic word problem",
	"task.gsm8k_001.desc": "Solve a basic arithmetic word problem",
	"task.gsm8k_002.name": "Multi-step calculation",
	"task.gsm8k_002.desc": "Solve a multi-step calculation problem",
}
