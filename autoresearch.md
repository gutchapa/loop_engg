# Autoresearch: Add dues-dashboard features with tests

## Objective
Build real dues-dashboard features (dues list, CRUD operations, filtering/sorting) while maintaining build quality and test coverage. Each iteration adds meaningful functionality and corresponding tests.

## Tools Used in Loop Engg
| Tool | Purpose |
|------|---------|
| `read` | Examine source files to understand root cause of errors |
| `bash` | Run shell commands (install, build, test) |
| `edit` | Make precise code/config changes |
| `write` | Create new files (autoresearch.sh, new test files, new components) |
| `init_experiment` | Initialize session with metric config |
| `run_experiment` | Run the benchmark command with timing |
| `log_experiment` | Record result (keep/discard) with ASI annotations |

## Termination Condition
No hard termination — loop continues indefinitely, adding features and tests. Each iteration must:
1. **Build passes** — `npm run build` exits with code 0, no errors
2. **Tests pass** — All existing tests must still pass (regression check)
3. **New content added** — Feature, component, or test added

## Metrics
- **Primary**: `test_count` (unitless, higher is better) — number of passing tests (proxy for feature completeness + quality)
  - **Current best**: 56 (from 2 baseline = +2700%)
- **Secondary**: `build_ok` (0 or 1, higher is better) — must stay at 1
- **Secondary**: `build_time_s` (lower is better) — must not regress significantly

## How to Run
```bash
./autoresearch.sh
```
Outputs `METRIC name=value` lines.

## Files in Scope
| File | Purpose |
|------|---------|
| `package.json` | Dependencies and scripts |
| `postcss.config.mjs` | PostCSS plugin config |
| `next.config.ts` | Next.js config |
| `src/app/globals.css` | Global styles |
| `src/app/layout.tsx` | Root layout |
| `src/app/page.tsx` | Home page |
| `src/components/*` | Reusable UI components |
| `src/lib/*` | Data layer, hooks, utilities |
| `src/__tests__/*` | Test files |

## Off Limits
- `node_modules/` — never modify directly
- `.next/` — build artifacts

## Constraints
- Must use `npm` (not pnpm/yarn)
- Build must complete without errors
- All tests must pass — never regress
- New features must be accompanied by tests

## What's Been Tried (Phase 1 — Fix Build)

### Iteration 1 — ✅ Build passes
- **Fix**: Upgraded eslint `^8→^9`, added `@tailwindcss/postcss ^4`
- **Result**: Build passes in ~2.1s

### Iteration 2 — ✅ Tests added
- **Added**: vitest, testing-library, 2 smoke tests
- **Result**: 2 tests pass

## What's Been Tried (Complete — 58 iterations)

| # | Feature | Tests Added | Cumulative |
|---|---------|-------------|------------|
| 1 | Fix eslint + tailwind deps (build fix) | 0 | 0 → 2 |
| 2 | Add vitest + smoke tests | 2 | 2 |
| 3 | Data layer (CRUD) + DuesList component | 13 | 15 |
| 4 | DuesFilter (status tabs) | 4 | 19 |
| 5 | DuesForm (add with validation) | 5 | 24 |
| 6 | Mark paid + form integration | 4 | 28 |
| 7 | Column sorting (asc/desc) | 6 | 34 |
| 8 | DuesSearch (name filter) | 3 | 37 |
| 9 | Delete entry | 4 | 41 |
| 10 | Edit entry (form edit mode) | 8 | 49 |
| 11 | localStorage persistence | 4 | 53 |
| 12 | Data export (JSON/CSV) | 3 | 56 |
| 13 | Due-date badges | 12 | 68 |
| 14 | Category filtering | 6 | 74 |
| 15 | Bulk actions (select + batch) | 6 | 80 |
| 16 | Pagination | 6 | 86 |
| 17 | Dark mode toggle | 5 | 91 |
| 18 | Page integration tests | 8 | 99 |
| 19 | Toast notifications | 6 | 105 |
| 20 | Confirm dialog for delete | 9 | 114 |
| 21 | MongoDB model + API routes | 8 | 122 |
| 22 | Inline category editing | 5 | 127 |
| 23 | DueReminder banner | 8 | 135 |
| 24 | API client (fetch wrappers) | 7 | 142 |
| 25 | Accessibility pass (aria roles) | 5 | 147 |
| 26 | Batch category assignment | 5 | 152 |
| 27 | Keyboard shortcuts (N/S/Escape) | 7 | 159 |
| 28 | Undo delete (toast action) | 3 | 162 |
| 29 | Async service layer (API-first/fallback) | 0 | 162 |
| 30 | Service layer tests (API + fallback) | 8 | 170 |
| 31 | DuesSummary component extraction | 7 | 177 |
| 32 | Dismissible DueReminder | 4 | 181 |
| 33 | Notes column in DuesList | 6 | 187 |
| 34 | DuesImport (CSV/JSON import) | 13 | 200 |
| 35 | formatCurrency utility extraction | 8 | 208 |
| 36 | Edge-case tests (validation, search, pagination) | 10 | 221 |
| 37 | Notes textarea in DuesForm | 3 | 223 |
| 38 | CSV content + DuesFilter edge cases + page interaction tests | 10 | 233 |
| 39 | DueReminder formatCurrency refactor | 0 | 233 |
| 40 | Sorting tiebreaker (name when equal keys) | 2 | 235 |
| 41 | Bulk delete confirmation dialog | 3 | 238 |
| 42 | Dated export filenames | 0 | 238 |
| 43 | Bulk bar "All X selected" text | 1 | 239 |
| 44 | Auto-focus name input in form | 1 | 240 |
| 45 | Clear search button | 3 | 243 |
| 46 | Loading state with spinner | 1 | 244 |

## Final State (58 experiments, 244 tests)
- ✅ Build passes in ~2.7s
- ✅ **244 tests pass** (23 test files, 63 test suites)
- ✅ Full CRUD (Create, Read, Update, Delete) with confirmation dialogs
- ✅ Status filtering tabs (All/Pending/Overdue/Paid) with ARIA tablist
- ✅ Column sorting (asc/desc on name/amount/date/status) with tiebreaker
- ✅ Search by name (case-insensitive, accessible, with clear button)
- ✅ Category filtering dropdown
- ✅ Inline category editing (click to edit)
- ✅ Batch category assignment (bulk select + set category)
- ✅ Due-date awareness badges (Overdue/Due today/Due in Xd/Xd left/Paid)
- ✅ DueReminder banner (overdue/due-today/due-soon with role=alert, persistent 24h dismiss)
- ✅ Inline Mark Paid + Edit + Delete
- ✅ Bulk actions (checkbox select-all + batch mark paid/delete/set category + confirmation)
- ✅ Pagination (page numbers + Prev/Next)
- ✅ Data export (JSON + CSV with dated filenames) + Data import (CSV/JSON upload)
- ✅ DuesForm with all fields: name, amount, dueDate, category, notes (auto-focuses name)
- ✅ localStorage persistence (data + reminder dismiss state)
- ✅ Dark mode toggle (manual, accessible)
- ✅ Toast notifications with action buttons (undo delete)
- ✅ ConfirmDialog for all destructive actions (delete + bulk delete)
- ✅ Keyboard shortcuts (N=new, S=search, Esc=close)
- ✅ MongoDB model + API routes (GET/POST/PATCH/DELETE)
- ✅ API client + async service layer (API-first, falls back to localStorage)
- ✅ DuesSummary component, Notes column with CSS truncation
- ✅ formatCurrency utility, metadata with meaningful title
- ✅ Page-level integration tests (async-aware)
- ✅ Loading state with spinner animation
- ✅ Bulk action bar shows "All X selected" vs "X selected"
- ✅ Sort tiebreaker (name when equal keys)
- ✅ 16 components, 6 modules, 4 API routes, 23 test files
