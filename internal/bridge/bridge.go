// Package bridge executes tool calls that an LLM requests during the AI agent loop.
package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ToolCall struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args,omitempty"`
}

type Result struct {
	Tool   string `json:"tool"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func Execute(tc ToolCall) Result {
	switch tc.Tool {
	case "read_file":
		return execReadFile(tc.Args)
	case "write_file":
		return execWriteFile(tc.Args)
	case "run_command":
		return execRunCommand(tc.Args)
	case "list_files":
		return execListFiles(tc.Args)
	case "read_config":
		return execReadConfig()
	case "get_metrics":
		return execGetMetrics()
	case "check_termination":
		return execCheckTermination()
	default:
		return Result{Tool: tc.Tool, Error: fmt.Sprintf("unknown tool: %s", tc.Tool)}
	}
}

// ParseToolCalls extracts tool calls from LLM response text.
// Finds {"tool": "xxx", "args": {...}} JSON blocks anywhere in the text.
func ParseToolCalls(response string) []ToolCall {
	var calls []ToolCall

	remaining := response
	for {
		idx := strings.Index(remaining, `{"tool":"`)
		if idx < 0 {
			idx = strings.Index(remaining, `{ "tool"`)
			if idx < 0 {
				idx = strings.Index(remaining, `{"tool": "`)
				if idx < 0 {
					break
				}
			}
		}

		start := idx
		depth := 0
		end := -1
		inString := false
		for i := start; i < len(remaining); i++ {
			ch := remaining[i]
			if ch == '"' && (i == 0 || remaining[i-1] != '\\') {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}
		if end < 0 {
			break
		}

		jsonStr := remaining[start:end]
		var tc ToolCall
		if err := json.Unmarshal([]byte(jsonStr), &tc); err == nil && tc.Tool != "" {
			calls = append(calls, tc)
		}
		remaining = remaining[end:]
	}

	if len(calls) > 0 {
		return calls
	}

	// Fall back to XML tags: <read_file><path>foo</path></read_file>, etc.
	// Use a simpler approach — find tool names and extract content between tags
	toolNames := []string{"read_file", "write_file", "run_command", "list_files", "read_config", "get_metrics", "check_termination"}
	remaining2 := response
	for _, toolName := range toolNames {
		openTag := "<" + toolName + ">"
		closeTag := "</" + toolName + ">"
		for {
			start := strings.Index(remaining2, openTag)
			if start < 0 {
				break
			}
			innerStart := start + len(openTag)
			end := strings.Index(remaining2[innerStart:], closeTag)
			if end < 0 {
				break
			}
			innerXML := remaining2[innerStart : innerStart+end]

			switch toolName {
			case "read_file":
				calls = append(calls, parseXMLReadFile(innerXML))
			case "write_file":
				calls = append(calls, parseXMLWriteFile(innerXML))
			case "run_command":
				calls = append(calls, parseXMLRunCommand(innerXML))
			case "list_files":
				calls = append(calls, parseXMLListFiles(innerXML))
			case "read_config":
				calls = append(calls, ToolCall{Tool: "read_config"})
			case "get_metrics":
				calls = append(calls, ToolCall{Tool: "get_metrics"})
			case "check_termination":
				calls = append(calls, ToolCall{Tool: "check_termination"})
			}
			remaining2 = remaining2[innerStart+end+len(closeTag):]
		}
	}

	return calls
}

func execReadFile(args json.RawMessage) Result {
	var p struct{ Path string }
	_ = json.Unmarshal(args, &p)
	if p.Path == "" {
		return Result{Tool: "read_file", Error: "path is required"}
	}
	path := resolvePath(p.Path)
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Tool: "read_file", Error: fmt.Sprintf("read: %v", err)}
	}
	text := string(data)
	if len(text) > 8000 {
		text = text[:8000] + "\n... [truncated]"
	}
	return Result{Tool: "read_file", Output: fmt.Sprintf("%s (%d bytes):\n%s", p.Path, len(data), text)}
}

func execWriteFile(args json.RawMessage) Result {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Path == "" {
		return Result{Tool: "write_file", Error: "path is required"}
	}
	path := resolvePath(p.Path)
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(p.Content), 0644); err != nil {
		return Result{Tool: "write_file", Error: fmt.Sprintf("write: %v", err)}
	}
	return Result{Tool: "write_file", Output: fmt.Sprintf("Wrote %d bytes to %s", len(p.Content), p.Path)}
}

func execRunCommand(args json.RawMessage) Result {
	var p struct {
		Command string  `json:"command"`
		Timeout float64 `json:"timeout,omitempty"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Command == "" {
		return Result{Tool: "run_command", Error: "command is required"}
	}
	if p.Timeout <= 0 {
		p.Timeout = 60
	}
	timeout := time.Duration(p.Timeout * float64(time.Second))
	cmd := exec.Command("sh", "-c", p.Command)
	wd, _ := os.Getwd()
	cmd.Dir = wd

	done := make(chan error, 1)
	start := time.Now()
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	go func() { done <- cmd.Run() }()

	var exitCode int
	timedOut := false
	select {
	case err := <-done:
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				exitCode = e.ExitCode()
			} else {
				exitCode = -1
			}
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		exitCode = -1
		timedOut = true
	}
	return Result{Tool: "run_command", Output: fmt.Sprintf(
		"Exit: %d | %dms%s\nSTDOUT:\n%s\nSTDERR:\n%s",
		exitCode, time.Since(start).Milliseconds(),
		map[bool]string{true: " | TIMEOUT"}[timedOut],
		truncStr(stdout.String(), 2000), truncStr(stderr.String(), 1000)),
	}
}

func execListFiles(args json.RawMessage) Result {
	var p struct {
		Dir     string `json:"dir,omitempty"`
		Pattern string `json:"pattern,omitempty"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Dir == "" {
		p.Dir = "."
	}
	dir := resolvePath(p.Dir)

	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			n := info.Name()
			if n == ".git" || n == "node_modules" || n == "vendor" || n == ".next" || strings.HasPrefix(n, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if p.Pattern != "" {
			if m, _ := filepath.Match(p.Pattern, info.Name()); !m {
				return nil
			}
		}
		rel, _ := filepath.Rel(dir, path)
		files = append(files, rel)
		return nil
	})
	return Result{Tool: "list_files", Output: fmt.Sprintf("%d files:\n%s", len(files), strings.Join(files, "\n"))}
}

func execReadConfig() Result {
	wd, _ := os.Getwd()
	data, err := os.ReadFile(filepath.Join(wd, "autoresearch.config.json"))
	if err != nil {
		return Result{Tool: "read_config", Error: fmt.Sprintf("no config: %v", err)}
	}
	return Result{Tool: "read_config", Output: string(data)}
}

func execGetMetrics() Result {
	wd, _ := os.Getwd()
	data, err := os.ReadFile(filepath.Join(wd, "autoresearch.jsonl"))
	if err != nil {
		return Result{Tool: "get_metrics", Output: "(no runs yet)"}
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var last []string
	for i := len(lines) - 1; i >= 0 && len(last) < 3; i-- {
		if strings.Contains(lines[i], `"metric"`) {
			last = append([]string{lines[i]}, last...)
		}
	}
	return Result{Tool: "get_metrics", Output: strings.Join(last, "\n")}
}

func execCheckTermination() Result {
	wd, _ := os.Getwd()
	logData, _ := os.ReadFile(filepath.Join(wd, "autoresearch.jsonl"))
	cfgData, _ := os.ReadFile(filepath.Join(wd, "autoresearch.config.json"))

	var logLines []struct{ Metric float64 `json:"metric"` }
	for _, line := range strings.Split(string(logData), "\n") {
		var e struct{ Metric float64 `json:"metric"` }
		if json.Unmarshal([]byte(line), &e) == nil {
			logLines = append(logLines, e)
		}
	}

	var cfg struct {
		Metric struct {
			Target struct {
				Metric   string  `json:"metric"`
				Operator string  `json:"operator"`
				Value    float64 `json:"value"`
			} `json:"target"`
		} `json:"metric"`
	}
	json.Unmarshal(cfgData, &cfg)

	lastVal := float64(0)
	if len(logLines) > 0 {
		lastVal = logLines[len(logLines)-1].Metric
	}

	if cfg.Metric.Target.Metric == "" {
		return Result{Tool: "check_termination", Output: fmt.Sprintf("Last metric: %.0f (no target)", lastVal)}
	}

	met := false
	switch cfg.Metric.Target.Operator {
	case ">=":
		met = lastVal >= cfg.Metric.Target.Value
	case "<=":
		met = lastVal <= cfg.Metric.Target.Value
	case "==":
		met = lastVal == cfg.Metric.Target.Value
	}

	if met {
		return Result{Tool: "check_termination", Output: fmt.Sprintf("TARGET MET: %s %s %.0f (got %.0f)", cfg.Metric.Target.Metric, cfg.Metric.Target.Operator, cfg.Metric.Target.Value, lastVal)}
	}
	return Result{Tool: "check_termination", Output: fmt.Sprintf("Not met: %s %s %.0f (got %.0f)", cfg.Metric.Target.Metric, cfg.Metric.Target.Operator, cfg.Metric.Target.Value, lastVal)}
}

func resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}

// XML parsers for fallback format
func parseXMLReadFile(inner string) ToolCall {
	re := regexp.MustCompile(`<path>(.*?)</path>`)
	m := re.FindStringSubmatch(inner)
	path := ""
	if m != nil {
		path = strings.TrimSpace(m[1])
	}
	args, _ := json.Marshal(map[string]string{"path": path})
	return ToolCall{Tool: "read_file", Args: args}
}

func parseXMLWriteFile(inner string) ToolCall {
	pathRe := regexp.MustCompile(`<path>(.*?)</path>`)
	cntRe := regexp.MustCompile(`<content>(.*?)</content>`)
	path := ""
	content := ""
	if m := pathRe.FindStringSubmatch(inner); m != nil {
		path = strings.TrimSpace(m[1])
	}
	if m := cntRe.FindStringSubmatch(inner); m != nil {
		content = m[1]
	}
	args, _ := json.Marshal(map[string]string{"path": path, "content": content})
	return ToolCall{Tool: "write_file", Args: args}
}

func parseXMLRunCommand(inner string) ToolCall {
	re := regexp.MustCompile(`<command>(.*?)</command>`)
	m := re.FindStringSubmatch(inner)
	cmd := ""
	if m != nil {
		cmd = strings.TrimSpace(m[1])
	}
	args, _ := json.Marshal(map[string]string{"command": cmd})
	return ToolCall{Tool: "run_command", Args: args}
}

func parseXMLListFiles(inner string) ToolCall {
	pathRe := regexp.MustCompile(`<path>(.*?)</path>`)
	patRe := regexp.MustCompile(`<pattern>(.*?)</pattern>`)
	dir := ""
	pattern := ""
	if m := pathRe.FindStringSubmatch(inner); m != nil {
		dir = strings.TrimSpace(m[1])
	}
	if m := patRe.FindStringSubmatch(inner); m != nil {
		pattern = strings.TrimSpace(m[1])
	}
	args, _ := json.Marshal(map[string]string{"dir": dir, "pattern": pattern})
	return ToolCall{Tool: "list_files", Args: args}
}
