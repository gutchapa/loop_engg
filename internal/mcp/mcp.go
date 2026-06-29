// Package mcp implements a Model Context Protocol (MCP) server over stdio.
// Exposes tools (read_file, write_file, run_command, etc.) for any MCP-compatible
// LLM client (Claude Code, Cursor, Cline, etc.).
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// JSON-RPC 2.0 message types
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *ErrorObj       `json:"error,omitempty"`
}

type ErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Tool definition for tools/list
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Content item for tool results
type ContentItem struct {
	Type string `json:"type"` // "text" or "resource"
	Text string `json:"text,omitempty"`
}

// ToolResult is the result of a tools/call
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Server represents an MCP server instance
type Server struct {
	reader *bufio.Scanner
	tools  map[string]ToolHandler
}

type ToolHandler func(args json.RawMessage) (any, error)

// NewServer creates a new MCP server
func NewServer() *Server {
	s := &Server{
		reader: bufio.NewScanner(os.Stdin),
		tools:  make(map[string]ToolHandler),
	}

	// Register built-in tools
	s.tools["read_file"] = s.handleReadFile
	s.tools["write_file"] = s.handleWriteFile
	s.tools["run_command"] = s.handleRunCommand
	s.tools["read_config"] = s.handleReadConfig
	s.tools["check_termination"] = s.handleCheckTermination
	s.tools["list_files"] = s.handleListFiles
	s.tools["get_metrics"] = s.handleGetMetrics

	return s
}

// Serve runs the MCP server, reading JSON-RPC requests from stdin
// and writing responses to stdout.
func (s *Server) Serve() {
	// Use a line-based scanner for JSON-RPC
	for s.reader.Scan() {
		line := strings.TrimSpace(s.reader.Text())
		if line == "" {
			continue
		}

		// Try to parse as request
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// Try as notification
			var notif Notification
			if err2 := json.Unmarshal([]byte(line), &notif); err2 != nil {
				s.writeError(nil, -32700, fmt.Sprintf("Parse error: %v", err))
				continue
			}
			s.handleNotification(notif)
			continue
		}

		s.handleRequest(req)
	}

	if err := s.reader.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP read error: %v\n", err)
		os.Exit(1)
	}
}

func (s *Server) handleRequest(req Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req.ID)
	case "tools/list":
		s.handleToolsList(req.ID)
	case "tools/call":
		s.handleToolCall(req.ID, req.Params)
	default:
		s.writeError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleNotification(notif Notification) {
	// Notifications have no response. We ignore them.
	// Common: "notifications/initialized"
}

func (s *Server) handleInitialize(id json.RawMessage) {
	result := map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]string{
			"name":    "loop-engg-mcp",
			"version": "1.1.0",
		},
	}
	s.writeResult(id, result)
}

func (s *Server) handleToolsList(id json.RawMessage) {
	tools := []ToolDefinition{}

	defs := []struct {
		name        string
		description string
		schema      string
	}{
		{
			"read_file",
			"Read a file from the project. Returns content lines. Use offset and limit for large files.",
			`{"type":"object","properties":{"path":{"type":"string","description":"File path relative to project root"},"offset":{"type":"number","description":"Line offset (1-indexed), optional"},"limit":{"type":"number","description":"Max lines to return, optional"}},"required":["path"]}`,
		},
		{
			"write_file",
			"Write or overwrite a file in the project. Creates parent directories automatically.",
			`{"type":"object","properties":{"path":{"type":"string","description":"File path relative to project root"},"content":{"type":"string","description":"File content"}},"required":["path","content"]}`,
		},
		{
			"run_command",
			"Execute a shell command in the project directory. Returns stdout, stderr, exit code, and duration.",
			`{"type":"object","properties":{"command":{"type":"string","description":"Shell command to execute"},"timeout":{"type":"number","description":"Timeout in seconds, optional (default 60)"}},"required":["command"]}`,
		},
		{
			"list_files",
			"List files in a directory matching optional pattern.",
			`{"type":"object","properties":{"dir":{"type":"string","description":"Directory to list, default '.'"},"pattern":{"type":"string","description":"Optional glob pattern (e.g., '*.go')"}},"required":[]}`,
		},
		{
			"read_config",
			"Read the autoresearch.config.json file for current experiment settings.",
			`{"type":"object","properties":{},"required":[]}`,
		},
		{
			"get_metrics",
			"Get current metrics from the last run (test count, exit code, duration).",
			`{"type":"object","properties":{},"required":[]}`,
		},
		{
			"check_termination",
			"Check if experiment termination conditions are met.",
			`{"type":"object","properties":{},"required":[]}`,
		},
	}

	for _, d := range defs {
		tools = append(tools, ToolDefinition{
			Name:        d.name,
			Description: d.description,
			InputSchema: json.RawMessage(d.schema),
		})
	}

	s.writeResult(id, map[string]any{"tools": tools})
}

func (s *Server) handleToolCall(id json.RawMessage, params json.RawMessage) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		s.writeError(id, -32602, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	handler, ok := s.tools[call.Name]
	if !ok {
		s.writeError(id, -32601, fmt.Sprintf("Tool not found: %s", call.Name))
		return
	}

	result, err := handler(call.Arguments)
	if err != nil {
		s.writeResult(id, ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		})
		return
	}

	s.writeResult(id, result)
}

// --- Tool Handlers ---

func (s *Server) handleReadFile(args json.RawMessage) (any, error) {
	var params struct {
		Path   string `json:"path"`
		Offset int    `json:"offset,omitempty"`
		Limit  int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Resolve relative to working directory
	path := params.Path
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	text := string(data)

	if params.Offset > 0 || params.Limit > 0 {
		lines := strings.Split(text, "\n")
		offset := params.Offset
		if offset < 1 {
			offset = 1
		}
		start := offset - 1
		if start >= len(lines) {
			text = ""
		} else {
			end := len(lines)
			if params.Limit > 0 && start+params.Limit < end {
				end = start + params.Limit
			}
			text = strings.Join(lines[start:end], "\n")
		}
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: text}},
	}, nil
}

func (s *Server) handleWriteFile(args json.RawMessage) (any, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Resolve relative to working directory
	path := params.Path
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(params.Content), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("✅ Written %d bytes to %s", len(params.Content), params.Path)}},
	}, nil
}

func (s *Server) handleRunCommand(args json.RawMessage) (any, error) {
	var params struct {
		Command string  `json:"command"`
		Timeout float64 `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Command == "" {
		return nil, fmt.Errorf("command is required")
	}
	if params.Timeout <= 0 {
		params.Timeout = 60
	}

	timeout := time.Duration(params.Timeout * float64(time.Second))

	cmd := exec.Command("sh", "-c", params.Command)
	wd, _ := os.Getwd()
	cmd.Dir = wd

	done := make(chan error, 1)
	start := time.Now()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	go func() {
		done <- cmd.Run()
	}()

	var exitCode int
	var timedOut bool

	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		} else {
			exitCode = 0
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		exitCode = -1
		timedOut = true
	}

	duration := time.Since(start)

	return ToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("Exit code: %d\nDuration: %dms\nTimed out: %v\n\nSTDOUT:\n%s\n\nSTDERR:\n%s",
				exitCode, duration.Milliseconds(), timedOut, truncate(stdout.String(), 2000), truncate(stderr.String(), 1000)),
		}},
	}, nil
}

func (s *Server) handleListFiles(args json.RawMessage) (any, error) {
	var params struct {
		Dir     string `json:"dir"`
		Pattern string `json:"pattern,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Dir == "" {
		params.Dir = "."
	}

	dir := params.Dir
	if !filepath.IsAbs(dir) {
		wd, _ := os.Getwd()
		dir = filepath.Join(wd, dir)
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}
		if params.Pattern != "" {
			matched, _ := filepath.Match(params.Pattern, info.Name())
			if !matched {
				return nil
			}
		}
		// Make path relative to working directory
		rel, _ := filepath.Rel(dir, path)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: strings.Join(files, "\n")}},
	}, nil
}

func (s *Server) handleReadConfig(json.RawMessage) (any, error) {
	// Read autoresearch.config.json from working directory
	wd, _ := os.Getwd()
	data, err := os.ReadFile(filepath.Join(wd, "autoresearch.config.json"))
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("No config found: %v", err)}},
		}, nil
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *Server) handleGetMetrics(json.RawMessage) (any, error) {
	// Read metrics from autoresearch.jsonl
	wd, _ := os.Getwd()
	data, err := os.ReadFile(filepath.Join(wd, "autoresearch.jsonl"))
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: "No metrics logged yet"}},
		}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var lastMetrics string
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], `"metric"`) {
			lastMetrics = lines[i]
			break
		}
	}
	if lastMetrics == "" {
		lastMetrics = "No experiment entries found"
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: lastMetrics}},
	}, nil
}

func (s *Server) handleCheckTermination(json.RawMessage) (any, error) {
	// Parse most recent experiment from log and check against config conditions
	wd, _ := os.Getwd()
	logData, err := os.ReadFile(filepath.Join(wd, "autoresearch.jsonl"))
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: "No log file — termination not applicable"}},
		}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	var lastMetric float64
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], `"metric"`) {
			var entry struct {
				Metric float64 `json:"metric"`
			}
			if err := json.Unmarshal([]byte(lines[i]), &entry); err == nil {
				lastMetric = entry.Metric
			}
			break
		}
	}

	// Read config conditions
	configData, err := os.ReadFile(filepath.Join(wd, "autoresearch.config.json"))
	if err != nil {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Last metric: %.0f (no config found)", lastMetric)}},
		}, nil
	}

	var cfg struct {
		Termination struct {
			Conditions []struct {
				Metric   string  `json:"metric"`
				Operator string  `json:"operator"`
				Value    float64 `json:"value"`
			} `json:"conditions"`
		} `json:"termination"`
	}
	json.Unmarshal(configData, &cfg)

	if len(cfg.Termination.Conditions) == 0 {
		return ToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Last metric: %.0f (no termination conditions configured)", lastMetric)}},
		}, nil
	}

	var results []string
	for _, cond := range cfg.Termination.Conditions {
		met := false
		switch cond.Operator {
		case ">=":
			met = lastMetric >= cond.Value
		case "<=":
			met = lastMetric <= cond.Value
		case "==":
			met = lastMetric == cond.Value
		default:
			results = append(results, fmt.Sprintf("Unknown operator: %s", cond.Operator))
			continue
		}
		if met {
			results = append(results, fmt.Sprintf("✅ %s %s %.0f (got %.0f) — MET", cond.Metric, cond.Operator, cond.Value, lastMetric))
		} else {
			results = append(results, fmt.Sprintf("🔄 %s %s %.0f (got %.0f) — NOT MET", cond.Metric, cond.Operator, cond.Value, lastMetric))
		}
	}

	return ToolResult{
		Content: []ContentItem{{Type: "text", Text: strings.Join(results, "\n")}},
	}, nil
}

// --- Response helpers ---

func (s *Server) writeResult(id json.RawMessage, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func (s *Server) writeError(id json.RawMessage, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObj{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...\n[truncated]"
}
