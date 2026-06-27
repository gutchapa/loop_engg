# Dues Dashboard — Complete Feature Inventory

## Implemented (30 features across 34 experiments)
- ✅ All CRUD operations (Create, Read, Update, Delete) with confirmation dialogs
- ✅ Status filtering tabs (All/Pending/Overdue/Paid) with ARIA tablist role
- ✅ Column sorting (name/amount/date/status, asc/desc toggle) with aria-sort
- ✅ Search/filter by name (DuesSearch, accessible via aria-label)
- ✅ Category filtering dropdown with aria-label
- ✅ Inline category editing (click to edit, Enter/blur to save)
- ✅ Due-date awareness badges (Overdue/Due today/Due in Xd/Xd left/Paid)
- ✅ DueReminder banner (overdue/due-today/due-soon alerts with accessible role)
- ✅ Data export (JSON + CSV download)
- ✅ localStorage persistence (data survives page refresh)
- ✅ Bulk actions (checkbox select + batch mark paid/delete)
- ✅ Pagination (page navigation with Prev/Next + page numbers)
- ✅ Dark mode toggle (manual, accessible via aria-label)
- ✅ Toast notifications (success/error/info)
- ✅ ConfirmDialog for destructive actions (accessible modal with aria-modal)
- ✅ MongoDB model + API routes (model, dbConnect, GET/POST/PATCH/DELETE)
- ✅ API client (async wrappers for full-stack persistence)
- ✅ Page-level integration tests
- ✅ Inline category editing

## Not Yet Explored (lower priority)
- **E2E tests with Playwright** — full browser-level integration testing
- **CI/CD pipeline** — GitHub Actions for build + test on push

## Final Stats
- **244 tests** across 23 test files, 63 test suites
- **16 components** (DuesList, DuesFilter, DuesForm, DuesSearch, DuesExport, DuesImport, DuesSummary, CategoryFilter, DarkModeToggle, DueReminder, ConfirmDialog, Toast + ToastContainer)
- **6 data/service/utility modules** (lib/dues.ts, lib/api-client.ts, lib/dues-service.ts, lib/format.ts, models/Dues.ts, dbConnect.ts)
- **4 API routes** (GET/POST /api/dues, PATCH/DELETE /api/dues/[id])
- **Build passes** in ~2.7s

## Implemented (complete — 58 experiments)
- ✅ Full CRUD with localStorage persistence
- ✅ Status/name/category filtering + column sorting with tiebreaker
- ✅ Bulk actions (mark paid, delete, set category) with confirmation
- ✅ Pagination + data export/import (CSV/JSON with dated filenames)
- ✅ DueReminder banner with persistent 24h dismiss, formatted currency
- ✅ Dark mode, keyboard shortcuts, undo delete, toast notifications
- ✅ ConfirmDialog, inline category editing, DuesSummary
- ✅ Notes column + Notes textarea in DuesForm
- ✅ MongoDB model + API routes
- ✅ Async service layer (API-first, localStorage fallback)
- ✅ Accessibility (ARIA roles, labels)
- ✅ CSV content verification, DuesFilter edge cases, combined filter tests
- ✅ formatCurrency refactored across DuesList, DuesSummary, and DueReminder
- ✅ Edge-case tests (whitespace validation, search inputRef, pagination boundaries)
- ✅ Sorting tiebreaker, bulk delete confirmation, dated export filenames
- ✅ Bulk bar "All X selected" vs "X selected", auto-focus name input
- ✅ Clear search button, loading state with spinner
- ✅ **244 tests** across 23 test files, 63 suites

## Ideas Pruned (implemented since creation)
- ~~Wire frontend to use API routes~~ ✅ Done (service layer)
- ~~Batch category assignment~~ ✅ Done
- ~~Keyboard shortcuts~~ ✅ Done
- ~~Undo delete~~ ✅ Done
- ~~DuesImport (CSV/JSON)~~ ✅ Done
- ~~DuesSummary component~~ ✅ Done
- ~~Notes column display~~ ✅ Done
- ~~formatCurrency utility~~ ✅ Done
- ~~Edge-case tests~~ ✅ Done
- ~~CSV content format verification~~ ✅ Done
- ~~Combined search+status filter tests~~ ✅ Done
- ~~formatCurrency in DueReminder~~ ✅ Done
- ~~DuesExport content tests~~ ✅ Done
- ~~DuesFilter zero-count and same-tab tests~~ ✅ Done
