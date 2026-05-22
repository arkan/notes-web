package app

const templates = `
{{define "layout-start"}}
<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · Notes Web</title><link rel="stylesheet" href="/_static/style.css"><script defer src="/_static/app.js"></script><script type="module">import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs'; mermaid.initialize({startOnLoad:true,theme:'neutral'});</script><script defer src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"></script></head><body><button class="mobile-menu" data-sidebar-toggle aria-label="Open sidebar" aria-expanded="false">☰</button><div class="sidebar-backdrop" data-sidebar-close></div><div class="shell"><aside class="side"><a class="brand" href="/">Notes Web</a><form class="search" action="/_search"><input name="q" placeholder="Search…" value="{{.Q}}"></form><label class="theme-picker">Theme<select data-theme-select aria-label="Theme"><option value="auto">Auto</option><option value="light">Light</option><option value="dark">Dark</option><option value="sepia">Sepia</option></select></label><section><h3>Favorites</h3>{{range .Favorites}}<a class="nav" href="{{index . "URL"}}">★ {{index . "Label"}}</a>{{end}}</section><section><h3>Vault</h3>{{template "tree" .Tree}}</section></aside><main class="main">
{{end}}
{{define "layout-end"}}</main></div></body></html>{{end}}
{{define "tree"}}<ul class="tree">{{range .}}<li>{{if .IsDir}}<details class="tree-folder{{if .ContainsActive}} active-branch{{end}}" data-tree-path="{{.Rel}}"{{if .ContainsActive}} open{{end}}><summary>📁 {{.Name}}</summary>{{if .Children}}{{template "tree" .Children}}{{end}}</details>{{else}}<a{{if .IsActive}} class="active" aria-current="page"{{end}} href="{{.URL}}">📄 {{.Name}}</a>{{end}}</li>{{end}}</ul>{{end}}
{{define "home"}}{{template "layout-start" .}}<h1>Home</h1><button class="copy-link" data-copy-link>Copy link</button><div class="cards"><section class="card"><h2>Latest daily note</h2>{{with .Latest}}<a class="biglink" href="{{url .RelPath}}">{{.RelPath}}</a>{{else}}<p>No daily note found.</p>{{end}}</section><section class="card"><h2>Favorites</h2>{{range .Favorites}}<a class="pill" href="{{index . "URL"}}">{{index . "Label"}}</a>{{end}}</section></div><h2>Recent notes</h2><ul class="list">{{range .Recent}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a><small>{{.ModTime.Format "2006-01-02 15:04"}}</small></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "note"}}{{template "layout-start" .}}<nav class="crumb"><a href="/">Home</a> / {{.Note.RelPath}}</nav><article class="note"><header><h1>{{.Doc.Title}}</h1><button class="copy-link" data-copy-link>Copy link</button></header>{{if .Doc.Toc}}<details class="toc" open><summary>Table of contents</summary><ul>{{range .Doc.Toc}}<li class="lvl{{.Level}}"><a href="#{{.ID}}">{{.Text}}</a></li>{{end}}</ul></details>{{end}}<div class="content">{{safe .Doc.HTML}}</div></article><section class="backlinks"><h2>Backlinks</h2>{{if .Backlinks}}<ul>{{range .Backlinks}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a></li>{{end}}</ul>{{else}}<p>No backlinks.</p>{{end}}</section>{{template "layout-end" .}}{{end}}
{{define "folder"}}{{template "layout-start" .}}<h1>📁 {{.Path}}</h1><button class="copy-link" data-copy-link>Copy link</button><ul class="list">{{range .Items}}<li><a href="{{index . "URL"}}">{{if index . "Dir"}}📁{{else}}📄{{end}} {{index . "Name"}}</a></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "search"}}{{template "layout-start" .}}<h1>Search</h1><form class="search big" action="/_search"><input name="q" value="{{.Q}}" autofocus><button>Search</button></form>{{if .Err}}<p class="error">{{.Err}}</p>{{end}}<ul class="results">{{range .Results}}<li><a href="{{.URL}}">{{.RelPath}}:{{.Line}}</a><p>{{.Snippet}}</p></li>{{else}}{{if .Q}}<p>No results.</p>{{end}}{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "resolve"}}{{template "layout-start" .}}<h1>Choose a note</h1><p>Multiple notes match <code>{{.Name}}</code>.</p><ul class="list">{{range .Matches}}<li><a href="{{url .RelPath}}">{{.RelPath}}</a></li>{{end}}</ul>{{template "layout-end" .}}{{end}}
{{define "missing"}}{{template "layout-start" .}}<h1>Note not found</h1><p>No note matches <code>{{.Name}}</code>.</p>{{template "layout-end" .}}{{end}}
`

const css = `
:root{color-scheme:light;--bg:#f7f5ef;--panel:#fffdfa;--ink:#27231d;--muted:#786f63;--line:#e2dccc;--accent:#6b5cff;--soft:#efecff;--code:#eee9dc;--pre:#f1eee6;--measure:78ch}[data-theme=dark]{color-scheme:dark;--bg:#11131a;--panel:#181b24;--ink:#ebe7df;--muted:#a9a197;--line:#303545;--accent:#9b8cff;--soft:#272340;--code:#242838;--pre:#171b26}[data-theme=sepia]{color-scheme:light;--bg:#f4ecd8;--panel:#fff8e8;--ink:#33291c;--muted:#7b6b55;--line:#dfd0ae;--accent:#8a5a12;--soft:#f1dfb8;--code:#efe1c5;--pre:#eadcbd}@media(prefers-color-scheme:dark){:root:not([data-theme]){color-scheme:dark;--bg:#11131a;--panel:#181b24;--ink:#ebe7df;--muted:#a9a197;--line:#303545;--accent:#9b8cff;--soft:#272340;--code:#242838;--pre:#171b26}}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:16px/1.6 ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif}.mobile-menu{display:none}.sidebar-backdrop{display:none}.shell{display:grid;grid-template-columns:300px minmax(0,1fr);min-height:100vh}.side{position:sticky;top:0;height:100vh;overflow:auto;padding:22px;background:var(--panel);border-right:1px solid var(--line)}.brand{display:block;font-weight:800;font-size:22px;text-decoration:none;color:var(--ink);margin-bottom:18px}.main{min-width:0;width:100%;padding:40px 56px}.search{display:flex;gap:8px;margin:12px 0 24px}.search input{width:100%;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:var(--panel);color:var(--ink)}.theme-picker{display:flex;align-items:center;justify-content:space-between;gap:10px;margin:0 0 24px;color:var(--muted);font-size:14px}.theme-picker select{border:1px solid var(--line);border-radius:999px;background:var(--panel);color:var(--ink);padding:6px 10px}.search.big input{font-size:18px}button{border:1px solid var(--line);background:var(--panel);color:var(--ink);border-radius:10px;padding:8px 12px;cursor:pointer}button.copied,.task-id.copied{border-color:#2b8a3e;color:#2b8a3e;background:#ebfbee}h1,h2,h3{line-height:1.2}h1{font-size:40px;margin:0 0 24px}h2{margin-top:32px}.nav,.tree a,.list a,.results a{color:var(--ink);text-decoration:none}.nav{display:block;padding:5px 0}.tree{list-style:none;padding-left:12px;margin:4px 0}.tree ul{border-left:1px solid var(--line);margin-left:8px}.tree li{margin:4px 0}.tree summary{cursor:pointer;user-select:none;border-radius:8px;padding:2px 4px}.tree summary:hover,.tree a:hover{background:var(--soft)}.tree a.active,.tree-folder.active-branch>summary{background:var(--soft);color:var(--accent);font-weight:700}.cards{display:grid;grid-template-columns:1fr 1fr;gap:16px}.card{background:var(--panel);border:1px solid var(--line);border-radius:18px;padding:20px}.biglink{font-size:18px;color:var(--accent)}.pill{display:inline-block;padding:7px 10px;margin:4px;border-radius:999px;background:var(--soft);color:var(--accent);text-decoration:none}.crumb{color:var(--muted);margin-bottom:16px}.note header{display:flex;align-items:start;justify-content:space-between;gap:20px;min-width:0}.content{font-size:17px;line-height:1.72;overflow-wrap:anywhere}.content a{overflow-wrap:anywhere}.content>:where(p,ul,ol,blockquote,details,dl){max-width:var(--measure)}.content>:where(pre,table,.mermaid,img){max-width:100%}.content p{margin:0 0 1.05rem}.content h2{margin:2.2rem 0 1rem}.content img{max-width:100%}.content pre{overflow:auto;max-width:100%;padding:14px;border-radius:12px;background:var(--pre)}.content code{background:var(--code);border-radius:5px;padding:1px 4px}.content table{display:block;overflow-x:auto;max-width:100%;border-collapse:collapse;width:100%;margin:16px 0}.content th,.content td{border:1px solid var(--line);padding:8px 10px}.frontmatter,.toc,.backlinks{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:12px 16px;margin:18px 0}.frontmatter dl{display:grid;grid-template-columns:150px 1fr;gap:6px}.frontmatter dt{font-weight:700}.callout{border-left:4px solid var(--accent);background:var(--soft);padding:10px 14px;border-radius:10px;margin:16px 0}.callout-title{font-weight:700}.task-id{display:inline-flex;align-items:center;gap:4px;margin-left:8px;padding:1px 7px;border:1px solid var(--line);border-radius:999px;background:var(--panel);color:var(--muted);font:12px ui-monospace,SFMono-Regular,Menlo,monospace;vertical-align:middle}.task-id:hover{color:var(--accent);border-color:var(--accent)}.lvl2{margin-left:12px}.lvl3{margin-left:24px}.list,.results{padding-left:0;list-style:none}.list li,.results li{padding:10px 0;border-bottom:1px solid var(--line)}small{display:block;color:var(--muted)}.error{color:#b00020}@media(max-width:850px){.shell{display:block}.mobile-menu{display:inline-flex;position:fixed;top:12px;left:12px;z-index:40;align-items:center;justify-content:center;width:42px;height:42px;padding:0;border-radius:999px;box-shadow:0 2px 10px #0002}.side{position:fixed;left:0;top:0;width:min(86vw,340px);max-width:340px;transform:translateX(-100%);z-index:30;height:100vh;transition:transform .18s ease;box-shadow:0 12px 35px #0003}.main{padding:64px 18px 28px}.cards{grid-template-columns:1fr}.sidebar-backdrop{position:fixed;inset:0;background:#0006;z-index:20}body.sidebar-open{overflow:hidden}body.sidebar-open .side{transform:translateX(0)}body.sidebar-open .sidebar-backdrop{display:block}}
`

const js = `
const sidebarStorageKey = 'notes-web:sidebar-open';
const themeStorageKey = 'notes-web:theme';
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
document.addEventListener('DOMContentLoaded', () => { initThemePicker(); restoreSidebarState(); initMobileSidebar(); });
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
