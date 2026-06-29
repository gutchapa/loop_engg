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

	"github.com/gutchapa/loop/internal/config"
	"github.com/gutchapa/loop/internal/log"
	"github.com/gutchapa/loop/internal/metric"
	"github.com/gutchapa/loop/internal/run"
)

const version = "1.1.0"

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
	fmt.Print(`Loop Engineering CLI — autonomous experiment loop tooling.

Usage:
  loop init <name> <metric_name> [--unit <unit>] [--direction <lower|higher>]
        Initialize a new experiment session. Writes a config header to
        autoresearch.jsonl.

  loop run <command> [--timeout <seconds>]
        Run a shell command, measure wall-clock duration, capture output.
        Outputs METRIC lines for duration, exit_code, and any METRIC
        lines found in the command's own output.

  loop auto [--timeout <seconds>]
        Run one experiment iteration autonomously:
          1. Load autoresearch.config.json for command + termination rules
          2. Execute the command with timing
          3. Parse primary metric from METRIC lines
          4. Check termination conditions (metric thresholds, max iterations)
          5. Log result to autoresearch.jsonl
          6. Exit with verdict: "LOOP COMPLETE" or "LOOP CONTINUE"

  loop bench <package> [--benchtime <duration>] [--count <n>]
        Run Go benchmarks for a package and output results as METRIC lines.
        Parses standard Go benchmark output (ns/op, allocs/op, B/op).
        Example: loop bench ./examples/fintech-pay/ --benchtime 100x

  loop check [--dir <path>]
        Pre-check a project: validate package.json exists, config loads,
        and log file is readable. Exits 0 on success.

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

	// 3. Determine current run number
	runNum := 0
	cfgDir := filepath.Dir(filepath.Join(workingDir, config.DefaultFileName))
	logPath := filepath.Join(cfgDir, log.DefaultFileName)
	lastRun, err := log.LastRun(logPath)
	if err == nil {
		runNum = lastRun + 1
	} else {
		runNum = 1
	}

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
	if cfg.MetricName != "" {
		primaryName = cfg.MetricName
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
		// Store in extra metrics
		if v, err := strconv.ParseFloat(m.Value, 64); err == nil {
			extraMetrics[m.Name] = v
		} else {
			extraMetrics[m.Name] = m.Value
		}
	}

	if !primaryFound {
		fmt.Fprintf(os.Stderr, "warning: primary metric '%s' not found in output\n", primaryName)
	}

	// 7. Check termination conditions
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

	// Check metric-based termination conditions
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
	if result.ExitCode == 0 {
		status = "keep"
	} else {
		status = "crash"
		reason = fmt.Sprintf("exit code %d", result.ExitCode)
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
