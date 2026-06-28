# 🧪 Loop Engineering

> *Build software through autonomous, self-correcting experiment loops.*
> Now rewritten in **Go** — single binary, zero dependencies, cross-platform.

## What's Inside

| File | Purpose |
|------|---------|
| [`LOOP_ENGINEERING.md`](./LOOP_ENGINEERING.md) | Full methodology — principles, loop diagram, tooling, ASI, real-world results |
| [`cmd/loop/main.go`](./cmd/loop/main.go) | **`loop` CLI** — the experiment engine (Go, single binary) |
| [`internal/`](./internal/) | Go packages: config, run, metric, log |
| [`autoresearch.config.json`](./autoresearch.config.json) | Loop configuration (working dir, max iterations) |
| [`autoresearch.md`](./autoresearch.md) | Loop rules — objective, metrics, files in scope, termination conditions |
| [`autoresearch.ideas.md`](./autoresearch.ideas.md) | Ideas backlog with stop-signal guards |

## The `loop` CLI

A single Go binary replaces the old bash-based autoresearch.sh:

```bash
# Initialize a new experiment session
./loop init "Optimize build" build_time_s --unit s --direction lower

# Run a benchmark command (timed, captures METRIC lines)
./loop run "npm run build && npx vitest run --reporter=json"

# Run with timeout
./loop run "python train.py" --timeout 300

# Validate project state
./loop check

# Build from source
go build -o loop ./cmd/loop/
```

### METRIC Protocol

The CLI outputs structured `METRIC name=value` lines that the agent parses:

```
METRIC exit_code=0
METRIC duration_ms=1240
METRIC duration_s=1.240
METRIC timed_out=0
METRIC build_ok=1
METRIC test_count=244
```

Any `METRIC` lines from the child command's output are forwarded automatically.

## The Repo That Proves It

The **dues-dashboard** — a full-featured payment tracking app — was built entirely through this methodology. [Check it out →](https://github.com/gutchapa/dues-dashboard)

58 sequential experiments. 244 passing tests. Zero regressions. One AI agent.

## Quick Start

```bash
# Build the CLI
go build -o loop ./cmd/loop/

# Apply to your own project
cp loop autoresearch.config.json your-project/
# Configure: set workingDir, maxIterations in autoresearch.config.json
# Run:
./loop run "npm run build && npm test"
```

## License

MIT
