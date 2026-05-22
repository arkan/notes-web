
const paletteShortcutClass = 'palette-shortcuts';
const sidebarStorageKey = 'notes-web:sidebar-open';
const panelStateStorageKey = 'notes-web:panel-open';
const themeStorageKey = 'notes-web:theme';
const fontSizeStorageKey = 'notes-web:font-size';
const readingFocusStorageKey = 'notes-web:reading-focus';
function applyInitialPreferences() {
  try {
    applyTheme(localStorage.getItem(themeStorageKey) || 'auto');
    applyFontSize(localStorage.getItem(fontSizeStorageKey) || 'normal');
    applyReadingFocus(localStorage.getItem(readingFocusStorageKey) === 'true');
  } catch {}
}
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
let paletteLoadPromise = null;
let paletteLoadError = false;
function loadPaletteItems() {
  if (paletteItems.length) return Promise.resolve(paletteItems);
  if (paletteLoadPromise) return paletteLoadPromise;
  paletteLoadError = false;
  paletteLoadPromise = fetch('/_api/palette')
    .then((r) => {
      if (!r.ok) throw new Error('palette load failed: ' + r.status);
      return r.json();
    })
    .then((items) => {
      paletteItems = Array.isArray(items) ? items : [];
      return paletteItems;
    })
    .catch((err) => {
      console.error(err);
      paletteLoadError = true;
      paletteLoadPromise = null;
      throw err;
    });
  return paletteLoadPromise;
}
function openPalette() {
  const palette = document.querySelector('[data-palette]');
  const input = document.querySelector('[data-palette-input]');
  if (!palette || !input) return;
  palette.hidden = false;
  renderPalette(input.value);
  loadPaletteItems().then(() => renderPalette(input.value)).catch(() => renderPalette(input.value));
  setTimeout(() => { input.focus(); input.select(); }, 0);
}
function closePalette() {
  const palette = document.querySelector('[data-palette]');
  if (palette) palette.hidden = true;
}
function setActivePaletteIndex(index) {
  paletteSelectedIndex = Math.max(0, Math.min(index, Math.max(0, paletteMatches.length - 1)));
  document.querySelectorAll('[data-palette-index]').forEach((button) => {
    const selected = Number(button.dataset.paletteIndex) === paletteSelectedIndex;
    button.classList.toggle('is-selected', selected);
    button.setAttribute('aria-selected', String(selected));
  });
}
function renderPalette(query) {
  const results = document.querySelector('[data-palette-results]');
  if (!results) return;
  if (!paletteItems.length) {
    paletteMatches = [];
    results.innerHTML = paletteLoadError ? '<p class="palette-empty empty-state">Unable to load search results.</p>' : '<p class="palette-empty empty-state">Loading…</p>';
    return;
  }
  const q = (query || '').trim().toLowerCase();
  paletteMatches = paletteItems.filter((item) => {
    const haystack = [item.title, item.path, item.kind].filter(Boolean).join(' ').toLowerCase();
    return !q || haystack.includes(q);
  }).slice(0, 30);
  if (paletteSelectedIndex >= paletteMatches.length) paletteSelectedIndex = 0;
  results.innerHTML = paletteMatches.map((item, index) => '<button type="button" role="option" aria-selected="' + (index === paletteSelectedIndex) + '" class="palette-item' + (index === paletteSelectedIndex ? ' is-selected' : '') + '" data-palette-index="' + index + '"><strong>' + escapeHTML(item.title || item.path || item.url || 'Untitled') + '</strong><small>' + escapeHTML(item.path || item.url || '') + '</small><span class="chip palette-kind">' + escapeHTML(item.kind || 'item') + '</span></button>').join('') || '<p class="palette-empty empty-state">No results.</p>';
}
function openPaletteMatch(index) {
  const item = paletteMatches[index];
  if (item && item.url) location.assign(item.url);
}
function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}
function initCommandPalette() {
  document.querySelector('[data-palette-open]')?.addEventListener('click', openPalette);
  document.querySelector('[data-palette]')?.addEventListener('click', (ev) => { if (ev.target.matches('[data-palette]')) closePalette(); });
  const results = document.querySelector('[data-palette-results]');
  results?.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-palette-index]');
    if (!button || !results.contains(button)) return;
    ev.preventDefault();
    openPaletteMatch(Number(button.dataset.paletteIndex));
  });
  results?.addEventListener('mousemove', (ev) => {
    const button = ev.target.closest('[data-palette-index]');
    if (!button || !results.contains(button)) return;
    setActivePaletteIndex(Number(button.dataset.paletteIndex));
  });
  document.querySelector('[data-palette-input]')?.addEventListener('input', (ev) => { paletteSelectedIndex = 0; renderPalette(ev.target.value); });
  document.querySelector('[data-palette-input]')?.addEventListener('keydown', (ev) => {
    if (ev.key === 'ArrowDown') { ev.preventDefault(); setActivePaletteIndex(paletteSelectedIndex + 1); }
    else if (ev.key === 'ArrowUp') { ev.preventDefault(); setActivePaletteIndex(paletteSelectedIndex - 1); }
    else if (ev.key === 'Enter') { ev.preventDefault(); openPaletteMatch(paletteSelectedIndex); }
  });
  document.addEventListener('keydown', (ev) => {
    const inField = ev.target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(ev.target.tagName);
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'k') { ev.preventDefault(); openPalette(); }
    else if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'b') { ev.preventDefault(); toggleReadingFocus(); }
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
  document.documentElement.dataset.readingFocus = String(Boolean(enabled));
  document.body?.classList.toggle('reading-focus', Boolean(enabled));
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

function readPanelState() {
  try {
    const raw = JSON.parse(localStorage.getItem(panelStateStorageKey) || '{}');
    return raw && typeof raw === 'object' && !Array.isArray(raw) ? raw : {};
  } catch { return {}; }
}
function writePanelState(state) {
  try { localStorage.setItem(panelStateStorageKey, JSON.stringify(state)); }
  catch {}
}
function restorePanelState() {
  const state = readPanelState();
  document.querySelectorAll('details[data-panel-state]').forEach((details) => {
    const key = details.dataset.panelState;
    if (!key) return;
    if (Object.prototype.hasOwnProperty.call(state, key)) details.open = Boolean(state[key]);
    details.addEventListener('toggle', () => {
      const current = readPanelState();
      current[key] = details.open;
      writePanelState(current);
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
  initTagFilter();
  document.querySelector('[data-hide-rare]')?.addEventListener('click', () => {
    document.querySelector('.rare-tags')?.toggleAttribute('hidden');
  });
}
function initTagFilter() {
  const tagFilter = document.querySelector('[data-tag-filter]');
  if (!tagFilter) return;
  tagFilter.addEventListener('input', () => {
    const q = tagFilter.value.trim().toLowerCase().replace(/^#/, '');
    document.querySelectorAll('[data-tag-chip][data-tag-name]').forEach((el) => {
      const tag = (el.dataset.tagName || el.textContent || '').toLowerCase().replace(/^#/, '');
      el.hidden = Boolean(q) && !tag.includes(q);
      el.closest('.tag-letter')?.toggleAttribute('data-filtered', Boolean(q));
    });
    document.querySelectorAll('.tag-letter').forEach((section) => {
      section.hidden = Boolean(q) && !section.querySelector('[data-tag-chip]:not([hidden])');
    });
    document.querySelectorAll('.rare-tags').forEach((section) => {
      section.hidden = Boolean(q) && !section.querySelector('[data-tag-chip]:not([hidden])');
    });
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
document.addEventListener('DOMContentLoaded', () => { applyInitialPreferences(); initThemePicker(); initReadingControls(); initSettingsModal(); initCommandPalette(); restoreSidebarState(); restorePanelState(); initMobileSidebar(); initListFilters(); });
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
