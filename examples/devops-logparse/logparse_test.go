package logparse

import (
	"testing"
)

func TestParseInfoLine(t *testing.T) {
	p := NewParser()
	e := p.ParseLine(`2024-01-15T10:30:00Z [INFO] api: request completed`)
	if e == nil {
		t.Fatal("expected parsed entry")
	}
	if e.Level != LevelInfo {
		t.Errorf("expected INFO, got %s", e.Level)
	}
	if e.Service != "api" {
		t.Errorf("expected api, got %s", e.Service)
	}
	if e.Message != "request completed" {
		t.Errorf("unexpected message: %s", e.Message)
	}
}

func TestParseErrorLine(t *testing.T) {
	p := NewParser()
	e := p.ParseLine(`2024-01-15T10:30:00Z [ERROR] db: connection timeout`)
	if e == nil || e.Level != LevelError {
		t.Error("expected ERROR entry")
	}
}

func TestParseAltFormat(t *testing.T) {
	p := NewParser()
	e := p.ParseLine(`2024-01-15 10:30:00,123 ERROR auth - login failed`)
	if e == nil || e.Level != LevelError {
		t.Error("expected ERROR entry from alt format")
	}
	if e.Service != "auth" {
		t.Errorf("expected auth, got %s", e.Service)
	}
}

func TestParseBracketFormat(t *testing.T) {
	p := NewParser()
	e := p.ParseLine(`[2024-01-15 10:30:00] [WARN] (cache) eviction threshold reached`)
	if e == nil || e.Level != LevelWarn {
		t.Error("expected WARN entry")
	}
	if e.Service != "cache" {
		t.Errorf("expected cache, got %s", e.Service)
	}
}

func TestParseEmptyLine(t *testing.T) {
	p := NewParser()
	if e := p.ParseLine(""); e != nil {
		t.Error("empty line should return nil")
	}
	if e := p.ParseLine("   "); e != nil {
		t.Error("whitespace line should return nil")
	}
}

func TestParseFallback(t *testing.T) {
	p := NewParser()
	e := p.ParseLine(`something ERROR happened somewhere`)
	if e == nil || e.Level != LevelError {
		t.Error("expected fallback to extract ERROR")
	}
}

func TestParseAll(t *testing.T) {
	p := NewParser()
	logs := `2024-01-15T10:30:00Z [INFO] api: ok
2024-01-15T10:31:00Z [ERROR] db: fail
[2024-01-15 10:32:00] [WARN] (cache) full
`
	entries := p.ParseAll(logs)
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestCountByLevel(t *testing.T) {
	p := NewParser()
	logs := `[INFO] svc: a
[INFO] svc: b
[ERROR] svc: c
[WARN] svc: d
`
	entries := p.ParseAll(logs)
	counts := CountByLevel(entries)
	if counts[LevelInfo] != 2 || counts[LevelError] != 1 || counts[LevelWarn] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}

func TestGenerateLogs(t *testing.T) {
	logs := GenerateLogs(100)
	if len(logs) == 0 {
		t.Error("expected generated logs")
	}
	lines := 0
	for _, c := range logs {
		if c == '\n' {
			lines++
		}
	}
	if lines != 100 {
		t.Errorf("expected 100 lines, got %d", lines)
	}
}

func BenchmarkParseLine(b *testing.B) {
	p := NewParser()
	line := `2024-01-15T10:30:00Z [INFO] api: request completed in 42ms`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.ParseLine(line)
	}
}

func BenchmarkParseAll(b *testing.B) {
	p := NewParser()
	logs := GenerateLogs(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.ParseAll(logs)
	}
}

func BenchmarkCountByLevel(b *testing.B) {
	p := NewParser()
	logs := GenerateLogs(1000)
	entries := p.ParseAll(logs)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CountByLevel(entries)
	}
}
