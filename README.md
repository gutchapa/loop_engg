# Loop Engineering — Autonomous AI Coding Agent

## What is Loop Engineering?
A **Go CLI tool** that serves as both:
1. **MCP Server** 🧩 — exposes tools (`read_file`, `write_file`, `run_command`, etc.) for any MCP-compatible LLM client (Claude Code, Cursor, Cline)
2. **Self-contained AI Agent** 🤖 — built-in LLM client that reads your project, plans changes, writes code, runs tests, iterates autonomously

Zero external dependencies. Single binary. Works with Grok, DeepSeek, OpenAI, or local Ollama.

## The Full Self-Learning Stack (v1.4)

```
┌──────────────────────────────────────────────────────────────────────┐
│                       loop (single Go binary)                        │
│                                                                      │
│  ┌──────────┐   ┌───────────┐   ┌───────────┐   ┌───────────┐      │
│  │  learn/  │   │ diagnose/ │   │  evolve/  │   │  planner/ │      │
│  │ Memory   │◄──┤ Classify  │◄──┤ Self-tune │──►│ Prompt    │      │
│  │ persists │   │ failures  │   │ strategy  │   │ builder   │      │
│  └────┬─────┘   └─────┬─────┘   └─────┬─────┘   └─────┬─────┘      │
│       │               │               │               │             │
│       │  remembers fix if seen before │               │             │
│       │◄──────────────────────────────┘               │             │
│       │                                              │             │
│       │      best strategies + anti-patterns         │             │
│       │─────────────────────────────────────────────►│             │
│       │                                              │             │
│       │      prompt patches (reinforce/explore)      │             │
│       │◄─────────────────────────────────────────────┘             │
│                                                                      │
│  Packages: 12 internal (all pure Go, zero external deps)             │
│  Storage:  autoresearch.knowledge.json (persistent memory)           │
└──────────────────────────────────────────────────────────────────────┘
```

## Architecture

```mermaid
flowchart TB
    subgraph CLI["loop binary (single Go binary)"]
        direction LR
        MCP["🔌 MCP Server<br/>loop mcp"]
        AI["🤖 AI Agent<br/>loop ai"]
        AUTO["🔄 Orchestrator<br/>loop auto"]
    end

    subgraph Clients["External MCP Clients"]
        CC["Claude Code"]
        CUR["Cursor"]
        CLI2["Cline"]
    end

    subgraph LLMs["LLM Providers"]
        GROK["Grok (xAI)"]
        DS["DeepSeek"]
        OAI["OpenAI"]
        OLL["Ollama (local)"]
    end

    subgraph SelfLearn["Self-Learning Engine"]
        direction TB
        LEARN["🧠 learn<br/>persistent knowledge<br/>strategies • anti-patterns • fixes"]
        DIAG["🔍 diagnose<br/>failure classifier<br/>8 types • trigger matching"]
        EVO["🧬 evolve<br/>self-modification<br/>trend analysis • prompt tuning"]
        LEARN -->|"remembers fixes"| DIAG
        DIAG -->|"queried for known fix"| LEARN
        LEARN -->|"best strategies"| EVO
        EVO -->|"prompt patches"| LEARN
    end

    subgraph Packages["Internal Packages"]
        SCN["🔒 scanner<br/>secrets detection"]
        LLM["🌐 llm<br/>OpenAI-compat client"]
        MCP2["🔌 mcp<br/>JSON-RPC server"]
        PLAN["📋 planner<br/>context builder"]
        BRG["🔧 bridge<br/>tool executor"]
        CFG["⚙️ config<br/>schema + loader"]
        RUN["▶️ run<br/>shell exec"]
        MET["📊 metric<br/>parser"]
        LOG["📝 log<br/>JSONL writer"]
    end

    subgraph Project["Your Project"]
        AR["autoresearch.md<br/>rules + objective"]
        AC["autoresearch.config.json<br/>guardrails + targets"]
        AK["autoresearch.knowledge.json<br/>persistent memory"]
        AJ["autoresearch.jsonl<br/>experiment log"]
        SRC["source code"]
    end

    CC -->|"stdio | MCP"| MCP
    CUR --> MCP
    CLI2 --> MCP

    MCP --> MCP2
    MCP2 --> SRC
    MCP2 --> AR
    MCP2 --> AJ

    AI --> SCN --> LLM
    LLM -->|"chat completion"| GROK
    LLM --> DS
    LLM --> OAI
    LLM --> OLL
    AI --> PLAN
    PLAN --> AR
    PLAN --> AC
    PLAN --> SRC
    PLAN --> LEARN
    AI --> BRG
    BRG --> SRC
    BRG --> RUN
    AI --> RUN
    RUN -->|"METRIC lines"| MET
    MET --> LOG
    LOG --> AJ

    AI --> LEARN
    AI --> DIAG
    AI --> EVO
    LEARN --> AK

    AUTO --> CFG
    AUTO --> RUN
    AUTO --> MET
    AUTO --> LOG
    AUTO -->|"guardrails"| CFG

    style CLI fill:#2d3748,stroke:#63b3ed,color:#e2e8f0
    style SelfLearn fill:#1e293b,stroke:#c084fc,color:#e2e8f0
    style Packages fill:#1a365d,stroke:#90cdf4,color:#e2e8f0
    style Project fill:#22543d,stroke:#68d391,color:#e2e8f0
    style Clients fill:#44337a,stroke:#b794f4,color:#e2e8f0
    style LLMs fill:#742a2a,stroke:#fc8181,color:#e2e8f0
```

### How `loop ai` works with the self-learning stack

```mermaid
sequenceDiagram
    participant AI as 🤖 loop ai
    participant SC as 🔒 Scanner
    participant KS as 🧠 Knowledge Store
    participant PL as 📋 Planner
    participant LLM as 🧠 LLM API
    participant BR as 🔧 Bridge
    participant FS as 📁 Filesystem
    participant TEST as 🧪 Test Runner
    participant DIAG as 🔍 Diagnose
    participant EVO as 🧬 Evolve

    loop each iteration
        AI->>SC: scan project
        SC-->>AI: clean / warnings
        AI->>KS: load knowledge
        KS-->>AI: 11 entries (strategies + anti-patterns)
        AI->>PL: build context
        PL-->>AI: system prompt + knowledge summary

        loop tool turns (max 5)
            AI->>LLM: chat(messages)
            LLM-->>AI: {"tool": "read_file", ...}
            AI->>BR: execute(toolCall)
            BR->>FS: read/write/run
            FS-->>BR: output
            BR-->>AI: result
        end

        AI->>TEST: run command
        TEST-->>AI: METRIC test_count=74

        alt tests failed
            AI->>DIAG: analyze failure
            DIAG->>KS: find known fix?
            KS-->>DIAG: npm peer conflict → --legacy-peer-deps
            DIAG-->>AI: Diagnosis + fix suggestion
        end

        AI->>EVO: record snapshot
        EVO->>EVO: evaluate trend
        EVO-->>AI: Assessment: stagnating → rotate strategy
        AI->>PL: inject prompt patch
    end
```

### How `evolve` self-modifies

```mermaid
flowchart LR
    SNAP["📸 Record<br/>Snapshot"] --> TREND["📈 Compute<br/>Trend"]
    TREND --> EVAL["🩺 Evaluate<br/>Status"]
    EVAL -->|"improving"| REIN["✅ Reinforce<br/>strategy"]
    EVAL -->|"stagnating"| NUDGE["🔄 Nudge<br/>exploration"]
    EVAL -->|"regressing"| CAUT["⚠️ Caution<br/>prompt"]
    EVAL -->|"stuck"| ESCAPE["🚨 Escape<br/>stuck loop"]
    REIN --> PROMPT["💉 Inject into<br/>system prompt"]
    NUDGE --> PROMPT
    CAUT --> PROMPT
    ESCAPE --> PROMPT

    EVAL -->|"diminishing"| TUNE["⚙️ Relax<br/>target"]
    EVAL -->|"consistently hitting"| TIGHT["🎯 Tighten<br/>target"]
```

### Failure Diagnosis Engine

| Category | Example Trigger | Known Fix Source |
|----------|----------------|------------------|
| `dep_missing` | `Module not found`, `cannot find package` | Memory store — install command |
| `dep_conflict` | `peer dependency conflict`, `ERESOLVE` | Memory store — `--legacy-peer-deps` |
| `build_error` | `syntax error`, `undefined:`, `import cycle` | LLM — fix source code |
| `test_failure` | `FAIL`, `assertion failed` | LLM — fix test or code |
| `config_error` | `config:`, `invalid config` | Suggest `loop init` |
| `env_issue` | `permission denied`, `command not found` (system tool) | Check PATH, permissions |
| `network_error` | `connection refused`, `DNS`, `timeout` | Retry, check connectivity |
| `code_bug` | Panic, nil pointer, index out of range | LLM — fix the bug |
| `unknown` | Unrecognized pattern | Full output to LLM + store |

### Knowledge Store (`autoresearch.knowledge.json`)

| Entry Type | Purpose | Example |
|-----------|---------|---------|
| `strategy` | Proven techniques that worked | "Add test stubs for untested modules" |
| `anti_pattern` | Approaches that consistently fail | "Rewrite entire module at once" |
| `infra_fix` | Environment/dependency fixes | "npm install --legacy-peer-deps" |

Automatic distillation from JSONL logs: `loop learn --distill`

### Metric types

| Type | Who checks | Example |
|------|-----------|---------|
| **Guardrail** (soft) | `loop auto` every run | `exit_code == 0` |
| **Qualitative** (human) | LLM uses judgment | "UI is clean" |
| **Hard** (numeric) | `loop auto` termination | `test_count >= 74` |

### Package dependency graph

```mermaid
flowchart LR
    MAIN[cmd/loop/main.go] --> CFG2[config]
    MAIN --> RUN2[run]
    MAIN --> MET2[metric]
    MAIN --> LOG2[log]
    MAIN --> MCP3[mcp]
    MAIN --> LLM2[llm]
    MAIN --> PLN2[planner]
    MAIN --> BRG2[bridge]
    MAIN --> SCN2[scanner]
    MAIN --> LEARN2[learn]
    MAIN --> DIAG2[diagnose]
    MAIN --> EVO2[evolve]

    PLN2 --> CFG2
    PLN2 --> LEARN2
    MCP3 --> RUN2
    MCP3 --> LOG2
    BRG2 --> RUN2
    DIAG2 --> LEARN2
    EVO2 --> LEARN2

    style MAIN fill:#e53e3e,stroke:#fc8181,color:#fff
    style CFG2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style RUN2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style MET2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style LOG2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style MCP3 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style LLM2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style PLN2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style BRG2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style SCN2 fill:#2b6cb0,stroke:#63b3ed,color:#fff
    style LEARN2 fill:#7e22ce,stroke:#c084fc,color:#fff
    style DIAG2 fill:#7e22ce,stroke:#c084fc,color:#fff
    style EVO2 fill:#7e22ce,stroke:#c084fc,color:#fff
```

## Quick Start

### 1. Build
```bash
go build -o loop ./cmd/loop/
```

### 2. MCP Server Mode (works with any LLM client)
```bash
# Start MCP server via stdio
./loop mcp
```
Then connect your LLM client:
- **Claude Code**: `claude mcp add -- stdio -- /path/to/loop mcp`
- **Cursor**: Configure as custom MCP server
- **Cline**: Add as MCP tool server

Exposed tools:
| Tool | Description |
|------|-------------|
| `read_file` | Read files with offset/limit |
| `write_file` | Write/create files |
| `run_command` | Execute shell commands |
| `list_files` | List project files |
| `read_config` | Read experiment config |
| `get_metrics` | Get test/benchmark results |
| `check_termination` | Check if goals met |

### 3. Self-contained AI Agent Mode
```bash
# With Grok (default)
./loop ai --api-key xai-...

# With DeepSeek
./loop ai --provider deepseek --api-key sk-...

# With local Ollama
./loop ai --provider ollama

# Configure in autoresearch.config.json:
# "ai": { "provider": { "provider": "grok", "model": "grok-4-20-0309-reasoning" } }
# Or set LOOP_API_KEY env var
```

### 4. Experiment Loop Mode
```bash
# Manual experiment
./loop run "go test ./..."

# Autonomous iteration from config
./loop auto
```

## Commands

| Command | Purpose |
|---------|---------|
| `loop mcp` | MCP server — tools for any LLM client |
| `loop ai` | Self-contained AI agent with full self-learning stack |
| `loop run` | Execute command with timing |
| `loop auto` | Autonomous experiment iteration |
| `loop bench` | Run Go benchmarks as METRIC lines |
| `loop check` | Validate project state |
| `loop learn` | Knowledge store management (`--distill`, `--anti`, `--infra`) |
| `loop init` | Initialize experiment session |
| `loop version` | Print version |

## Self-Learning Capabilities

### 🧠 Persistent Memory (`learn/` — v1.3.0)
- Distills patterns from experiment logs (`loop learn --distill`)
- Tracks strategy confidence scores
- Maintains anti-pattern blacklist
- Stores infrastructure fixes with trigger matching

### 🔍 Failure Diagnosis (`diagnose/` — v1.3.1)
- Classifies 8 failure categories (dep_missing, dep_conflict, build_error, test_failure, config_error, env_issue, network_error, code_bug, unknown)
- Pattern matchers for npm, Go, Python, and shell errors
- Queries knowledge store for known fixes (e.g., `--legacy-peer-deps` for npm conflicts)
- Auto-feeds diagnosis back to LLM on test failures

### 🧬 Self-Evolution (`evolve/` — v1.4.0)
- Records performance snapshots every iteration
- Detects trends: improving / stagnating / regressing / stuck
- Generates prompt patches: reinforce, explore, caution, escape
- Auto-tunes config targets (tighten if hitting, relax if stuck)
- Detects diminishing returns and stuck loops
- Suggests strategy rotation from knowledge store

## Security
Before any file content is sent to a cloud LLM API, the **security scanner** runs:
- Detects API keys, passwords, tokens, private keys
- Flags `.env`, `*.pem`, `secrets.*` files
- Warns and asks confirmation on critical findings

## Configuration

### `autoresearch.config.json`
```json
{
  "metricName": "test_count",
  "direction": "higher",
  "command": "go test ./...",
  "maxIterations": 50,
  "termination": {
    "conditions": [
      { "metric": "test_count", "operator": ">=", "value": 50 }
    ]
  },
  "ai": {
    "maxIterations": 10,
    "filesInScope": ["*.go", "*.ts"],
    "provider": {
      "provider": "grok",
      "model": "grok-4-20-0309-reasoning",
      "endpoint": "https://api.x.ai/v1",
      "apiKey": ""
    }
  },
  "objectives": ["improve test coverage", "optimize performance"],
  "guardrails": [
    { "check": "exit_code == 0" }
  ]
}
```

### `autoresearch.md`
Defines the project objective, rules, and metrics for the AI agent.

### `autoresearch.knowledge.json` (auto-generated)
Persistent memory — stores strategies, anti-patterns, and infrastructure fixes across sessions.

## Supported LLM Providers

| Provider | Flag | Env Var | Default Model |
|----------|------|---------|---------------|
| Grok (xAI) | `--provider grok` | `LOOP_API_KEY` | `grok-4-20-0309-reasoning` |
| DeepSeek | `--provider deepseek` | `LOOP_API_KEY` | `deepseek-v4-flash` |
| OpenAI | `--provider openai` | `LOOP_API_KEY` | `gpt-4o` |
| Ollama (local) | `--provider ollama` | — | `gemma4-hermes` |

## Example Projects
See `examples/` for industry-specific Go demos:
- Fintech payment processing
- Healthcare search
- E-commerce catalog
- DevOps log parsing
- Media thumbnail
- Logistics route optimization

## Package Tree
```
cmd/loop/main.go          CLI dispatch + all commands
internal/bridge/bridge.go Tool execution (LLM → real actions)
internal/config/config.go Goal, Guardrail, HardMetric, AI schema
internal/diagnose/diagnose.go Failure classifier (8 types, 4 language matchers)
internal/evolve/evolve.go Self-modification engine (trends, prompt tuning, config tuning)
internal/learn/store.go   Persistent knowledge store (JSON)
internal/llm/llm.go       OpenAI-compatible client
internal/log/log.go       JSONL experiment logger
internal/mcp/mcp.go       MCP server (JSON-RPC 2.0)
internal/metric/metric.go METRIC parser
internal/patch/patch.go   Smart code patcher
internal/planner/planner.go System prompt + knowledge injection
internal/run/run.go       Shell command runner
internal/scanner/scanner.go Secrets detection
```

## License
MIT
