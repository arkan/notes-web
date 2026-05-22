package app

const templates = `
{{define "layout-start"}}
<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · Notes Web</title><link rel="stylesheet" href="/_static/style.css"><script defer src="/_static/app.js"></script><script type="module">import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs'; mermaid.initialize({startOnLoad:true,theme:'neutral'});</script><script defer src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"></script></head><body><button class="mobile-menu" data-sidebar-toggle aria-label="Open sidebar" aria-expanded="false">☰</button><button class="palette-button" data-palette-open>⌘K</button><div class="palette" data-palette hidden><div class="palette-panel"><input data-palette-input aria-label="Command palette" placeholder="Search notes, tags, and favorites…"><div class="palette-results" data-palette-results></div></div></div><div class="sidebar-backdrop" data-sidebar-close></div><div class="shell"><aside class="side"><a class="brand" href="/">Notes Web</a><form class="search" action="/_search"><input name="q" placeholder="Search…" value="{{.Q}}"></form><label class="theme-picker">Theme<select data-theme-select aria-label="Theme"><option value="auto">Auto</option><option value="light">Light</option><option value="dark">Dark</option><option value="sepia">Sepia</option></select></label><section><h3>Favorites</h3>{{range .Favorites}}<a class="nav" href="{{index . "URL"}}">★ {{index . "Label"}}</a>{{end}}</section><section><h3>Explore</h3><a class="nav" href="/_todo">☑ TODOs</a><a class="nav" href="/_tags"># Tags</a><a class="nav" href="/_broken-links">⛓ Broken links</a><a class="nav" href="/_orphans">◌ Orphans</a></section><section><h3>Vault</h3>{{template "tree" .Tree}}</section></aside><main class="main">
{{end}}
{{define "layout-end"}}</main></div></body></html>{{end}}
{{define "tree"}}<ul class="tree">{{range .}}<li>{{if .IsDir}}<details class="tree-folder{{if .ContainsActive}} active-branch{{end}}" data-tree-path="{{.Rel}}"{{if .ContainsActive}} open{{end}}><summary>📁 {{.Name}}</summary>{{if .Children}}{{template "tree" .Children}}{{end}}</details>{{else}}<a{{if .IsActive}} class="active" aria-current="page"{{end}} href="{{.URL}}">📄 {{.Name}}</a>{{end}}</li>{{end}}</ul>{{end}}
{{define "home"}}{{template "layout-start" .}}<h1>Home</h1><button class="copy-link" data-copy-link>Copy link</button>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<div class="cards"><section class="card"><h2>Latest daily note</h2>{{with .Dashboard.LatestDaily}}<a class="biglink" href="{{url .RelPath}}">{{.RelPath}}</a>{{else}}<p>No daily note found.</p>{{end}}</section><section class="card"><h2>Open TODOs</h2><a class="biglink" href="/_todo">Open TODO dashboard</a>{{if .Dashboard.OpenTasks}}<ul class="compact-list">{{range .Dashboard.OpenTasks}}<li><a href="{{.SourceURL}}">{{.Text}}</a>{{if .Due}} <small>📅 {{.Due}}</small>{{end}}</li>{{end}}</ul>{{else}}<p>No open tasks.</p>{{end}}</section><section class="card metric"><h2>Broken links</h2><a class="metric-link" href="/_broken-links"><strong>{{.Dashboard.BrokenLinkCount}}</strong></a></section><section class="card metric"><h2>Orphan notes</h2><a class="metric-link" href="/_orphans"><strong>{{.Dashboard.OrphanNoteCount}}</strong></a></section><section class="card"><h2>Favorites</h2>{{range .Favorites}}<a class="pill" href="{{index . "URL"}}">{{index . "Label"}}</a>{{end}}</section></div><h2>Recent notes</h2><ul class="list">{{range .Recent}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a><small>{{.ModTime.Format "2006-01-02 15:04"}}</small></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "note"}}{{template "layout-start" .}}<nav class="crumb"><a href="/">Home</a> / {{.Note.RelPath}}</nav><article class="note reading-surface"><header><h1>{{.Doc.Title}}</h1><div class="reading-controls"><label>Font size<select data-font-size-select aria-label="Font size"><option value="normal">Normal</option><option value="small">Small</option><option value="large">Large</option></select></label><button class="focus-toggle" data-focus-toggle aria-label="Toggle reading focus">Focus</button><button class="copy-link" data-copy-link>Copy link</button></div></header>{{if .Doc.Tags}}<div class="tag-row">{{range .Doc.Tags}}<a class="tag-badge" href="/_tags/{{.}}">#{{.}}</a>{{end}}</div>{{end}}{{if .Doc.Toc}}<details class="toc" open><summary>Table of contents</summary><ul>{{range .Doc.Toc}}<li class="lvl{{.Level}}"><a href="#{{.ID}}">{{.Text}}</a></li>{{end}}</ul></details>{{end}}<div class="content">{{safe .Doc.HTML}}</div></article><section class="forward-links"><h2>Forward links</h2>{{if .ForwardLinks}}<ul class="link-context-list">{{range .ForwardLinks}}<li>{{if eq .Kind "unique"}}<a href="{{.URL}}">{{.Display}}</a>{{else}}<span class="missing-link">{{.Display}}</span>{{end}} <small>{{.Kind}}</small></li>{{end}}</ul>{{else}}<p>No forward links.</p>{{end}}</section><section class="backlinks"><h2>Backlinks</h2>{{if .Backlinks}}<ul class="link-context-list">{{range .Backlinks}}<li><a href="{{url .Source.RelPath}}">{{.Source.RelPath}}</a><blockquote>{{.Context}}</blockquote></li>{{end}}</ul>{{else}}<p>No backlinks.</p>{{end}}</section>{{template "layout-end" .}}{{end}}
{{define "folder"}}{{template "layout-start" .}}<h1>📁 {{.Path}}</h1><button class="copy-link" data-copy-link>Copy link</button><ul class="list">{{range .Items}}<li><a href="{{index . "URL"}}">{{if index . "Dir"}}📁{{else}}📄{{end}} {{index . "Name"}}</a></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "search"}}{{template "layout-start" .}}<h1>Search</h1><form class="search big" action="/_search"><input name="q" value="{{.Q}}" autofocus><button>Search</button></form><details class="search-help"><summary>Search syntax</summary><ul><li><code>tag:daily</code> filters by tag.</li><li><code>path:&quot;Areas/Daily Briefings&quot;</code> filters by path.</li><li><code>title:Target</code> filters by title.</li><li><code>frontmatter:title=&quot;Daily Briefing&quot;</code> filters frontmatter.</li><li><code>&quot;exact phrase&quot;</code> searches an exact phrase.</li></ul></details>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<ul class="results">{{range .Results}}<li><a href="{{.URL}}">{{.RelPath}}:{{.Line}}</a><p>{{safe .SnippetHTML}}</p></li>{{else}}{{if .Q}}<p>No results.</p>{{end}}{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "broken-links"}}{{template "layout-start" .}}<h1>Broken links</h1>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<p class="muted">Unresolved wikilinks found in the vault.</p><ul class="link-context-list">{{range .BrokenLinks}}<li><strong>{{.Target}}</strong> in <a href="{{.Source.URL}}">{{.Source.Title}}</a><small>{{.Source.RelPath}}:{{.LineNo}}</small><blockquote>{{.Context}}</blockquote></li>{{else}}<li class="empty">No broken links.</li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "orphans"}}{{template "layout-start" .}}<h1>Orphan notes</h1>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<p class="muted">Notes with no incoming wikilinks.</p><ul class="list">{{range .Orphans}}<li><a href="{{.URL}}">{{.RelPath}}</a><small>{{.Title}}</small></li>{{else}}<li>No orphan notes.</li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "task-list"}}<ul class="todo-list">{{range .}}<li><a href="{{.SourceURL}}">{{.Text}}</a><span>{{if .Due}}📅 {{.Due}}{{end}}{{if .Done}}✅ {{.Done}}{{end}}</span>{{if .ID}}<button class="task-id" data-copy="{{.ID}}">tid:{{.ID}}</button>{{end}}</li>{{else}}<li class="empty">None.</li>{{end}}</ul>{{end}}
{{define "todo"}}{{template "layout-start" .}}<h1>TODOs</h1>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<p class="muted">Grouped for {{.Today}} from TODO.md files.</p><section class="todo-section overdue"><h2>Overdue</h2>{{template "task-list" .Board.Overdue}}</section><section class="todo-section today"><h2>Today</h2>{{template "task-list" .Board.Today}}</section><section class="todo-section upcoming"><h2>Upcoming</h2>{{template "task-list" .Board.Upcoming}}</section><section class="todo-section no-date"><h2>No due date</h2>{{template "task-list" .Board.NoDate}}</section><section class="todo-section done"><h2>Done</h2>{{template "task-list" .Board.Done}}</section>{{template "layout-end" .}}{{end}}
{{define "tags"}}{{template "layout-start" .}}<h1>Tags</h1>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<div class="tag-cloud">{{range .Tags}}<a class="tag-badge" href="{{index . "URL"}}">#{{index . "Tag"}} <small>{{index . "Count"}}</small></a>{{else}}<p>No tags found.</p>{{end}}</div>{{template "layout-end" .}}{{end}}
{{define "tag"}}{{template "layout-start" .}}<h1>#{{.Tag}}</h1>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<ul class="list">{{range .Notes}}<li><a href="{{.URL}}">{{.Title}}</a><small>{{.RelPath}}</small></li>{{else}}<p>No notes for this tag.</p>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "resolve"}}{{template "layout-start" .}}<h1>Choose a note</h1><p>Multiple notes match <code>{{.Name}}</code>.</p><ul class="list">{{range .Matches}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "missing"}}{{template "layout-start" .}}<h1>Note not found</h1><p>No note matches <code>{{.Name}}</code>.</p>{{template "layout-end" .}}{{end}}
`

const css = `
:root{color-scheme:light;--bg:#f7f5ef;--panel:#fffdfa;--ink:#27231d;--muted:#786f63;--line:#e2dccc;--accent:#6b5cff;--soft:#efecff;--code:#eee9dc;--pre:#f1eee6;--measure:78ch}[data-theme=dark]{color-scheme:dark;--bg:#11131a;--panel:#181b24;--ink:#ebe7df;--muted:#a9a197;--line:#303545;--accent:#9b8cff;--soft:#272340;--code:#242838;--pre:#171b26}[data-theme=sepia]{color-scheme:light;--bg:#f4ecd8;--panel:#fff8e8;--ink:#33291c;--muted:#7b6b55;--line:#dfd0ae;--accent:#8a5a12;--soft:#f1dfb8;--code:#efe1c5;--pre:#eadcbd}@media(prefers-color-scheme:dark){:root:not([data-theme]){color-scheme:dark;--bg:#11131a;--panel:#181b24;--ink:#ebe7df;--muted:#a9a197;--line:#303545;--accent:#9b8cff;--soft:#272340;--code:#242838;--pre:#171b26}}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:16px/1.6 ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif}.mobile-menu{display:none}.palette-button{position:fixed;right:18px;bottom:18px;z-index:15;border-radius:999px;box-shadow:0 6px 24px #0002}.palette[hidden]{display:none}.palette{position:fixed;inset:0;z-index:60;background:#0005;padding:10vh 18px}.palette-panel{max-width:720px;margin:0 auto;background:var(--panel);border:1px solid var(--line);border-radius:18px;box-shadow:0 24px 80px #0005;overflow:hidden}.palette input{width:100%;border:0;border-bottom:1px solid var(--line);background:var(--panel);color:var(--ink);font-size:20px;padding:18px}.palette-results{max-height:55vh;overflow:auto}.palette-item{display:block;width:100%;text-align:left;border:0;border-radius:0;border-bottom:1px solid var(--line);padding:12px 16px}.palette-item small{color:var(--muted)}.sidebar-backdrop{display:none}.shell{display:grid;grid-template-columns:300px minmax(0,1fr);min-height:100vh}.side{position:sticky;top:0;height:100vh;overflow:auto;padding:22px;background:var(--panel);border-right:1px solid var(--line)}.brand{display:block;font-weight:800;font-size:22px;text-decoration:none;color:var(--ink);margin-bottom:18px}.main{min-width:0;width:100%;padding:40px 56px}.search{display:flex;gap:8px;margin:12px 0 24px}.search input{width:100%;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:var(--panel);color:var(--ink)}.theme-picker{display:flex;align-items:center;justify-content:space-between;gap:10px;margin:0 0 24px;color:var(--muted);font-size:14px}.theme-picker select{border:1px solid var(--line);border-radius:999px;background:var(--panel);color:var(--ink);padding:6px 10px}.search.big input{font-size:18px}button{border:1px solid var(--line);background:var(--panel);color:var(--ink);border-radius:10px;padding:8px 12px;cursor:pointer}button.copied,.task-id.copied{border-color:#2b8a3e;color:#2b8a3e;background:#ebfbee}h1,h2,h3{line-height:1.2}h1{font-size:40px;margin:0 0 24px}h2{margin-top:32px}.nav,.tree a,.list a,.results a{color:var(--ink);text-decoration:none}.nav{display:block;padding:5px 0}.tree{list-style:none;padding-left:12px;margin:4px 0}.tree ul{border-left:1px solid var(--line);margin-left:8px}.tree li{margin:4px 0}.tree summary{cursor:pointer;user-select:none;border-radius:8px;padding:2px 4px}.tree summary:hover,.tree a:hover{background:var(--soft)}.tree a.active,.tree-folder.active-branch>summary{background:var(--soft);color:var(--accent);font-weight:700}.cards{display:grid;grid-template-columns:1fr 1fr;gap:16px}.card{background:var(--panel);border:1px solid var(--line);border-radius:18px;padding:20px}.biglink{font-size:18px;color:var(--accent)}.compact-list{margin:10px 0 0;padding-left:18px}.compact-list li{margin:6px 0}.metric strong{display:block;font-size:38px;line-height:1;color:var(--accent)}.metric-link{text-decoration:none}.metric-link:hover strong{text-decoration:underline}.todo-section{margin:24px 0;padding:18px;border:1px solid var(--line);border-radius:18px;background:var(--panel)}.todo-list{list-style:none;margin:0;padding:0}.todo-list li{display:flex;gap:10px;align-items:center;justify-content:space-between;padding:8px 0;border-top:1px solid var(--line)}.todo-list li:first-child{border-top:0}.todo-list .empty{color:var(--muted);justify-content:flex-start}.muted{color:var(--muted)}.link-context-list{list-style:none;margin:0;padding:0}.link-context-list li{padding:10px 0;border-top:1px solid var(--line)}.link-context-list li:first-child{border-top:0}.link-context-list blockquote{margin:8px 0 0;padding:8px 12px;border-left:3px solid var(--line);color:var(--muted);background:var(--panel)}.missing-link{color:#b54747;text-decoration:line-through;text-decoration-thickness:1px}.pill{display:inline-block;padding:7px 10px;margin:4px;border-radius:999px;background:var(--soft);color:var(--accent);text-decoration:none}.tag-row{display:flex;flex-wrap:wrap;gap:8px;margin:-8px 0 18px}.tag-cloud{display:flex;flex-wrap:wrap;gap:10px}.tag-badge{display:inline-flex;align-items:center;gap:6px;padding:4px 10px;border-radius:999px;background:var(--soft);color:var(--accent);text-decoration:none;font-size:14px}.tag-badge small{display:inline;color:var(--muted)}.crumb{color:var(--muted);margin-bottom:16px}.note header{display:flex;align-items:start;justify-content:space-between;gap:20px;min-width:0}.content{font-size:17px;line-height:1.72;overflow-wrap:anywhere}.content a{overflow-wrap:anywhere}.content>:where(p,ul,ol,blockquote,details,dl){max-width:var(--measure)}.content>:where(pre,table,.mermaid,img){max-width:100%}.content p{margin:0 0 1.05rem}.content h2{margin:2.2rem 0 1rem}.content img{max-width:100%}.content pre{overflow:auto;max-width:100%;padding:14px;border-radius:12px;background:var(--pre)}.content code{background:var(--code);border-radius:5px;padding:1px 4px}.content table{display:block;overflow-x:auto;max-width:100%;border-collapse:collapse;width:100%;margin:16px 0}.content th,.content td{border:1px solid var(--line);padding:8px 10px}.frontmatter,.toc,.backlinks{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:12px 16px;margin:18px 0}.frontmatter dl{display:grid;grid-template-columns:150px 1fr;gap:6px}.frontmatter dt{font-weight:700}.callout{border-left:4px solid var(--accent);background:var(--soft);padding:12px 14px;border-radius:12px;margin:16px 0}.callout-title{display:flex;align-items:center;gap:8px;font-weight:700}.callout-icon{flex:0 0 auto}.callout-body{margin-top:8px}.callout-body p{margin:0}.callout-note{border-color:var(--accent)}.callout-info{border-color:#228be6}.callout-tip{border-color:#2b8a3e}.callout-warning{border-color:#f08c00}.callout-danger,.callout-error{border-color:#e03131}.callout-success{border-color:#2b8a3e}.callout-question{border-color:#7048e8}.callout-bug{border-color:#c2255c}.task-id{display:inline-flex;align-items:center;gap:4px;margin-left:8px;padding:1px 7px;border:1px solid var(--line);border-radius:999px;background:var(--panel);color:var(--muted);font:12px ui-monospace,SFMono-Regular,Menlo,monospace;vertical-align:middle}.contains-task-list{padding-left:0;list-style:none}.task-list-item{display:flex;align-items:baseline;gap:10px;margin:8px 0}.task-list-item input[type=checkbox]{flex:0 0 auto;transform:translateY(1px)}.task-meta{display:inline-flex;align-items:center;gap:4px;margin-left:6px;padding:1px 7px;border:1px solid var(--line);border-radius:999px;background:var(--soft);color:var(--muted);font-size:12px;white-space:nowrap}.due-date{color:var(--accent)}.done-date{color:#2b8a3e}.task-id:hover{color:var(--accent);border-color:var(--accent)}.lvl2{margin-left:12px}.lvl3{margin-left:24px}.list,.results{padding-left:0;list-style:none}.list li,.results li{padding:10px 0;border-bottom:1px solid var(--line)}small{display:block;color:var(--muted)}.error{color:#b00020}.search-help{margin:12px 0 22px;padding:12px 14px;border:1px solid var(--line);border-radius:12px;background:var(--panel)}.search-help summary{cursor:pointer;font-weight:700}.search-help ul{margin:10px 0 0}.search-help code{white-space:nowrap}.reading-surface{max-width:var(--measure);margin-inline:auto}.reading-controls{display:flex;gap:10px;flex-wrap:wrap;align-items:center}.reading-controls label{display:inline-flex;gap:6px;align-items:center;color:var(--muted);font-size:13px}.note .content>p,.note .content>ul,.note .content>ol,.note .content>blockquote,.note .content>details,.note .content>h1,.note .content>h2,.note .content>h3,.note .content>h4,.note .content>h5,.note .content>h6{width:min(100%,var(--measure));margin-left:auto;margin-right:auto}.note .content table,.note .content pre,.note .content .mermaid,.note .content .contains-task-list{max-width:none;width:100%;margin-left:0;margin-right:0}.note .content img{max-width:100%;height:auto}body.reading-focus .side{display:none}body.reading-focus .shell{grid-template-columns:minmax(0,1fr)}body.reading-focus .main{padding-inline:max(24px,8vw)}[data-font-size="small"] body{font-size:15px}[data-font-size="large"] body{font-size:18px}@media(max-width:850px){.shell{display:block}.mobile-menu{display:inline-flex;position:fixed;top:12px;left:12px;z-index:40;align-items:center;justify-content:center;width:42px;height:42px;padding:0;border-radius:999px;box-shadow:0 2px 10px #0002}.side{position:fixed;left:0;top:0;width:min(86vw,340px);max-width:340px;transform:translateX(-100%);z-index:30;height:100vh;transition:transform .18s ease;box-shadow:0 12px 35px #0003}.main{padding:64px 18px 28px}.cards{grid-template-columns:1fr}.sidebar-backdrop{position:fixed;inset:0;background:#0006;z-index:20}body.sidebar-open{overflow:hidden}body.sidebar-open .side{transform:translateX(0)}body.sidebar-open .sidebar-backdrop{display:block}}
`

const js = `
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
  const matches = paletteItems.filter(item => !q || item.title.toLowerCase().includes(q) || (item.path || '').toLowerCase().includes(q)).slice(0, 30);
  results.innerHTML = matches.map((item, index) => '<button class="palette-item" data-palette-index="' + index + '"><strong>' + escapeHTML(item.title) + '</strong><small>' + escapeHTML(item.kind) + ' ' + escapeHTML(item.path || '') + '</small></button>').join('') || '<p class="palette-empty">No results.</p>';
  results.querySelectorAll('[data-palette-index]').forEach((button) => {
    button.addEventListener('click', () => {
      const item = matches[Number(button.dataset.paletteIndex)];
      if (item) location.href = item.url;
    });
  });
}
function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}
function initCommandPalette() {
  document.querySelector('[data-palette-open]')?.addEventListener('click', openPalette);
  document.querySelector('[data-palette]')?.addEventListener('click', (ev) => { if (ev.target.matches('[data-palette]')) closePalette(); });
  document.querySelector('[data-palette-input]')?.addEventListener('input', (ev) => renderPalette(ev.target.value));
  document.addEventListener('keydown', (ev) => {
    const inField = ev.target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(ev.target.tagName);
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'k') { ev.preventDefault(); openPalette(); }
    else if (ev.key === '/' && !inField) { ev.preventDefault(); openPalette(); }
    else if (ev.key === 'Escape') closePalette();
  });
}

function applyFontSize(size) {
  const normalized = ['small', 'normal', 'large'].includes(size) ? size : 'normal';
  if (normalized === 'normal') document.documentElement.removeAttribute('data-font-size');
  else document.documentElement.dataset.fontSize = normalized;
}
function applyReadingFocus(enabled) {
  document.body.classList.toggle('reading-focus', Boolean(enabled));
  const toggle = document.querySelector('[data-focus-toggle]');
  if (toggle) toggle.setAttribute('aria-pressed', String(Boolean(enabled)));
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
  const focusToggle = document.querySelector('[data-focus-toggle]');
  const savedFocus = localStorage.getItem(readingFocusStorageKey) === 'true';
  applyReadingFocus(savedFocus);
  focusToggle?.addEventListener('click', () => {
    const enabled = !document.body.classList.contains('reading-focus');
    localStorage.setItem(readingFocusStorageKey, String(enabled));
    applyReadingFocus(enabled);
  });
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
    details.open = openPaths.has(path);
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
document.addEventListener('DOMContentLoaded', () => { initThemePicker(); initReadingControls(); initCommandPalette(); restoreSidebarState(); initMobileSidebar(); });
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
