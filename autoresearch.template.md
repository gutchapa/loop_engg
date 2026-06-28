# Autoresearch: [Your Project Name]

## Objective
[Describe what you're building. Example: "Build a high-throughput Go HTTP proxy with sub-millisecond P99 latency."]

## Tools
| Tool | Purpose |
|------|---------|
| `read` | Examine source files |
| `bash` | Run shell commands |
| `edit` | Make precise code changes |
| `write` | Create new files |
| `init_experiment` | Initialize session |
| `run_experiment` | Run the benchmark |
| `log_experiment` | Record result |

## Metrics

### Primary (optimization target)
| Metric | Unit | Direction | Description |
|--------|------|-----------|-------------|
| [your_metric] | [ms / µs / tx/s / kb / —] | [lower / higher] | [what it measures] |

> ⚠️ Replace `[your_metric]` above with the actual metric name. Delete this row.

### Secondary (monitoring, not optimization targets)
| Metric | Unit | Target | Description |
|--------|------|--------|-------------|
| build_ok | 0/1 | must stay 1 | Build passes without errors |
| [other_metric] | [unit] | [target] | [description] |

### Example metrics for common domains
| Domain | Primary Metric | Unit | Direction |
|--------|---------------|------|-----------|
| API server | P99 latency | ms | lower |
| Payment processing | tx_per_sec | tx/s | higher |
| Log parser | lines_per_sec | lines/s | higher |
| Image pipeline | process_time | µs | lower |
| Route optimizer | compute_time | ms | lower |
| Compiler | build_time | s | lower |
| ML training | val_accuracy | % | higher |
| Binary size | binary_size | kb | lower |

## Termination Conditions

> ⚠️ DELETE THE OPTIONS THAT DON'T APPLY. Keep the ones relevant to your project.

### Hard stops — loop ends immediately when any is true:
1. **User says to stop** — any variant of "stop", "done", "enough"
2. **Project delivered** — the feature/artifact is in the user's hands
3. **No more requests** — all asked-for features are complete
4. **Ideas backlog empty** — nothing left worth exploring
5. **Hit max iterations** — configured in autoresearch.config.json

### Soft guard — stop unless there's a real need:
- If tests pass, build passes, no bugs, no user requests, no backlog ideas → **STOP**. Do not invent busywork.
- The metric is a quality proxy, not a score to grind.

### Starting the experiment loop

```bash
# Initialize
./loop init "[Experiment Name]" [primary_metric] --unit [unit] --direction [lower|higher]

# Baseline
./loop run "go build ./... && go test ./... -count=1"

# Each iteration: implement change, then:
./loop run "go build ./... && go test ./... -count=1"
```

## Files in Scope
[List files and directories the agent is allowed to modify]
- `cmd/` — CLI entry point
- `internal/` — core packages
- `pkg/` — public API
- `go.mod`, `go.sum`
- etc.

## Off Limits
[List files the agent must never touch]
- `vendor/`
- `node_modules/` (if applicable)
- `.git/`
- `third_party/`

## How to Run
```bash
./loop run "your benchmark command"
```
Outputs `METRIC name=value` lines. The agent parses these automatically.
