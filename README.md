# 🧪 Loop Engineering

> *Build software through autonomous, self-correcting experiment loops.*

58 sequential experiments. 244 passing tests. Zero regressions. One AI agent.

This repo contains the **methodology, tooling, and documentation** for Loop Engineering — a disciplined approach to building software where every change is a measured experiment.

## What's Inside

| File | Purpose |
|------|---------|
| [`LOOP_ENGINEERING.md`](./LOOP_ENGINEERING.md) | Full methodology — principles, loop diagram, tooling, ASI, real-world results |
| [`autoresearch.sh`](./autoresearch.sh) | Benchmark script — runs build + tests, outputs structured metrics |
| [`autoresearch.config.json`](./autoresearch.config.json) | Loop configuration (working dir, max iterations) |
| [`autoresearch.md`](./autoresearch.md) | Complete experiment log — 58 iterations, what was tried, results |
| [`autoresearch.ideas.md`](./autoresearch.ideas.md) | Feature inventory — implemented vs. future ideas |

## The Repo That Proves It

The **dues-dashboard** — a full-featured payment tracking app — was built entirely through this methodology. [Check it out →](https://github.com/gutchapa/dues-dashboard)

## Quick Start

```bash
# Apply to your own project
cp autoresearch.sh autoresearch.config.json your-project/
# Configure: set workingDir, maxIterations, benchmark command
# Run:
./autoresearch.sh
```

## License

MIT
