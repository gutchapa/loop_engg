// Package learn provides a persistent knowledge store for the AI agent.
// It remembers past strategies, infrastructure fixes, error patterns,
// and project context — surviving across sessions so the agent gets
// smarter over time.
//
// Stored in autoresearch.knowledge.json alongside the experiment log.
package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// KnowledgeStore is the in-memory representation of the persistent store.
type KnowledgeStore struct {
	Version  int       `json:"version"`
	Updated  time.Time `json:"updated"`
	Entries  []Entry   `json:"entries"`
	path     string    // file path, not serialized
}

// Entry represents one piece of learned knowledge.
type Entry struct {
	ID        string    `json:"id"`        // unique slug: "fix:eslint-version-conflict"
	Kind      string    `json:"kind"`      // "strategy", "anti_pattern", "infra_fix", "error_pattern", "context"
	Created   time.Time `json:"created"`
	LastHit   time.Time `json:"last_hit"`  // last time this was useful
	HitCount  int       `json:"hit_count"` // how many times it helped
	Project   string    `json:"project"`   // which project (empty = global)

	// Human-readable fields
	Title       string `json:"title"`       // one-line summary
	Description string `json:"description"` // longer explanation
	Fix         string `json:"fix"`         // what action to take ("upgrade eslint to ^9", etc.)

	// Machine-matching fields
	Triggers     []string          `json:"triggers"`      // error strings that match ("conflicts with", "peer dependency")
	Conditions   map[string]string `json:"conditions"`    // context conditions ("language": "typescript", "runner": "vitest")
	SuccessRate  float64           `json:"success_rate"`  // 0.0-1.0: how often this fix works
	Confidence   float64           `json:"confidence"`    // 0.0-1.0: how confident we are in this knowledge
}

const DefaultFileName = "autoresearch.knowledge.json"

// Load reads the knowledge store from disk (or creates a fresh one).
func Load(dir string) (*KnowledgeStore, error) {
	path := filepath.Join(dir, DefaultFileName)
	store := &KnowledgeStore{
		Version: 1,
		Updated: time.Now(),
		Entries: []Entry{},
		path:    path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil // fresh store
		}
		return nil, fmt.Errorf("read knowledge store: %w", err)
	}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parse knowledge store: %w", err)
	}
	store.path = path
	return store, nil
}

// Save persists the knowledge store to disk.
func (ks *KnowledgeStore) Save() error {
	ks.Updated = time.Now()
	ks.Version++
	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(ks.path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// Add inserts a new knowledge entry and saves.
func (ks *KnowledgeStore) Add(e Entry) error {
	if e.ID == "" {
		e.ID = slug(e.Title)
	}
	if e.Created.IsZero() {
		e.Created = time.Now()
	}
	if e.LastHit.IsZero() {
		e.LastHit = e.Created
	}
	ks.Entries = append(ks.Entries, e)
	return ks.Save()
}

// RecordHit bumps the hit count for an entry and saves.
func (ks *KnowledgeStore) RecordHit(id string) error {
	for i := range ks.Entries {
		if ks.Entries[i].ID == id {
			ks.Entries[i].HitCount++
			ks.Entries[i].LastHit = time.Now()
			return ks.Save()
		}
	}
	return fmt.Errorf("entry not found: %s", id)
}

// RecordResult updates success rate based on whether a fix worked.
func (ks *KnowledgeStore) RecordResult(id string, worked bool) error {
	for i := range ks.Entries {
		if ks.Entries[i].ID == id {
			e := &ks.Entries[i]
			if worked {
				// Move toward 1.0
				e.SuccessRate = e.SuccessRate*0.8 + 1.0*0.2
				e.Confidence = min(1.0, e.Confidence+0.1)
			} else {
				// Move toward 0.0
				e.SuccessRate = e.SuccessRate * 0.5
				e.Confidence = max(0.0, e.Confidence-0.2)
			}
			return ks.Save()
		}
	}
	return fmt.Errorf("entry not found: %s", id)
}

// Query searches the knowledge store for relevant entries given a situation.
// Returns entries sorted by relevance (highest first).
func (ks *KnowledgeStore) Query(kind string, errorText string, conditions map[string]string) []Entry {
	var results scoredEntries
	for _, e := range ks.Entries {
		if kind != "" && e.Kind != kind {
			continue
		}
		score := ks.matchScore(e, errorText, conditions)
		if score > 0 {
			results = append(results, scoredEntry{entry: e, score: score})
		}
	}
	sort.Sort(sort.Reverse(results))

	out := make([]Entry, len(results))
	for i, se := range results {
		out[i] = se.entry
	}
	return out
}

// FindInfraFix looks for infrastructure fixes matching the error text.
func (ks *KnowledgeStore) FindInfraFix(errorText, project string) *Entry {
	entries := ks.Query("infra_fix", errorText, map[string]string{"project": project})
	if len(entries) > 0 && entries[0].Confidence >= 0.3 {
		return &entries[0]
	}
	// Fall back to global entries
	entries = ks.Query("infra_fix", errorText, nil)
	if len(entries) > 0 && entries[0].Confidence >= 0.3 {
		return &entries[0]
	}
	return nil
}

// AntiPatterns returns known strategies that failed multiple times.
func (ks *KnowledgeStore) AntiPatterns() []Entry {
	var out []Entry
	for _, e := range ks.Entries {
		if e.Kind == "anti_pattern" && e.SuccessRate < 0.3 && e.HitCount >= 2 {
			out = append(out, e)
		}
	}
	return out
}

// BestStrategies returns the most successful strategies ordered by confidence.
func (ks *KnowledgeStore) BestStrategies() []Entry {
	var scored scoredEntries
	for _, e := range ks.Entries {
		if e.Kind == "strategy" && e.Confidence > 0 {
			score := e.Confidence * float64(e.HitCount+1)
			scored = append(scored, scoredEntry{entry: e, score: score})
		}
	}
	sort.Sort(sort.Reverse(scored))
	out := make([]Entry, len(scored))
	for i, se := range scored {
		out[i] = se.entry
	}
	return out
}

// DistillFromLog analyzes the experiment log (JSONL) and extracts patterns.
// Call this after experiments accumulate 10+ runs.
func (ks *KnowledgeStore) DistillFromLog(logPath string) error {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("read log: %w", err)
	}

	type logEntry struct {
		Run         int               `json:"run"`
		Status      string            `json:"status"`
		Description string            `json:"description"`
		Metric      float64           `json:"metric"`
		Metrics     map[string]float64 `json:"metrics"`
		ASI         map[string]any    `json:"asi"`
		Commit      string            `json:"commit"`
		Timestamp   int64             `json:"timestamp"`
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var experiments []logEntry
	for _, line := range lines {
		if !strings.Contains(line, `"status"`) {
			continue
		}
		var e logEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		experiments = append(experiments, e)
	}

	// Pattern 1: Find strategies that consistently improve metrics
	strategySeen := map[string]struct{}{}
	for i := 1; i < len(experiments); i++ {
		curr := experiments[i]
		prev := experiments[i-1]
		if curr.Status == "keep" && prev.Status == "keep" {
			improvement := curr.Metric - prev.Metric
			if improvement > 0 {
				desc := strings.ToLower(curr.Description)
				id := slug("strategy:" + curr.Description)
				if _, exists := strategySeen[id]; exists {
					continue
				}
				strategySeen[id] = struct{}{}

				strategy := Entry{
					ID:          id,
					Kind:        "strategy",
					Created:     time.Now(),
					Title:       curr.Description,
					Description: fmt.Sprintf("Improved metric from %.0f to %.0f (+%.0f)", prev.Metric, curr.Metric, improvement),
					Confidence:  0.5,
					SuccessRate: 0.8,
				}
				// Check if already exists
				found := false
				for _, e := range ks.Entries {
					if e.ID == id {
						found = true
						break
					}
				}
				if !found {
					ks.Entries = append(ks.Entries, strategy)
				}
				_ = desc
			}
		}
	}

	// Pattern 2: Find failed strategies → anti_patterns
	for _, e := range experiments {
		if e.Status == "discard" || e.Status == "crash" {
			id := slug("anti:" + e.Description)
			found := false
			for i, ke := range ks.Entries {
				if ke.ID == id {
					ks.Entries[i].HitCount++
					ks.Entries[i].LastHit = time.Now()
					ks.Entries[i].SuccessRate *= 0.5
					found = true
					break
				}
			}
			if !found {
				ap := Entry{
					ID:          id,
					Kind:        "anti_pattern",
					Created:     time.Now(),
					Title:       e.Description,
					Description: fmt.Sprintf("This approach failed: %s", e.Description),
					Fix:         "Avoid this strategy",
					SuccessRate: 0.1,
					Confidence:  0.4,
				}
				ks.Entries = append(ks.Entries, ap)
			}
		}
	}

	// Pattern 3: Infrastructure fixes from ASI
	for _, e := range experiments {
		if e.ASI == nil {
			continue
		}
		if errorType, ok := e.ASI["error_type"].(string); ok {
			id := slug("infra:" + errorType)
			found := false
			for _, ke := range ks.Entries {
				if ke.ID == id {
					found = true
					break
				}
			}
			if !found {
				fix := ""
				if f, ok := e.ASI["fix"].(string); ok {
					fix = f
				}
				rootCause := ""
				if rc, ok := e.ASI["root_cause"].(string); ok {
					rootCause = rc
				}
				triggers := []string{errorType}
				if rootCause != "" {
					triggers = append(triggers, rootCause)
				}

				entry := Entry{
					ID:          id,
					Kind:        "infra_fix",
					Created:     time.Now(),
					Title:       fmt.Sprintf("Fix: %s", errorType),
					Description: rootCause,
					Fix:         fix,
					Triggers:    triggers,
					SuccessRate: 0.8,
					Confidence:  0.6,
				}
				ks.Entries = append(ks.Entries, entry)
			}
		}
	}

	return ks.Save()
}

// Summary returns a human-readable summary of the knowledge store.
func (ks *KnowledgeStore) Summary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Knowledge Store: %d entries\n\n", len(ks.Entries)))

	strategies := ks.BestStrategies()
	if len(strategies) > 0 {
		b.WriteString("📈 Best Strategies:\n")
		for i, s := range strategies {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("  • %s (confidence: %.1f, hits: %d)\n", s.Title, s.Confidence, s.HitCount))
		}
		b.WriteString("\n")
	}

	anti := ks.AntiPatterns()
	if len(anti) > 0 {
		b.WriteString("🚫 Anti-Patterns (avoid):\n")
		for _, a := range anti {
			b.WriteString(fmt.Sprintf("  • %s\n", a.Title))
		}
		b.WriteString("\n")
	}

	infraCount := 0
	for _, e := range ks.Entries {
		if e.Kind == "infra_fix" {
			infraCount++
		}
	}
	if infraCount > 0 {
		b.WriteString(fmt.Sprintf("🔧 Infrastructure fixes: %d known\n", infraCount))
	}

	return b.String()
}

// --- Internal ---

func (ks *KnowledgeStore) matchScore(e Entry, errorText string, conditions map[string]string) float64 {
	score := 0.0

	// Trigger match: how many trigger strings appear in error text
	if errorText != "" {
		errorLower := strings.ToLower(errorText)
		for _, t := range e.Triggers {
			if strings.Contains(errorLower, strings.ToLower(t)) {
				score += 2.0
			}
		}
	}

	// Condition match: how many context conditions match
	if len(conditions) > 0 && len(e.Conditions) > 0 {
		for k, v := range conditions {
			if ev, ok := e.Conditions[k]; ok && strings.EqualFold(ev, v) {
				score += 1.0
			}
		}
	}

	// Boost by confidence
	score += e.Confidence

	// Boost by hit count
	score += float64(e.HitCount) * 0.5

	return score
}

// --- Helpers ---

type scoredEntry struct {
	entry Entry
	score float64
}

type scoredEntries []scoredEntry

func (s scoredEntries) Len() int           { return len(s) }
func (s scoredEntries) Less(i, j int) bool { return s[i].score < s[j].score }
func (s scoredEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func slug(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		s = s[:60]
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		case r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '_':
			return '-'
		default:
			return -1
		}
	}, s)
	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
