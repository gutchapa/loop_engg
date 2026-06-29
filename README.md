# Loop Engineering — Autonomous AI Coding Agent

## What is Loop Engineering?
A **Go CLI tool** that serves as both:
1. **MCP Server** 🧩 — exposes tools (`read_file`, `write_file`, `run_command`, etc.) for any MCP-compatible LLM client (Claude Code, Cursor, Cline)
2. **Self-contained AI Agent** 🤖 — built-in LLM client that reads your project, plans changes, writes code, runs tests, and iterates autonomously

Zero external dependencies. Single binary. Works with Grok, DeepSeek, OpenAI, or local Ollama.

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

    subgraph Packages["Internal Packages"]
        SCN["scanner<br/>secrets detection"]
        LLM["llm<br/>OpenAI-compat client"]
        MCP2["mcp<br/>JSON-RPC server"]
        PLAN["planner<br/>context builder"]
        BRG["bridge<br/>tool executor"]
        CFG["config<br/>schema + loader"]
        RUN["run<br/>shell exec"]
        MET["metric<br/>parser"]
        LOG["log<br/>JSONL writer"]
    end

    subgraph Project["Your Project"]
        AR["autoresearch.md<br/>rules + objective"]
        AC["autoresearch.config.json<br/>guardrails + targets"]
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
    AI --> BRG
    BRG --> SRC
    BRG --> RUN
    AI --> RUN
    RUN -->|"METRIC lines"| MET
    MET --> LOG
    LOG --> AJ

    AUTO --> CFG
    AUTO --> RUN
    AUTO --> MET
    AUTO --> LOG
    AUTO -->|"guardrails"| CFG

    style CLI fill:#2d3748,stroke:#63b3ed,color:#e2e8f0
    style Packages fill:#1a365d,stroke:#90cdf4,color:#e2e8f0
    style Project fill:#22543d,stroke:#68d391,color:#e2e8f0
    style Clients fill:#44337a,stroke:#b794f4,color:#e2e8f0
    style LLMs fill:#742a2a,stroke:#fc8181,color:#e2e8f0
```

### How `loop ai` works (inner loop)

```mermaid
sequenceDiagram
    participant AI as 🤖 loop ai
    participant SC as 🔒 Scanner
    participant PL as 📋 Planner
    participant LLM as 🧠 LLM API
    participant BR as 🔧 Bridge
    participant FS as 📁 Filesystem
    participant TEST as 🧪 Test Runner

    loop each iteration
        AI->>SC: scan project
        SC-->>AI: clean / warnings
        AI->>PL: build context
        PL-->>AI: system + user prompts

        loop tool turns (max 5)
            AI->>LLM: chat(messages)
            LLM-->>AI: {"tool": "read_file", ...}
            AI->>BR: execute(toolCall)
            BR->>FS: read/write/run
            FS-->>BR: output
            BR-->>AI: result
            AI->>AI: append to messages
        end

        AI->>TEST: run command
        TEST-->>AI: METRIC test_count=74
    end
```

### Metric types

| Type | Who checks | Example |
|------|-----------|---------|
| **Guardrail** (soft) | `loop auto` every run | `exit_code == 0` |
| **Qualitative** (human) | LLM uses judgment | “UI is clean” |
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

    PLN2 --> CFG2
    MCP3 --> RUN2
    MCP3 --> LOG2
    BRG2 --> RUN2

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
| `loop ai` | Self-contained AI agent (OODA loop) |
| `loop run` | Execute command with timing |
| `loop auto` | Autonomous experiment iteration |
| `loop bench` | Run Go benchmarks as METRIC lines |
| `loop check` | Validate project state |
| `loop init` | Initialize experiment session |
| `loop version` | Print version |

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
  }
}
```

### `autoresearch.md`
Defines the project objective, rules, and metrics for the AI agent.

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

## License
MIT
