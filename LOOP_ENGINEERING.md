# 🧪 Loop Engineering

> *An autonomous, self-correcting methodology for building software one verified experiment at a time.*

---

## What Is Loop Engineering?

**Loop Engineering** (or *Loop Engg*) is a disciplined approach to building software where every change is treated as an **experiment**. Each iteration follows a strict cycle:

```
1. OBSERVE   → Read the codebase, identify gaps, review past experiments
2. HYPOTHESIZE → Form a testable hypothesis ("Adding X will increase test_count by Y")
3. IMPLEMENT → Write code, add tests, update existing tests if needed
4. VERIFY    → Build + run all tests (zero regression allowed)
5. MEASURE   → Record metrics (test count, build time, pass/fail)
6. COMMIT or REVERT → Keep winners, discard losers, always learn
```

This repo is the **living proof** of this methodology — built entirely by an AI agent running autonomous loop engineering over 58 sequential experiments.

---

## Core Principles

### 1. Atomic Experiments
Every change is a single, focused experiment. No sprawling PRs. No "while I'm here" fixes. Each experiment does one thing and measures the impact.

### 2. Zero Regression
All existing tests must still pass after every change. If a change breaks anything, it's discarded. This creates a **strict monotonic improvement curve** — coverage and quality only go up.

### 3. Measurable Primary Metric
Every experiment optimizes toward a single primary metric:
- **Phase 1**: `build_ok` (binary — does it compile?)
- **Phase 2+**: `test_count` (higher is better — proxy for feature completeness + quality)

Secondary metrics (`build_time_s`, `tests_ok`) are monitored for regressions.

### 4. Every Experiment Logged
Each experiment is recorded with:
- **Hypothesis** — what we expected to happen
- **Metric delta** — did it improve or regress?
- **ASI** (Actionable Side Information) — context, learnings, dead ends
- **Decision** — keep or discard

This creates a **searchable knowledge base** of what works and what doesn't.

### 5. Confidence Scoring
After 3+ runs, the system calculates a confidence score:
```
confidence = best_improvement / noise_floor
```
- **≥2.0×** — likely real improvement
- **1.0–2.0×** — possible, needs confirmation
- **<1.0×** — within noise, re-run to confirm

---

## The Experiment Loop (Detailed)

```
┌─────────────────────────────────────────────────────────────┐
│                      START                                  │
│  "We have a codebase. Let's improve it."                    │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  1. OBSERVE                                                 │
│  ─────────                                                 │
│  • Read current codebase state                              │
│  • Read past experiment logs (autoresearch.jsonl)           │
│  • Identify gaps, bugs, missing features                   │
│  • Read autoresearch.ideas.md for queued ideas              │
│  • Form hypothesis: "Adding X will improve Y"              │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  2. IMPLEMENT                                               │
│  ──────────                                                 │
│  • Write the feature / fix                                  │
│  • Write tests for it                                       │
│  • Keep changes focused and minimal                         │
│  • Follow existing code patterns                            │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  3. VERIFY                                                  │
│  ──────                                                     │
│  • Run full build (npm run build)                           │
│  • Run full test suite (npm test / vitest run)              │
│  • If either fails → diagnose, fix, re-verify               │
│  • No partial successes — all or nothing                    │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  4. MEASURE                                                 │
│  ───────                                                    │
│  • Run the benchmark script (autoresearch.sh)               │
│  • Record primary metric (test_count)                       │
│  • Record secondary metrics (build_ok, build_time_s)        │
│  • Parse METRIC lines from output                           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  5. DECIDE                                                  │
│  ──────                                                     │
│  • Metric improved?  → KEEP (commit)                        │
│  • Metric unchanged? → DISCARD (revert code)                │
│  • Metric regressed? → DISCARD (revert code)                │
│  • Build/test fail?  → CRASH (revert, log error)            │
│                                                              │
│  Log experiment with:                                       │
│  • Hypothesis (what we tried)                                │
│  • Metric (numeric value)                                   │
│  • Status (keep/discard/crash)                              │
│  • Description (what happened)                              │
│  • ASI (learnings, dead ends, next steps)                   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Check termination (STOP SIGNALS):                             │
│  - User said to stop? → STOP                                  │
│  - App delivered to user for testing? → STOP                  │
│  - No more user-requested features? → STOP                    │
│  - Ideas backlog empty/stale? → STOP                          │
│  - Max iterations reached? → STOP                             │
│  - Otherwise → LOOP back to OBSERVE                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Tooling Stack

| Tool | Role in Loop Engg |
|------|-------------------|
| **read** | Examine source files, understand root causes |
| **`loop` CLI** | Go binary — init, run (timed), check, version. Replaces bash autoresearch.sh |
| **bash** | Run shell commands (install, build, test, git) |
| **edit** | Make precise, targeted code changes |
| **write** | Create new files (components, tests, docs) |
| **init_experiment** | Initialize experiment session with metric config |
| **run_experiment** | Run benchmark, capture timing + output + exit code |
| **log_experiment** | Record result with full context (keep/discard + ASI) |

---

## Metrics

### Primary
| Metric | Phase | Direction | Rationale |
|--------|-------|-----------|-----------|
| `build_ok` | 1 (init) | higher ↑ | Binary — does the project compile? |
| `test_count` | 2+ (iterative) | higher ↑ | Proxy for feature completeness + quality |

### Secondary
| Metric | Direction | Use |
|--------|-----------|-----|
| `build_time_s` | lower ↓ | Detect build regressions |
| `tests_ok` | higher ↑ | All tests must pass (1) or fail (0) |

---

## ASI (Actionable Side Information)

Every experiment is annotated with structured ASI:

```json
{
  "hypothesis": "Adding keyboard shortcuts will improve UX without breaking tests",
  "files_modified": ["src/components/DuesSearch.tsx", "src/lib/useKeyboardShortcuts.ts"],
  "new_tests": 7,
  "total_test_count": 159,
  "dead_ends": [
    "useEffect cleanup for global listeners — forgot to return removeEventListener",
    "Esc key to close form also closes ConfirmDialog — had to whitelist"
  ],
  "next_action": "Add undo delete — toast action button + restore logic"
}
```

ASI is the **institutional memory** of the loop. It survives code reverts. It teaches future iterations what didn't work and why.

---

## Real-World Results: 58 Experiments

### What was built
A full-featured dues management dashboard — **entirely one experiment at a time**:

```
Phase 1 (2 exp): Fix build, add test framework
Phase 2 (56 exp): Build features in order of dependency:

CRUD → Filter → Sort → Search → Delete → Edit → Persist →
Export → Badges → Category filter → Bulk actions → Pagination →
Dark mode → Toasts → Confirm dialogs → MongoDB/API → 
Inline editing → Reminder banner → API client → A11y →
Batch category → Keyboard shortcuts → Undo delete →
Service layer → Summary component → Dismissible reminder →
Notes column → CSV/JSON import → formatCurrency →
Persistent dismiss → Edge-case tests → CSV content checks →
Sort tiebreaker → Bulk confirm → Dated exports →
All-selected text → Auto-focus → Clear search → Loading spinner
```

### Final State
| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Build | ❌ Broken | ✅ ~2.7s | Fixed |
| Tests | 0 | **244** | +244 tests |
| Test files | 0 | **23** | +23 files |
| Components | 0 | **16** | +16 components |
| Modules | 0 | **6** | +6 modules |
| API routes | 0 | **4** | +4 routes |

### Keep Rate
- **Kept**: 58 / 58 (100%)
- **Discarded**: 0
- **Crashed**: 0

Every experiment improved or maintained the primary metric.

---

## Why "Loop Engineering"?

The name draws from three metaphors:

1. **Control loop** (cybernetics) — observe, decide, act, measure, repeat
2. **Experiment loop** (scientific method) — hypothesis, test, measure, conclude
3. **Dev loop** (engineering) — code, build, test, iterate

Combined: a **self-correcting autonomous system** that builds software through repeated, measured experiments.

---

## When to Use Loop Engineering

✅ **Greenfield projects** — build from scratch with zero-regression guarantees
✅ **Refactoring** — improve code quality while maintaining all existing tests
✅ **Bug fixing** — add a failing test first, then fix, then prove it passes
✅ **Feature addition** — add features incrementally with companion tests
✅ **Codebase rescue** — add tests to untested code one file at a time

❌ **Exploratory prototyping** — the discipline slows down rapid exploration
❌ **Hotfixes** — the full loop takes minutes, not seconds
❌ **One-time migrations** — not designed for bulk refactors

---

## Reusing This Methodology

To apply Loop Engineering to your own project:

1. **Build the CLI:**
   ```bash
   go build -o loop ./cmd/loop/
   ```

2. **Set up the infrastructure:**
   ```bash
   # Copy the Go binary and config to your project
   cp loop autoresearch.config.json your-project/
   # Set workingDir and maxIterations in autoresearch.config.json
   # Create autoresearch.md with your rules
   ```

3. **Define your metric** — what are you optimizing?
   - Test count? Build time? Bundle size? Performance?

4. **Create a baseline:**
   ```bash
   ./loop init "Baseline" test_count --direction higher
   ./loop run "npm run build && npx vitest run --reporter=json"
   ```

5. **Start the loop:**
   ```
   Observe → Hypothesize → Implement → Verify → Measure → Decide → Repeat
   ```

6. **Track everything** — every experiment is a commit or a lesson learned

---

## Lessons Learned: The Infinite Loop Bug

### The Problem

The original loop had no **terminal condition** for "done." It said "No hard termination — continues indefinitely" and treated `test_count` as a target to chase forever. After auto-compaction, the agent would resume generating hypotheses even though:

- The user had explicitly said to stop
- The app had been delivered for testing
- No new features were requested

This is a design flaw: a self-correcting loop that doesn't know when to stop isn't self-correcting — it's **endless busywork**. The loop optimizes for its own continuation rather than for shipping value.

### The Fix

Added **hard stop signals** to the termination check:

1. **User says stop** — any variant of "r u done", "stop", "lets not overdo"
2. **App delivered to user** for testing → STOP
3. **No more user-requested features** → STOP (don't generate busywork)
4. **Ideas backlog empty/stale** → STOP
5. **Max iterations reached** → STOP (original check retained)

Also added a **safety check** at session start: *"Is there an actual user need, or is this busywork?"* If busywork → prune the ideas backlog and exit.

### How to Implement in Your Loop

In your `autoresearch.md`, replace a simple termination line with explicit stop signals:

```
│  Check termination (STOP SIGNALS):                             │
│  - User said to stop? → STOP                                  │
│  - App delivered to user for testing? → STOP                  │
│  - No more user-requested features? → STOP                    │
│  - Ideas backlog empty/stale? → STOP                          │
│  - Max iterations reached? → STOP                             │
│  - Otherwise → LOOP back to OBSERVE                           │
```

Also add a guard at the top of `autoresearch.ideas.md`:

```
# HALT — Do not start new experiments unless user asks for new features.
```

### The Meta-Lesson

A methodology that automates decision-making must also automate **when to stop**. The termination condition is as important as the iteration logic. Without it, the agent will optimize for metric-chasing over user satisfaction — the ultimate anti-pattern in autonomous engineering.

---

## The Bottom Line

> *Loop Engineering turns software development into a **repeatable, measurable, self-documenting process**. Instead of chaotic sprints and sprawling PRs, you get a clean chain of atomic experiments — each one a yes/no, keep/discard decision that either improves the codebase or teaches you something.*

**58 experiments across a real project. 244 tests. Zero regressions. 100% keep rate.**

This repo is the proof.
