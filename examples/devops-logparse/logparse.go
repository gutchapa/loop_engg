// Package logparse parses unstructured log lines into structured entries.
// Loop Engineering target: maximize lines/sec parsed.
package logparse

import (
	"regexp"
	"strings"
	"time"
)

type Level int

const (
	LevelUnknown Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Entry struct {
	Timestamp time.Time
	Level     Level
	Service   string
	Message   string
	TraceID   string
}

// Parser extracts structured entries from raw log text.
type Parser struct {
	// Compiled regex patterns for different log formats
	formats []*regexp.Regexp
}

// NewParser creates a parser that recognizes common log formats.
func NewParser() *Parser {
	return &Parser{
		formats: []*regexp.Regexp{
			// 2024-01-15T10:30:00Z [INFO] service: message
			regexp.MustCompile(`^(\S+)\s+\[(\w+)\]\s+(\S+):\s+(.+)$`),
			// 2024-01-15 10:30:00,123 ERROR service - message
			regexp.MustCompile(`^(\S+\s+\S+)\s+(\w+)\s+(\S+)\s*-\s*(.+)$`),
			// [2024-01-15 10:30:00] [INFO] (service) message
			regexp.MustCompile(`^\[(\S+\s+\S+)\]\s+\[(\w+)\]\s+\((\S+)\)\s+(.+)$`),
		},
	}
}

// ParseLine parses a single log line into an Entry.
// Returns nil if the line doesn't match any known format.
func (p *Parser) ParseLine(line string) *Entry {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	for _, re := range p.formats {
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339, m[1])
		if ts.IsZero() {
			ts, _ = time.Parse("2006-01-02 15:04:05", m[1])
		}
		if ts.IsZero() {
			ts, _ = time.Parse("2006-01-02 15:04:05,000", m[1])
		}

		level := parseLevel(m[2])

		return &Entry{
			Timestamp: ts,
			Level:     level,
			Service:   m[3],
			Message:   m[4],
		}
	}

	// Fallback: extract level and service via simple contains
	return parseFallback(line)
}

func parseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelUnknown
	}
}

// parseFallback uses simpler heuristics when regex fails.
func parseFallback(line string) *Entry {
	entry := &Entry{}

	// Try to find a level keyword
	upper := strings.ToUpper(line)
	for _, l := range []struct {
		kw    string
		level Level
	}{
		{"FATAL", LevelFatal},
		{"ERROR", LevelError},
		{"WARN", LevelWarn},
		{"INFO", LevelInfo},
		{"DEBUG", LevelDebug},
	} {
		if strings.Contains(upper, l.kw) {
			entry.Level = l.level
			break
		}
	}

	entry.Message = line
	return entry
}

// ParseAll parses all lines of a log text.
func (p *Parser) ParseAll(text string) []Entry {
	lines := strings.Split(text, "\n")
	entries := make([]Entry, 0, len(lines))
	for _, line := range lines {
		if e := p.ParseLine(line); e != nil {
			entries = append(entries, *e)
		}
	}
	return entries
}

// CountByLevel returns a count of entries per log level.
func CountByLevel(entries []Entry) map[Level]int {
	counts := make(map[Level]int)
	for _, e := range entries {
		counts[e.Level]++
	}
	return counts
}

// GenerateLogs creates n sample log lines for benchmarking.
func GenerateLogs(n int) string {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	services := []string{"api", "db", "cache", "auth", "worker"}
	messages := []string{
		"request completed in 42ms",
		"connection pool depleted",
		"user authentication failed",
		"cache miss for key",
		"background job finished",
		"rate limit exceeded",
		"database query timeout",
		"health check passed",
	}

	var b strings.Builder
	now := time.Now()
	for i := 0; i < n; i++ {
		ts := now.Add(-time.Duration(n-i) * time.Second).Format(time.RFC3339)
		level := levels[i%len(levels)]
		svc := services[(i/len(levels))%len(services)]
		msg := messages[i%len(messages)]
		b.WriteString(ts + " [" + level + "] " + svc + ": " + msg + "\n")
	}
	return b.String()
}
