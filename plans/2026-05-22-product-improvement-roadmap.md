# Notes Web Product Improvement Roadmap

**Goal:** Turn Notes Web from a functional Markdown/Obsidian vault viewer into a highly polished, fast, pleasant daily-use reading and navigation tool.

**Current baseline:** The first version works well: Markdown rendering, wikilinks, backlinks, search, sidebar, Basic Auth, mobile drawer, copy links, copy task IDs, private deployment over Tailscale/LAN.

---

## 1. Better Markdown / Obsidian-style rendering

### Ideas

- Improve Obsidian callouts:
  - `> [!note]`
  - `> [!warning]`
  - `> [!tip]`
  - `> [!danger]`
  - `> [!info]`
  - `> [!quote]`
- Add icons and colors per callout type.
- Support collapsible callouts when possible.
- Improve task list rendering:
  - cleaner checkbox alignment;
  - better spacing;
  - subtle styling for completed items;
  - better visual treatment for dates, tags, recurrence markers, and task IDs.
- Improve wikilinks:
  - support `[[note#heading|alias]]` cleanly;
  - style missing links differently;
  - style ambiguous links differently;
  - add optional hover previews later.
- Support Obsidian embeds:
  - `![[image.png]]`;
  - `![[other-note.md]]` as embedded note preview;
  - possibly support PDFs/images inline when safe.
- Improve Mermaid integration:
  - theme matching light/dark mode;
  - nicer error fallback;
  - option to disable external CDN.
- Improve MathJax integration:
  - better inline/block rendering;
  - option to disable if unused.

### Priority

High for callouts, task lists, wikilinks. Medium for embeds, Mermaid, MathJax.

---

## 2. Reading comfort and typography

### Ideas

- Keep the overall app layout full-width, but limit long prose blocks to a comfortable reading width, around `70–85ch`.
- Let wide elements use more space:
  - tables;
  - code blocks;
  - Mermaid diagrams;
  - images;
  - TODO lists.
- Improve heading hierarchy:
  - better vertical rhythm;
  - less cramped sections;
  - consistent heading sizes.
- Add a “reading focus” mode:
  - hide sidebar;
  - center content;
  - keep copy/search controls accessible.
- Add font-size controls:
  - small;
  - normal;
  - large;
  - persisted in `localStorage`.
- Add themes:
  - light;
  - dark;
  - sepia;
  - auto via `prefers-color-scheme`.

### Priority

High. This directly affects daily comfort.

---

## 3. Faster navigation

### Ideas

- Add a command palette:
  - shortcut `/` or `Cmd+K`;
  - search notes by title/path;
  - open recent notes;
  - jump to favorites;
  - copy current link;
  - open backlinks.
- Improve search:
  - instant search UI;
  - highlighted snippets;
  - better ranking;
  - boost exact title/path matches;
  - boost favorites and recently viewed notes.
- Add recently viewed notes.
- Preserve scroll position per note during back/forward navigation.
- Improve breadcrumbs:
  - each path segment clickable;
  - dropdown for sibling files/folders.
- Add previous/next navigation inside a folder.

### Priority

High for command palette and recent notes. Medium for scroll restoration and breadcrumb dropdowns.

---

## 4. Sidebar improvements

### Ideas

- Highlight the current note in the sidebar.
- Auto-open the folder path for the current note.
- Keep manually opened folders persisted, but always reveal the active file.
- Add a sidebar search/filter.
- Make sidebar resizable on desktop.
- Add virtual sections:
  - Favorites;
  - Recent;
  - Daily;
  - TODOs;
  - Tags.
- Improve mobile drawer:
  - close button inside the drawer;
  - swipe-to-open from left edge;
  - swipe-to-close;
  - smoother animation;
  - better safe-area handling on iPhone.

### Priority

High for active note highlighting and auto-open current path. Medium for resize/search/swipe.

---

## 5. Better home page / dashboard

### Ideas

- Turn the home page into a private dashboard:
  - latest daily note;
  - today’s note;
  - open TODOs;
  - overdue TODOs;
  - recently modified notes;
  - recently viewed notes;
  - favorites;
  - broken links count;
  - orphan notes count.
- Make sections configurable from `.notes-web.yaml`.
- Add quick actions:
  - open TODO;
  - open latest daily;
  - open search;
  - copy local service URL.

### Priority

Medium-high. Useful once navigation basics are polished.

---

## 6. TODO-specific experience

### Ideas

- Add a dedicated TODO view:
  - all tasks from all `TODO.md` files or selected files;
  - grouped by status;
  - overdue;
  - today;
  - upcoming;
  - no due date;
  - done/archive.
- Parse visible task metadata:
  - due date;
  - done date;
  - recurrence markers;
  - tags;
  - task ID from `<!-- tid:... -->`.
- Add visual badges:
  - due date;
  - recurrence;
  - tags;
  - task ID.
- Keep the current copy-task-ID behavior.
- Optional future write mode:
  - mark done;
  - postpone;
  - edit task;
  - only if explicitly wanted, because it changes the app from read-only to write-capable.

### Priority

Medium-high because TODO.md is clearly central to daily use.

---

## 7. Backlinks, forward links, and graph features

### Ideas

- Improve backlinks:
  - show surrounding context;
  - group by folder;
  - sort by recently modified;
  - collapse/expand section.
- Add forward links:
  - list all outgoing wikilinks and markdown links.
- Add broken links page:
  - all unresolved wikilinks;
  - source notes;
  - link text.
- Add orphan notes page:
  - notes with no backlinks.
- Add local graph around current note:
  - current note;
  - backlinks;
  - forward links;
  - maybe depth 2.
- Avoid a huge global graph unless it proves useful.

### Priority

Medium. Backlink context is likely more valuable than a global graph.

---

## 8. Tags and metadata

### Ideas

- Add a `/tags` page.
- Add pages for individual tags:
  - `/tags/fp`;
  - `/tags/veille`;
  - `/tags/admin`.
- Make tags clickable wherever rendered.
- Improve frontmatter display:
  - hide empty fields;
  - show structured fields as badges;
  - render dates nicely;
  - support aliases;
  - support `type`, `status`, `tags`.
- Add a toggle to hide/show frontmatter by default.
- Add config for frontmatter fields to display prominently.

### Priority

Medium.

---

## 9. Search improvements

### Ideas

- Add search modes:
  - content;
  - title;
  - path;
  - tag;
  - frontmatter.
- Add query syntax:
  - `tag:fp`;
  - `path:Areas`;
  - `type:note`;
  - `status:active`;
  - quoted exact phrases.
- Add filters in the UI:
  - folder;
  - tag;
  - type;
  - modified date.
- Add highlighted snippets.
- Add keyboard-first search navigation.

### Priority

Medium-high. Search is one of the core app loops.

---

## 10. Performance and indexing

### Ideas

- Build an in-memory index at startup:
  - file paths;
  - titles;
  - headings;
  - frontmatter;
  - tags;
  - wikilinks;
  - outgoing markdown links;
  - backlinks.
- Add optional filesystem watcher:
  - update index when files change;
  - avoid full rescans.
- Cache rendered HTML by file mtime.
- Precompute backlinks instead of scanning on each request.
- Lazy-load or virtualize the sidebar for very large vaults.
- Add simple internal metrics/logging:
  - render time;
  - search time;
  - index build time;
  - number of notes indexed.

### Priority

Medium now, high if the vault grows or search/backlinks become slow.

---

## 11. Mobile and PWA polish

### Ideas

- Add a web app manifest:
  - name;
  - short name;
  - icon;
  - theme color;
  - standalone display mode.
- Make the app installable on iPhone/Android home screen.
- Add iPhone safe-area support:
  - `env(safe-area-inset-top)`;
  - `env(safe-area-inset-bottom)`.
- Improve mobile tap targets.
- Add mobile-specific typography and spacing.
- Add offline shell caching later:
  - app frame;
  - CSS/JS;
  - recent notes if useful.

### Priority

Medium-high if the app is used often from phone.

---

## 12. Configuration and deployment

### Ideas

- Expand `.notes-web.yaml`:

```yaml
site_title: Notes Web
favorites:
  - Areas/Daily Briefings
  - Areas/TODO.md
daily_glob: Areas/Daily Briefings/*-briefing.md
hidden:
  - .git
  - .obsidian
  - node_modules
theme: auto
features:
  mermaid: true
  mathjax: true
  backlinks: true
  task_ids: true
```

- Add `/healthz`.
- Add better structured logs.
- Add example `systemd --user` service.
- Add example Tailscale/LAN deployment section.
- Add option to bind Unix socket if reverse-proxied locally.
- Document security model clearly:
  - intended for trusted private network;
  - Basic Auth available;
  - raw HTML trusted local content;
  - not hardened for direct public exposure.

### Priority

Medium.

---

## 13. UI polish details

### Ideas

- Consistent icon set.
- Better empty states:
  - no backlinks;
  - no search results;
  - missing note;
  - ambiguous note.
- Toast notifications:
  - link copied;
  - task ID copied;
  - search error;
  - sidebar action.
- Subtle transitions:
  - sidebar drawer;
  - collapsible sections;
  - hover previews.
- Better code block design:
  - copy code button;
  - language label;
  - horizontal scroll only inside code block.
- Better table design:
  - sticky header maybe;
  - horizontal scroll inside table wrapper only.

### Priority

Medium. Lots of small wins.

---

## 14. Suggested roadmap

### Phase 1 — Immediate comfort wins

1. Improve typography and content spacing.
2. Add light/dark/auto themes.
3. Improve callout rendering.
4. Improve task list rendering.
5. Highlight current note in sidebar.
6. Auto-open current note path in sidebar.

### Phase 2 — Navigation power

1. Add command palette.
2. Add recently viewed notes.
3. Improve search UI and result snippets.
4. Add clickable tag pages.
5. Add better backlinks with context.
6. Add forward links.

### Phase 3 — Daily workflow

1. Build dedicated TODO dashboard.
2. Add daily note navigation.
3. Improve home dashboard.
4. Add configurable dashboard sections.
5. Add broken links and orphan notes pages.

### Phase 4 — Mobile/PWA

1. Add PWA manifest.
2. Add safe-area iPhone CSS.
3. Polish mobile drawer.
4. Add swipe gestures.
5. Add installable app docs.

### Phase 5 — Performance and architecture

1. Add in-memory index.
2. Add filesystem watcher.
3. Cache rendered Markdown.
4. Precompute backlinks.
5. Add metrics/logging.

---

## 15. Recommended next implementation order

If the goal is maximum perceived improvement with limited work, start with:

1. **Typography and note layout polish**
2. **Dark/light/auto theme**
3. **Current note highlight + auto-open sidebar path**
4. **Better task list rendering**
5. **Better callouts**
6. **Command palette**
7. **Recently viewed notes**
8. **Backlinks with context**
9. **TODO dashboard**
10. **PWA/mobile polish**

This sequence improves daily comfort first, then navigation speed, then deeper knowledge-management features.
