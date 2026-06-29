// Package planner reads the project context (autoresearch.md, config, files in scope)
// and builds structured prompts for the LLM agent.
package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gutchapa/loop/internal/config"
)

// ProjectContext holds the full context for the AI agent.
type ProjectContext struct {
	Rules     string // from autoresearch.md
	Config    config.Config
	Files     []FileEntry  // files in scope with their content
	Metrics   string       // recent metrics from log
	Objective string       // parsed objective from rules
}

// FileEntry holds a file's path and content.
type FileEntry struct {
	Path    string
	Content string
}

// LoadContext reads the project context from the working directory.
func LoadContext(workingDir string) (*ProjectContext, error) {
	ctx := &ProjectContext{}

	// 1. Load autoresearch.md
	rulesPath := filepath.Join(workingDir, "autoresearch.md")
	if data, err := os.ReadFile(rulesPath); err == nil {
		ctx.Rules = string(data)
		ctx.Objective = extractObjective(string(data))
	}

	// 2. Load config
	if cfg, err := config.Load(filepath.Join(workingDir, config.DefaultFileName)); err == nil {
		ctx.Config = cfg
	}

	// 3. Load files in scope
	if ctx.Config.AI != nil && len(ctx.Config.AI.FilesInScope) > 0 {
		for _, pattern := range ctx.Config.AI.FilesInScope {
			matches, err := filepath.Glob(filepath.Join(workingDir, pattern))
			if err != nil {
				continue
			}
			for _, match := range matches {
				relPath, _ := filepath.Rel(workingDir, match)
				data, err := os.ReadFile(match)
				if err != nil {
					continue
				}
				ctx.Files = append(ctx.Files, FileEntry{
					Path:    relPath,
					Content: string(data),
				})
			}
		}
	}

	// 4. Load recent metrics from log
	logPath := filepath.Join(workingDir, "autoresearch.jsonl")
	if data, err := os.ReadFile(logPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		// Get last 3 experiment lines
		var lastLines []string
		for i := len(lines) - 1; i >= 0 && len(lastLines) < 3; i-- {
			if strings.Contains(lines[i], `"metric"`) {
				lastLines = append([]string{lines[i]}, lastLines...)
			}
		}
		ctx.Metrics = strings.Join(lastLines, "\n")
	}

	return ctx, nil
}

// BuildSystemPrompt creates the system prompt for the LLM agent.
func (ctx *ProjectContext) BuildSystemPrompt() string {
	prompt := `You are an autonomous coding agent running inside the Loop Engineering CLI.
Your goal is to improve the project based on the rules and context below.

## Metrics can be HARD or SOFT
- HARD metrics: numeric targets (e.g. "test_count >= 80", "latency_ms <= 50")
- SOFT metrics (guardrails): must always pass (e.g. "build passes", "no regressions")
- QUALITATIVE goals: subjective (e.g. "clean UI", "no data mismatches") — use your judgment

## Your capabilities
You can use MCP tools to:
1. read_file — read any file in the project
2. write_file — write/modify any file
3. run_command — execute shell commands (tests, builds, benchmarks)
4. list_files — explore the project structure
5. read_config — read experiment configuration
6. get_metrics — get current test/benchmark results
7. check_termination — check if experiment goals are met

## Your loop
1. OBSERVE — read files, run tests, understand current state
2. ORIENT — analyze metrics, decide what to improve
3. DECIDE — plan a specific code change
4. ACT — write code, run tests, verify
5. Repeat until termination conditions are met

## Rules
- Write clean, tested code
- Run tests after every change to verify nothing breaks
- If tests fail, fix them before moving on
- Never break a guardrail (soft metric that must pass)
- Qualitative improvements are valid — clean code, better UX, fewer bugs
- Check termination conditions periodically`
	return prompt
}

// BuildUserPrompt creates the user prompt with project-specific context.
func (ctx *ProjectContext) BuildUserPrompt() string {
	var b strings.Builder

	b.WriteString("## Project Objective\n")
	if ctx.Objective != "" {
		b.WriteString(ctx.Objective)
		b.WriteString("\n")
	} else {
		b.WriteString("Improve the project based on the rules in autoresearch.md\n")
	}

	b.WriteString("\n## Experiment Config\n")

	// Qualitative goals (soft — human judgment)
	if ctx.Config.Goal != nil {
		if ctx.Config.Goal.Summary != "" {
			b.WriteString(fmt.Sprintf("Goal: %s\n", ctx.Config.Goal.Summary))
		}
		if len(ctx.Config.Goal.Qualitative) > 0 {
			b.WriteString("Qualitative objectives:\n")
			for _, q := range ctx.Config.Goal.Qualitative {
				b.WriteString(fmt.Sprintf("  - %s\n", q))
			}
		}
	}

	// Guardrails (must always pass)
	if len(ctx.Config.Guardrails) > 0 {
		b.WriteString("\n🛡️ Guardrails (must NEVER break):\n")
		for _, g := range ctx.Config.Guardrails {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", g.Name, g.Check))
		}
	}

	// Hard metric (numeric target)
	if ctx.Config.Metric != nil && ctx.Config.Metric.Name != "" {
		b.WriteString(fmt.Sprintf("\nHard metric: %s (direction: %s)\n", ctx.Config.Metric.Name, ctx.Config.Metric.Direction))
		if ctx.Config.Metric.Target.Metric != "" {
			b.WriteString(fmt.Sprintf("Target: %s %s %.0f\n", ctx.Config.Metric.Target.Metric, ctx.Config.Metric.Target.Operator, ctx.Config.Metric.Target.Value))
		}
	}

	// Fallback for legacy config
	b.WriteString(fmt.Sprintf("Command: %s\n", ctx.Config.Command))
	if ctx.Config.MaxIterations > 0 {
		b.WriteString(fmt.Sprintf("Max iterations: %d\n", ctx.Config.MaxIterations))
	}

	b.WriteString("\n## Recent Metrics\n")
	if ctx.Metrics != "" {
		b.WriteString(ctx.Metrics)
		b.WriteString("\n")
	} else {
		b.WriteString("(no metrics yet)\n")
	}

	b.WriteString("\n## Files in Scope\n")
	if len(ctx.Files) > 0 {
		for _, f := range ctx.Files {
			b.WriteString(fmt.Sprintf("\n### %s\n", f.Path))
			b.WriteString("```\n")
			b.WriteString(truncateContent(f.Content, 5000))
			b.WriteString("\n```\n")
		}
	} else {
		b.WriteString("(explore the project using list_files and read_file)\n")
	}

	b.WriteString("\n## Your Task\n")
	b.WriteString("Analyze the current state and decide what code change would most improve the metric. ")
	b.WriteString("Make one focused change at a time, run tests, and iterate.\n")

	return b.String()
}

// extractObjective parses the objective from autoresearch.md.
func extractObjective(rules string) string {
	lines := strings.Split(rules, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Objective") || strings.HasPrefix(trimmed, "# Objective") {
			// Return the next non-empty, non-header line
			found := false
			for _, l := range lines {
				if strings.TrimSpace(l) == trimmed {
					found = true
					continue
				}
				if found && strings.TrimSpace(l) != "" && !strings.HasPrefix(l, "#") && !strings.HasPrefix(l, "|") {
					return strings.TrimSpace(l)
				}
			}
		}
	}
	return ""
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n... [truncated]"
}
