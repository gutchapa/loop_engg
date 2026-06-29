// Package diagnose parses command output to classify failures and suggest fixes.
// It's the "infrastructure intelligence" layer — it knows the difference between
// a code bug, a missing dependency, a version conflict, and an environment issue.
// It queries the learn/ knowledge store for known fixes before falling back to
// pattern-based suggestions.
package diagnose

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gutchapa/loop/internal/learn"
)

// ProblemClass categorizes the type of failure.
type ProblemClass string

const (
	ClassDepMissing  ProblemClass = "dep_missing"  // package/module not found
	ClassDepConflict ProblemClass = "dep_conflict"  // version conflict
	ClassBuildError  ProblemClass = "build_error"   // compile/syntax error
	ClassTestFailure ProblemClass = "test_failure"  // assertion/test failed
	ClassConfigError ProblemClass = "config_error"  // config file or flag issue
	ClassEnvIssue    ProblemClass = "env_issue"     // OS, permission, missing tool
	ClassNetworkErr  ProblemClass = "network_error" // connection refused/timeout
	ClassCodeBug     ProblemClass = "code_bug"      // runtime panic, nil pointer
	ClassUnknown     ProblemClass = "unknown"
)

// Severity indicates how bad the failure is.
type Severity string

const (
	SevFatal   Severity = "fatal"   // cannot proceed without fixing
	SevWarning Severity = "warning" // can ignore but should fix
	SevInfo    Severity = "info"    // FYI
)

// Finding is a single detected problem in the output.
type Finding struct {
	Class    ProblemClass `json:"class"`
	Severity Severity     `json:"severity"`
	Message  string       `json:"message"`  // human-readable one-liner
	Detail   string       `json:"detail"`   // full error line from output
	File     string       `json:"file"`     // file path if parseable
	Line     int          `json:"line"`     // line number if parseable
	Package  string       `json:"package"`  // npm/go/pip package name
	Version  string       `json:"version"`  // version if mentioned
}

// Diagnosis is the full result of analyzing a command failure.
type Diagnosis struct {
	ExitCode   int       `json:"exit_code"`
	IsCodeBug  bool      `json:"is_code_bug"`  // true = need to fix source code
	IsInfra    bool      `json:"is_infra"`     // true = env/dep/config issue (not code)
	Findings   []Finding `json:"findings"`
	KnownFix   *learn.Entry `json:"known_fix,omitempty"` // from knowledge store
	Suggestion string    `json:"suggestion"`  // recommended action
}

// Analyze examines the combined stdout+stderr of a failed command and
// produces a diagnosis. It queries the knowledge store for known fixes.
func Analyze(exitCode int, combined string, store *learn.KnowledgeStore) *Diagnosis {
	d := &Diagnosis{
		ExitCode: exitCode,
		Findings: []Finding{},
	}

	if exitCode == 0 {
		d.Suggestion = "no errors detected"
		return d
	}

	// Run all pattern matchers
	d.Findings = append(d.Findings, matchNPM(combined)...)
	d.Findings = append(d.Findings, matchGo(combined)...)
	d.Findings = append(d.Findings, matchPython(combined)...)
	d.Findings = append(d.Findings, matchShell(combined)...)
	d.Findings = append(d.Findings, matchGeneric(combined)...)

	// Classify: is this infrastructure or code?
	for _, f := range d.Findings {
		switch f.Class {
		case ClassDepMissing, ClassDepConflict, ClassConfigError, ClassEnvIssue, ClassNetworkErr:
			d.IsInfra = true
		case ClassBuildError, ClassTestFailure, ClassCodeBug:
			d.IsCodeBug = true
		}
	}

	// If we couldn't classify but exit code != 0, it's probably a code bug
	if len(d.Findings) == 0 && exitCode != 0 {
		d.IsCodeBug = true
		d.Findings = append(d.Findings, Finding{
			Class:    ClassUnknown,
			Severity: SevWarning,
			Message:  fmt.Sprintf("command failed with exit code %d (cause unknown)", exitCode),
			Detail:   firstLines(combined, 3),
		})
	}

	// Query knowledge store for known fixes
	if store != nil {
		for _, f := range d.Findings {
			entry := store.FindInfraFix(f.Detail, "")
			if entry != nil {
				d.KnownFix = entry
				d.Suggestion = fmt.Sprintf("Known fix: %s → %s", entry.Title, entry.Fix)
				return d
			}
		}
		// Also try the full error text
		if d.KnownFix == nil {
			if entry := store.FindInfraFix(combined, ""); entry != nil {
				d.KnownFix = entry
				d.Suggestion = fmt.Sprintf("Known fix: %s → %s", entry.Title, entry.Fix)
				return d
			}
		}
	}

	// Generate suggestion from patterns
	d.Suggestion = suggestAction(d)
	return d
}

// Summary returns a human-readable one-liner for the diagnosis.
func (d *Diagnosis) Summary() string {
	if d.ExitCode == 0 {
		return "✅ All checks passed"
	}
	if d.KnownFix != nil {
		return fmt.Sprintf("🔧 Known fix available: %s", d.KnownFix.Title)
	}
	switch {
	case d.IsInfra:
		return fmt.Sprintf("🏗️  Infrastructure issue (%d findings) — not a code bug", len(d.Findings))
	case d.IsCodeBug:
		return fmt.Sprintf("🐛 Code bug or test failure (%d findings)", len(d.Findings))
	default:
		return fmt.Sprintf("❌ Failed with exit %d (cause unknown)", d.ExitCode)
	}
}

// ShouldRetry returns true if the error is likely transient (network, timeout).
func (d *Diagnosis) ShouldRetry() bool {
	for _, f := range d.Findings {
		if f.Class == ClassNetworkErr {
			return true
		}
	}
	return false
}

// --- Pattern Matchers ---

var (
	// NPM / Node.js
	npmErrRe       = regexp.MustCompile(`npm ERR!\s+(.+)`)
	npmMissingRe   = regexp.MustCompile(`(?i)(cannot find module|module not found)\s+['"]?([^'"]+)['"]?`)
	npmConflictRe  = regexp.MustCompile(`(?i)(conflicting peer dependency|unable to resolve dependency).*?(@?[\w@/.-]+)`)
	npmERESOLVERe  = regexp.MustCompile(`ERESOLVE.*?(peer\s+[\w@/.-]+)`)
	nodeModuleRe   = regexp.MustCompile(`(?i)(cannot find module)\s+'([^']+)'`)
)

func matchNPM(output string) []Finding {
	var findings []Finding
	for _, m := range npmErrRe.FindAllStringSubmatch(output, -1) {
		msg := strings.TrimSpace(m[1])
		class := classifyNPMError(msg)
		findings = append(findings, Finding{
			Class:    class,
			Severity: SevFatal,
			Message:  msg,
			Detail:   m[0],
		})
	}
	for _, m := range npmMissingRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassDepMissing,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Missing module: %s", m[2]),
			Detail:   m[0],
			Package:  m[2],
		})
	}
	for _, m := range npmConflictRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassDepConflict,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Dependency conflict: %s", m[2]),
			Detail:   m[0],
		})
	}
	for _, m := range npmERESOLVERe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassDepConflict,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Dependency resolution failed: %s", m[1]),
			Detail:   m[0],
		})
	}
	for _, m := range nodeModuleRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassDepMissing,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Cannot find module: %s", m[2]),
			Detail:   m[0],
			Package:  m[2],
		})
	}
	return findings
}

func classifyNPMError(msg string) ProblemClass {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "peer dep") || strings.Contains(lower, "conflict") || strings.Contains(lower, "eresolve"):
		return ClassDepConflict
	case strings.Contains(lower, "cannot find") || strings.Contains(lower, "enoent") || strings.Contains(lower, "404"):
		return ClassDepMissing
	case strings.Contains(lower, "permission") || strings.Contains(lower, "eacces"):
		return ClassEnvIssue
	case strings.Contains(lower, "syntax") || strings.Contains(lower, "unexpected token"):
		return ClassBuildError
	default:
		return ClassUnknown
	}
}

// Go
var (
	goBuildErrRe  = regexp.MustCompile(`^(.+\.go):(\d+):(\d+):\s+(.+)`)
	goMissingRe   = regexp.MustCompile(`(?i)cannot find package\s+"([^"]+)"`)
	goUndefinedRe = regexp.MustCompile(`(?i)undefined:\s+(\S+)`)
	goImportCycle = regexp.MustCompile(`(?i)import cycle not allowed`)
)

func matchGo(output string) []Finding {
	var findings []Finding
	for _, m := range goBuildErrRe.FindAllStringSubmatch(output, -1) {
		file, lineStr, _, msg := m[1], m[2], m[3], m[4]
		line := 0
		fmt.Sscanf(lineStr, "%d", &line)
		findings = append(findings, Finding{
			Class:    ClassBuildError,
			Severity: SevFatal,
			Message:  fmt.Sprintf("%s:%d: %s", file, line, msg),
			Detail:   m[0],
			File:     file,
			Line:     line,
		})
	}
	for _, m := range goMissingRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassDepMissing,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Missing Go package: %s", m[1]),
			Detail:   m[0],
			Package:  m[1],
		})
	}
	for _, m := range goUndefinedRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassBuildError,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Undefined identifier: %s", m[1]),
			Detail:   m[0],
		})
	}
	if goImportCycle.MatchString(output) {
		findings = append(findings, Finding{
			Class:    ClassBuildError,
			Severity: SevFatal,
			Message:  "Import cycle detected",
			Detail:   goImportCycle.FindString(output),
		})
	}
	return findings
}

// Python
var (
	pyTracebackRe  = regexp.MustCompile(`Traceback \(most recent call last\)`)
	pyModuleErrRe  = regexp.MustCompile(`(?i)(ModuleNotFoundError|ImportError):\s*(.+)`)
	pySyntaxErrRe  = regexp.MustCompile(`(?i)SyntaxError:\s*(.+)`)
	pipMissingRe   = regexp.MustCompile(`(?i)(ERROR: Could not find a version|No matching distribution)\s+(.+)`)
)

func matchPython(output string) []Finding {
	var findings []Finding
	if pyTracebackRe.MatchString(output) {
		for _, m := range pyModuleErrRe.FindAllStringSubmatch(output, -1) {
			findings = append(findings, Finding{
				Class:    ClassDepMissing,
				Severity: SevFatal,
				Message:  fmt.Sprintf("%s", strings.TrimSpace(m[2])),
				Detail:   m[0],
			})
		}
		for _, m := range pySyntaxErrRe.FindAllStringSubmatch(output, -1) {
			findings = append(findings, Finding{
				Class:    ClassBuildError,
				Severity: SevFatal,
				Message:  fmt.Sprintf("Syntax error: %s", strings.TrimSpace(m[1])),
				Detail:   m[0],
			})
		}
		if len(findings) == 0 {
			findings = append(findings, Finding{
				Class:    ClassCodeBug,
				Severity: SevFatal,
				Message:  "Python traceback (unclassified)",
				Detail:   firstLines(output, 5),
			})
		}
	}
	for _, m := range pipMissingRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassDepMissing,
			Severity: SevFatal,
			Message:  fmt.Sprintf("pip missing package: %s", strings.TrimSpace(m[2])),
			Detail:   m[0],
		})
	}
	return findings
}

// Generic Shell / System
var (
	cmdNotFoundRe  = regexp.MustCompile(`(?i)(\w+): command not found`)
	permDeniedRe   = regexp.MustCompile(`(?i)permission denied`)
	noSpaceRe      = regexp.MustCompile(`(?i)(no space left|disk full)`)
	connRefusedRe  = regexp.MustCompile(`(?i)(connection refused|connection reset|no route to host|network is unreachable)`)
	timeoutRe      = regexp.MustCompile(`(?i)(timeout|timed out|deadline exceeded)`)
	oomRe          = regexp.MustCompile(`(?i)(out of memory|OOM|killed)`)
	fileNotFoundRe = regexp.MustCompile(`(?i)(no such file or directory|cannot access|cannot open)`)
)

func matchShell(output string) []Finding {
	var findings []Finding
	for _, m := range cmdNotFoundRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassEnvIssue,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Command not found: %s (is it installed?)", m[1]),
			Detail:   m[0],
			Package:  m[1],
		})
	}
	if permDeniedRe.MatchString(output) {
		findings = append(findings, Finding{
			Class:    ClassEnvIssue,
			Severity: SevFatal,
			Message:  "Permission denied — check file permissions or use sudo",
			Detail:   permDeniedRe.FindString(output),
		})
	}
	if noSpaceRe.MatchString(output) {
		findings = append(findings, Finding{
			Class:    ClassEnvIssue,
			Severity: SevFatal,
			Message:  "No disk space left",
			Detail:   noSpaceRe.FindString(output),
		})
	}
	for _, m := range connRefusedRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassNetworkErr,
			Severity: SevFatal,
			Message:  fmt.Sprintf("Network error: %s", m[1]),
			Detail:   m[0],
		})
	}
	for _, m := range timeoutRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassNetworkErr,
			Severity: SevWarning,
			Message:  fmt.Sprintf("Timeout: %s", m[1]),
			Detail:   m[0],
		})
	}
	if oomRe.MatchString(output) {
		findings = append(findings, Finding{
			Class:    ClassEnvIssue,
			Severity: SevFatal,
			Message:  "Out of memory — process was killed",
			Detail:   oomRe.FindString(output),
		})
	}
	for _, m := range fileNotFoundRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassConfigError,
			Severity: SevFatal,
			Message:  fmt.Sprintf("File not found: %s", m[0]),
			Detail:   m[0],
		})
	}
	return findings
}

// Generic: catch-all for exit codes and common failure patterns.
var (
	assertFailRe  = regexp.MustCompile(`(?i)(assertion|assert|expected.*but got|FAIL|FAILED)`)
	panicRe       = regexp.MustCompile(`(?i)(panic:|runtime error|segmentation fault|null pointer|index out of range)`)
	configErrRe   = regexp.MustCompile(`(?i)(invalid configuration|config error|bad config|unknown flag|invalid flag)`)
	exitNonZeroRe = regexp.MustCompile(`(?i)exit status (\d+)`)
)

func matchGeneric(output string) []Finding {
	var findings []Finding
	for _, m := range assertFailRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassTestFailure,
			Severity: SevWarning,
			Message:  m[0],
			Detail:   m[0],
		})
	}
	for _, m := range panicRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassCodeBug,
			Severity: SevFatal,
			Message:  m[0],
			Detail:   m[0],
		})
	}
	for _, m := range configErrRe.FindAllStringSubmatch(output, -1) {
		findings = append(findings, Finding{
			Class:    ClassConfigError,
			Severity: SevFatal,
			Message:  m[0],
			Detail:   m[0],
		})
	}
	return findings
}

// --- Suggestion Engine ---

func suggestAction(d *Diagnosis) string {
	if d.ExitCode == 0 {
		return "no action needed"
	}
	if len(d.Findings) == 0 {
		return fmt.Sprintf("unexplained failure (exit %d) — check the output manually", d.ExitCode)
	}

	// Prioritize by class
	classes := map[ProblemClass]int{}
	for _, f := range d.Findings {
		classes[f.Class]++
	}

	switch {
	case classes[ClassDepConflict] > 0:
		return fmt.Sprintf("🔧 Dependency conflict detected. Try resolving version constraints. If npm: check package.json versions or use --legacy-peer-deps.")
	case classes[ClassDepMissing] > 0:
		pkgs := []string{}
		for _, f := range d.Findings {
			if f.Class == ClassDepMissing && f.Package != "" {
				pkgs = append(pkgs, f.Package)
			}
		}
		if len(pkgs) > 0 {
			return fmt.Sprintf("📦 Missing packages: %s. Run install/update.", strings.Join(uniqueStrs(pkgs), ", "))
		}
		return "📦 Missing dependency detected. Run install/update."
	case classes[ClassEnvIssue] > 0:
		return "🏗️  Environment issue. Check: is the tool installed? Are permissions correct? Is there disk space?"
	case classes[ClassNetworkErr] > 0:
		return "🌐 Network error. Retry the command — this may be transient."
	case classes[ClassBuildError] > 0:
		fileRefs := []string{}
		for _, f := range d.Findings {
			if f.File != "" {
				ref := f.File
				if f.Line > 0 {
					ref = fmt.Sprintf("%s:%d", f.File, f.Line)
				}
				fileRefs = append(fileRefs, ref)
			}
		}
		if len(fileRefs) > 0 {
			return fmt.Sprintf("🏗️  Build error in: %s. Fix the syntax/type errors.", strings.Join(uniqueStrs(fileRefs), ", "))
		}
		return "🏗️  Build error. Check the compiler output for syntax/type errors."
	case classes[ClassTestFailure] > 0:
		return "🧪 Test failure. Examine the assertion errors and fix the code or tests."
	case classes[ClassCodeBug] > 0:
		return "🐛 Runtime error detected. Check the stack trace and fix the code."
	case classes[ClassConfigError] > 0:
		return "⚙️  Configuration error. Check config file format, flags, and paths."
	default:
		return fmt.Sprintf("❌ Command failed (exit %d). %d findings, review output.", d.ExitCode, len(d.Findings))
	}
}

// Report returns a detailed multi-line diagnostic report.
func (d *Diagnosis) Report() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("═ Diagnosis (exit %d) ═\n", d.ExitCode))
	b.WriteString(fmt.Sprintf("  Type:    %s\n", d.classLabel()))
	b.WriteString(fmt.Sprintf("  Fix:     %s\n", d.Suggestion))
	if d.KnownFix != nil {
		b.WriteString(fmt.Sprintf("  Memory:  %s → %s\n", d.KnownFix.Title, d.KnownFix.Fix))
	}
	if len(d.Findings) > 0 {
		b.WriteString(fmt.Sprintf("  Findings (%d):\n", len(d.Findings)))
		for i, f := range d.Findings {
			b.WriteString(fmt.Sprintf("    %d. [%s] %s\n", i+1, f.Class, f.Message))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// ToKnowledgeEntry converts the primary finding into a learnable entry for the knowledge store.
func (d *Diagnosis) ToKnowledgeEntry(fix string) learn.Entry {
	entry := learn.Entry{
		Kind:     "infra_fix",
		Title:    d.Summary(),
		Fix:      fix,
	}
	if len(d.Findings) > 0 {
		f := d.Findings[0]
		entry.Description = f.Detail
		entry.Triggers = []string{f.Detail, f.Message}
		entry.Conditions = map[string]string{
			"class": string(f.Class),
		}
	}
	return entry
}

// AddToKnowledge saves the diagnosis to the knowledge store as a learnable entry.
func (d *Diagnosis) AddToKnowledge(store *learn.KnowledgeStore, fix string) error {
	entry := d.ToKnowledgeEntry(fix)
	return store.Add(entry)
}

func (d *Diagnosis) classLabel() string {
	switch {
	case d.IsInfra:
		return "infrastructure"
	case d.IsCodeBug:
		return "code bug / test failure"
	default:
		return "unknown"
	}
}

// --- Helpers ---

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func uniqueStrs(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
