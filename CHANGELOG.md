# Changelog

## v1.4.0 — Self-Evolution Engine (2026-06-29)
- **Added**: `evolve/` package — self-modification engine
  - Records performance `Snapshot`s every AI iteration (metric, tools, exit code, output)
  - `Evaluate()` detects 5 states: improving, stagnating, regressing, stuck, done
  - Generates prompt patches: reinforce strategy, nudge exploration, caution, escape stuck loop
  - `TuneConfig()` auto-adjusts targets — tightens when consistently hitting, relaxes when stuck
  - `GeneratePrompt()` creates evolved system prompt variants from knowledge
  - Detects diminishing returns and stuck loops (same metric + exit code for 4+ iterations)
  - Suggests strategy rotation from knowledge store
- **Integrated** into `loop ai` command: snapshots recorded every iteration, evaluation at intervals, prompt patches injected into context
- **Full stack tested**: learn → diagnose → evolve, end-to-end with DeepSeek

## v1.3.1 — Failure Diagnosis Engine (2026-06-29)
- **Added**: `diagnose/` package — classifies command failures into 8 categories
  - `dep_missing`, `dep_conflict`, `build_error`, `test_failure`, `config_error`, `env_issue`, `network_error`, `code_bug`, `unknown`
  - Pattern matchers for npm, Go, Python, shell errors
  - Queries knowledge store for known fixes (trigger-based matching)
  - Auto-feeds diagnosis + memory suggestions back to LLM on test failures
- Tested on 7 real failure types, all classified correctly, npm peer conflict matches known fix

## v1.3.0 — Persistent Knowledge Store (2026-06-28)
- **Added**: `learn/` package — persistent knowledge store (`autoresearch.knowledge.json`)
  - Strategy entries with confidence scoring (success rate × hit count)
  - Anti-pattern blacklist (approaches to avoid)
  - Infrastructure fix entries (error → solution with trigger matching)
  - `DistillFromLog()` auto-extracts patterns from JSONL experiment history
  - `Query()` with type + triggers + confidence filters
  - `BestStrategies()`, `AntiPatterns()`, `FindInfraFix()` query helpers
- **Command**: `loop learn --distill --anti --infra`
- **Integrated** into `loop ai` — knowledge summary injected into system prompt each iteration
- Tested with 11 distilled entries (9 strategies, 1 anti-pattern, 2 infra fixes)

## v1.2.2 — Tool Execution Bridge (2026-06-28)
- **Added**: `bridge/` package — executes JSON tool calls from LLM responses
  - Parses `{"tool":"read_file","path":"..."}` from LLM messages
  - 7 tools: read_file, write_file, run_command, list_files, read_config, get_metrics, check_termination
  - Multi-turn conversation: up to 5 tool turns per AI iteration
  - Results fed back to LLM as user messages
- Tested with DeepSeek: 25+ tool calls across 5 iterations, zero crashes

## v1.2.1 — Soft Metrics + Qualitative Goals (2026-06-28)
- **Added**: Goal, Guardrail, HardMetric types in config
  - Guardrails: checked silently every iteration (pass/fail only)
  - Qualitative goals: LLM uses judgment to assess
  - Hard metrics: numeric targets for termination
- `loop auto` checks all three types

## v1.2.0 — MCP Server + Self-contained AI (2026-06-28)
- **Added**: `mcp/` package — MCP server (JSON-RPC 2.0 over stdio)
  - 7 tools: read_file, write_file, run_command, list_files, read_config, get_metrics, check_termination
- **Added**: `llm/` package — OpenAI-compatible client (Grok, DeepSeek, OpenAI, Ollama)
- **Added**: `planner/` package — context builder from autoresearch.md + config + source
- **Added**: `scanner/` package — secrets detection before cloud API calls
- **Commands**: `loop mcp`, `loop ai`, `loop auto`, `loop bench`

## v1.0.0 — Initial Release (2026-06-26)
- **Commands**: `loop init`, `loop run`, `loop check`, `loop version`
- **Packages**: config, run, metric, log
- Multi-metric parsing from command output
- JSONL experiment logging
- Autonomous experiment orchestration (`loop auto`)
- Go benchmark integration (`loop bench`)
- 6 industry-specific Go demo projects
