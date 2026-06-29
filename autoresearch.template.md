# Autoresearch: [Your Project Name]

## Objective
[Describe what you're optimizing. Example: "Reduce P99 latency of the payment API to under 50ms."]

## How It Works
The `loop` binary (Go CLI) is the orchestrator:

1. **Configure** — set `command`, `metricName`, and `termination` in `autoresearch.config.json`
2. **Run** — `./loop auto` executes one iteration:
   - Runs your benchmark command
   - Parses `METRIC name=value` lines from output
   - Checks termination conditions
   - Logs result to `autoresearch.jsonl`
   - Outputs `✅ LOOP COMPLETE` (target met) or `🔄 LOOP CONTINUE` (keep iterating)

## Setup

```bash
# 1. Initialize experiment session (writes config header to autoresearch.jsonl)
./loop init "Reduce API latency" "p99_latency_ms" --unit "ms" --direction "lower"

# 2. Edit autoresearch.config.json with your command and termination conditions:
#    {
#      "metricName": "p99_latency_ms",
#      "command": "go test -bench=. -benchtime=1x ./internal/api",
#      "termination": {
#        "maxIterations": 30,
#        "conditions": [
#          { "metric": "p99_latency_ms", "operator": "<=", "value": 50 }
#        ]
#      }
#    }

# 3. The AI writes/improves code, then:
./loop auto

# 4. Read the verdict:
#    ✅ LOOP COMPLETE: p99_latency_ms <= 50 (got 42)   → DONE
#    🔄 LOOP CONTINUE: conditions not yet met            → iterate again
```

## Metrics

### Primary (optimization target)
| Metric | Unit | Direction | Description |
|--------|------|-----------|-------------|
| [your_metric] | [ms/µs/tx/s/kb/—] | [lower/higher] | [what it measures] |

> ⚠️ Replace `[your_metric]` above. Must match `metricName` in `autoresearch.config.json`.

### Secondary (monitoring only)
| Metric | Description |
|--------|-------------|
| `exit_code` | Must be 0 (build/test pass) |
| `duration_ms` | Benchmark execution time |

## Termination Conditions
Defined in `autoresearch.config.json` under `termination`:

```json
"termination": {
  "maxIterations": 30,
  "conditions": [
    { "metric": "p99_latency_ms", "operator": "<=", "value": 50 }
  ]
}
```

**Operators**: `>=`, `<=`, `==`, `>`, `<`

Multiple conditions = AND (all must be met).

The `loop auto` command automatically checks these and outputs the verdict.

## Output Protocol
Your benchmark command must output `METRIC name=value` lines:
```
METRIC p99_latency_ms=42.5
METRIC p50_latency_ms=12.3
```

Built-in METRIC lines from `loop auto`:
```
METRIC exit_code=0
METRIC duration_ms=3211
```

## Files in Scope
[List files the agent is allowed to modify]
- `cmd/` — CLI entry point
- `internal/` — core packages
- `pkg/` — public API
- `go.mod`, `go.sum`
- etc.

## Off Limits
[List files the agent must never touch]
- `vendor/`
- `.git/`
- `third_party/`

## Hard Stop
- If `loop auto` says `LOOP COMPLETE` — the goal is met, stop
- If user says to stop — stop immediately
- If no more meaningful optimizations to try — stop
