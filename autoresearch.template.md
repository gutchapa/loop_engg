# Autoresearch: [Your Project Name]

## Objective
[Describe what you're optimizing. Example: "Reduce P99 latency of the payment API to under 50ms."]

## Modes

### Mode 1: MCP Server (works with any LLM client)
```bash
./loop mcp
```
Exposes tools to any MCP-compatible client (Claude Code, Cursor, Cline, etc.):
- `read_file` — read project files
- `write_file` — write/modify code
- `run_command` — execute tests/builds
- `list_files` — explore project structure
- `read_config` — read experiment config

### Mode 2: Self-contained AI Agent
```bash
./loop ai --provider grok --api-key xai-...
```
The binary reads your project, calls an LLM, writes code, runs tests, and loops autonomously.

### Mode 3: Simple Experiment Loop
```bash
./loop auto
```
Runs your command, checks termination conditions, logs results.

## Setup

### 1. Configure `autoresearch.config.json`
```json
{
  "metricName": "[your_metric]",
  "command": "[your_benchmark_command]",
  "termination": {
    "maxIterations": 30,
    "conditions": [
      { "metric": "[your_metric]", "operator": "<=", "value": 50 }
    ]
  },
  "ai": {
    "maxIterations": 10,
    "filesInScope": ["*.go", "*.ts", "*.json"],
    "provider": {
      "provider": "grok",
      "model": "grok-4-20-0309-reasoning",
      "endpoint": "https://api.x.ai/v1",
      "apiKey": ""
    }
  }
}
```

### 2. Initialize
```bash
./loop init "[project_name]" "[metric_name]" --direction "lower"
```

### 3. Run
```bash
# MCP mode (any LLM client connects)
./loop mcp

# Or self-contained AI agent
./loop ai --provider grok

# Or simple experiment loop
./loop auto
```

## Metrics

### Primary (optimization target)
| Metric | Unit | Direction | Description |
|--------|------|-----------|-------------|
| [your_metric] | [ms/µs/tx/s/kb/—] | [lower/higher] | [what it measures] |

### Secondary
| Metric | Description |
|--------|-------------|
| `exit_code` | Must be 0 (build/test pass) |
| `duration_ms` | Benchmark execution time |

## Termination Conditions
Defined in `autoresearch.config.json` under `termination`:
- **Operators**: `>=`, `<=`, `==`, `>`, `<`
- Multiple conditions = AND (all must be met)
- `loop auto` checks these automatically

## Output Protocol
Your command must output `METRIC name=value` lines:
```
METRIC p99_latency_ms=42.5
```

## Files in Scope
[List files the agent is allowed to modify]
- `cmd/`
- `internal/`
- `pkg/`
- `go.mod`, `go.sum`

## Off Limits
[List files the agent must never touch]
- `vendor/`
- `.git/`
- `third_party/`

## Hard Stop
- If `loop auto` says `LOOP COMPLETE` — stop
- If user says to stop — stop immediately
- If no more meaningful optimizations — stop
