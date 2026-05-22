package app

const templates = `
{{define "layout-start"}}
<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · Notes Web</title><link rel="stylesheet" href="/_static/style.css"><script defer src="/_static/app.js"></script><script type="module">import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs'; mermaid.initialize({startOnLoad:true,theme:'neutral'});</script><script defer src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"></script></head><body><button class="mobile-menu btn icon-btn" data-sidebar-toggle aria-label="Open sidebar" aria-expanded="false">☰</button><button class="palette-button btn" data-palette-open aria-label="Open command palette">⌘K</button><div class="palette" data-palette hidden><div class="palette-panel" role="dialog" aria-label="Command palette"><input data-palette-input aria-label="Command palette" placeholder="Search notes, tags, and favorites…"><div class="palette-results" data-palette-results role="listbox"></div><div class="palette-shortcuts" aria-hidden="true"><span>↑↓ Navigate</span><span>Enter Open</span><span>Esc Close</span></div></div></div><div class="sidebar-backdrop" data-sidebar-close></div><div class="shell"><aside class="side"><div class="side-header"><a class="brand" href="/">Notes Web</a><span class="side-subtitle">Personal knowledge base</span></div><form class="search sidebar-search" action="/_search"><input name="q" placeholder="Search…" value="{{.Q}}"><span class="kbd">/</span></form><section><h3>Favorites</h3>{{range .Favorites}}<a class="nav" href="{{index . "URL"}}"><span class="nav-icon">★</span>{{index . "Label"}}</a>{{end}}</section><section><h3>Explore</h3><a class="nav" href="/_todo"><span class="nav-icon">☑</span>TODOs</a><a class="nav" href="/_tags"><span class="nav-icon">#</span>Tags</a><a class="nav" href="/_broken-links"><span class="nav-icon">⛓</span>Broken links</a><a class="nav" href="/_orphans"><span class="nav-icon">◌</span>Orphans</a></section><section><h3>Vault</h3>{{template "tree" .Tree}}</section><div class="sidebar-footer"><button class="settings-button btn ghost" data-settings-open aria-haspopup="dialog"><span aria-hidden="true">⚙</span><span>Settings</span></button></div></aside><div class="settings-modal" data-settings-modal hidden><div class="settings-backdrop" data-settings-close></div><section class="settings-dialog" role="dialog" aria-modal="true" aria-labelledby="settings-title"><header><div><p class="eyebrow">Preferences</p><h2 id="settings-title">Settings</h2></div><button class="btn ghost icon-btn" data-settings-close aria-label="Close settings">×</button></header><section class="settings-section"><h3>Appearance</h3><label class="setting-row"><span><strong>Theme</strong><small>Choose the color palette for the app chrome.</small></span><select data-theme-select aria-label="Theme"><option value="auto">Auto</option><option value="light">Light</option><option value="dark">Dark</option><option value="sepia">Sepia</option></select></label><label class="setting-row"><span><strong>Font size</strong><small>Adjust note reading size across pages.</small></span><select data-font-size-select aria-label="Font size"><option value="normal">Normal</option><option value="small">Small</option><option value="large">Large</option></select></label></section><section class="settings-section"><h3>Keyboard shortcuts</h3><dl class="shortcut-list"><div><dt>⌘/Ctrl K</dt><dd>Open command palette</dd></div><div><dt>/</dt><dd>Open command palette when not typing</dd></div><div><dt>⌘/Ctrl F</dt><dd>Toggle reading focus</dd></div><div><dt>Esc</dt><dd>Close dialogs</dd></div></dl></section></section></div><main class="main">
{{end}}
{{define "layout-end"}}</main></div></body></html>{{end}}
{{define "tree"}}<ul class="tree">{{range .}}<li>{{if .IsDir}}<details class="tree-folder{{if .ContainsActive}} active-branch{{end}}" data-tree-path="{{.Rel}}"{{if .ContainsActive}} open{{end}}><summary><span aria-hidden="true">📁</span> {{.Name}}</summary>{{if .Children}}{{template "tree" .Children}}{{end}}</details>{{else}}<a{{if .IsActive}} class="active" aria-current="page"{{end}} href="{{.URL}}"><span aria-hidden="true">📄</span> {{.Name}}</a>{{end}}</li>{{end}}</ul>{{end}}
{{define "home"}}{{template "layout-start" .}}<div class="page-header"><div><p class="eyebrow">Overview</p><h1>Home</h1></div><button class="copy-link btn ghost" data-copy-link>Copy link</button></div>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<div class="cards"><section class="card"><h2>Latest daily note</h2>{{with .Dashboard.LatestDaily}}<a class="biglink" href="{{url .RelPath}}">{{.RelPath}}</a>{{else}}<p class="empty-state">No daily note found.</p>{{end}}</section><section class="card"><h2>Open TODOs</h2><a class="biglink" href="/_todo">Open TODO dashboard</a>{{if .Dashboard.OpenTasks}}<ul class="compact-list">{{range .Dashboard.OpenTasks}}<li><a href="{{.SourceURL}}">{{.Text}}</a>{{if .Due}} <small>Due {{.Due}}</small>{{end}}</li>{{end}}</ul>{{else}}<p class="empty-state">No open tasks.</p>{{end}}</section><section class="card metric"><h2>Broken links</h2><a class="metric-link" href="/_broken-links"><strong>{{.Dashboard.BrokenLinkCount}}</strong><span>Review diagnostics</span></a></section><section class="card metric"><h2>Orphan notes</h2><a class="metric-link" href="/_orphans"><strong>{{.Dashboard.OrphanNoteCount}}</strong><span>Review notes</span></a></section><section class="card"><h2>Favorites</h2>{{range .Favorites}}<a class="pill chip" href="{{index . "URL"}}">{{index . "Label"}}</a>{{end}}</section></div><h2>Recent notes</h2><ul class="list note-card-list">{{range .Recent}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a><small>{{.ModTime.Format "2006-01-02 15:04"}}</small></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "note"}}{{template "layout-start" .}}<nav class="crumb reading-surface" aria-label="Breadcrumb">{{range $i, $crumb := .Breadcrumbs}}{{if $i}}<span class="crumb-separator" aria-hidden="true">/</span>{{end}}<a href="{{$crumb.URL}}"{{if $crumb.Current}} aria-current="page"{{end}}>{{$crumb.Label}}</a>{{end}}</nav><article class="note reading-surface"><header><div><h1>{{.Doc.Title}}</h1>{{if .Doc.Tags}}<div class="tag-row">{{range .Doc.Tags}}<a class="tag-badge chip" href="/_tags/{{.}}">#{{.}}</a>{{end}}</div>{{end}}</div><div class="note-actions"><button class="copy-link btn ghost" data-copy-link>Copy link</button></div></header>{{if .Doc.Toc}}<details class="toc" open><summary>Table of contents</summary><ul>{{range .Doc.Toc}}<li class="lvl{{.Level}}"><a href="#{{.ID}}">{{.Text}}</a></li>{{end}}</ul></details>{{end}}<div class="content">{{safe .Doc.HTML}}</div></article><section class="link-panel forward-links"><h2>Forward links <span class="count">{{len .ForwardLinks}}</span></h2>{{if .ForwardLinks}}<ul class="link-context-list">{{range .ForwardLinks}}<li>{{if eq .Kind "unique"}}<a href="{{.URL}}">{{.Display}}</a>{{else}}<span class="missing-link">{{.Display}}</span>{{end}} <small>{{.Kind}}</small></li>{{end}}</ul>{{else}}<p class="empty-state">No forward links.</p>{{end}}</section><section class="link-panel backlinks"><h2>Backlinks <span class="count">{{len .Backlinks}}</span></h2>{{if .Backlinks}}<ul class="link-context-list">{{range .Backlinks}}<li><a href="{{url .Source.RelPath}}">{{.Source.RelPath}}</a><blockquote>{{.Context}}</blockquote></li>{{end}}</ul>{{else}}<p class="empty-state">No backlinks.</p>{{end}}</section>{{template "layout-end" .}}{{end}}
{{define "folder"}}{{template "layout-start" .}}<nav class="crumb reading-surface" aria-label="Breadcrumb">{{range $i, $crumb := .Breadcrumbs}}{{if $i}}<span class="crumb-separator" aria-hidden="true">/</span>{{end}}<a href="{{$crumb.URL}}"{{if $crumb.Current}} aria-current="page"{{end}}>{{$crumb.Label}}</a>{{end}}</nav><article class="folder-view reading-surface"><header><div><p class="eyebrow">Folder</p><h1>{{.FolderName}}</h1><small>{{.Path}}</small></div><div class="note-actions"><button class="copy-link btn ghost" data-copy-link>Copy link</button></div></header><ul class="list folder-list">{{range .Items}}<li><a href="{{index . "URL"}}">{{if index . "Dir"}}📁{{else}}📄{{end}} {{index . "Name"}}</a></li>{{else}}<li class="empty-state">This folder is empty.</li>{{end}}</ul></article>{{template "layout-end" .}}{{end}}
{{define "search"}}{{template "layout-start" .}}<div class="page-header"><div><p class="eyebrow">Find anything</p><h1>Search</h1></div></div><form class="search big search-page-form" action="/_search"><input name="q" value="{{.Q}}" placeholder="Search notes, tags, paths, or exact phrases…" autofocus><button class="btn primary">Search</button></form><details class="search-help"><summary>Search syntax</summary><p class="muted">Examples: <code>tag:daily</code>, <code>path:&quot;Areas/Daily Briefings&quot;</code>, <code>title:Target</code>, <code>frontmatter:title=&quot;Daily Briefing&quot;</code>, <code>&quot;exact phrase&quot;</code>.</p><ul><li><code>tag:daily</code> filters by tag.</li><li><code>path:&quot;Areas/Daily Briefings&quot;</code> filters by path.</li><li><code>title:Target</code> filters by title.</li><li><code>frontmatter:title=&quot;Daily Briefing&quot;</code> filters frontmatter.</li><li><code>&quot;exact phrase&quot;</code> searches an exact phrase.</li></ul></details>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}{{if .Results}}<ul class="results rich-results">{{range .Results}}<li class="search-result-card"><a class="result-title" href="{{.URL}}">{{.Title}}</a><div class="result-path">{{.RelPath}}{{if .LineNo}} · line {{.LineNo}}{{end}}</div><p class="result-snippet">{{safe .SnippetHTML}}</p></li>{{end}}</ul>{{else}}{{if .Q}}<div class="empty-state"><h2>No results for <code>{{.Q}}</code></h2><p>Try a broader search, remove a filter, or open Search syntax for examples.</p></div>{{end}}{{end}}{{template "layout-end" .}}{{end}}
{{define "broken-links"}}{{template "layout-start" .}}<div class="page-header"><div><p class="eyebrow">Diagnostics</p><h1>Broken links</h1></div></div>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<p class="muted">Unresolved wikilinks found in the vault.</p><div class="diagnostic-summary"><div><strong>{{.BrokenTotal}}</strong><span>occurrences</span></div><div><strong>{{.BrokenDistinctTargets}}</strong><span>distinct targets</span></div><div><strong>{{.BrokenAffectedNotes}}</strong><span>affected notes</span></div></div><div class="filter-bar"><input data-list-filter placeholder="Filter target or source…"><span class="muted">Show top {{.BrokenTopLimit}} groups by occurrence count.</span></div>{{if .BrokenGroups}}<div class="diagnostic-groups">{{range .BrokenGroups}}<details class="broken-link-group"{{if .Open}} open{{end}}><summary><strong>{{.Target}}</strong><span class="count">{{.Total}} occurrences</span></summary><ul class="link-context-list">{{range .Items}}<li><a href="{{.Source.URL}}">{{.Source.Title}}</a><small>{{.Source.RelPath}}:{{.LineNo}}</small><blockquote>{{.Context}}</blockquote></li>{{end}}</ul></details>{{end}}</div>{{else}}<p class="empty-state">No broken links.</p>{{end}}{{template "layout-end" .}}{{end}}
{{define "orphans"}}{{template "layout-start" .}}<div class="page-header"><div><p class="eyebrow">Diagnostics</p><h1>Orphan notes</h1></div></div>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<p class="muted">Notes with no incoming wikilinks.</p><div class="diagnostic-summary"><div><strong>{{.OrphanTotal}}</strong><span>orphan notes</span></div></div><div class="filter-bar"><input data-list-filter placeholder="Filter path or title…"></div>{{if .Orphans}}<ul class="note-card-grid">{{range .Orphans}}<li class="note-card"><a href="{{.URL}}">{{.Title}}</a><small>{{.RelPath}}</small></li>{{end}}</ul>{{else}}<p class="empty-state">No orphan notes.</p>{{end}}{{template "layout-end" .}}{{end}}
{{define "task-list"}}<ul class="todo-list">{{range .}}<li class="task-card{{if .Completed}} completed{{end}}"><span class="task-checkbox" aria-label="{{.StatusLabel}}">{{if .Completed}}✓{{end}}</span><div class="task-main"><a class="task-title" href="{{.SourceURL}}">{{.Text}}</a><div class="task-meta-row">{{if .Due}}<span class="task-date {{.DateClass}}">Due {{.Due}}</span>{{end}}{{if .Done}}<span class="task-date done-date">Done {{.Done}}</span>{{end}}{{if .ID}}<button class="task-id" data-copy="{{.ID}}" title="Copy task ID">tid:{{.ID}}</button>{{end}}</div></div></li>{{else}}<li class="empty-state">None.</li>{{end}}</ul>{{end}}
{{define "todo"}}{{template "layout-start" .}}<div class="page-header"><div><p class="eyebrow">Task dashboard</p><h1>TODOs</h1></div></div>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<p class="muted">Grouped for {{.Today}} from TODO.md files.</p><div class="todo-board"><section class="todo-column overdue"><h2>Overdue <span class="count">{{len .Board.Overdue}}</span></h2>{{template "task-list" .Board.Overdue}}</section><section class="todo-column today"><h2>Today <span class="count">{{len .Board.Today}}</span></h2>{{template "task-list" .Board.Today}}</section><section class="todo-column upcoming"><h2>Upcoming <span class="count">{{len .Board.Upcoming}}</span></h2>{{template "task-list" .Board.Upcoming}}</section></div><div class="todo-secondary"><details class="todo-column no-date"><summary><h2>No due date <span class="count">{{len .Board.NoDate}}</span></h2></summary>{{template "task-list" .Board.NoDate}}</details><details class="todo-column done"><summary><h2>Done <span class="count">{{len .Board.Done}}</span></h2></summary>{{template "task-list" .Board.Done}}</details></div>{{template "layout-end" .}}{{end}}
{{define "tags"}}{{template "layout-start" .}}<div class="page-header"><div><p class="eyebrow">Explore</p><h1>Tags</h1></div></div>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<div class="tag-stats"><div><strong>{{.TagTotal}}</strong><span>total tags</span></div><div><strong>{{.OneOffTagCount}}</strong><span>one-off tags</span></div></div><div class="tag-controls"><input data-tag-filter placeholder="Filter tags…"><button class="btn ghost" data-hide-rare>Hide one-off tags</button></div><section><h2>Popular tags</h2><div class="tag-cloud popular-tags">{{range .PopularTags}}<a class="tag-badge chip" data-tag-chip href="{{.URL}}">#{{.Tag}} <small>{{.Count}}</small></a>{{else}}<p class="empty-state">No tags found.</p>{{end}}</div></section><section><h2>Alphabetical index</h2>{{range .TagGroups}}<section class="tag-letter"><h3>{{.Letter}}</h3><div class="tag-cloud">{{range .Tags}}<a class="tag-badge chip" data-tag-chip href="{{.URL}}">#{{.Tag}} <small>{{.Count}}</small></a>{{end}}</div></section>{{end}}</section><details class="rare-tags"><summary>Rare tags <span class="count">{{.OneOffTagCount}}</span></summary><div class="tag-cloud">{{range .RareTags}}<a class="tag-badge chip muted-chip" data-tag-chip href="{{.URL}}">#{{.Tag}} <small>{{.Count}}</small></a>{{end}}</div></details>{{template "layout-end" .}}{{end}}
{{define "tag"}}{{template "layout-start" .}}<h1>#{{.Tag}}</h1>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<ul class="list">{{range .Notes}}<li><a href="{{.URL}}">{{.Title}}</a><small>{{.RelPath}}</small></li>{{else}}<p class="empty-state">No notes for this tag.</p>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "resolve"}}{{template "layout-start" .}}<h1>Choose a note</h1><p>Multiple notes match <code>{{.Name}}</code>.</p><ul class="list">{{range .Matches}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "missing"}}{{template "layout-start" .}}<h1>Note not found</h1><p>No note matches <code>{{.Name}}</code>.</p>{{template "layout-end" .}}{{end}}
`

const css = `
:root{color-scheme:light;--bg:#f7f5ef;--surface:#fffdfa;--surface-raised:#ffffff;--panel:var(--surface);--ink:#27231d;--muted:#786f63;--line:#e2dccc;--accent:#6b5cff;--accent-ink:#3d32c2;--danger:#b54747;--warning:#b76e00;--success:#2b8a3e;--soft:#efecff;--code:#eee9dc;--pre:#f1eee6;--measure:1180px;--space-1:4px;--space-2:8px;--space-3:12px;--space-4:16px;--space-5:24px;--space-6:32px;--radius-sm:8px;--radius-md:12px;--radius-lg:18px;--radius-pill:999px;--shadow-sm:0 2px 10px #00000014;--shadow-modal:0 24px 80px #0005}[data-theme=dark]{color-scheme:dark;--bg:#11131a;--surface:#181b24;--surface-raised:#1f2430;--panel:var(--surface);--ink:#ebe7df;--muted:#a9a197;--line:#303545;--accent:#9b8cff;--accent-ink:#d4ceff;--danger:#ff8787;--warning:#ffc078;--success:#69db7c;--soft:#272340;--code:#242838;--pre:#171b26}[data-theme=sepia]{color-scheme:light;--bg:#f4ecd8;--surface:#fff8e8;--surface-raised:#fffaf0;--panel:var(--surface);--ink:#33291c;--muted:#7b6b55;--line:#dfd0ae;--accent:#8a5a12;--accent-ink:#5f3b09;--soft:#f1dfb8;--code:#efe1c5;--pre:#eadcbd}@media(prefers-color-scheme:dark){:root:not([data-theme]){color-scheme:dark;--bg:#11131a;--surface:#181b24;--surface-raised:#1f2430;--panel:var(--surface);--ink:#ebe7df;--muted:#a9a197;--line:#303545;--accent:#9b8cff;--accent-ink:#d4ceff;--soft:#272340;--code:#242838;--pre:#171b26}}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:16px/1.6 ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif}a{color:var(--accent-ink);text-underline-offset:3px}a:hover{text-decoration-thickness:2px}:focus-visible{outline:3px solid color-mix(in srgb,var(--accent),transparent 45%);outline-offset:3px}.btn,button{border:1px solid var(--line);background:var(--surface-raised);color:var(--ink);border-radius:var(--radius-md);padding:8px 12px;cursor:pointer;box-shadow:0 1px 0 #00000008}.btn.primary{background:var(--accent);border-color:var(--accent);color:white}.btn.ghost{background:transparent}.icon-btn{display:inline-flex;align-items:center;justify-content:center}.chip{display:inline-flex;align-items:center;gap:6px;padding:4px 10px;border-radius:var(--radius-pill);background:var(--soft);color:var(--accent-ink);text-decoration:none;font-size:14px}.muted-chip{opacity:.72}.empty-state{padding:18px;border:1px dashed var(--line);border-radius:var(--radius-lg);background:color-mix(in srgb,var(--surface),transparent 30%);color:var(--muted)}.mobile-menu{display:none}.palette-button{position:fixed;right:18px;bottom:18px;z-index:15;border-radius:999px;box-shadow:0 6px 24px #0002}.palette[hidden]{display:none}.palette{position:fixed;inset:0;z-index:60;background:#0007;padding:10vh 18px}.palette-panel{max-width:760px;margin:0 auto;background:var(--surface-raised);border:1px solid var(--line);border-radius:22px;box-shadow:var(--shadow-modal);overflow:hidden}.palette input{width:100%;border:0;border-bottom:1px solid var(--line);background:var(--surface-raised);color:var(--ink);font-size:20px;padding:18px}.palette-results{max-height:55vh;overflow:auto;padding-bottom:12px}.palette-item{display:grid;grid-template-columns:1fr auto;gap:4px 12px;width:100%;text-align:left;border:0;border-radius:0;border-bottom:1px solid var(--line);padding:13px 16px;background:transparent}.palette-item small{grid-column:1;color:var(--muted)}.palette-kind{grid-column:2;grid-row:1 / span 2;align-self:center}.palette-item.is-selected,.palette-item:hover{background:var(--soft)}.palette-shortcuts{display:flex;gap:14px;justify-content:flex-end;padding:10px 14px;color:var(--muted);font-size:12px;border-top:1px solid var(--line)}.sidebar-backdrop{display:none}.shell{display:grid;grid-template-columns:300px minmax(0,1fr);min-height:100vh}.side{position:sticky;top:0;height:100vh;overflow:auto;padding:22px;background:var(--surface);border-right:1px solid var(--line)}.side-header{margin-bottom:18px}.brand{display:block;font-weight:800;font-size:22px;text-decoration:none;color:var(--ink)}.side-subtitle{display:block;color:var(--muted);font-size:13px}.main{min-width:0;width:100%;padding:40px 56px}.search{display:flex;gap:8px;margin:12px 0 24px}.search input,.filter-bar input,.tag-controls input{width:100%;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:var(--surface-raised);color:var(--ink)}.sidebar-search{position:relative}.sidebar-search .kbd{position:absolute;right:10px;top:50%;transform:translateY(-50%);font-size:12px;color:var(--muted);border:1px solid var(--line);border-radius:6px;padding:0 5px}.theme-picker{display:flex;align-items:center;justify-content:space-between;gap:10px;margin:0 0 24px;color:var(--muted);font-size:14px}.theme-picker select,.reading-controls select{border:1px solid var(--line);border-radius:999px;background:var(--surface-raised);color:var(--ink);padding:6px 10px}.search.big input{font-size:18px}.page-header{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:24px}.eyebrow{text-transform:uppercase;letter-spacing:.08em;color:var(--muted);font-size:12px;font-weight:800;margin:0 0 4px}button.copied,.task-id.copied{border-color:var(--success);color:var(--success);background:#ebfbee}h1,h2,h3{line-height:1.2}h1{font-size:40px;margin:0 0 12px}h2{margin-top:32px}.nav,.tree a,.list a,.results a{text-decoration:none}.nav{display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:var(--radius-md);color:var(--ink)}.nav:hover,.nav[aria-current=page]{background:var(--soft);color:var(--accent-ink)}.nav-icon{width:20px;text-align:center}.tree{list-style:none;padding-left:12px;margin:4px 0}.tree ul{border-left:1px solid var(--line);margin-left:8px}.tree li{margin:4px 0}.tree summary{cursor:pointer;user-select:none;border-radius:8px;padding:2px 4px}.tree summary:hover,.tree a:hover{background:var(--soft)}.tree a.active,.tree-folder.active-branch>summary{background:var(--soft);color:var(--accent);font-weight:700}.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:16px}.card{background:var(--surface-raised);border:1px solid var(--line);border-radius:18px;padding:20px;box-shadow:var(--shadow-sm)}.biglink{font-size:18px;color:var(--accent-ink)}.compact-list{margin:10px 0 0;padding-left:18px}.compact-list li{margin:6px 0}.metric strong{display:block;font-size:38px;line-height:1;color:var(--accent)}.metric-link{text-decoration:none}.metric-link:hover strong{text-decoration:underline}.count{display:inline-flex;align-items:center;justify-content:center;min-width:1.8em;padding:1px 8px;border-radius:999px;background:var(--soft);color:var(--accent-ink);font-size:.78em}.todo-section{margin:24px 0;padding:18px;border:1px solid var(--line);border-radius:18px;background:var(--panel)}.todo-group{margin:18px 0;padding:18px;border:1px solid var(--line);border-radius:18px;background:var(--surface-raised);box-shadow:var(--shadow-sm)}details.todo-group summary{cursor:pointer;list-style:none}details.todo-group summary::-webkit-details-marker{display:none}.todo-list{list-style:none;margin:0;padding:0}.todo-list li{display:flex;gap:10px;align-items:center;justify-content:space-between;padding:8px 0;border-top:1px solid var(--line)}.todo-list li:first-child{border-top:0}.task-row{display:grid!important;grid-template-columns:auto minmax(0,1fr) auto;gap:10px;align-items:center}.task-title{overflow-wrap:anywhere;color:var(--ink)}.task-meta-row{display:inline-flex;gap:8px;align-items:center;justify-content:flex-end;flex-wrap:wrap}.task-status{color:var(--muted)}.task-date{display:inline-flex;align-items:center;white-space:nowrap;border:1px solid var(--line);border-radius:999px;padding:1px 8px;font-size:12px;color:var(--muted);background:var(--surface)}.overdue-date{color:var(--danger);border-color:color-mix(in srgb,var(--danger),var(--line) 65%)}.today-date{color:var(--accent-ink)}.upcoming-date{color:var(--muted)}.muted{color:var(--muted)}.link-context-list{list-style:none;margin:0;padding:0}.link-context-list li{padding:10px 0;border-top:1px solid var(--line)}.link-context-list li:first-child{border-top:0}.link-context-list blockquote{margin:8px 0 0;padding:8px 12px;border-left:3px solid var(--line);color:var(--muted);background:var(--surface)}.missing-link{color:var(--danger);text-decoration:line-through;text-decoration-thickness:1px}.pill{display:inline-block;padding:7px 10px;margin:4px;border-radius:999px;background:var(--soft);color:var(--accent);text-decoration:none}.tag-row{display:flex;flex-wrap:wrap;gap:8px;margin:8px 0 18px}.tag-cloud{display:flex;flex-wrap:wrap;gap:10px}.tag-badge small{display:inline;color:var(--muted)}.tag-stats,.diagnostic-summary{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px;margin:18px 0}.tag-stats>div,.diagnostic-summary>div{background:var(--surface-raised);border:1px solid var(--line);border-radius:var(--radius-lg);padding:16px}.tag-stats strong,.diagnostic-summary strong{display:block;font-size:30px;color:var(--accent)}.tag-stats span,.diagnostic-summary span{color:var(--muted)}.tag-controls,.filter-bar{display:flex;gap:12px;align-items:center;margin:18px 0}.tag-letter{margin-top:18px}.rare-tags,.broken-link-group,.toc,.frontmatter,.link-panel{background:var(--surface-raised);border:1px solid var(--line);border-radius:14px;padding:12px 16px;margin:18px 0}.broken-link-group summary{cursor:pointer;display:flex;justify-content:space-between;gap:12px}.note-card-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:12px;list-style:none;padding:0}.note-card{border:1px solid var(--line);background:var(--surface-raised);border-radius:var(--radius-lg);padding:14px}.crumb{color:var(--muted);margin-bottom:16px}.crumb.reading-surface{display:flex;align-items:center;gap:8px;flex-wrap:wrap}.crumb a[aria-current="page"]{font-weight:650}.crumb-separator{color:var(--muted)}.note header{display:flex;align-items:start;justify-content:space-between;gap:20px;min-width:0}.folder-view header{display:flex;align-items:start;justify-content:space-between;gap:20px;min-width:0}.folder-view h1{margin:0 0 8px}.folder-list{margin-top:32px}.folder-list li{display:flex;align-items:center;min-height:58px}.folder-list a{display:block;width:100%;font-size:20px}.content{font-size:17px;line-height:1.72;overflow-wrap:anywhere}.content a{overflow-wrap:anywhere}.content>:where(p,ul,ol,blockquote,details,dl){max-width:var(--measure)}.content>:where(pre,table,.mermaid,img){max-width:100%}.content p{margin:0 0 1.05rem}.content h2{margin:2.2rem 0 1rem}.content img{max-width:100%}.content pre{overflow:auto;max-width:100%;padding:14px;border-radius:12px;background:var(--pre)}.content code{background:var(--code);border-radius:5px;padding:1px 4px}.content table{display:block;overflow-x:auto;max-width:100%;border-collapse:collapse;width:100%;margin:16px 0}.content th,.content td{border:1px solid var(--line);padding:8px 10px}.frontmatter dl{display:grid;grid-template-columns:150px 1fr;gap:6px}.frontmatter dt{font-weight:700}.callout{border-left:4px solid var(--accent);background:var(--soft);padding:12px 14px;border-radius:12px;margin:16px 0}.callout-title{display:flex;align-items:center;gap:8px;font-weight:700}.callout-icon{flex:0 0 auto}.callout-body{margin-top:8px}.callout-body p{margin:0}.callout-note{border-color:var(--accent)}.callout-info{border-color:#228be6}.callout-tip{border-color:#2b8a3e}.callout-warning{border-color:#f08c00}.callout-danger,.callout-error{border-color:#e03131}.callout-success{border-color:#2b8a3e}.callout-question{border-color:#7048e8}.callout-bug{border-color:#c2255c}.task-id{display:inline-flex;align-items:center;gap:4px;margin-left:8px;padding:1px 7px;border:1px solid var(--line);border-radius:999px;background:var(--panel);color:var(--muted);font:12px ui-monospace,SFMono-Regular,Menlo,monospace;vertical-align:middle}.contains-task-list{padding-left:0;list-style:none}.task-list-item{display:flex;align-items:baseline;gap:10px;margin:8px 0}.task-list-item input[type=checkbox]{flex:0 0 auto;transform:translateY(1px)}.task-meta{display:inline-flex;align-items:center;gap:4px;margin-left:6px;padding:1px 7px;border:1px solid var(--line);border-radius:999px;background:var(--soft);color:var(--muted);font-size:12px;white-space:nowrap}.due-date{color:var(--accent)}.done-date{color:var(--success)}.task-id:hover{color:var(--accent);border-color:var(--accent)}.lvl2{margin-left:12px}.lvl3{margin-left:24px}.list,.results{padding-left:0;list-style:none}.list li,.results li{padding:10px 0;border-bottom:1px solid var(--line)}small{display:block;color:var(--muted)}.error{color:#b00020}.search-help{margin:12px 0 22px;padding:12px 14px;border:1px solid var(--line);border-radius:12px;background:var(--surface-raised)}.search-help summary{cursor:pointer;font-weight:700}.search-help ul{margin:10px 0 0}.search-help code{white-space:nowrap}.search-result-card{background:var(--surface-raised);border:1px solid var(--line)!important;border-radius:var(--radius-lg);padding:16px!important;margin:12px 0;box-shadow:var(--shadow-sm)}.result-title{display:block;font-weight:800;color:var(--ink);font-size:18px}.result-path{color:var(--muted);font-size:13px;margin-top:2px}.result-snippet{margin:10px 0 0}.reading-surface{max-width:var(--measure);margin-inline:auto}.reading-controls{display:flex;gap:10px;flex-wrap:wrap;align-items:center}.reading-controls label{display:inline-flex;gap:6px;align-items:center;color:var(--muted);font-size:13px}.note .content>p,.note .content>ul,.note .content>ol,.note .content>blockquote,.note .content>details,.note .content>h1,.note .content>h2,.note .content>h3,.note .content>h4,.note .content>h5,.note .content>h6{width:min(100%,var(--measure));margin-left:auto;margin-right:auto}.note .content table,.note .content pre,.note .content .mermaid,.note .content .contains-task-list{max-width:none;width:100%;margin-left:0;margin-right:0}.note .content img{max-width:100%;height:auto}body.reading-focus .side{display:none}body.reading-focus .shell{grid-template-columns:minmax(0,1fr)}body.reading-focus .main{padding-inline:max(24px,8vw)}[data-font-size="small"] body{font-size:15px}[data-font-size="large"] body{font-size:18px}@media(max-width:850px){.shell{display:block}.mobile-menu{display:inline-flex;position:fixed;top:12px;left:12px;z-index:40;align-items:center;justify-content:center;width:42px;height:42px;padding:0;border-radius:999px;box-shadow:0 2px 10px #0002}.side{position:fixed;left:0;top:0;width:min(86vw,340px);max-width:340px;transform:translateX(-100%);z-index:30;height:100vh;transition:transform .18s ease;box-shadow:0 12px 35px #0003}.main{padding:64px 18px 28px}.cards{grid-template-columns:1fr}.sidebar-backdrop{position:fixed;inset:0;background:#0006;z-index:20}body.sidebar-open{overflow:hidden}body.sidebar-open .side{transform:translateX(0)}body.sidebar-open .sidebar-backdrop{display:block}.task-row{grid-template-columns:auto minmax(0,1fr)}.task-meta-row{grid-column:2;justify-content:flex-start}.tag-controls,.filter-bar,.page-header{display:block}}

.side{padding-bottom:96px}.sidebar-footer{position:sticky;bottom:0;margin-top:auto;padding:14px 0 0;background:linear-gradient(180deg,transparent,var(--surface) 28%);border-top:1px solid var(--line)}.settings-button{width:100%;display:flex;align-items:center;justify-content:space-between;gap:10px;padding:11px 12px}.settings-modal[hidden]{display:none}.settings-modal{position:fixed;inset:0;z-index:60;display:grid;place-items:center;padding:24px}.settings-backdrop{position:absolute;inset:0;background:#0008;backdrop-filter:blur(3px)}.settings-dialog{position:relative;width:min(620px,calc(100vw - 32px));max-height:min(760px,calc(100vh - 32px));overflow:auto;background:var(--surface-raised);border:1px solid var(--line);border-radius:24px;box-shadow:var(--shadow-modal);padding:24px}.settings-dialog header{display:flex;align-items:start;justify-content:space-between;gap:16px;margin-bottom:20px}.settings-dialog h2{margin:0}.settings-section{border-top:1px solid var(--line);padding-top:18px;margin-top:18px}.settings-section h3{margin:0 0 12px}.setting-row{display:grid;grid-template-columns:1fr auto;align-items:center;gap:18px;padding:14px 0}.setting-row small{display:block;color:var(--muted);margin-top:2px}.shortcut-list{display:grid;gap:8px;margin:0}.shortcut-list div{display:grid;grid-template-columns:120px 1fr;gap:14px;align-items:center}.shortcut-list dt{font-weight:800;background:var(--soft);border:1px solid var(--line);border-radius:var(--radius-sm);padding:6px 8px;text-align:center}.shortcut-list dd{margin:0;color:var(--muted)}.note-actions{display:flex;gap:10px}.frontmatter{width:min(100%,var(--measure));max-width:var(--measure);margin-left:auto;margin-right:auto}.todo-board{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:18px;align-items:start;margin-top:24px}.todo-secondary{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px;align-items:start;margin-top:18px}.todo-column{background:linear-gradient(180deg,var(--surface-raised),color-mix(in srgb,var(--surface),var(--soft) 16%));border:1px solid var(--line);border-radius:22px;padding:18px;box-shadow:var(--shadow-sm)}.todo-column>h2,.todo-column summary h2{margin:0;font-size:18px;display:flex;align-items:center;justify-content:space-between;gap:10px}.todo-column summary{cursor:pointer}.todo-column.overdue{border-color:color-mix(in srgb,var(--danger),var(--line) 55%)}.todo-column.today{border-color:color-mix(in srgb,var(--success),var(--line) 60%)}.todo-column.upcoming{border-color:color-mix(in srgb,var(--accent),var(--line) 62%)}.todo-list{list-style:none;margin:14px 0 0;padding:0;display:grid;gap:10px}.task-card{display:grid;grid-template-columns:32px 1fr;gap:14px;align-items:start;background:var(--surface);border:1px solid color-mix(in srgb,var(--line),transparent 20%);border-radius:16px;padding:12px 13px;box-shadow:0 1px 0 #00000008}.task-card:hover{border-color:color-mix(in srgb,var(--accent),var(--line) 55%);box-shadow:var(--shadow-sm)}.task-card.completed{opacity:.72}.task-checkbox{width:22px;height:22px;margin-top:2px;border:1.5px solid var(--line);border-radius:6px;display:grid;place-items:center;color:var(--success);background:var(--surface-raised);font-size:13px;font-weight:900}.task-main{min-width:0}.task-title{display:block;color:var(--ink);font-weight:650;text-decoration:none;line-height:1.42;overflow-wrap:anywhere}.task-title:hover{color:var(--accent-ink);text-decoration:underline}.task-meta-row{display:flex;flex-wrap:wrap;align-items:center;gap:7px;margin-top:8px}.task-date,.task-id,.task-meta{font-size:12px;line-height:1;border-radius:999px;padding:5px 8px;background:var(--soft);border:1px solid var(--line);color:var(--muted)}.task-id{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.overdue-date{color:var(--danger);background:color-mix(in srgb,var(--danger),transparent 88%)}.today-date{color:var(--success);background:color-mix(in srgb,var(--success),transparent 88%)}.done-date{color:var(--success)}.content .contains-task-list{padding-left:0;list-style:none;display:grid;gap:12px;max-width:var(--measure);margin-left:auto;margin-right:auto}.content .task-list-item{display:grid;grid-template-columns:32px 1fr;gap:12px;align-items:start;background:var(--surface-raised);border:1px solid var(--line);border-radius:18px;padding:13px 14px;box-shadow:var(--shadow-sm);line-height:1.5}.content .task-list-item input[type=checkbox]{appearance:none;width:22px;height:22px;margin:2px 0 0;border:1.5px solid var(--line);border-radius:6px;background:var(--surface)}.content .task-list-item input[type=checkbox]:checked{background:var(--success);border-color:var(--success);box-shadow:inset 0 0 0 4px var(--surface-raised)}@media(max-width:1100px){.todo-board,.todo-secondary{grid-template-columns:1fr}}.side{padding-bottom:92px}.side>section:last-of-type{padding-bottom:88px;margin-bottom:74px}.sidebar-footer{position:absolute;left:22px;right:22px;bottom:22px;margin:0;padding:12px 0 0;background:var(--surface);border-top:1px solid var(--line);z-index:2}
.side{display:flex;flex-direction:column;overflow:hidden;padding-bottom:92px}.side>section:last-of-type{flex:1;min-height:0;overflow:auto;padding-bottom:88px;margin-bottom:74px}.sidebar-footer{position:absolute;left:22px;right:22px;bottom:22px;margin:0;padding:12px 0 0;background:var(--surface);border-top:1px solid var(--line);box-shadow:0 -10px 24px color-mix(in srgb,var(--surface),transparent 45%);z-index:2}.task-checkbox{inline-size:22px;min-inline-size:22px;width:22px;block-size:22px;min-block-size:22px;display:inline-grid;place-items:center;justify-self:center}.content .task-list-item input[type=checkbox]{inline-size:22px;min-inline-size:22px;width:22px;block-size:22px;min-block-size:22px}.palette-button{top:18px;right:18px;bottom:auto}.reading-focus .palette-button{display:none}
`

const js = `
const paletteShortcutClass = 'palette-shortcuts';
const sidebarStorageKey = 'notes-web:sidebar-open';
const themeStorageKey = 'notes-web:theme';
const fontSizeStorageKey = 'notes-web:font-size';
const readingFocusStorageKey = 'notes-web:reading-focus';
function applyTheme(theme) {
  if (!theme || theme === 'auto') document.documentElement.removeAttribute('data-theme');
  else document.documentElement.dataset.theme = theme;
}
function initThemePicker() {
  const select = document.querySelector('[data-theme-select]');
  const saved = localStorage.getItem(themeStorageKey) || 'auto';
  applyTheme(saved);
  if (select) select.value = saved;
  select?.addEventListener('change', () => {
    localStorage.setItem(themeStorageKey, select.value);
    applyTheme(select.value);
  });
}

let paletteItems = [];
let paletteMatches = [];
let paletteSelectedIndex = 0;
function openPalette() {
  const palette = document.querySelector('[data-palette]');
  const input = document.querySelector('[data-palette-input]');
  if (!palette || !input) return;
  palette.hidden = false;
  if (!paletteItems.length) fetch('/_api/palette').then(r => r.json()).then(items => { paletteItems = items; renderPalette(input.value); });
  renderPalette(input.value);
  setTimeout(() => input.focus(), 0);
}
function closePalette() {
  const palette = document.querySelector('[data-palette]');
  if (palette) palette.hidden = true;
}
function renderPalette(query) {
  const results = document.querySelector('[data-palette-results]');
  if (!results) return;
  const q = (query || '').toLowerCase();
  paletteMatches = paletteItems.filter(item => !q || item.title.toLowerCase().includes(q) || (item.path || '').toLowerCase().includes(q)).slice(0, 30);
  if (paletteSelectedIndex >= paletteMatches.length) paletteSelectedIndex = 0;
  results.innerHTML = paletteMatches.map((item, index) => '<button role="option" aria-selected="' + (index === paletteSelectedIndex) + '" class="palette-item' + (index === paletteSelectedIndex ? ' is-selected' : '') + '" data-palette-index="' + index + '"><strong>' + escapeHTML(item.title) + '</strong><small>' + escapeHTML(item.path || '') + '</small><span class="chip palette-kind">' + escapeHTML(item.kind) + '</span></button>').join('') || '<p class="palette-empty empty-state">No results.</p>';
  results.querySelectorAll('[data-palette-index]').forEach((button) => {
    button.addEventListener('mouseenter', () => { paletteSelectedIndex = Number(button.dataset.paletteIndex); renderPalette(document.querySelector('[data-palette-input]')?.value || ''); });
    button.addEventListener('click', () => openPaletteMatch(Number(button.dataset.paletteIndex)));
  });
}
function openPaletteMatch(index) {
  const item = paletteMatches[index];
  if (item) location.href = item.url;
}
function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}
function initCommandPalette() {
  document.querySelector('[data-palette-open]')?.addEventListener('click', openPalette);
  document.querySelector('[data-palette]')?.addEventListener('click', (ev) => { if (ev.target.matches('[data-palette]')) closePalette(); });
  document.querySelector('[data-palette-input]')?.addEventListener('input', (ev) => { paletteSelectedIndex = 0; renderPalette(ev.target.value); });
  document.querySelector('[data-palette-input]')?.addEventListener('keydown', (ev) => {
    if (ev.key === 'ArrowDown') { ev.preventDefault(); paletteSelectedIndex = Math.min(paletteSelectedIndex + 1, Math.max(0, paletteMatches.length - 1)); renderPalette(ev.target.value); }
    else if (ev.key === 'ArrowUp') { ev.preventDefault(); paletteSelectedIndex = Math.max(0, paletteSelectedIndex - 1); renderPalette(ev.target.value); }
    else if (ev.key === 'Enter') { ev.preventDefault(); openPaletteMatch(paletteSelectedIndex); }
  });
  document.addEventListener('keydown', (ev) => {
    const inField = ev.target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(ev.target.tagName);
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'k') { ev.preventDefault(); openPalette(); }
    else if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'f') { ev.preventDefault(); toggleReadingFocus(); }
    else if (ev.key === '/' && !inField) { ev.preventDefault(); openPalette(); }
    else if (ev.key === 'Escape') { closePalette(); closeSettingsModal(); }
  });
}

function applyFontSize(size) {
  const normalized = ['small', 'normal', 'large'].includes(size) ? size : 'normal';
  if (normalized === 'normal') document.documentElement.removeAttribute('data-font-size');
  else document.documentElement.dataset.fontSize = normalized;
}
function applyReadingFocus(enabled) {
  document.body.classList.toggle('reading-focus', Boolean(enabled));
}
function toggleReadingFocus() {
  const enabled = !document.body.classList.contains('reading-focus');
  localStorage.setItem(readingFocusStorageKey, String(enabled));
  applyReadingFocus(enabled);
}
function initReadingControls() {
  const fontSelect = document.querySelector('[data-font-size-select]');
  const savedFont = localStorage.getItem(fontSizeStorageKey) || 'normal';
  applyFontSize(savedFont);
  if (fontSelect) fontSelect.value = savedFont;
  fontSelect?.addEventListener('change', () => {
    localStorage.setItem(fontSizeStorageKey, fontSelect.value);
    applyFontSize(fontSelect.value);
  });
  applyReadingFocus(localStorage.getItem(readingFocusStorageKey) === 'true');
}
function openSettingsModal() {
  const modal = document.querySelector('[data-settings-modal]');
  if (!modal) return;
  modal.hidden = false;
  setTimeout(() => modal.querySelector('select,button')?.focus(), 0);
}
function closeSettingsModal() {
  const modal = document.querySelector('[data-settings-modal]');
  if (modal) modal.hidden = true;
}
function initSettingsModal() {
  document.querySelectorAll('[data-settings-open]').forEach((el) => el.addEventListener('click', openSettingsModal));
  document.querySelectorAll('[data-settings-close]').forEach((el) => el.addEventListener('click', closeSettingsModal));
}

function readSidebarState() {
  try { return new Set(JSON.parse(localStorage.getItem(sidebarStorageKey) || '[]')); }
  catch { return new Set(); }
}
function writeSidebarState(openPaths) {
  try { localStorage.setItem(sidebarStorageKey, JSON.stringify([...openPaths].sort())); }
  catch {}
}
function restoreSidebarState() {
  const openPaths = readSidebarState();
  document.querySelectorAll('details.tree-folder[data-tree-path]').forEach((details) => {
    const path = details.dataset.treePath;
    if (!details.classList.contains('active-branch')) details.open = openPaths.has(path);
    details.addEventListener('toggle', () => {
      const current = readSidebarState();
      if (details.open) current.add(path); else current.delete(path);
      writeSidebarState(current);
    });
  });
}

function openSidebar() {
  document.body.classList.add('sidebar-open');
  document.querySelector('[data-sidebar-toggle]')?.setAttribute('aria-expanded', 'true');
}
function closeSidebar() {
  document.body.classList.remove('sidebar-open');
  document.querySelector('[data-sidebar-toggle]')?.setAttribute('aria-expanded', 'false');
}
function initMobileSidebar() {
  const toggle = document.querySelector('[data-sidebar-toggle]');
  const closeTargets = document.querySelectorAll('[data-sidebar-close]');
  const side = document.querySelector('.side');
  toggle?.addEventListener('click', () => {
    if (document.body.classList.contains('sidebar-open')) closeSidebar(); else openSidebar();
  });
  closeTargets.forEach((target) => target.addEventListener('click', closeSidebar));
  side?.addEventListener('click', (ev) => {
    const target = ev.target;
    if (target.closest('a')) closeSidebar();
  });
  document.addEventListener('keydown', (ev) => {
    if (ev.key === 'Escape') closeSidebar();
  });
}

function initListFilters() {
  document.querySelectorAll('[data-list-filter]').forEach((input) => {
    input.addEventListener('input', () => {
      const q = input.value.toLowerCase();
      document.querySelectorAll('.broken-link-group,.note-card').forEach((el) => {
        el.hidden = q && !el.textContent.toLowerCase().includes(q);
      });
    });
  });
  const tagFilter = document.querySelector('[data-tag-filter]');
  tagFilter?.addEventListener('input', () => {
    const q = tagFilter.value.toLowerCase();
    document.querySelectorAll('[data-tag-chip]').forEach((el) => { el.hidden = q && !el.textContent.toLowerCase().includes(q); });
  });
  document.querySelector('[data-hide-rare]')?.addEventListener('click', () => {
    document.querySelector('.rare-tags')?.toggleAttribute('hidden');
  });
}

async function copyText(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.setAttribute('readonly', '');
  ta.style.position = 'fixed';
  ta.style.left = '-9999px';
  document.body.appendChild(ta);
  ta.select();
  document.execCommand('copy');
  ta.remove();
}
function markCopied(el, label) {
  const old = el.textContent;
  el.classList.add('copied');
  if (label) el.textContent = label;
  setTimeout(() => { el.classList.remove('copied'); if (label) el.textContent = old; }, 1200);
}
document.addEventListener('DOMContentLoaded', () => { initThemePicker(); initReadingControls(); initSettingsModal(); initCommandPalette(); restoreSidebarState(); initMobileSidebar(); initListFilters(); });
document.addEventListener('click', async (ev) => {
  const copy = ev.target.closest('[data-copy]');
  if (copy) {
    ev.preventDefault();
    await copyText(copy.dataset.copy);
    markCopied(copy, 'copied');
    return;
  }
  const link = ev.target.closest('[data-copy-link]');
  if (link) {
    ev.preventDefault();
    await copyText(location.href);
    markCopied(link, 'Link copied');
  }
});
`
