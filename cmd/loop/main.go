package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gutchapa/loop/internal/bridge"
	"github.com/gutchapa/loop/internal/config"
	"github.com/gutchapa/loop/internal/llm"
	"github.com/gutchapa/loop/internal/log"
	"github.com/gutchapa/loop/internal/mcp"
	"github.com/gutchapa/loop/internal/metric"
	"github.com/gutchapa/loop/internal/planner"
	"github.com/gutchapa/loop/internal/run"
	"github.com/gutchapa/loop/internal/scanner"
)

const version = "1.2.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	case "auto":
		cmdAuto(os.Args[2:])
	case "bench":
		cmdBench(os.Args[2:])
	case "check":
		cmdCheck(os.Args[2:])
	case "mcp":
		cmdMCP(os.Args[2:])
	case "ai":
		cmdAI(os.Args[2:])
	case "version":
		fmt.Printf("loop v%s\n", version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`Loop Engineering CLI — autonomous AI coding agent + experiment loop tooling.

Usage:
  loop init <name> <metric_name> [--unit <unit>] [--direction <lower|higher>]
        Initialize a new experiment session.

  loop run <command> [--timeout <seconds>]
        Run a shell command with timing and METRIC output.

  loop auto [--timeout <seconds>]
        Run one autonomous experiment iteration from config.

  loop bench <package> [--benchtime <duration>] [--count <n>]
        Run Go benchmarks and output METRIC lines.

  loop mcp
        Run as MCP (Model Context Protocol) server over stdio.
        Exposes tools (read_file, write_file, run_command, etc.) for any
        MCP-compatible LLM client (Claude Code, Cursor, Cline, etc.).
        Connect via: claude mcp add -- stdio -- /path/to/loop mcp

  loop ai [--timeout <seconds>] [--provider <name>] [--model <name>]
        Run as a self-contained AI agent. Reads autoresearch.md and
        autoresearch.config.json, then autonomously:
          1. OBSERVE — read project files, run tests, get metrics
          2. ORIENT — analyze state via LLM
          3. DECIDE — plan code changes
          4. ACT — write files, run tests, verify
          5. LOOP — repeat until termination conditions met
        Requires an LLM API key (config file, env var, or --api-key).
        Supported providers: grok, deepseek, openai, ollama
        Example: loop ai --provider grok --api-key xai-...

  loop check [--dir <path>]
        Pre-check project state.

  loop version
        Print version and exit.

  loop help
        Print this help.
`)
}

// cmdInit initializes a new experiment session.
func cmdInit(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: loop init <name> <metric_name> [--unit <unit>] [--direction <lower|higher>]\n")
		os.Exit(1)
	}

	name := args[0]
	metricName := args[1]

	unit := ""
	direction := "lower"

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--unit":
			if i+1 < len(args) {
				i++
				unit = args[i]
			}
		case "--direction":
			if i+1 < len(args) {
				i++
				direction = args[i]
			}
		}
	}

	logger, err := log.NewLogger("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	header := log.ConfigHeader{
		Type:          "config",
		Name:          name,
		MetricName:    metricName,
		MetricUnit:    unit,
		BestDirection: direction,
	}
	if err := logger.AppendConfig(header); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized experiment session: %s (metric: %s, direction: %s)\n", name, metricName, direction)
}

// cmdAuto runs one autonomous iteration: run command, check termination, log.
func cmdAuto(args []string) {
	timeout := 0 * time.Second
	for i := 0; i < len(args); i++ {
		if args[i] == "--timeout" && i+1 < len(args) {
			i++
			secs, err := time.ParseDuration(args[i] + "s")
			if err == nil {
				timeout = secs
			}
		}
	}

	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading autoresearch.config.json: %v\n", err)
		os.Exit(1)
	}

	if cfg.Command == "" {
		fmt.Fprintf(os.Stderr, "error: no \"command\" set in autoresearch.config.json\n")
		os.Exit(1)
	}

	// 2. Resolve working directory
	workingDir := cfg.WorkingDir
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			workingDir = cwd
		}
	}
	if !filepath.IsAbs(workingDir) {
		cwd, _ := os.Getwd()
		workingDir = filepath.Join(cwd, workingDir)
	}

	// 3. Determine current run number — count runs since last config header
	cfgDir := filepath.Dir(filepath.Join(workingDir, config.DefaultFileName))
	logPath := filepath.Join(cfgDir, log.DefaultFileName)
	runNum := countRunsSinceLastConfig(logPath) + 1

	// 4. Run the experiment command
	opts := run.Options{
		Timeout: timeout,
		Dir:     workingDir,
	}
	result := run.Run(cfg.Command, opts)

	// 5. Parse all METRIC lines from output
	allMetrics := metric.ParseAll(result.Combined)

	// Always include built-in metrics
	exitCodeMetric := metric.Metric{Name: "exit_code", Value: strconv.Itoa(result.ExitCode)}
	durMsMetric := metric.Metric{Name: "duration_ms", Value: strconv.FormatInt(result.Duration.Milliseconds(), 10)}
	allMetrics = append([]metric.Metric{exitCodeMetric, durMsMetric}, allMetrics...)

	// 6. Find primary metric value
	primaryName := "test_count" // default
	hasHardMetric := cfg.Metric != nil && cfg.Metric.Name != ""
	if hasHardMetric {
		primaryName = cfg.Metric.Name
	}

	var primaryValue float64 = 0
	primaryFound := false
	extraMetrics := map[string]any{}

	for _, m := range allMetrics {
		if m.Name == primaryName {
			if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
				primaryValue = v
				primaryFound = true
			}
		}
		if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
			extraMetrics[m.Name] = v
		} else {
			extraMetrics[m.Name] = m.Value
		}
	}

	if !primaryFound && hasHardMetric {
		fmt.Fprintf(os.Stderr, "warning: primary metric '%s' not found in output\n", primaryName)
	}

	// 7. Check guardrails (soft metrics — must always pass)
	guardrailsFailed := false
	guardrailFailures := []string{}

	for _, g := range cfg.Guardrails {
		passed := evaluateGuardrail(g.Check, allMetrics)
		if !passed {
			guardrailsFailed = true
			guardrailFailures = append(guardrailFailures, g.Name)
			fmt.Fprintf(os.Stderr, "🛑 Guardrail FAILED: %s (%s)\n", g.Name, g.Check)
		}
	}

	// 8. Check termination conditions
	terminated := false
	reason := ""

	// Check max iterations
	maxIters := cfg.MaxIterations
	if cfg.Termination.MaxIterations > 0 {
		maxIters = cfg.Termination.MaxIterations
	}
	if runNum >= maxIters {
		terminated = true
		reason = fmt.Sprintf("max iterations reached (%d)", maxIters)
	}

	// Also check the hard metric target
	if !terminated && hasHardMetric && cfg.Metric.Target.Metric != "" {
		cond := cfg.Metric.Target
		var metricValue float64
		found := false
		for _, m := range allMetrics {
			if m.Name == cond.Metric {
				if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
					metricValue = v
					found = true
				}
				break
			}
		}
		if found {
			condMet := evaluateCondition(cond.Operator, metricValue, cond.Value)
			if condMet {
				terminated = true
				reason = fmt.Sprintf("%s %s %.0f (got %.0f)", cond.Metric, cond.Operator, cond.Value, metricValue)
			}
		}
	}

	// Check legacy metric-based termination conditions
	if !terminated {
		for _, cond := range cfg.Termination.Conditions {
			var metricValue float64
			found := false
			for _, m := range allMetrics {
				if m.Name == cond.Metric {
					if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
						metricValue = v
						found = true
					}
					break
				}
			}
			if !found {
				continue
			}

			condMet := false
			switch cond.Operator {
			case ">=":
				condMet = metricValue >= cond.Value
			case "<=":
				condMet = metricValue <= cond.Value
			case "==":
				condMet = metricValue == cond.Value
			case ">":
				condMet = metricValue > cond.Value
			case "<":
				condMet = metricValue < cond.Value
			default:
				fmt.Fprintf(os.Stderr, "warning: unknown operator '%s' in termination condition\n", cond.Operator)
			}

			if condMet {
				terminated = true
				reason = fmt.Sprintf("%s %s %.0f (got %.0f)", cond.Metric, cond.Operator, cond.Value, metricValue)
				break
			}
		}
	}

	if reason == "" {
		reason = "conditions not yet met (run " + strconv.Itoa(runNum) + ")"
	}

	// 8. Determine status
	status := "discard"
	if guardrailsFailed {
		status = "crash"
		reason = fmt.Sprintf("guardrail(s) failed: %s", strings.Join(guardrailFailures, ", "))
	} else if result.ExitCode != 0 {
		status = "crash"
		reason = fmt.Sprintf("exit code %d", result.ExitCode)
	} else {
		status = "keep"
	}

	// 9. Log the result
	logger, err := log.NewLogger(logPath)
	if err == nil {
		exp := log.Experiment{
			Type:        "experiment",
			Run:         runNum,
			Metric:      primaryValue,
			MetricName:  primaryName,
			Metrics:     extraMetrics,
			Status:      status,
			Description: reason,
		}
		if err := logger.AppendExperiment(exp); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to log: %v\n", err)
		}
	}

	// 10. Output result
	fmt.Printf("---\n")
	fmt.Printf("METRIC exit_code=%d\n", result.ExitCode)
	fmt.Printf("METRIC duration_ms=%d\n", result.Duration.Milliseconds())
	fmt.Printf("METRIC timed_out=%d\n", boolToInt(result.TimedOut))

	// Forward primary metric
	if primaryFound {
		fmt.Printf("METRIC %s=%s\n", primaryName, formatFloat(primaryValue))
	}

	// Print output tail for debugging
	if result.Stdout != "" {
		fmt.Println("---STDOUT_TAIL---")
		fmt.Println(tail(result.Stdout, 10))
	}
	if result.Stderr != "" {
		fmt.Println("---STDERR_TAIL---")
		fmt.Println(tail(result.Stderr, 10))
	}

	// 11. Final verdict — machine-parseable
	if terminated {
		fmt.Printf("✅ LOOP COMPLETE: %s\n", reason)
	} else {
		fmt.Printf("🔄 LOOP CONTINUE: %s\n", reason)
	}
}

// cmdMCP runs the MCP server over stdio.
func cmdMCP(args []string) {
	mcpServer := mcp.NewServer()
	mcpServer.Serve()
}

// cmdAI runs the self-contained AI agent.
func cmdAI(args []string) {
	// Parse flags
	var provider, model, apiKey string
	timeout := 120 * time.Second

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 < len(args) {
				i++
				provider = args[i]
			}
		case "--model":
			if i+1 < len(args) {
				i++
				model = args[i]
			}
		case "--api-key":
			if i+1 < len(args) {
				i++
				apiKey = args[i]
			}
		case "--timeout":
			if i+1 < len(args) {
				i++
				if d, err := time.ParseDuration(args[i] + "s"); err == nil {
					timeout = d
				}
			}
		}
	}

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Determine working directory
	workingDir := cfg.WorkingDir
	if workingDir == "" {
		wd, _ := os.Getwd()
		workingDir = wd
	} else if !filepath.IsAbs(workingDir) {
		cwd, _ := os.Getwd()
		workingDir = filepath.Join(cwd, workingDir)
	}

	// Resolve provider/model/endpoint from config or flags
	aiCfg := cfg.AI
	ep := ""
	if aiCfg != nil {
		if provider == "" && aiCfg.Provider.Provider != "" {
			provider = aiCfg.Provider.Provider
		}
		if model == "" && aiCfg.Provider.Model != "" {
			model = aiCfg.Provider.Model
		}
		if aiCfg.Provider.Endpoint != "" {
			ep = aiCfg.Provider.Endpoint
		}
		if aiCfg.Provider.APIKey != "" && apiKey == "" {
			apiKey = aiCfg.Provider.APIKey
		}
	}

	// Fall back to env var for API key
	if apiKey == "" {
		apiKey = os.Getenv("LOOP_API_KEY")
	}

	// If provider is set but no endpoint, use defaults
	if provider != "" && ep == "" {
		defEp, defModel := llm.ProviderDefaults(provider)
		if defEp != "" {
			ep = defEp
			if model == "" {
				model = defModel
			}
		}
	}

	if ep == "" || model == "" {
		fmt.Fprintf(os.Stderr, "error: no AI provider configured. Set in autoresearch.config.json,\n")
		fmt.Fprintf(os.Stderr, "  or use flags: --provider <name> --model <name> --api-key <key>\n")
		fmt.Fprintf(os.Stderr, "  Supported: grok, deepseek, openai, ollama\n")
		fmt.Fprintf(os.Stderr, "  Or set LOOP_API_KEY env var.\n")
		os.Exit(1)
	}

	// Run security scan on the project before starting
	fmt.Fprintf(os.Stderr, "🔍 Scanning for sensitive data before AI loop...\n")
	findings, err := scanner.ScanDir(workingDir, []string{".git", "node_modules", "vendor", ".next"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: scan error: %v\n", err)
	}
	hasCritical := scanner.PrintFindings(findings)
	if hasCritical {
		abort := scanner.ConfirmScan(findings)
		if abort {
			fmt.Fprintf(os.Stderr, "🛑 Aborted by user — sensitive data detected\n")
			os.Exit(1)
		}
	}

	// Initialize LLM client
	client := llm.NewClient(ep, apiKey, model)

	// Load project context
	ctx, err := planner.LoadContext(workingDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load context: %v\n", err)
	}

	maxIter := 10
	if aiCfg != nil && aiCfg.MaxIterations > 0 {
		maxIter = aiCfg.MaxIterations
	}
	if cfg.MaxIterations > 0 && cfg.MaxIterations < maxIter {
		maxIter = cfg.MaxIterations
	}

	fmt.Fprintf(os.Stderr, "🤖 AI Agent starting: %s (%s)\n", model, ep)
	fmt.Fprintf(os.Stderr, "📋 Objective: %s\n", ctx.Objective)
	fmt.Fprintf(os.Stderr, "🔄 Max iterations: %d\n\n", maxIter)

	// Main AI loop
	for iter := 1; iter <= maxIter; iter++ {
		fmt.Fprintf(os.Stderr, "\n=== Iteration %d/%d ===\n", iter, maxIter)

		// Build conversation
		systemPrompt := ctx.BuildSystemPrompt()
		userPrompt := ctx.BuildUserPrompt()

		messages := []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}

		// Multi-turn tool execution loop
		maxTurns := 5
		for turn := 1; turn <= maxTurns; turn++ {
			fmt.Fprintf(os.Stderr, "🤔 LLM turn %d...\n", turn)

			resp, err := client.Chat(llm.ChatRequest{Messages: messages})
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ LLM error: %v\n", err)
				break
			}

			choice := resp.Choices[0].Message
			messages = append(messages, choice)

			// Parse tool calls
			calls := bridge.ParseToolCalls(choice.Content)

			if len(calls) == 0 {
				// No tool calls — LLM is done thinking
				fmt.Fprintf(os.Stderr, "📝 Plan:\n%s\n", truncate(choice.Content, 400))
				break
			}

			// Execute tool calls
			for _, tc := range calls {
				fmt.Fprintf(os.Stderr, "🔧 %s → ", tc.Tool)
				result := bridge.Execute(tc)
				if result.Error != "" {
					fmt.Fprintf(os.Stderr, "❌ %s\n", result.Error)
				} else {
					fmt.Fprintf(os.Stderr, "%s\n", truncate(strings.Split(result.Output, "\n")[0], 80))
				}

				// Feed result back to conversation
				feedback := fmt.Sprintf("Tool result for %s:\n%s", tc.Tool, result.Output)
				if result.Error != "" {
					feedback = fmt.Sprintf("Tool error for %s: %s", tc.Tool, result.Error)
				}
				messages = append(messages, llm.Message{Role: "user", Content: feedback})
			}
		}

		// Run tests after LLM conversation to verify changes
		if cfg.Command != "" {
			fmt.Fprintf(os.Stderr, "\n🧪 Running tests...\n")
			opts := run.Options{Timeout: timeout, Dir: workingDir}
			result := run.Run(cfg.Command, opts)

			allMetrics := metric.ParseAll(result.Combined)
			for _, m := range allMetrics {
				fmt.Fprintf(os.Stderr, "   METRIC %s=%s\n", m.Name, m.Value)
			}

			if result.ExitCode != 0 {
				fmt.Fprintf(os.Stderr, "⚠️  Tests FAILED (exit %d)\n", result.ExitCode)
				// Feed test failure back to LLM next iteration
			}
		}

		// Refresh context for next iteration
		ctx, _ = planner.LoadContext(workingDir)
	}

	fmt.Fprintf(os.Stderr, "\n✅ AI LOOP COMPLETE (%d iterations)\n", maxIter)
}

// cmdBench runs Go benchmarks and outputs METRIC lines.
func cmdBench(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: loop bench <package> [--benchtime <duration>] [--count <n>]\n")
		os.Exit(1)
	}

	pkg := args[0]
	benchtime := "1x"
	count := "1"

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--benchtime":
			if i+1 < len(args) {
				i++
				benchtime = args[i]
			}
		case "--count":
			if i+1 < len(args) {
				i++
				count = args[i]
			}
		}
	}

	cfg, _ := config.Load("")
	workingDir := cfg.WorkingDir
	if workingDir == "" {
		wd, _ := os.Getwd()
		workingDir = wd
	}
	if !filepath.IsAbs(workingDir) {
		cwd, _ := os.Getwd()
		workingDir = filepath.Join(cwd, workingDir)
	}

	cmd := fmt.Sprintf("go test -bench=. -benchtime=%s -count=%s %s", benchtime, count, pkg)
	opts := run.Options{Dir: workingDir}
	result := run.Run(cmd, opts)

	fmt.Printf("METRIC exit_code=%d\n", result.ExitCode)
	fmt.Printf("METRIC duration_ms=%d\n", result.Duration.Milliseconds())

	// Parse Go benchmark output lines
	// Format: BenchmarkName-10   1000000   123.4 ns/op   64 B/op   6 allocs/op
	benchRe := regexp.MustCompile(`^(Benchmark\w+)(?:-\d+)?\s+\d+\s+([\d.]+)\s+ns/op`)
	allocRe := regexp.MustCompile(`([\d.]+)\s+allocs/op`)
	bytesRe := regexp.MustCompile(`([\d.]+)\s+B/op`)

	lines := strings.Split(result.Combined, "\n")
	for _, line := range lines {
		if m := benchRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			nsop := m[2]
			fmt.Printf("METRIC %s_ns_per_op=%s\n", name, nsop)

			if am := allocRe.FindStringSubmatch(line); am != nil {
				fmt.Printf("METRIC %s_allocs_per_op=%s\n", name, am[1])
			}
			if bm := bytesRe.FindStringSubmatch(line); bm != nil {
				fmt.Printf("METRIC %s_bytes_per_op=%s\n", name, bm[1])
			}
		}
	}

	// Print tail for debugging
	fmt.Println("---STDOUT_TAIL---")
	fmt.Println(tail(result.Stdout, 10))
	if result.Stderr != "" {
		fmt.Println("---STDERR_TAIL---")
		fmt.Println(tail(result.Stderr, 5))
	}

	os.Exit(result.ExitCode)
}

// cmdRun runs a shell command with timing and captures metrics.
func cmdRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: loop run <command> [--timeout <seconds>]\n")
		os.Exit(1)
	}

	timeout := 0 * time.Second
	commandParts := []string{}

	for i := 0; i < len(args); i++ {
		if args[i] == "--timeout" {
			if i+1 < len(args) {
				i++
				secs, err := time.ParseDuration(args[i] + "s")
				if err == nil {
					timeout = secs
				}
			}
		} else {
			commandParts = append(commandParts, args[i])
		}
	}

	command := strings.Join(commandParts, " ")

	// Load config for working directory
	cfg, _ := config.Load("")
	workingDir := cfg.WorkingDir
	if workingDir == "" {
		wd, err := os.Getwd()
		if err == nil {
			workingDir = wd
		}
	}

	// If workingDir is relative, make it absolute relative to cwd
	if !filepath.IsAbs(workingDir) {
		cwd, _ := os.Getwd()
		workingDir = filepath.Join(cwd, workingDir)
	}

	opts := run.Options{
		Timeout: timeout,
		Dir:     workingDir,
	}

	result := run.Run(command, opts)

	// Always output METRIC lines
	fmt.Printf("METRIC exit_code=%d\n", result.ExitCode)
	fmt.Printf("METRIC duration_ms=%d\n", result.Duration.Milliseconds())
	fmt.Printf("METRIC duration_s=%.3f\n", result.Duration.Seconds())

	if result.TimedOut {
		fmt.Println("METRIC timed_out=1")
	} else {
		fmt.Println("METRIC timed_out=0")
	}

	// Parse and forward any METRIC lines from command output
	extracted := metric.ParseAll(result.Combined)
	for _, m := range extracted {
		fmt.Printf("METRIC %s=%s\n", m.Name, m.Value)
	}

	// Print tails for debugging
	fmt.Println("---STDOUT_TAIL---")
	fmt.Println(tail(result.Stdout, 15))
	if result.Stderr != "" {
		fmt.Println("---STDERR_TAIL---")
		fmt.Println(tail(result.Stderr, 10))
	}

	os.Exit(result.ExitCode)
}

// cmdCheck validates the project state.
func cmdCheck(args []string) {
	dir := "."
	for i := 0; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			i++
			dir = args[i]
		}
	}

	checks := 0
	failed := 0

	// Check package.json
	pkgPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgPath); err == nil {
		data, _ := os.ReadFile(pkgPath)
		var pkg map[string]any
		if json.Unmarshal(data, &pkg) == nil {
			fmt.Println("✓ package.json: valid")
			checks++
		} else {
			fmt.Println("✗ package.json: invalid JSON")
			failed++
		}
	} else {
		fmt.Println("✗ package.json: not found")
		failed++
	}

	// Check config
	cfgPath := filepath.Join(dir, config.DefaultFileName)
	if _, err := os.Stat(cfgPath); err == nil {
		if _, err := config.Load(cfgPath); err == nil {
			fmt.Println("✓ autoresearch.config.json: valid")
			checks++
		} else {
			fmt.Println("✗ autoresearch.config.json: invalid")
			failed++
		}
	} else {
		fmt.Println("~ autoresearch.config.json: not present (optional)")
		checks++
	}

	// Check log file
	logPath := filepath.Join(dir, log.DefaultFileName)
	if _, err := os.Stat(logPath); err == nil {
		exps, err := log.ReadAll(logPath)
		if err == nil {
			fmt.Printf("✓ autoresearch.jsonl: %d experiment(s) logged\n", len(exps))
			checks++
		} else {
			fmt.Println("✗ autoresearch.jsonl: unreadable")
			failed++
		}
	} else {
		fmt.Println("~ autoresearch.jsonl: not present (new session)")
		checks++
	}

	if failed > 0 {
		fmt.Printf("\n%d check(s) passed, %d failed\n", checks, failed)
		os.Exit(1)
	}
	fmt.Printf("\nAll %d check(s) passed.\n", checks)
}

func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// evaluateCondition checks if two values satisfy an operator.
func evaluateCondition(op string, actual, target float64) bool {
	switch op {
	case ">=":
		return actual >= target
	case "<=":
		return actual <= target
	case "==":
		return actual == target
	case ">":
		return actual > target
	case "<":
		return actual < target
	default:
		return false
	}
}

// evaluateGuardrail checks a simple guardrail expression like "exit_code == 0" or "test_count == total_tests".
func evaluateGuardrail(check string, metrics []metric.Metric) bool {
	check = strings.TrimSpace(check)

	// Parse: METRIC_A OPERATOR METRIC_B_OR_NUMBER
	// e.g. "exit_code == 0", "test_count == total_tests"
	parts := strings.Fields(check)
	if len(parts) < 3 {
		return false
	}

	leftName := parts[0]
	op := parts[1]
	rightExpr := parts[2]

	// Find left metric
	var leftVal float64
	foundLeft := false
	for _, m := range metrics {
		if m.Name == leftName {
			if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
				leftVal = v
				foundLeft = true
			}
			break
		}
	}
	if !foundLeft {
		return false
	}

	// Parse right side (either a number or another metric name)
	var rightVal float64
	if v, err := strconv.ParseFloat(rightExpr, 64); err == nil {
		rightVal = v
	} else {
		// Try as a metric name
		foundRight := false
		for _, m := range metrics {
			if m.Name == rightExpr {
				if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
					rightVal = v
					foundRight = true
				}
				break
			}
		}
		if !foundRight {
			return false
		}
	}

	return evaluateCondition(op, leftVal, rightVal)
}

// countRunsSinceLastConfig counts experiment entries since the last config header.
func countRunsSinceLastConfig(logPath string) int {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	count := 0
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], `"type":"config"`) {
			break
		}
		if strings.Contains(lines[i], `"type":"experiment"`) {
			count++
		}
	}
	return count
}
