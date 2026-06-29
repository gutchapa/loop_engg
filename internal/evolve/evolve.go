// Package evolve is the self-modification engine. It analyzes the agent's
// performance over time and automatically improves itself by:
//
//   1. Tuning the system prompt based on what works/fails
//   2. Adjusting experiment config (targets, guardrails, max iterations)
//   3. Rotating strategies when current approach stalls
//   4. Self-critiquing after each iteration
//
// All changes are versioned and reversible — the engine never destroys
// the original config, it creates variants that can be rolled back.
package evolve

import (
	"fmt"
	"strings"
	"time"

	"github.com/gutchapa/loop/internal/learn"
)

// Snapshot captures the agent's state at a point in time for evaluation.
type Snapshot struct {
	Iteration     int               `json:"iteration"`
	Metric        float64           `json:"metric"`         // current primary metric
	PrevMetric    float64           `json:"prev_metric"`    // previous iteration's metric
	Direction     string            `json:"direction"`      // "higher" or "lower"
	ToolCalls     []string          `json:"tool_calls"`     // tools used this iteration
	LLMResponse   string            `json:"llm_response"`   // truncated LLM response
	TestOutput    string            `json:"test_output"`    // test/stderr output
	ExitCode      int               `json:"exit_code"`
	StrategyUsed  string            `json:"strategy_used"`  // what the agent tried
	KnowledgeHits int               `json:"knowledge_hits"` // how many knowledge entries helped
	Duration      time.Duration     `json:"duration"`
}

// Assessment is the engine's evaluation of the current state.
type Assessment struct {
	Status       string   `json:"status"`        // "improving", "stagnating", "regressing", "done"
	Confidence   float64  `json:"confidence"`    // 0.0-1.0: how confident in current strategy
	Insight      string   `json:"insight"`       // human-readable analysis
	Suggestions  []string `json:"suggestions"`   // what to try next
	PromptPatch  string   `json:"prompt_patch"`  // text to inject into system prompt
	ConfigPatch  map[string]any `json:"config_patch"` // config changes to apply
	NewStrategy  string   `json:"new_strategy"`  // alternative strategy if rotating
}

// Engine is the self-modification engine.
type Engine struct {
	Store     *learn.KnowledgeStore
	History   []Snapshot          // recent iteration snapshots
	Window    int                 // how many iterations to look back
	PromptGen int                 // generation counter for prompt variants
	ConfigGen int                 // generation counter for config variants
}

// New creates a new evolution engine.
func New(store *learn.KnowledgeStore) *Engine {
	return &Engine{
		Store:  store,
		Window: 5,
	}
}

// Record adds a snapshot to the history.
func (e *Engine) Record(s Snapshot) {
	e.History = append(e.History, s)
	if len(e.History) > e.Window*2 {
		e.History = e.History[len(e.History)-e.Window*2:]
	}
}

// Evaluate analyzes the recent history and produces an assessment.
func (e *Engine) Evaluate() Assessment {
	if len(e.History) == 0 {
		return Assessment{
			Status:  "starting",
			Insight: "No history yet — collecting baseline data.",
		}
	}

	last := e.History[len(e.History)-1]
	a := Assessment{}

	// Determine trend
	trend := e.computeTrend()
	improved := e.metricImproved(last.Metric, last.PrevMetric, last.Direction)

	switch {
	case improved:
		a.Status = "improving"
		a.Confidence = 0.7
	case trend < -0.1:
		a.Status = "regressing"
		a.Confidence = 0.2
	default:
		a.Status = "stagnating"
		a.Confidence = 0.4
	}

	// Check termination
	if e.isDone() {
		a.Status = "done"
		a.Insight = "Target achieved. No further optimization needed."
		return a
	}

	// Generate insights
	switch a.Status {
	case "improving":
		a.Insight = "Current strategy is working — continue with adjustments."
		a.Suggestions = []string{"Double down on current approach", "Increase iteration budget"}
		a.PromptPatch = e.reinforceStrategy()

	case "stagnating":
		a.Insight = "Progress has plateaued. Consider strategy rotation."
		alt := e.pickAlternativeStrategy()
		a.Suggestions = []string{
			fmt.Sprintf("Rotate to: %s", alt),
			"Try a different optimization angle",
			"Check if target is too difficult",
		}
		a.NewStrategy = alt
		a.PromptPatch = e.nudgeExploration()

	case "regressing":
		a.Insight = "Recent changes made things worse. Roll back strategy."
		a.Suggestions = []string{
			"Revert last change",
			"Check guardrails — something broke",
			"Simplify: try smaller, incremental changes",
		}
		a.PromptPatch = e.cautionPrompt()
	}

	// Check for diminishing returns
	if e.isDiminishing() {
		a.Suggestions = append(a.Suggestions, "Diminishing returns detected — try a different class of optimization")
		a.ConfigPatch = map[string]any{
			"relax_target": true,
			"note":         "Current target may be near the optimal value",
		}
	}

	// Check for stuck loop
	if e.isStuck() {
		a.Status = "stuck"
		a.Insight = "Agent is stuck in a loop — same files, same errors, no progress."
		a.Suggestions = []string{"Change optimization goal", "Ask for human guidance", "Try a completely different approach"}
		a.PromptPatch = e.escapeStuckLoop()
	}

	return a
}

// GeneratePrompt creates a new system prompt variant based on learnings.
// Returns the prompt text and a generation ID for tracking.
func (e *Engine) GeneratePrompt(base string) (string, int) {
	e.PromptGen++
	gen := e.PromptGen

	// Get knowledge to inject
	best := e.Store.BestStrategies()
	anti := e.Store.AntiPatterns()

	var additions strings.Builder
	additions.WriteString("\n\n## Evolved Instructions (gen %d)\n\n")

	if len(best) > 0 {
		additions.WriteString("### Proven Strategies (use these patterns):\n")
		for i, s := range best {
			if i >= 5 {
				break
			}
			additions.WriteString(fmt.Sprintf("- %s ✅\n", s.Title))
		}
		additions.WriteString("\n")
	}

	if len(anti) > 0 {
		additions.WriteString("### Anti-Patterns (NEVER do these):\n")
		for _, a := range anti {
			additions.WriteString(fmt.Sprintf("- %s ❌\n", a.Title))
		}
		additions.WriteString("\n")
	}

	// Add adaptive guidance based on history
	if len(e.History) >= 3 {
		additions.WriteString("### Adaptive Guidance:\n")
		trend := e.computeTrend()
		switch {
		case trend > 0.05:
			additions.WriteString("- Current approach is working well. Continue refining.\n")
		case trend < -0.05:
			additions.WriteString("- Recent changes caused regression. Try a different approach.\n")
		default:
			additions.WriteString("- Progress is slow. Explore: what haven't you tried?\n")
		}
	}

	return base + fmt.Sprintf(additions.String(), gen), gen
}

// TuneConfig analyzes metric trends and suggests config adjustments.
func (e *Engine) TuneConfig(target float64, direction string) map[string]any {
	if len(e.History) < 3 {
		return nil
	}

	changes := map[string]any{}
	e.ConfigGen++
	changes["generation"] = e.ConfigGen

	// If consistently hitting target, tighten it
	if e.isDone() && e.History[len(e.History)-1].Metric != 0 {
		adjustment := 0.05 // 5% tighter
		if direction == "lower" {
			changes["new_target"] = target * (1 - adjustment)
		} else {
			changes["new_target"] = target * (1 + adjustment)
		}
		changes["reason"] = "Target consistently met — tightening for further optimization"
		return changes
	}

	// If target seems unreachable, suggest relaxation
	if e.isStuck() && len(e.History) >= 5 {
		adjustment := 0.1 // 10% looser
		if direction == "lower" {
			changes["new_target"] = target * (1 + adjustment)
		} else {
			changes["new_target"] = target * (1 - adjustment)
		}
		changes["reason"] = "Target appears unreachable after multiple attempts — consider relaxing"
	}

	return changes
}

// Summary returns a human-readable status of the evolution engine.
func (e *Engine) Summary() string {
	if len(e.History) == 0 {
		return "🧬 Evolution engine: idle (no history)"
	}

	a := e.Evaluate()
	return fmt.Sprintf("🧬 Evolution: %s (confidence: %.1f) | %s", a.Status, a.Confidence, a.Insight)
}

// --- Internal ---

func (e *Engine) computeTrend() float64 {
	if len(e.History) < 2 {
		return 0
	}
	// Linear trend over the window
	recent := e.History
	if len(recent) > e.Window {
		recent = recent[len(recent)-e.Window:]
	}

	var sumDelta, sumAbs float64
	for i := 1; i < len(recent); i++ {
		delta := recent[i].Metric - recent[i-1].Metric
		sumDelta += delta
		sumAbs += abs(delta)
	}
	if sumAbs == 0 {
		return 0
	}
	return sumDelta / sumAbs // normalized -1 to 1
}

func (e *Engine) metricImproved(current, previous float64, direction string) bool {
	if direction == "higher" {
		return current > previous
	}
	return current < previous
}

func (e *Engine) isDone() bool {
	// Check if last 3 iterations all hit same metric (plateau at target)
	if len(e.History) < 3 {
		return false
	}
	recent := e.History[len(e.History)-3:]
	val := recent[0].Metric
	for _, s := range recent[1:] {
		if s.Metric != val {
			return false
		}
	}
	return true
}

func (e *Engine) isDiminishing() bool {
	if len(e.History) < 4 {
		return false
	}
	// Check if improvement per iteration is shrinking
	improvements := []float64{}
	for i := 1; i < len(e.History); i++ {
		delta := abs(e.History[i].Metric - e.History[i-1].Metric)
		improvements = append(improvements, delta)
	}
	if len(improvements) < 3 {
		return false
	}
	// Are the last 2 improvements smaller than the first 2?
	return improvements[len(improvements)-1] < improvements[0]*0.3
}

func (e *Engine) isStuck() bool {
	if len(e.History) < 4 {
		return false
	}
	// Same exit code, same tools, same metric for 4+ iterations
	recent := e.History[len(e.History)-4:]
	first := recent[0]
	for _, s := range recent[1:] {
		if s.ExitCode != first.ExitCode {
			return false
		}
		if s.Metric != first.Metric {
			return false
		}
	}
	return true
}

func (e *Engine) pickAlternativeStrategy() string {
	strategies := e.Store.BestStrategies()
	if len(strategies) == 0 {
		return "explore project files and look for untested code paths"
	}
	// Pick a strategy that hasn't been tried recently
	recent := map[string]bool{}
	for _, s := range e.History {
		if s.StrategyUsed != "" {
			recent[s.StrategyUsed] = true
		}
	}
	for _, s := range strategies {
		if !recent[s.Title] {
			return s.Title
		}
	}
	return strategies[len(strategies)%max(len(strategies), 1)].Title
}

func (e *Engine) reinforceStrategy() string {
	if len(e.History) == 0 {
		return ""
	}
	last := e.History[len(e.History)-1]
	return fmt.Sprintf("The previous approach worked well (metric: %.1f). Build on it — make a similar improvement to another area.", last.Metric)
}

func (e *Engine) nudgeExploration() string {
	return "Progress has plateaued. Try something different: explore a new area of the codebase, or try a completely different optimization technique."
}

func (e *Engine) cautionPrompt() string {
	return "Recent changes caused a regression. Be cautious: make small, incremental changes and verify after each one. Revert anything that breaks tests."
}

func (e *Engine) escapeStuckLoop() string {
	return "You appear to be stuck in a loop — repeating the same actions with no progress. STOP. Try one of these instead:\n1. Explore a completely different file or module\n2. Add a new feature instead of optimizing existing code\n3. If the target seems unreachable, focus on qualitative improvements instead"
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
