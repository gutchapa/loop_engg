# 🧪 Loop Engineering

> *Build software through autonomous, self-correcting experiment loops.*
> **Pure Go** — single binary, zero dependencies, cross-platform. No bash/Python/Node.

## Philosophy

**The `loop` binary is the orchestrator.** It reads config, runs benchmarks, checks termination conditions, and decides when to stop. The AI agent writes code; `loop` evaluates it.

## Quick Start

```bash
go build -o loop ./cmd/loop/

# Initialize a session
./loop init "Optimize latency" p99_latency_ms --unit "ms" --direction lower

# Edit autoresearch.config.json:
#   {
#     "metricName": "p99_latency_ms",
#     "command": "go test -bench=. -benchtime=10x ./api/",
#     "termination": {
#       "conditions": [{"metric": "p99_latency_ms", "operator": "<=", "value": 50}]
#     }
#   }

# Run one iteration — loop auto handles everything:
./loop auto
# → ✅ LOOP COMPLETE: target reached
# → 🔄 LOOP CONTINUE: keep iterating
```

## Commands

| Command | What it does |
|---------|-------------|
| `loop init <name> <metric> [--unit <u>] [--direction <d>]` | Initialize experiment session |
| `loop run <command> [--timeout <s>]` | Run a command, time it, capture METRIC lines |
| **`loop auto`** | **Run one iteration: execute command → parse metrics → check termination → log → verdict** |
| `loop bench <pkg> [--benchtime <d>] [--count <n>]` | Run Go benchmarks, output as METRIC lines |
| `loop check` | Validate project health |
| `loop version` | Print version |

## METRIC Protocol

Your benchmark must output `METRIC name=value` lines:

```
METRIC p99_latency_ms=42.5
METRIC throughput_tps=18500
```

Built-in METRIC lines from `loop`:
```
METRIC exit_code=0
METRIC duration_ms=3211
METRIC timed_out=0
```

## `loop auto` — The Orchestrator

Reads `autoresearch.config.json`:

```json
{
  "metricName": "test_count",
  "command": "npx vitest run --reporter=json | python3 -c \"...\"",
  "termination": {
    "maxIterations": 50,
    "conditions": [
      { "metric": "test_count", "operator": ">=", "value": 74 }
    ]
  }
}
```

On each run it:
1. Executes the command
2. Parses all `METRIC` lines
3. Checks every termination condition
4. Logs to `autoresearch.jsonl`
5. Outputs verdict: **`✅ LOOP COMPLETE`** or **`🔄 LOOP CONTINUE`**

## `loop bench` — Go Benchmark Runner

Runs `go test -bench` and converts results to METRIC lines:

```bash
./loop bench ./examples/fintech-pay/ --benchtime 100x
```

Output:
```
METRIC exit_code=0
METRIC duration_ms=986
METRIC BenchmarkLuhnCheck_ns_per_op=35.84
METRIC BenchmarkProcess_ns_per_op=123.3
```

## Industry Examples

Each example has `autoresearch.config.json` ready — just `cd` and run:

```bash
cd /Users/gutchapa/loop_engg

# FinTech — Luhn validation throughput
./loop bench ./examples/fintech-pay/ --benchtime 100x

# Healthcare — patient record search
./loop bench ./examples/healthcare-search/ --benchtime 100x

# E-Commerce — product catalog filter+sort
./loop bench ./examples/ecommerce-catalog/ --benchtime 100x

# DevOps — log parsing throughput
./loop bench ./examples/devops-logparse/ --benchtime 100x

# Media — thumbnail generation
./loop bench ./examples/media-thumb/ --benchtime 100x

# Logistics — route optimization
./loop bench ./examples/logistics-route/ --benchtime 100x
```

Or use `loop auto` with an example's config:
```bash
cp examples/fintech-pay/autoresearch.config.json .
./loop auto
```

## Use on Your Own Project

```bash
# 1. Build
go build -o loop ./cmd/loop/

# 2. Copy to your project
cp loop autoresearch.config.json autoresearch.template.md ../your-project/
cd ../your-project

# 3. Edit autoresearch.config.json with your command + termination
# 4. Initialize
./loop init "My Opt" my_metric --direction lower

# 5. Code → auto → code → auto → done
./loop auto
```

## Contents

| File | Purpose |
|------|---------|
| [`LOOP_ENGINEERING.md`](./LOOP_ENGINEERING.md) | Full methodology — principles, loop diagram, ASI |
| [`cmd/loop/main.go`](./cmd/loop/main.go) | `loop` CLI (Go, single binary) |
| [`internal/`](./internal/) | Go packages: config, run, metric, log |
| [`examples/`](./examples/) | 6 industry demo projects with configs |
| [`autoresearch.config.json`](./autoresearch.config.json) | Config template |
| [`autoresearch.template.md`](./autoresearch.template.md) | Bring your own rules template |
| [`autoresearch.ideas.md`](./autoresearch.ideas.md) | Ideas backlog |

## License

MIT
