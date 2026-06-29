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
| [`autoresearch.template.md`](./autoresearch.template.md) | **Bring your own rules** — fill in your metrics, termination conditions, scope |

## The `loop` CLI (Updated 2026-06-29)

```bash
# Build the CLI (now with security hardening + clean build)
go build -o loop ./cmd/loop/

# Initialize a new experiment session
./loop init "Optimize validation" tx_per_sec --unit "tx/s" --direction higher

# Run a benchmark (timed + METRIC parsing)
./loop run "go test -bench=BenchmarkBatchProcess -benchmem -count=3" --timeout 30

# Project health check
./loop check

# Version
./loop version
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

## Use It on Your Own Project (Updated)

```bash
# 1. Build (now security-hardened + warning on shell exec)
go build -o loop ./cmd/loop/

# 2. Copy core files to your project
cp loop autoresearch.config.json autoresearch.template.md ../your-new-project/
cd ../your-new-project
mv autoresearch.template.md autoresearch.md

# 3. Edit autoresearch.md with your rules, metric, and stop signals
# 4. Start the loop
./loop init "Baseline" test_count --direction higher
./loop run "go test ./... -count=1"
```

**Security Note**: The `run` command uses `sh -c`. Only use it with trusted benchmark commands. Never pass raw user input.

## License

MIT

## License

MIT
