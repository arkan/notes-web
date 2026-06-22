# Edit mode CRUD implementation

## Plan

- [x] Load PRD, Tier 1 rules, UI/backend/testing/tooling rules, security model, current tasks, and lessons.
- [x] Reuse prior edit-mode exploration context for implementation mapping.
- [x] Create deepwork state at `.slim/deepwork/edit-mode-crud.md`.
- [x] Draft implementation plan at `docs/plans/2026-06-20-edit-mode-crud-implementation.md`.
- [x] Review the implementation plan with `@oracle`.
- [x] Reconcile review findings.
- [x] Get execution confirmation before code.
- [x] Phase 0: context/docs, config, path policy, CSRF foundation with Go tests.
- [x] Phase 1: existing-note edit, preview, save, dirty/conflict behavior with Go + Playwright tests.
- [x] Phase 2: create/templates/rename/link rewrite/missing-link creation with tests.
- [x] Phase 3: trash/restore/restore-as with tests.
- [x] Phase 4: command palette, badges, mobile shell, designer review with tests.
- [x] Phase 5: docs, oracle review, full validation.
- [x] Condense note/folder header actions into a single tool dropdown.
- [x] Validate actions dropdown with Go/Playwright checks.
- [x] Grill modern premium UI/UX direction for Notes Web.
- [x] Create prototype plan and output directory.
- [x] Create 5 interactive modern UI/UX prototype variants under `docs/plans/`.
- [x] Create prototype index/scorecard and reconcile links.
- [x] Validate prototypes are self-contained and run `git diff --check`.
- [x] Review prototype set with `@designer`.

## Current status

- Wide content border polish complete: Trash/Inbox full-page `reading-surface` frames were replaced with unboxed wide `app-page` surfaces while keeping child cards and edit hooks intact.
- Folder surface border polish complete: folder pages no longer use the generic boxed reading-surface frame; dedicated quiet list styling is covered by Go/Playwright checks.
- Media placeholder polish complete: local missing images and local non-previewable media now render as compact flat `.media-placeholder` links; valid images keep simple borderless rendering; full `make test` passed with 154 Playwright tests.
- Modern Workbench Hermes visual polish complete: sidebar independent scroll, TODO toolbar in-flow, calmer task rows/context pane/search recents, focused Playwright coverage, and targeted validation passed.
- Phase 1 passed oracle + UI recheck.
- Phase 2 passed oracle recheck and designer recheck after `.note-actions` wrap fix.
- Phase 3 passed oracle/designer after symlink/metadata fixes.
- Phase 4 passed oracle/designer after template action, dirty palette, global Trash, and empty-folder rename fixes.
- Phase 5 complete: README/context docs updated, final oracle review PASS, `make test` + `git diff --check` passed.
- UI polish complete: note/folder action buttons are condensed into one `⚙` dropdown while preserving existing action hooks.
- Modern UI/UX grill complete: validated prototype brief written to `docs/plans/2026-06-21-modern-ui-ux-prototype-brief.md`.
- Prototype execution plan written to `docs/plans/2026-06-21-modern-ui-ux-prototypes-plan.md`.
- Prototype set complete under `docs/plans/modern-ui-ux-prototypes/`: index plus Linear, Raycast, Readwise, Obsidian, and Hybrid variants.
- Prototype validation passed: expected files exist, no external network references, inline JS parses, `git diff --check` passed for prototype paths.
- Designer review of prototype set: PASS.
- Prototype variants cleaned up to app-only fullscreen renderings: prototype rails, explanatory intros, scorecards, and mobile showcase clutter are hidden/removed from the main variant view; final designer recheck PASS.
- UI/UX direction selected: `variant-linear.html`.
- Linear spec plan written to `docs/plans/2026-06-21-modern-ui-linear-spec-plan.md`.
- Linear UI/UX spec written to `docs/plans/2026-06-21-modern-ui-linear-ui-ux-spec.md`; Trash purge corrected out of v1 scope.
- Modern Workbench Refactor umbrella plan drafted at `docs/plans/2026-06-21-modern-workbench-refactor.md`.
- Modern Workbench Refactor plan reviewed by oracle (`PASS_WITH_CHANGES`) and reconciled: Phase 0.5 inventory, design-source gate, split Phase 5, nav rollout guard, mobile/test/review-size gates.
- Modern Workbench Phase 0 baseline passed: `make test` with 113 Playwright tests.
- Modern Workbench Phase 0.5 route/template/action inventory written to `docs/plans/2026-06-21-modern-workbench-route-action-inventory.md`.
- Modern Workbench Phase 1 shell plan drafted at `docs/plans/2026-06-21-modern-workbench-phase1-shell-plan.md`; oracle returned PASS_WITH_CHANGES and corrections were reconciled.
- Modern Workbench UI code is blocked until user confirms migration boundary and design source-of-truth handling.
- User decisions for Modern Workbench Phase 1: use current working tree, update `DESIGN.md`/`.context/rules/ui.md` before code, hide unimplemented nav routes, use Note as first golden path, preserve Mermaid/MathJax CDN for Phase 1.
- Modern Workbench Phase 1 shell/tokens implemented and validated: `make test` passed with 117 Playwright tests; oracle PASS; designer recheck PASS after mobile sidebar inert/aria-hidden fix.
- Modern Workbench Phase 2 nav/context/palette plan drafted at `docs/plans/2026-06-21-modern-workbench-phase2-nav-context-palette-plan.md`; awaiting oracle review before code.
- Modern Workbench Phase 2 implemented and validated: full `make test` passed with 122 Playwright tests; oracle final recheck PASS; designer recheck PASS.
- Modern Workbench Phase 3 plan reviewed by oracle (`PASS_WITH_CHANGES`) and reconciled. Quick capture is deferred to Phase 4; Phase 3 renders no Quick capture field/button/placeholder.
- Modern Workbench Phase 3A Note migration passed oracle/designer after mobile anchor offset fix; full `make test` passed with 124 Playwright tests.
- Modern Workbench Phase 3B Home migration passed oracle/designer after Home single-column and due-now summary fixes; full `make test` passed with 126 Playwright tests.
- Modern Workbench Phase 4 Inbox/Capture decisions collected and plan drafted at `docs/plans/2026-06-21-modern-workbench-phase4-inbox-capture-plan.md`; oracle plan review PASS after corrections.
- Modern Workbench Phase 4A backend passed oracle after collision, structured-error, template-leak, todo-dir, and Inbox-list hardening.
- Modern Workbench Phase 4B Inbox UI passed oracle/designer after busy-state, symlink gating, human titles, and responsive-card fixes; full `make test` passed with 134 Playwright tests.
- Modern Workbench Phase 5a Search/Maintenance/Trash passed oracle/designer; full `make test` passed with 136 Playwright tests.
- Modern Workbench Phase 5b Tasks/Calendar/Tags passed oracle/designer after Calendar local-time/template/safe-render fixes and center-pane overflow fixes; full `make test` passed with 138 Playwright tests.
- Modern Workbench Phase 5c Projects passed oracle/designer after active-project semantic test and nav update; full `make test` passed with 141 Playwright tests.
- Modern Workbench Phase 6 Mobile/Settings/Prefs passed oracle/designer after safe palette recents, reading-focus sync, Dataview mobile overflow, and density comfort fixes; full `make test` passed with 148 Playwright tests.
- Modern Workbench Phase 7 cleanup/docs/context passed oracle/designer after CSS dead-selector cleanup, `Object.hasOwn` update, README/DESIGN/Tier2 context updates, and full `make test` with 148 Playwright tests.
- No GitHub issues created and no commits made.

## Phase 5a Search/Maintenance/Trash review

- Added read-only `/_maintenance` route with counts/links only; no path examples.
- Search page was modernized as an inspection surface while preserving query semantics.
- Trash page was visually polished while preserving Restore / Restore as only and no purge.
- App nav after Phase 5a: Home, Tasks, Search, Tags, Maintenance when editing disabled; Home, Inbox, Tasks, Search, Tags, Maintenance, Trash when editing enabled.
- Maintenance path-policy/count tests cover hidden/template/trash diagnostics and count parity.
- Validation: `make test` + `git diff --check` passed with 136 Playwright tests.
- Reviews: oracle PASS after must-fix recheck; designer PASS.

## Phase 5b Tasks/Calendar/Tags review

- Added server-backed read-only `/_calendar` route for daily notes only.
- Calendar query contract: `month=YYYY-MM`, `date=YYYY-MM-DD`, invalid params safe fallback, local-time parsing with `time.ParseInLocation`.
- Calendar uses `daily_notes_glob`/daily-note path policy and explicitly excludes `_template.md` even when `editing.hide_templates=false`.
- Calendar selected-day preview now renders Markdown HTML safely via `safe`.
- Tasks and Tags received Modern Workbench polish while preserving `data-todo-*`, task menus/filters/grouping, `data-tag-filter`, `data-hide-rare`, `data-tag-chip`, and `/_tags` routes.
- Review fixes: Calendar and Tasks stack/contain earlier under the right context pane to avoid internal `.main` horizontal overflow.
- Validation: `make test` + `git diff --check` passed with 138 Playwright tests.
- Reviews: oracle PASS after full-suite gate; designer PASS.

## Phase 5c Projects checklist

- [x] Polish `/_projects` as a read-only Modern Workbench page.
- [x] Add client-only project filtering with a filtered empty state.
- [x] Preserve `ActiveProjectsAll` semantics and app nav order.
- [x] Add focused Playwright contracts for Projects rendering, filtering, nav, and mobile overflow.
- [x] Run targeted validation, full `make test`, and `git diff --check`.

### Review

- Projects now renders a compact Modern Workbench page with header chips, overview counts, read-only search/filter, and active project cards showing label, description/path, note count, updated date, and latest note.
- Client filtering is DOM-only; it does not change URLs, server state, or project semantics.
- ActiveProjects semantics are covered: a project with one active note can still count/show a more recent non-active note in the same project.
- Visual checks covered desktop 1440 dark mode and mobile 390 dark mode; overflow metrics passed at 1280, 1366, 1440, 390, and 320 widths.
- Validation passed: `node --check internal/app/static/app.js`, `go test ./internal/app`, `npm run lint`, targeted Playwright Projects/nav tests, full `make test` with 141 Playwright tests, and `git diff --check`.
- Reviews: oracle PASS after test/full-suite gate; designer PASS.

## Phase 6 Mobile/Settings/Prefs checklist

- [x] Plan and review Phase 6 before implementation.
- [x] Add density preference as local UI state orthogonal to font size.
- [x] Add safe palette recents backed by current server palette data.
- [x] Keep Settings accessible on mobile and in reading-focus mode.
- [x] Fix mobile overflow regressions surfaced during review.
- [x] Run targeted checks, full `make test`, and `git diff --check`.

### Review

- Settings now exposes density, reading focus, and palette recent management while preserving existing theme/font controls.
- Palette recents store only URL-backed server palette items, drop stale/unsafe localStorage entries, and cap mobile recents to reduce clutter.
- Reading focus removes `data-reading-focus` when off, keeps palette accessible, hides the unusable hamburger, and syncs the Settings select after keyboard toggles.
- Comfortable density now preserves/increases key surface spacing instead of shrinking Home/Calendar cards.
- Dataview diagnostics now wrap/scroll query blocks internally on mobile.
- Validation passed: targeted Go/Playwright checks, full `make test` with 148 Playwright tests, and `git diff --check`.
- Reviews: oracle PASS; designer PASS after density rechecks.

## Phase 7 Cleanup/docs/context checklist

- [x] Plan and review Phase 7 cleanup before implementation.
- [x] Fix stale Settings shortcut copy.
- [x] Remove proved-dead legacy TODO layout CSS selectors.
- [x] Replace remaining `Object.prototype.hasOwnProperty.call` usages with `Object.hasOwn`.
- [x] Update README, DESIGN.md, `.context/rules/ui.md`, and relevant Tier 2 context.
- [x] Preserve deferred items: `map.go`, legacy root tokens, broad class renames, clipboard fallback.
- [x] Run targeted checks, `node --check`, full `make test`, and `git diff --check`.

### Review

- CSS cleanup removed `.todo-board`, `.todo-secondary`, and `.todo-column` references after confirming no template/JS/test usage.
- Settings shortcuts now consistently describe `⌘/Ctrl+B` as reading focus.
- Docs now describe Modern Workbench app surfaces, Settings preferences, safe palette recents, Inbox/capture, Projects, Calendar, Maintenance, Trash, and testing gates.
- Context now records browser-local preference rules, safe palette recents, density constraints, Dataview mobile overflow expectations, route map, and validation strategy.
- Validation passed: targeted Go/lint/Playwright, `node --check internal/app/static/app.js`, full `make test` with 148 Playwright tests, and `git diff --check`.
- Reviews: oracle PASS after final recheck; designer PASS.

## Phase 4B Inbox UI checklist

- [x] Load Phase 4 plan, backend/UI/testing/tooling rules, current tasks, and lessons.
- [x] Write local implementation plan.
- [x] Gate Inbox visibility from server data: route, app nav, context links, Home Quick capture.
- [x] Add `/_inbox` server-rendered page with current captures and action affordances.
- [x] Add Home Quick capture only when editing and Inbox are enabled.
- [x] Add vanilla JS for capture, archive, move confirmations, convert-to-task, and mobile-safe actions.
- [x] Add focused Modern Workbench CSS.
- [x] Add Go/template contracts and Playwright coverage on temporary vaults.
- [x] Run targeted validation and `git diff --check`.

### Review

- Inbox route/app nav/Home Quick capture now render only when editing and Inbox are enabled; editing-disabled returns 404 and hidden Inbox returns 403.
- Home Quick capture is additive above configured homepage blocks and preserves Phase 3B due-now summary/block order.
- Inbox page lists current captures with Open, Archive, Move, and Convert-to-task actions; Convert displays backend disabled reasons when unavailable.
- JS uses existing CSRF/edit JSON helpers and supports missing-folder/hidden move confirmations from backend 409 responses.
- Validation passed: `make test-go`, `npm run lint`, targeted Playwright `edit-mode` + `internal-pages`, full `make test` with 134 Playwright tests, and `git diff --check`.
- Review: oracle PASS after must-fix recheck; designer PASS after must-fix recheck.

## Phase 3B Home daily cockpit checklist

- [x] Load Modern Workbench Phase 3 plan, Linear spec, refactor plan, design system, UI/testing rules, tasks, and lessons.
- [x] Write local implementation plan.
- [x] Preserve homepage block model, ordering, visibility, and CSS order contracts.
- [x] Modernize Home into a calm daily cockpit with daily note hero and prominent urgent tasks.
- [x] Keep Quick capture absent and add contract coverage.
- [x] Improve responsive Home layout for narrow desktop/tablet/mobile.
- [x] Add/update Go and Playwright contracts.
- [x] Run targeted validation and `git diff --check`; run full `make test` if feasible.

### Review

- Home now uses a daily cockpit header, centered dashboard width, larger today hero date, prominent due-now task block, and calmer secondary cards while preserving all configured `data-home-block`, `data-home-order`, and `--home-block-order` contracts.
- Quick capture remains absent by markup and Playwright contracts.
- Visual spot checks completed on desktop 1440×900 and mobile 390×844 dark mode.
- Validation passed: `make test-go`, `make lint`, targeted Playwright for `internal-pages` + `edit-mode`, `git diff --check`, and full `make test` with 126 Playwright tests.

## Phase 3A Note migration checklist

- [x] Load Modern Workbench Phase 3A plan, Linear spec, refactor plan, design system, UI/testing rules, tasks, and lessons.
- [x] Write local implementation plan.
- [x] Preserve note action dropdown, edit hooks, copy/Open URL/reading prompt hooks, and edit-mode behavior.
- [x] Modernize note reading composition, metadata/link panels, and mobile note spacing.
- [x] Keep context pane note additions read-only and free of write/destructive actions.
- [x] Add/update note visual/behavior contract tests.
- [x] Run targeted validation and `git diff --check`.

## Phase 3A Note migration review

- Note page now renders inside a centered reading stack with modern breadcrumb, title, prose, TOC, frontmatter, and link-panel treatment.
- Main-content note action dropdown and edit/copy/Open URL/reading prompt hooks remain in place; context pane stays read-only.
- Mobile note layout prevents horizontal page overflow and keeps the gear action reachable.
- Dataview multi-filter menu focus now avoids immediate scroll-close when the taller note reading stack pushes controls near the viewport edge.
- Validation passed: `make test-go`, `make lint`, targeted Playwright note suites, `npx playwright test tests/e2e/dataview-filters.spec.ts`, full `make test` with 124 Playwright tests, and `git diff --check`.

## Phase 2 UI/UX checklist

- [x] Render New in editable note/folder context and Rename on editable note pages.
- [x] Render Create this note on missing pages with source context.
- [x] Implement create note/folder dialog with title-to-path sync, manual path freeze, confirmations, and success navigation.
- [x] Implement rename dialog with preview-first impact, hidden confirmation, execute, and success navigation.
- [x] Implement missing-link create dialog with preview-first impact/template hints, execute, and success navigation.
- [x] Add calm mobile-safe dialog styling.
- [x] Add lightweight UI contract tests and targeted validation.

## Phase 2 UI/UX review

- Added Phase 2 UI hooks and vanilla modal flows only; Trash and command palette remain untouched.
- Validation: `node --check internal/app/static/app.js`, targeted UI tests, `go test ./internal/...`, `go test ./cmd/... ./internal/...`, `make lint`, `git diff --check`.

## Phase 1 UI/UX checklist

- [x] Render Edit action on Markdown note pages only when editing is enabled.
- [x] Build inline vanilla JS edit workbench with Source/Preview tabs.
- [x] Implement manual Preview, stale indicators, Save reload, Cancel/Esc dirty confirmation, beforeunload/internal navigation dirty guards.
- [x] Implement save conflict message with Copy draft and confirmed Reload disk.
- [x] Add calm desktop/mobile styling.
- [x] Add lightweight UI contract tests and run targeted validation.

## Phase 1 UI/UX review

- Existing-note edit UI is in templates/static assets only; no Create/Rename/Trash/command palette actions added.
- Validation: `go test ./internal/...`, `go test ./cmd/... ./internal/...`, `make lint`, `git diff --check`.
- Playwright edit-mode E2E added with a temporary copied vault and dedicated server process.
- Validation: `make test-go`, `make lint`, `make test-e2e`, `git diff --check`.
- Review: oracle PASS, designer PASS after targeted fixes.

---

# Edit mode CRUD PRD

## Plan

- [x] Reuse the completed edit-mode codebase exploration, security review, and UX review.
- [x] Load the documentation, UI, backend, testing, and security context rules.
- [x] Write a local plan under `docs/plans/2026-06-20-edit-mode-prd.md`.
- [x] Write the PRD under `docs/edit-mode-crud-prd.md`.
- [x] Run `git diff --check`.

## Result

- Local PRD created; no GitHub issue was created per request.

---

# Speed optimization to <50 ms/page

## Plan

- [x] Establish repeatable benchmark against `/home/arkan/hermes` on `http://100.79.17.90:18080`.
- [x] Remove full-vault scans from note forward/backlink rendering.
- [x] Reuse `VaultIndex` for homepage/TODO/Dataview/notes-map work.
- [x] Reduce repeated layout payload with a top-level + active-branch sidebar.
- [x] Add gzip for text responses.
- [x] Fix encoded `?` filenames by resolving `EscapedPath()` with `PathUnescape()`.
- [x] Cap expensive initial renders: Dataview tables, large folders, large code fences.
- [x] Keep Dataview `FILTER` tables uncapped so initial render and AJAX filters use the same result set.
- [x] Remove server-side syntax highlighting cost for large fenced-code notes.
- [x] Precompute backlink contexts in `VaultIndex`.
- [x] Verify every discovered page loads in under 50 ms.

## Final benchmark

Command shape: run server with `go run cmd/notes-web/main.go --vault /home/arkan/hermes --host 100.79.17.90 --port 18080`, then run `tmp/perf/bench-pages.go` with `-n 1 -warmups 0`.

Latest result after the Dataview filter-cap fix: `7018` pages, `failed=0`, `status_failures=0`.

Slowest page: `/_orphans` at `44.90ms`; slowest note: `/Areas/MOC%20Ressources.md` at `41.29ms`.

## Validation

- [x] `make test-go`
- [x] `make lint`
- [x] `npx playwright test tests/e2e/dataview-filters.spec.ts`
- [x] `git diff --check`
