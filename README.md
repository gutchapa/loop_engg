# 🧪 Loop Engineering

> *Build software through autonomous, self-correcting experiment loops.*
> Written in **Go** — single binary, zero dependencies, cross-platform.

## What's Inside

| File | Purpose |
|------|---------|
| [`LOOP_ENGINEERING.md`](./LOOP_ENGINEERING.md) | Full methodology — principles, loop diagram, tooling, ASI, real-world results |
| [`cmd/loop/main.go`](./cmd/loop/main.go) | **`loop` CLI** — the experiment engine (Go, single binary) |
| [`internal/`](./internal/) | Go packages: config, run, metric, log |
| [`examples/`](./examples/) | **6 industry demo projects** — ready-to-run benchmarks |
| [`autoresearch.config.json`](./autoresearch.config.json) | Loop configuration (working dir, max iterations) |
| [`autoresearch.ideas.md`](./autoresearch.ideas.md) | Ideas backlog with stop-signal guards |

## The `loop` CLI

```bash
# Initialize a new experiment session
./loop init "Optimize search" search_latency_us --unit µs --direction lower

# Run a benchmark command (timed, captures METRIC lines)
./loop run "go test -bench=. ./..."

# Run with timeout
./loop run "go test -bench=BenchmarkBatch -count=5 ./..." --timeout 120

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
METRIC bench_items_per_sec=85000
```

Any `METRIC` lines from the child command's output are forwarded automatically.

## Industry Examples

Pick an example that matches your domain and run it:

| Industry | Example | Optimization Target |
|----------|---------|-------------------|
| 💳 **FinTech** | `examples/fintech-pay/` | Max transactions/sec through Luhn validation pipeline |
| 🏥 **Healthcare** | `examples/healthcare-search/` | Min search latency (ms) for patient records |
| 🛒 **E-Commerce** | `examples/ecommerce-catalog/` | Min filter+sort latency (µs) for product catalog |
| 🔧 **DevOps** | `examples/devops-logparse/` | Max lines/sec parsed from unstructured logs |
| 🎬 **Media** | `examples/media-thumb/` | Min processing time per thumbnail (µs) |
| 🚚 **Logistics** | `examples/logistics-route/` | Min route computation time with quality constraints |

```bash
# Try one out
cd examples/fintech-pay
go test -bench=. -benchmem -count=3
go test -v -count=1 ./...
```

## Quick Start

```bash
# Build the CLI
go build -o loop ./cmd/loop/

# Point it at any project
cd examples/logistics-route
../../loop init "Optimize route" compute_us --unit µs --direction lower
../../loop run "go test -bench=BenchmarkNearestNeighbor -benchmem -count=3"
```

## License

MIT
