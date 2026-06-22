# PRD: Inline Edit Mode and Vault CRUD

## Problem Statement

Notes Web is a calm, keyboard-first workbench for reading, retrieving, and maintaining a trusted Markdown vault, but the current browser experience is read-only. The user can browse notes, inspect folders, search, review tasks, follow wikilinks, and view diagnostics, but any actual note maintenance still requires switching to another editor or file manager.

That context switch breaks the product promise of a fast, actionable vault workbench. It is especially painful when the user finds a typo while reading, follows a missing wikilink, wants to create a note in the current folder, needs to rename a note while preserving links, or wants to move stale notes out of the normal vault view. The browser UI should support these maintenance actions while preserving Notes Web's design principles: warm reading comfort, calm density, keyboard-native flow, explicit safety, and no heavy frontend stack.

The feature also requires a deliberate security and product-model change. Notes Web currently treats hidden paths as blocked from normal pages, actions, and diagnostics. The validated direction changes configured hidden paths into non-enumerated paths that can still be accessed by direct URL, while dot-prefixed paths remain blocked. Because this feature introduces write access to the vault, the product needs an explicit PRD-level contract for write activation, CSRF, path safety, trash behavior, conflicts, templates, link rewrites, and tests.

## Solution

Add an explicitly enabled inline edit mode and CRUD surface for Markdown notes and empty folders in the vault.

When editing is enabled in the vault configuration, note pages gain a calm toolbar with creation, editing, rename, and move-to-trash actions. Clicking Edit transforms the current note page into an inline workbench without changing the URL. The workbench keeps the surrounding app context on desktop, uses a full-screen editing shell on mobile, and provides Source and Preview tabs. Source uses a vanilla Markdown textarea. Preview is manually generated from the current unsaved textarea content using the same renderer as normal reading, so the user can verify Dataview blocks, Mermaid blocks, wikilinks, callouts, GFM, and trusted raw HTML as they will appear after saving.

The feature supports creating notes, creating empty folders, editing existing Markdown notes, renaming notes or empty folders, moving notes or empty folders to a hidden trash, restoring from trash, and creating notes from missing wikilinks. Note creation uses nearest-parent `_template.md` resolution with a blank fallback, slugifies new note filenames to kebab-case lowercase with transliterated accents, and preserves folder segment names. Rename and missing-wikilink creation include impact previews and rewrite exact wikilinks and exact Markdown links where validated. Destructive behavior is deliberately softened: the UI uses “Move to Trash”, stores dated snapshots under `_trash`, and does not support permanent purge in the first version.

Write access is disabled by default and is activated only by vault configuration. Basic Auth is not required when editing is enabled, but every write action must use CSRF protection. Writes must remain root-bound, reject traversal, block symlink writes, preserve existing file permissions, use atomic file replacement, invalidate the vault index after successful writes, and detect conflicts before overwriting disk changes.

## User Stories

1. As a vault owner, I want to enable editing explicitly in configuration, so that a read-only Notes Web instance never gains write powers by accident.
2. As a vault owner, I want editing disabled by default, so that existing deployments keep their current safety model until I opt in.
3. As a vault owner, I want write actions protected by CSRF tokens, so that another website cannot silently mutate my vault through my browser.
4. As a vault owner, I want optional Basic Auth to remain optional for editing, so that a trusted local deployment can still work without credentials.
5. As a reader, I want an Edit action on a note page, so that I can fix a note at the moment I notice a problem.
6. As a reader, I want editing to happen inline without changing the URL, so that the page keeps its place in my browser history and feels like the same workspace.
7. As a reader, I want the sidebar and surrounding context preserved on desktop while editing, so that I do not lose orientation in the vault.
8. As a mobile user, I want editing to use a full-screen mobile shell, so that the textarea has enough space to be usable.
9. As a keyboard-oriented user, I want `e` to enter edit mode, so that editing is reachable without the mouse.
10. As a keyboard-oriented user, I want `Ctrl/Cmd+S` to save, so that the edit flow matches common editor expectations.
11. As a keyboard-oriented user, I want `Ctrl/Cmd+Enter` to render Preview, so that I can verify Markdown without leaving the keyboard.
12. As a keyboard-oriented user, I want `Esc` to behave like Cancel, so that I have a predictable way to leave edit mode.
13. As a careful editor, I want `Esc` on unsaved changes to ask for confirmation, so that a quick keypress never discards my draft silently.
14. As a careful editor, I want Cancel on unsaved changes to ask for confirmation, so that I do not lose work by mistake.
15. As a careful editor, I want browser refresh, tab close, and internal navigation to warn me when changes are unsaved, so that I do not accidentally abandon a draft.
16. As an editor, I want the frontmatter to remain raw Markdown/YAML inside the textarea, so that Notes Web does not silently transform my vault format.
17. As an editor, I want Save to write exactly the textarea content without automatic Markdown or YAML formatting, so that the app does not introduce unwanted diffs.
18. As an editor, I want Save to return me to the rendered note, so that I immediately see the saved reading result.
19. As an editor, I want the source Markdown to be fetched only when I click Edit, so that normal note reading stays fast and lightweight.
20. As an editor, I want a Source tab, so that I can edit the raw Markdown directly.
21. As an editor, I want a Preview tab, so that I can inspect the rendered result before saving.
22. As an editor, I want Preview to be manual rather than live, so that large notes do not constantly re-render while I type.
23. As an editor, I want Preview to use the same renderer as read mode, so that Dataview, Mermaid, wikilinks, callouts, GFM, raw HTML, and heading IDs match the saved page.
24. As an editor, I want Preview to render the current unsaved textarea content, so that I can verify what I am about to save.
25. As an editor, I want a visible stale indicator when I type after rendering Preview, so that I know the preview no longer matches the source.
26. As an editor, I want the stale indicator shown both on the Preview tab and near the Preview action, so that it is visible from either editing state.
27. As an editor, I want conflicts detected if the file changed on disk since I began editing, so that Notes Web does not overwrite external edits.
28. As an editor, I want a save conflict to keep my draft in the textarea, so that my unsaved work is never lost.
29. As an editor, I want a Copy Draft action when a conflict occurs, so that I can preserve my version outside the browser before reloading.
30. As an editor, I want Reload Disk to replace the textarea only after confirmation, so that I understand the draft will be overwritten by the current disk version.
31. As a note creator, I want New to default to the current folder, so that notes are created where I am working.
32. As a note creator, I want the creation path to be editable, so that I can place a note somewhere else without leaving the flow.
33. As a note creator, I want a title field to generate a kebab-case lowercase Markdown filename, so that new notes follow the chosen naming convention.
34. As a note creator, I want accents transliterated during slugging, so that generated filenames remain simple and predictable.
35. As a note creator, I want `.md` applied to generated note filenames, so that created notes are Markdown notes.
36. As a note creator, I want non-Markdown note extensions blocked, so that the edit mode stays scoped to Markdown pages.
37. As a note creator, I want folder path segments to preserve the text I type, so that existing vault folder naming conventions are not normalized unexpectedly.
38. As a note creator, I want manual edits to the path field to freeze automatic title-to-slug updates, so that the title field does not overwrite a path I adjusted.
39. As a note creator, I want missing intermediate folders to require confirmation before creation, so that a typo does not silently create a new tree.
40. As a note creator, I want filename collisions to block creation, so that Notes Web never overwrites an existing note.
41. As a note creator, I want new note content to come from the nearest `_template.md` found by walking up from the target folder to the vault root, so that folder-local templates guide local workflows.
42. As a note creator, I want creation to fall back to a blank note if no template exists, so that the feature still works without template setup.
43. As a note creator, I want template variables to support a small explicit set, so that templates can include useful context without becoming a programming language.
44. As a note creator, I want creation Preview to use the current target path as render context, so that relative links behave like they will after creation.
45. As a folder maintainer, I want a path ending in `/` to create an empty folder, so that folder creation stays lightweight and keyboard-friendly.
46. As a folder maintainer, I want newly created folder names preserved rather than slugified, so that folder naming remains under my control.
47. As a folder maintainer, I want only empty folders to be renamed or moved to trash, so that Notes Web does not perform risky recursive mutations in the first version.
48. As a folder maintainer, I want non-empty folder actions to be disabled with the reason “cannot delete non-empty folders”, so that the limitation is clear.
49. As a note maintainer, I want Rename to offer a title-driven slug path and a raw path override, so that common renames are easy while advanced moves remain precise.
50. As a note maintainer, I want rename collisions to block the operation, so that an existing target is never overwritten.
51. As a note maintainer, I want Rename to show an impact preview before it writes anything, so that I can verify the scope of link changes.
52. As a note maintainer, I want Rename to rewrite exact wikilinks to the renamed note, so that internal navigation remains intact.
53. As a note maintainer, I want Rename to preserve wikilink aliases, so that user-facing link text stays unchanged.
54. As a note maintainer, I want Rename to preserve wikilink heading anchors, so that heading-level links still point to the intended section.
55. As a note maintainer, I want Rename to rewrite exact Markdown links to the renamed note, so that standard Markdown links remain intact.
56. As a note maintainer, I want rewritten Markdown links to be relative from the source note, so that links remain portable inside the vault.
57. As a note maintainer, I want free text never rewritten during Rename, so that ordinary prose is not corrupted.
58. As a note maintainer, I want rename impact grouped by visible notes, addressable hidden notes, and non-touched items, so that I can understand exactly what will change.
59. As a note maintainer, I want addressable hidden paths included in the impact preview, so that hidden does not hide write scope from me.
60. As a note maintainer, I want multi-file rename operations to preflight all conflicts before writing, so that the operation does not start if it cannot finish safely.
61. As a note maintainer, I want multi-file operations to attempt best-effort rollback if an unexpected write failure occurs mid-operation, so that partial states are minimized.
62. As a reader of broken links, I want a missing wikilink page to offer “Create this note”, so that I can repair a missing target in context.
63. As a reader of broken links, I want creating from a missing wikilink to slugify the new note filename and rewrite exact wikilinks to the new target, so that the original link stops being broken.
64. As a reader of broken links, I want missing-link creation to block if the source note is unknown, so that Notes Web does not create a note that leaves the link broken.
65. As a reader of broken links, I want missing-link creation to show an impact preview, so that I know which exact wikilinks will be rewritten.
66. As a reader of broken links, I want missing-link creation to rewrite all exact matching wikilinks, so that repeated references to the same missing target are repaired together.
67. As a vault maintainer, I want Move to Trash instead of permanent Delete, so that removing a note from the normal vault view is recoverable.
68. As a vault maintainer, I want the UI to say “Move to Trash” rather than “Delete”, so that the action accurately describes what will happen.
69. As a vault maintainer, I want trash entries stored as dated snapshots containing the original path, so that repeated removals do not collide and restores know where to go.
70. As a vault maintainer, I want empty folders moved to trash with the same model as notes, so that removal behavior is consistent.
71. As a vault maintainer, I want `_trash` hidden from normal folder browsing, search, command palette, backlinks, Dataview, and diagnostics, so that removed material does not clutter normal work.
72. As a vault maintainer, I want a dedicated Trash view, so that I can see removed snapshots intentionally.
73. As a vault maintainer, I want Trash to support Restore only in the first version, so that Notes Web does not introduce permanent purge behavior yet.
74. As a vault maintainer, I want direct CRUD access to `_trash` blocked, so that the Trash view remains the single safe restoration surface.
75. As a vault maintainer, I want restore collisions to block the original restore, so that restore never overwrites an existing file or folder.
76. As a vault maintainer, I want restore collisions to offer Restore As, so that I can recover the item under a different path when the original path is occupied.
77. As a template user, I want `_template.md` files hidden from normal lists, search, palette, backlinks, Dataview, and diagnostics, so that templates do not behave like ordinary notes.
78. As a template user, I want `_template.md` to be editable by direct URL, so that I can maintain a template in Notes Web when I intentionally open it.
79. As a template user, I want `_template.md` edits to show a Template badge, so that I know I am editing a creation template rather than a normal note.
80. As a hidden-note user, I want configured hidden paths to mean non-enumerated rather than confidential, so that I can keep them out of normal surfaces while still opening them by URL.
81. As a hidden-note user, I want direct URL access to configured hidden paths to allow CRUD, so that I can maintain non-enumerated notes when I intentionally navigate to them.
82. As a hidden-note user, I want dot-prefixed paths to remain blocked, so that sensitive implementation folders are not opened or mutated by the edit mode.
83. As a hidden-note user, I want configured hidden notes excluded from search, palette, backlinks, Dataview, and diagnostics, so that hidden keeps its non-enumerated meaning.
84. As a hidden-note user, I want hidden notes to show a Hidden badge in reading and editing contexts, so that I remember they are outside normal enumeration.
85. As a hidden-note user, I want moving a note between visible and hidden areas to require explicit confirmation, so that I understand the enumeration behavior is changing.
86. As a hidden-note user, I want that confirmation to state that hidden notes remain accessible by direct URL, so that I do not mistake hidden for private.
87. As a command palette user, I want Edit current, New here, Rename current, Move to Trash, and Open Trash available from the palette, so that edit actions stay keyboard-native.
88. As a command palette user, I want unavailable actions shown disabled with a short reason, so that I understand why an action cannot run in the current context.
89. As a command palette user, I want destructive actions reachable but clearly labeled, so that keyboard flow does not hide safety-critical behavior.
90. As a security-conscious vault owner, I want all write paths to pass the same root-boundary checks as read paths, so that traversal cannot escape the vault.
91. As a security-conscious vault owner, I want symlink writes blocked, so that a symlink cannot redirect a write outside the vault.
92. As a security-conscious vault owner, I want app-generated labels, errors, URLs, and paths escaped, so that malformed vault content cannot become app-generated HTML injection.
93. As a security-conscious vault owner, I want write endpoints to reject unsupported methods and malformed JSON, so that the API fails closed.
94. As a security-conscious vault owner, I want write endpoints to require editing to be enabled, so that API URLs do not mutate the vault when the feature is off.
95. As a vault owner, I want the vault index invalidated after successful writes, so that reading, search, dashboards, links, and Dataview reflect saved changes promptly.
96. As a vault owner, I want writes to preserve existing file permissions, so that editing through the browser does not unexpectedly loosen or change permissions.
97. As a vault owner, I want file writes to be atomic, so that an interrupted save is less likely to leave a partial file.
98. As a vault owner, I want no automatic versioning on normal saves in the first version, so that the feature stays simpler and does not create hidden storage growth.
99. As an open-source maintainer, I want this feature implemented without runtime frontend frameworks or npm runtime dependencies, so that Notes Web remains a small self-contained Go web server.
100. As an open-source maintainer, I want server-rendered templates and the Go renderer to remain the source of truth for HTML, so that JavaScript enhances the app rather than replacing the architecture.
101. As an open-source maintainer, I want the changed hidden-path semantics documented and reviewed before code lands, so that future agents do not apply the older hidden-path rule accidentally.
102. As an open-source maintainer, I want the edit mode to follow the existing warm paper, fresh ink, restrained violet, calm density, and visible-focus design system, so that editing feels native to Notes Web.

## Implementation Decisions

- Editing is a first-version CRUD feature focused on Markdown notes and empty folders.
- Editing is opt-in through vault configuration under an `editing` namespace.
- The validated configuration shape is:
  - `enabled: true` to activate editing.
  - `trash_path: _trash` for the internal trash location.
  - `template_name: _template.md` for nearest-parent template lookup.
  - `hide_templates: true` to keep templates out of normal enumeration.
  - `slug: kebab_lowercase` for generated note filenames.
- Configuration in the vault is the only activation source. There is no second CLI, Nix, or environment flag in the current product decision.
- Basic Auth remains optional. If editing is enabled without auth, the instance owner accepts that anyone with network access and a valid URL can mutate permitted vault paths.
- CSRF protection is mandatory for every write action and is not optional configuration.
- The edit UI is inline. It transforms the note surface into an edit workbench without switching to a dedicated edit URL.
- Desktop editing keeps the surrounding app context. Mobile editing uses a full-screen shell with a way back to context.
- Editing requires JavaScript. Reading remains server-rendered.
- The source editor is a vanilla textarea, not CodeMirror, Monaco, or another runtime frontend dependency.
- Frontmatter is edited as raw Markdown/YAML in the textarea.
- Save writes the textarea content exactly, with no Markdown/YAML formatting or automatic normalization.
- Source is fetched on demand when the user starts editing, not embedded into every read-mode page.
- The edit workbench has Source and Preview tabs everywhere.
- Preview is manual, not live.
- Preview renders the current unsaved textarea content.
- Preview uses the same Markdown renderer as normal reading.
- If the source changes after Preview renders, the UI marks Preview stale in both the tab and the Preview action.
- Save returns to rendered read mode.
- Cancel, Escape, internal navigation, refresh, and tab close protect unsaved changes.
- Save conflicts are detected by comparing the edit baseline with disk state before writing.
- A save conflict keeps the draft, offers Copy Draft, and allows Reload Disk only after confirmation.
- New note defaults to the current folder, but the target path is editable.
- New note title generates a kebab-case lowercase filename with transliterated accents and `.md` extension.
- If the path field is edited manually, title-to-slug synchronization stops.
- Folder path segments are preserved; only the note filename generated from a title is slugified.
- A target path ending in `/` creates a folder rather than a note.
- Folder creation preserves the typed folder name.
- Missing intermediate folders can be created only after visible confirmation.
- Creation and rename collisions block the operation.
- Notes with extensions other than `.md` are blocked from edit-mode create, edit, and rename flows.
- Empty folders can be created, renamed, and moved to trash. Non-empty folder mutations are disabled with a clear reason.
- New note templates are resolved by looking for the nearest `_template.md` from the target folder upward to the vault root, with a blank fallback.
- Template variables use a small explicit set rather than a template language with conditionals or loops.
- `_template.md` files are hidden from normal folder browsing, search, palette, backlinks, Dataview, and diagnostics.
- `_template.md` files remain editable by direct URL and show a Template badge when opened.
- Rename supports a title-driven slug path and a raw path override.
- Rename rewrites exact wikilinks and exact Markdown links that resolve to the renamed note.
- Wikilink rewrites preserve aliases and heading anchors.
- Markdown link rewrites produce paths relative to the source note.
- Rewrite never changes free text.
- Rename impact preview groups visible notes, addressable hidden notes, and non-touched items.
- Missing wikilink creation creates a slugified note and rewrites all exact wikilinks to the new target.
- Missing wikilink creation blocks if the source note is unknown.
- Missing wikilink creation always shows an impact preview.
- Multi-file operations preflight all conflicts before writing anything.
- Multi-file operations write atomically per file and attempt best-effort rollback if an unexpected mid-operation failure occurs.
- Move to Trash is the deletion model for notes and empty folders.
- The UI uses “Move to Trash” rather than “Delete” for the primary action.
- Trash stores dated snapshots under `_trash` with the original path preserved inside the snapshot structure.
- Trash is hidden from normal vault enumeration and is accessed through a dedicated Trash view.
- The first version supports Restore only from Trash. Permanent purge and empty-trash behavior are out of scope.
- Direct CRUD access to `_trash` is blocked even if the user knows the URL.
- Restore collisions block the original restore and offer Restore As.
- Hidden-path semantics change for configured hidden paths: configured hidden means non-enumerated, not confidential.
- Configured hidden paths are accessible by direct URL and allow CRUD when editing is enabled.
- Dot-prefixed path segments remain blocked and are not made addressable by the hidden-path semantic change.
- Configured hidden paths remain excluded from normal search, command palette, backlinks, Dataview, diagnostics, and folder enumeration.
- Reading or editing a configured hidden path shows a Hidden badge.
- Moving a note between visible and configured-hidden areas requires confirmation that explicitly says the hidden item remains accessible by direct URL.
- Link rewrite scans visible notes and configured hidden addressable notes, but excludes dot-prefixed paths, `_trash`, and `_template.md`.
- Command palette additions are scoped to current context plus Trash: Edit current, New here, Rename current, Move to Trash, and Open Trash.
- Command palette actions that are not available in the current context are displayed disabled with a short reason rather than silently hidden.
- Write APIs use JSON endpoints under an edit API namespace.
- HTTP methods are semantic: reads use safe methods, creation uses create semantics, save uses update semantics, rename/restore use partial-change semantics, and trash uses delete semantics that move to trash rather than permanently purge.
- Preview may return rendered HTML generated by the server renderer while other edit actions return JSON state and errors.
- All vault mutation routes must use centralized path resolution, root-boundary checks, CSRF validation, editing-enabled checks, extension checks, and hidden/dotpath policy checks.
- Writes through symlinks are blocked.
- Saves are atomic and preserve existing file permissions.
- Successful writes invalidate cached vault index state.
- App-generated output in edit UI, API errors, rendered fragments, paths, labels, and URLs must be escaped.
- The existing design system remains binding: warm paper, fresh ink, restrained violet, flat tonal layering, calm density, visible focus, keyboard-first interactions, no glassmorphism, no gradients, no generic SaaS chrome, and no pure black or pure white in new UI surfaces.
- The existing rule that hidden paths are always blocked must be updated or superseded before implementation, because this PRD intentionally changes configured hidden-path semantics.

## Testing Decisions

- Tests should validate external behavior and observable safety guarantees rather than private helper implementation details.
- The highest-value primary seam is the browser-visible edit workflow through Playwright against a fixture vault. This seam should cover the user-visible contract: entering edit mode, editing source, previewing, stale state, saving, creating, renaming with impact preview, moving to trash, restoring, command palette actions, dirty navigation protection, mobile shell behavior, hidden badges, and disabled folder actions.
- Browser tests must use the fixture vault, not the real Hermes vault.
- Because this feature adds write access and multi-file mutations, additional Go tests are warranted at the HTTP/vault behavior seam for cases that are expensive or brittle to exhaustively verify through Playwright.
- Go tests should cover configuration parsing and defaults for the editing namespace.
- Go tests should cover path policy: root traversal rejection, configured hidden direct access, dot-prefixed path blocking, `_trash` direct CRUD blocking, `_template.md` direct edit allowance, Markdown-only constraints, and symlink write blocking.
- Go tests should cover write behavior: atomic save, permission preservation, conflict detection, index invalidation trigger, and error responses.
- Go tests should cover trash behavior: dated snapshot destination, original-path preservation, empty-folder trashing, restore, restore collision, and Restore As.
- Go tests should cover template resolution: nearest-parent lookup, root fallback, blank fallback, hidden template enumeration exclusion, and variable substitution.
- Go tests should cover slugging: kebab-case lowercase, accent transliteration, `.md` application, path freeze after manual edit, folder segment preservation, and extension blocking.
- Go tests should cover link rewrite behavior: exact wikilinks, aliases, heading anchors, exact Markdown links, relative Markdown link output, no free-text rewrites, hidden addressable scan inclusion, and dotpath/trash/template exclusion.
- Go tests should cover missing-wikilink creation: source-known requirement, impact preview, slugified creation, exact multi-source rewrite, and all-or-nothing preflight failure.
- Go tests should cover CSRF enforcement for every mutation endpoint.
- Prior art in the codebase includes Go tests for config, app routing, Markdown rendering, hidden path behavior, Dataview AJAX action validation, search/dashboard behavior, and UI contract checks.
- Prior art in the browser suite includes Playwright coverage for Dataview filters, Markdown rendering, internal pages, and Dataview gallery. The edit-mode E2E tests should follow the same fixture-driven style and assert user-visible behavior.
- For browser-facing changes, validation should include `make test-go`, `make lint`, targeted Playwright edit-mode specs, and `git diff --check`. Full `make test` should run before claiming the implementation complete when feasible.

## Out of Scope

- Permanent purge from Trash.
- Empty Trash.
- Recursive folder rename, move, or delete.
- Editing non-Markdown text files.
- WYSIWYG editing.
- CodeMirror, Monaco, runtime frontend frameworks, frontend bundling, or runtime npm dependencies.
- Live preview on every keystroke.
- Automatic Markdown or YAML formatting.
- Automatic version history or per-save backups.
- Force Save over conflicts.
- Diff/merge UI for conflicting saves.
- Editing dot-prefixed paths such as implementation or application-private directories.
- Direct CRUD access to `_trash` outside the dedicated Trash view.
- Including templates, configured hidden paths, or trash snapshots in normal search, command palette, backlinks, Dataview, diagnostics, or folder enumeration.
- Treating configured hidden paths as confidential/private access control.
- Adding a dedicated Templates management view.
- Adding auth requirements beyond the existing optional Basic Auth model.
- Publishing this PRD to GitHub issues or applying issue labels.

## Further Notes

- This PRD was synthesized from the completed grill-me session and the repo exploration/security/UX reviews. It intentionally does not ask new interview questions.
- The user explicitly requested local Markdown output and no GitHub issue creation.
- The hidden-path decision is the highest-risk product and architecture change. Implementation should start by updating the documented security model and centralizing a path policy that distinguishes dot-prefixed blocked paths from configured hidden non-enumerated paths.
- The implementation should be planned as a high-risk multi-file feature with specialist security and UI review gates before code lands.
- UI copy should stay direct and calm. Prefer “Move to Trash”, “Hidden — accessible by direct URL”, “Preview stale”, “Cannot delete non-empty folders”, and short disabled reasons in the command palette.
- Because templates, CSS, and JavaScript are embedded assets, any implementation that touches edit UI assets must rebuild and restart the Go binary before live verification.
