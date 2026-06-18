
const paletteShortcutClass = 'palette-shortcuts';
const sidebarStorageKey = 'notes-web:sidebar-open';
const panelStateStorageKey = 'notes-web:panel-open';
const themeStorageKey = 'notes-web:theme';
const fontSizeStorageKey = 'notes-web:font-size';
const readingFocusStorageKey = 'notes-web:reading-focus';
const todoFilterStorageKey = 'notes-web:todo-filters';
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
    results.setAttribute('aria-busy', String(!paletteLoadError));
    results.innerHTML = paletteLoadError ? renderPaletteState('error', 'Unable to load search results.', 'Check the server, then reopen search.') : renderPaletteState('loading', 'Loading…', 'Preparing notes, tags, and favorites.');
    return;
  }
  results.setAttribute('aria-busy', 'false');
  const q = (query || '').trim().toLowerCase();
  paletteMatches = paletteItems.filter((item) => {
    const haystack = [item.title, item.path, item.kind].filter(Boolean).join(' ').toLowerCase();
    return !q || haystack.includes(q);
  }).slice(0, 30);
  if (paletteSelectedIndex >= paletteMatches.length) paletteSelectedIndex = 0;
  results.innerHTML = paletteMatches.map((item, index) => '<button type="button" role="option" aria-selected="' + (index === paletteSelectedIndex) + '" class="palette-item' + (index === paletteSelectedIndex ? ' is-selected' : '') + '" data-palette-index="' + index + '"><strong>' + escapeHTML(item.title || item.path || item.url || 'Untitled') + '</strong><small>' + escapeHTML(item.path || item.url || '') + '</small><span class="chip palette-kind">' + escapeHTML(item.kind || 'item') + '</span></button>').join('') || renderPaletteState('empty', 'No results.', 'Try a note title, tag, or favorite.');
}
function renderPaletteState(kind, title, detail) {
  return '<div class="palette-empty empty-state palette-state palette-state-' + kind + '" role="status"><strong>' + escapeHTML(title) + '</strong><small>' + escapeHTML(detail) + '</small><span class="palette-state-rule" aria-hidden="true"></span></div>';
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

function initHomepageProjectFilter() {
  document.querySelectorAll('[data-home-project-filter]').forEach((input) => {
    const block = input.closest('[data-home-block="active_projects"]') || document;
    const rows = Array.from(block.querySelectorAll('[data-home-project-row]'));
    const empty = block.querySelector('[data-home-project-empty]');
    const apply = () => {
      const q = (input.value || '').trim().toLowerCase();
      let visibleCount = 0;
      rows.forEach((row) => {
        const haystack = (row.dataset.homeProjectSearchText || row.textContent || '').toLowerCase();
        const visible = !q || haystack.includes(q);
        row.hidden = !visible;
        if (visible) visibleCount++;
      });
      if (empty) empty.hidden = visibleCount > 0 || rows.length === 0;
    };
    input.addEventListener('input', apply);
    apply();
  });
}


function readTodoFilterState() {
  try {
    const raw = JSON.parse(localStorage.getItem(todoFilterStorageKey) || '{}');
    return raw && typeof raw === 'object' && !Array.isArray(raw) ? raw : {};
  } catch { return {}; }
}
function writeTodoFilterState(state) {
  try { localStorage.setItem(todoFilterStorageKey, JSON.stringify(state)); }
  catch {}
}
function selectTodoOption(select, value) {
  if (!select || value === undefined || value === null) return;
  const normalized = String(value);
  if ([...select.options].some((option) => option.value === normalized)) select.value = normalized;
}
function restoreTodoFilterState({ tag, priority, date, group, hideNoDate, hideDone }) {
  const state = readTodoFilterState();
  selectTodoOption(tag, state.tag || '');
  selectTodoOption(priority, state.priority || '');
  selectTodoOption(date, state.date || '');
  selectTodoOption(group, state.group || 'Due date');
  if (hideNoDate && Object.prototype.hasOwnProperty.call(state, 'hideNoDate')) hideNoDate.checked = Boolean(state.hideNoDate);
  if (hideDone && Object.prototype.hasOwnProperty.call(state, 'hideDone')) hideDone.checked = Boolean(state.hideDone);
}

function initTodoFilters() {
  const shell = document.querySelector('.todo-shell');
  if (!shell) return;
  const search = shell.querySelector('[data-todo-search]');
  const tag = shell.querySelector('[data-todo-filter="tag"]');
  const priority = shell.querySelector('[data-todo-filter="priority"]');
  const date = shell.querySelector('[data-todo-filter="date"]');
  const group = shell.querySelector('[data-todo-filter="group"]');
  const hideNoDate = shell.querySelector('[data-todo-hide-nodate]');
  const hideDone = shell.querySelector('[data-todo-hide-done]');
  const rows = Array.from(shell.querySelectorAll('.task-row'));
  restoreTodoFilterState({ tag, priority, date, group, hideNoDate, hideDone });
  function persistTodoFilters() {
    writeTodoFilterState({ tag: tag?.value || '', priority: priority?.value || '', date: date?.value || '', group: group?.value || 'Due date', hideNoDate: Boolean(hideNoDate?.checked), hideDone: Boolean(hideDone?.checked) });
  }
  function apply() {
    const q = (search?.value || '').trim().toLowerCase();
    const selectedTag = tag?.value || '';
    const selectedPriority = priority?.value || '';
    const selectedDate = date?.value || '';
    const today = new URLSearchParams(location.search).get('today') || new Date().toISOString().slice(0, 10);
    updateTodoTagCounts(rows, { q, selectedPriority, selectedDate, hideNoDate: Boolean(hideNoDate?.checked), hideDone: Boolean(hideDone?.checked), today });
    rows.forEach((row) => {
      const visible = todoRowMatchesFilters(row, { q, selectedTag, selectedPriority, selectedDate, hideNoDate: Boolean(hideNoDate?.checked), hideDone: Boolean(hideDone?.checked), today }, true);
      row.hidden = !visible;
      row.dataset.todoVisible = String(visible);
    });
    renderTodoGroupedView(shell, rows, group?.value || 'Due date');
  }
  search?.addEventListener('input', apply);
  [tag, priority, date, group, hideNoDate, hideDone].forEach((el) => {
    el?.addEventListener('input', () => { persistTodoFilters(); apply(); });
    el?.addEventListener('change', () => { persistTodoFilters(); apply(); });
  });
  apply();
}
function todoRowTags(row) {
  return (row.dataset.tags || '').trim().split(/\s+/).filter(Boolean);
}
function todoRowMatchesFilters(row, filters, includeTag = true) {
  const tags = todoRowTags(row);
  const due = row.dataset.due || '';
  const isNoDate = !due;
  const isDone = row.classList.contains('completed') || Boolean(row.closest('.done'));
  const matchesSearch = !filters.q || row.textContent.toLowerCase().includes(filters.q);
  const matchesTag = !includeTag || !filters.selectedTag || tags.includes(filters.selectedTag);
  const matchesPriority = !filters.selectedPriority || row.dataset.priority === filters.selectedPriority;
  const matchesDate = !filters.selectedDate || todoDateGroup(due, filters.today) === filters.selectedDate.toLowerCase().replace(' ', '-');
  const matchesNoDate = !filters.hideNoDate || !isNoDate || isDone;
  const matchesDone = !filters.hideDone || !isDone;
  return matchesSearch && matchesTag && matchesPriority && matchesDate && matchesNoDate && matchesDone;
}
function countTodoTags(rows, filters) {
  const counts = new Map();
  rows.forEach((row) => {
    if (!todoRowMatchesFilters(row, filters, false)) return;
    new Set(todoRowTags(row)).forEach((value) => counts.set(value, (counts.get(value) || 0) + 1));
  });
  return counts;
}
function updateTodoTagCounts(rows, filters) {
  const shell = rows[0]?.closest('.todo-shell') || document;
  const counts = countTodoTags(rows, filters);
  shell.querySelectorAll('[data-todo-filter="tag"] option').forEach((option) => {
    if (!option.value) return;
    const count = counts.get(option.value) || 0;
    option.hidden = count === 0;
    option.disabled = count === 0;
    option.textContent = '#' + option.value + ' (' + String(counts.get(option.value) || 0) + ')';
  });
}
function todoDateGroup(due, today) {
  if (!due) return 'no-date';
  if (due < today) return 'overdue';
  if (due === today) return 'today';
  return 'upcoming';
}

function renderTodoGroupedView(shell, rows, mode) {
  const sections = Array.from(shell.querySelectorAll('.todo-section:not(.todo-dynamic-group)'));
  let dynamic = shell.querySelector('.todo-dynamic-groups');
  if (!dynamic) {
    dynamic = document.createElement('div');
    dynamic.className = 'todo-dynamic-groups';
    dynamic.setAttribute('aria-label', 'Grouped tasks');
    shell.querySelector('.todo-sections')?.appendChild(dynamic);
  }
  dynamic.replaceChildren();
  if (mode === 'Due date') {
    dynamic.hidden = true;
    sections.forEach((section) => {
      const hasVisibleRows = Boolean(section.querySelector('.task-row:not([hidden])'));
      const hasRows = Boolean(section.querySelector('.task-row'));
      section.hidden = hasRows && !hasVisibleRows;
    });
    updateTodoSectionCounts(sections);
    sortTodoRows(shell, mode);
    return;
  }
  sections.forEach((section) => { section.hidden = true; });
  const visibleRows = rows.filter((row) => row.dataset.todoVisible === 'true');
  const grouped = new Map();
  visibleRows.forEach((row) => {
    const key = todoGroupKey(row, mode);
    if (!grouped.has(key)) grouped.set(key, []);
    grouped.get(key).push(row);
  });
  const keys = [...grouped.keys()].sort((a, b) => todoGroupSortKey(a, mode).localeCompare(todoGroupSortKey(b, mode)));
  keys.forEach((key) => {
    const section = document.createElement('section');
    section.className = 'todo-section todo-dynamic-group ' + mode.toLowerCase();
    section.setAttribute('aria-labelledby', 'todo-group-' + slugifyTodoGroup(key));
    const header = document.createElement('div');
    header.className = 'todo-section-header';
    const heading = document.createElement('h2');
    heading.id = 'todo-group-' + slugifyTodoGroup(key);
    heading.textContent = key;
    const count = document.createElement('span');
    count.className = 'count';
    count.textContent = String(grouped.get(key).length);
    header.append(heading, count);
    const list = document.createElement('div');
    list.className = 'task-list';
    list.setAttribute('role', 'list');
    grouped.get(key)
      .slice()
      .sort((a, b) => todoSortKey(a, mode).localeCompare(todoSortKey(b, mode)))
      .forEach((row) => {
        const clone = row.cloneNode(true);
        clone.hidden = false;
        list.appendChild(clone);
      });
    section.append(header, list);
    dynamic.appendChild(section);
  });
  dynamic.hidden = false;
}
function updateTodoSectionCounts(sections) {
  sections.forEach((section) => {
    const count = section.querySelector('.todo-section-header .count');
    if (count) count.textContent = String(section.querySelectorAll('.task-row:not([hidden])').length);
  });
}
function todoGroupKey(row, mode) {
  if (mode === 'Priority') return row.dataset.priority && row.dataset.priority !== '—' ? row.dataset.priority : 'No priority';
  if (mode === 'Project') return row.dataset.project || 'Inbox';
  return todoDateGroup(row.dataset.due || '', new URLSearchParams(location.search).get('today') || new Date().toISOString().slice(0, 10));
}
function todoGroupSortKey(key, mode) {
  if (mode === 'Priority') return String(prioritySortRank(key === 'No priority' ? '' : key)) + '|' + key;
  return key.toLowerCase();
}
function slugifyTodoGroup(value) {
  return String(value).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'group';
}

function sortTodoRows(shell, mode) {
  shell.querySelectorAll('.task-list').forEach((list) => {
    const rows = Array.from(list.querySelectorAll('.task-row'));
    if (list.closest('.todo-section.done')) rows.sort(compareDoneRows);
    else rows.sort((a, b) => todoSortKey(a, mode).localeCompare(todoSortKey(b, mode)));
    rows.forEach((row) => list.appendChild(row));
  });
}
function compareDoneRows(a, b) {
  const aDone = a.dataset.done || '';
  const bDone = b.dataset.done || '';
  if (aDone !== bDone) {
    if (!aDone) return 1;
    if (!bDone) return -1;
    return bDone.localeCompare(aDone);
  }
  return (a.dataset.source + '|' + a.textContent).localeCompare(b.dataset.source + '|' + b.textContent);
}
function todoSortKey(row, mode) {
  if (mode === 'Priority') return String(prioritySortRank(row.dataset.priority || '')) + '|' + (row.dataset.due || '9999-99-99') + '|' + row.textContent;
  if (mode === 'Project') return (row.dataset.project || 'zzzz') + '|' + (row.dataset.due || '9999-99-99') + '|' + row.textContent;
  return (row.dataset.due || '9999-99-99') + '|' + String(prioritySortRank(row.dataset.priority || '')) + '|' + row.textContent;
}
function prioritySortRank(priority) {
  if (priority === 'P1') return 1;
  if (priority === 'P2') return 2;
  if (priority === 'P3') return 3;
  if (priority === 'P4') return 4;
  return 9;
}

function resetTodoDropdown(dropdown) {
  dropdown.hidden = true;
  dropdown.style.removeProperty('left');
  dropdown.style.removeProperty('top');
}
function closeTodoMenus(exceptMenu) {
  document.querySelectorAll('[data-task-menu][aria-expanded="true"]').forEach((button) => {
    if (button === exceptMenu) return;
    button.setAttribute('aria-expanded', 'false');
    const dropdown = button.closest('.task-actions')?.querySelector('.task-menu-dropdown');
    if (dropdown) resetTodoDropdown(dropdown);
  });
}
function positionTodoDropdown(menu, dropdown) {
  const buttonRect = menu.getBoundingClientRect();
  dropdown.style.position = 'fixed';
  dropdown.style.left = '0px';
  dropdown.style.top = '0px';
  dropdown.hidden = false;
  const dropdownRect = dropdown.getBoundingClientRect();
  const left = Math.max(8, Math.min(window.innerWidth - dropdownRect.width - 8, buttonRect.right - dropdownRect.width));
  const opensDown = buttonRect.bottom + 6 + dropdownRect.height <= window.innerHeight - 8;
  const top = opensDown ? Math.min(window.innerHeight - dropdownRect.height - 8, buttonRect.bottom + 6) : Math.min(window.innerHeight - dropdownRect.height - 8, Math.max(8, buttonRect.top - dropdownRect.height - 6));
  dropdown.style.left = left + 'px';
  dropdown.style.top = top + 'px';
}
function initTodoActions() {
  document.addEventListener('click', (ev) => {
    const menu = ev.target.closest('[data-task-menu]');
    if (!menu) {
      if (!ev.target.closest('.task-menu-dropdown')) closeTodoMenus();
      return;
    }
    ev.preventDefault();
    const dropdown = menu.closest('.task-actions')?.querySelector('.task-menu-dropdown');
    if (!dropdown) return;
    const expanded = menu.getAttribute('aria-expanded') === 'true';
    closeTodoMenus(menu);
    menu.setAttribute('aria-expanded', String(!expanded));
    if (expanded) resetTodoDropdown(dropdown);
    else positionTodoDropdown(menu, dropdown);
  });
  window.addEventListener('resize', () => closeTodoMenus());
  window.addEventListener('scroll', () => closeTodoMenus(), true);
  document.addEventListener('keydown', (ev) => {
    if (ev.key === 'Escape') closeTodoMenus();
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
function currentCopyPath() {
  const path = location.pathname || '/';
  return path === '/' ? '/' : path.replace(/^\/+/, '');
}
function initNotesMaps() {
  document.querySelectorAll('[data-notes-map]').forEach((el) => {
    let data;
    try { data = JSON.parse(el.getAttribute('data-notes-map') || '{}'); }
    catch (err) { el.innerHTML = '<div class="notes-map-error">Unable to parse map data.</div>'; return; }
    const points = Array.isArray(data.points) ? data.points : [];
    if (!points.length) {
      el.classList.add('notes-map-empty');
      el.textContent = data.skippedMissingCoords ? 'No geocoded points found (' + data.skippedMissingCoords + ' skipped without coordinates).' : 'No map points found.';
      return;
    }
    renderNotesMap(el, points, data.skippedMissingCoords || 0);
  });
}
function renderNotesMap(el, points, skipped) {
  const zoom = chooseMapZoom(points);
  const center = mapCenter(points);
  const size = {w: Math.max(320, el.clientWidth || 900), h: 420};
  const centerPx = lonLatToPixel(center.lon, center.lat, zoom);
  el.innerHTML = '<div class="notes-map-canvas" role="img" aria-label="Map with ' + points.length + ' note markers"></div><div class="notes-map-status"></div>';
  const canvas = el.querySelector('.notes-map-canvas');
  canvas.style.height = size.h + 'px';
  renderMapTiles(canvas, centerPx, zoom, size);
  const colors = {'en attente':'#f59f00','à relancer':'#e67700','a relancer':'#e67700','contactée':'#339af0','contacte':'#339af0','contacté':'#339af0','refusée':'#e03131','refusee':'#e03131','refusé':'#e03131','place':'#2f9e44','place obtenue':'#2f9e44','acceptée':'#2f9e44'};
  points.forEach((point) => {
    const px = lonLatToPixel(point.lon, point.lat, zoom);
    const left = (size.w / 2) + (px.x - centerPx.x);
    const top = (size.h / 2) + (px.y - centerPx.y);
    const marker = document.createElement('button');
    marker.type = 'button';
    marker.className = 'notes-map-marker';
    marker.style.left = left + 'px';
    marker.style.top = top + 'px';
    marker.style.setProperty('--marker-color', colors[String(point.colorValue || point.status || '').toLowerCase()] || '#6b5cff');
    marker.setAttribute('aria-label', point.title || 'Map point');
    marker.innerHTML = '<span></span>';
    marker.addEventListener('click', () => showNotesMapPopup(canvas, point, left, top));
    canvas.appendChild(marker);
  });
  const status = el.querySelector('.notes-map-status');
  status.textContent = points.length + ' point' + (points.length > 1 ? 's' : '') + (skipped ? ' · ' + skipped + ' skipped without coordinates' : '');
}
function chooseMapZoom(points) {
  if (points.length <= 1) return 13;
  const lats = points.map(p => Number(p.lat)), lons = points.map(p => Number(p.lon));
  const span = Math.max(Math.max(...lats) - Math.min(...lats), Math.max(...lons) - Math.min(...lons));
  if (span > 1) return 8;
  if (span > 0.4) return 10;
  if (span > 0.12) return 11;
  if (span > 0.04) return 12;
  return 13;
}
function mapCenter(points) {
  return {lat: points.reduce((sum, p) => sum + Number(p.lat), 0) / points.length, lon: points.reduce((sum, p) => sum + Number(p.lon), 0) / points.length};
}
function lonLatToPixel(lon, lat, zoom) {
  const sin = Math.sin(Number(lat) * Math.PI / 180);
  const scale = 256 * Math.pow(2, zoom);
  return {x: (Number(lon) + 180) / 360 * scale, y: (0.5 - Math.log((1 + sin) / (1 - sin)) / (4 * Math.PI)) * scale};
}
function renderMapTiles(canvas, centerPx, zoom, size) {
  const startX = Math.floor((centerPx.x - size.w / 2) / 256);
  const endX = Math.floor((centerPx.x + size.w / 2) / 256);
  const startY = Math.floor((centerPx.y - size.h / 2) / 256);
  const endY = Math.floor((centerPx.y + size.h / 2) / 256);
  for (let x = startX; x <= endX; x++) for (let y = startY; y <= endY; y++) {
    const img = document.createElement('img');
    img.className = 'notes-map-tile';
    img.alt = '';
    img.loading = 'lazy';
    img.src = 'https://tile.openstreetmap.org/' + zoom + '/' + x + '/' + y + '.png';
    img.style.left = (x * 256 - (centerPx.x - size.w / 2)) + 'px';
    img.style.top = (y * 256 - (centerPx.y - size.h / 2)) + 'px';
    canvas.appendChild(img);
  }
}
function showNotesMapPopup(canvas, point, left, top) {
  canvas.querySelector('.notes-map-popup')?.remove();
  const popup = document.createElement('div');
  popup.className = 'notes-map-popup';
  popup.style.left = Math.max(12, Math.min(left + 12, canvas.clientWidth - 260)) + 'px';
  popup.style.top = Math.max(12, top - 12) + 'px';
  const bits = [point.subtitle, point.address, point.distanceHome ? 'Maison: ' + point.distanceHome : '', point.distanceNoorderpark ? 'Noorderpark: ' + point.distanceNoorderpark : ''].filter(Boolean);
  popup.innerHTML = '<strong><a href="' + escapeHTML(point.url || '#') + '">' + escapeHTML(point.title || 'Untitled') + '</a></strong>' + bits.map(b => '<small>' + escapeHTML(b) + '</small>').join('') + (point.website ? '<a class="notes-map-popup-link" href="' + escapeHTML(point.website) + '">Website</a>' : '') + (point.mapUrl ? '<a class="notes-map-popup-link" href="' + escapeHTML(point.mapUrl) + '">Map</a>' : '');
  canvas.appendChild(popup);
}
document.addEventListener('DOMContentLoaded', () => { applyInitialPreferences(); initThemePicker(); initReadingControls(); initSettingsModal(); initCommandPalette(); restoreSidebarState(); restorePanelState(); initMobileSidebar(); initListFilters(); initHomepageProjectFilter(); initTodoActions(); initTodoFilters();
  initDataviewTables(); initNotesMaps(); });
document.addEventListener('click', async (ev) => {
  const codeCopy = ev.target.closest('[data-copy-code]');
  if (codeCopy) {
    ev.preventDefault();
    const code = codeCopy.closest('pre')?.querySelector('code');
    if (!code) return;
    await copyText(code.innerText);
    markCopied(codeCopy, '✓');
    return;
  }
  const copy = ev.target.closest('[data-copy]');
  if (copy) {
    ev.preventDefault();
    await copyText(copy.dataset.copy);
    markCopied(copy, 'copied');
    setTimeout(() => closeTodoMenus(), 150);
    return;
  }
  const pathCopy = ev.target.closest('[data-copy-path]');
  if (pathCopy) {
    ev.preventDefault();
    await copyText(currentCopyPath());
    markCopied(pathCopy, 'Path copied');
  }
});

let dataviewGlobalHandlersReady = false;

function initDataviewTables(root = document) {
  ensureDataviewGlobalHandlers();
  const wrappers = root.matches?.('.dataview-table-wrap') ? [root] : Array.from(root.querySelectorAll('.dataview-table-wrap'));
  wrappers.forEach((wrap) => initDataviewWrapper(wrap));
}

function initDataviewWrapper(wrap, existingState) {
  if (!wrap || wrap.dataset.dataviewInitialized === 'true') return;
  wrap.dataset.dataviewInitialized = 'true';
  const state = existingState || createDataviewState(wrap);
  state.wrap = wrap;
  wrap.__dataviewState = state;
  restoreDataviewTextFilter(wrap, state);
  restoreDataviewPageSize(wrap, state);
  initDataviewPagination(wrap, state);
  initDataviewAjaxControls(wrap, state);
  initDataviewMultiFilters(wrap, state);
  initDataviewSortHeaders(wrap, state);
  setDataviewLoading(wrap, false);
}

function createDataviewState(wrap) {
  return {
    page: 1,
    q: wrap.querySelector('input.dataview-filter[data-dataview-filter]')?.value || '',
    pageSize: wrap.querySelector('[data-dataview-page-size]')?.value || '0',
    sortField: '',
    sortDir: '',
    requestId: 0,
    abortController: null,
    debounceTimer: null,
    loading: false,
    wrap: wrap,
  };
}

function initDataviewPagination(wrap, state) {
  const table = wrap.querySelector('.dataview-table');
  const pageSizeSelect = wrap.querySelector('[data-dataview-page-size]');
  if (!table || !table.tBodies[0]) return;
  pageSizeSelect?.addEventListener('change', () => {
    state.pageSize = pageSizeSelect.value || '0';
    state.page = 1;
    updateDataviewPager(wrap, state);
  });
  updateDataviewPager(wrap, state);
}

function updateDataviewPager(wrap, state) {
  const table = wrap.querySelector('.dataview-table');
  const pager = wrap.querySelector('[data-dataview-pager]');
  const pageSizeSelect = wrap.querySelector('[data-dataview-page-size]');
  const tbody = table?.tBodies[0];
  if (!tbody) return;
  const allRows = Array.from(tbody.rows);
  const isGroupRow = (row) => row.classList.contains('dataview-group');
  const isNoRowsRow = (row) => Boolean(row.querySelector('.dataview-no-rows'));
  const dataRows = allRows.filter((row) => !isGroupRow(row) && !isNoRowsRow(row));
  const pageSize = Number(pageSizeSelect?.value || state.pageSize || 0);
  const maxPage = pageSize > 0 ? Math.max(1, Math.ceil(dataRows.length / pageSize)) : 1;
  state.page = Math.min(Math.max(1, state.page || 1), maxPage);
  let seen = 0;
  allRows.forEach((row) => {
    if (isGroupRow(row) || isNoRowsRow(row) || pageSize === 0) {
      row.hidden = false;
      return;
    }
    seen += 1;
    row.hidden = seen <= (state.page - 1) * pageSize || seen > state.page * pageSize;
  });
  if (!pager) return;
  if (pageSize > 0 && dataRows.length > pageSize) {
    pager.innerHTML = '<button type="button" data-dataview-prev>Prev</button><span>Page ' + state.page + ' / ' + maxPage + ' · ' + dataRows.length + ' rows</span><button type="button" data-dataview-next>Next</button>';
    pager.querySelector('[data-dataview-prev]')?.addEventListener('click', () => { state.page = Math.max(1, state.page - 1); updateDataviewPager(wrap, state); });
    pager.querySelector('[data-dataview-next]')?.addEventListener('click', () => { state.page = Math.min(maxPage, state.page + 1); updateDataviewPager(wrap, state); });
  } else {
    pager.innerHTML = '<span>' + dataRows.length + ' rows</span>';
  }
}

function restoreDataviewPageSize(wrap, state) {
  const select = wrap.querySelector('[data-dataview-page-size]');
  if (!select) return;
  const value = state.pageSize || select.value || '0';
  if ([...select.options].some((option) => option.value === value)) select.value = value;
  state.pageSize = select.value || '0';
}

function restoreDataviewTextFilter(wrap, state) {
  const input = wrap.querySelector('input.dataview-filter[data-dataview-filter]');
  if (input && state.q !== undefined) input.value = state.q;
}

function initDataviewAjaxControls(wrap, state) {
  if (!isAjaxDataviewWrapper(wrap)) return;
  const textFilter = wrap.querySelector('input.dataview-filter[data-dataview-filter]');
  textFilter?.addEventListener('input', () => {
    state.page = 1;
    window.clearTimeout(state.debounceTimer);
    state.debounceTimer = window.setTimeout(() => requestDataviewTable(wrap, state), 200);
  });
  wrap.querySelectorAll('select[data-dataview-filter]').forEach((select) => {
    select.addEventListener('change', () => {
      state.page = 1;
      requestDataviewTable(wrap, state);
    });
  });
}

function initDataviewSortHeaders(wrap, state) {
  if (!isAjaxDataviewWrapper(wrap)) return;
  wrap.querySelectorAll('th[data-dataview-sort][data-dataview-sort-field]').forEach((th) => {
    th.tabIndex = 0;
    const sort = () => {
      if (state.loading) return;
      const field = th.dataset.dataviewSortField || '';
      if (!field) return;
      const nextDir = state.sortField === field && state.sortDir === 'desc' ? 'asc' : 'desc';
      state.sortField = field;
      state.sortDir = nextDir;
      wrap.querySelectorAll('th[data-dataview-sort]').forEach((header) => {
        header.setAttribute('aria-sort', header === th ? (nextDir === 'desc' ? 'descending' : 'ascending') : 'none');
      });
      state.page = 1;
      requestDataviewTable(wrap, state);
    };
    th.addEventListener('click', sort);
    th.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter' || ev.key === ' ') {
        ev.preventDefault();
        sort();
      }
    });
  });
}

function isAjaxDataviewWrapper(wrap) {
  return wrap?.dataset.dataviewAction === 'renderDataviewTable' && Boolean(wrap.dataset.dataviewTable);
}

function buildDataviewRequestURL(wrap, state) {
  const url = new URL(location.pathname, location.origin);
  url.searchParams.set('action', wrap.dataset.dataviewAction || 'renderDataviewTable');
  url.searchParams.set('table', wrap.dataset.dataviewTable || '');
  const q = (wrap.querySelector('input.dataview-filter[data-dataview-filter]')?.value || '').trim();
  if (q) url.searchParams.set('q', q);
  wrap.querySelectorAll('select[data-dataview-filter]').forEach((select) => {
    const field = select.getAttribute('data-dataview-filter') || '';
    if (field && select.value) url.searchParams.append('filter.' + field, select.value);
  });
  wrap.querySelectorAll('select[multiple][data-dataview-filter-multi]').forEach((select) => {
    const field = select.getAttribute('data-dataview-filter-multi') || '';
    if (!field) return;
    Array.from(select.selectedOptions).forEach((option) => {
      if (option.value) url.searchParams.append('filter.' + field, option.value);
    });
  });
  if (state.sortField && state.sortDir) {
    url.searchParams.set('sort', state.sortField);
    url.searchParams.set('dir', state.sortDir);
  }
  return url;
}

function requestDataviewTable(wrap, state) {
  if (!isAjaxDataviewWrapper(wrap)) return;
  window.clearTimeout(state.debounceTimer);
  state.q = wrap.querySelector('input.dataview-filter[data-dataview-filter]')?.value || '';
  state.pageSize = wrap.querySelector('[data-dataview-page-size]')?.value || state.pageSize || '0';
  closeDataviewMultiMenus(wrap);
  if (state.abortController) state.abortController.abort();
  const requestId = state.requestId + 1;
  state.requestId = requestId;
  const controller = new AbortController();
  state.abortController = controller;
  state.loading = true;
  setDataviewLoading(wrap, true);
  const url = buildDataviewRequestURL(wrap, state);
  fetch(url.toString(), {signal: controller.signal, credentials: 'same-origin', headers: {'X-Requested-With': 'fetch'}})
    .then(async (response) => {
      const html = await response.text();
      if (requestId !== state.requestId) return;
      state.loading = false;
      state.abortController = null;
      replaceDataviewResponse(wrap, state, html, response.ok);
    })
    .catch((err) => {
      if (err?.name === 'AbortError' || requestId !== state.requestId) return;
      state.loading = false;
      state.abortController = null;
      replaceDataviewResponse(wrap, state, '', false);
    });
}

function replaceDataviewResponse(wrap, state, html, ok) {
  const fragment = document.createElement('template');
  fragment.innerHTML = html || '';
  const newWrap = fragment.content.querySelector('.dataview-table-wrap[data-dataview-action="renderDataviewTable"]');
  if (ok && newWrap) {
    state.page = 1;
    wrap.replaceWith(newWrap);
    initDataviewWrapper(newWrap, state);
    return;
  }
  const error = fragment.content.querySelector('.dataview-error') || fragment.content.firstElementChild || createDataviewErrorFragment();
  wrap.replaceWith(error);
}

function createDataviewErrorFragment() {
  const div = document.createElement('div');
  div.className = 'dataview-error';
  div.innerHTML = '<strong>Dataview non rendu</strong><p>Unable to update this table.</p>';
  return div;
}

function setDataviewLoading(wrap, loading) {
  wrap.classList.toggle('is-loading', Boolean(loading));
  wrap.toggleAttribute('data-dataview-loading', Boolean(loading));
  wrap.setAttribute('aria-busy', String(Boolean(loading)));
  wrap.querySelectorAll('input,select,button').forEach((control) => { control.disabled = Boolean(loading); });
  wrap.querySelectorAll('th[data-dataview-sort]').forEach((th) => { th.setAttribute('aria-disabled', String(Boolean(loading))); });
}

function initDataviewMultiFilters(wrap, state) {
  wrap.querySelectorAll('.dataview-multi-filter').forEach((multi) => {
    const button = multi.querySelector('.dataview-multi-btn[data-dataview-filter]');
    const menu = multi.querySelector('.dataview-multi-menu');
    const select = multi.querySelector('select[multiple][data-dataview-filter-multi]');
    if (!button || !menu || !select) return;
    button.dataset.dataviewFilterLabel = dataviewMultiLabel(button);
    syncDataviewMultiFromSelect(multi);
    button.addEventListener("click", () => {
      if (state.loading) return;
      if (button.getAttribute('aria-expanded') === 'true') closeDataviewMulti(multi, false);
      else openDataviewMulti(multi, true);
    });
    button.addEventListener('keydown', (ev) => {
      if (ev.key === 'ArrowDown') {
        ev.preventDefault();
        openDataviewMulti(multi, true, 0);
      } else if (ev.key === 'ArrowUp') {
        ev.preventDefault();
        openDataviewMulti(multi, true, getDataviewMultiInputs(menu).length - 1);
      }
    });
    menu.addEventListener('change', (ev) => {
      if (!ev.target.matches('input[type="checkbox"]')) return;
      syncDataviewMultiAfterToggle(multi, ev.target);
      state.page = 1;
      requestDataviewTable(wrap, state);
    });
    menu.addEventListener('keydown', (ev) => handleDataviewMultiKeydown(ev, multi, state));
  });
}

function dataviewMultiLabel(button) {
  const text = (button.textContent || '').trim();
  const match = text.match(/^(.*):\s*(?:All|\d+\s+selected)$/);
  return match ? match[1] : text.replace(/:\s*$/, '') || 'Filter';
}

function getDataviewMultiInputs(menu) {
  return Array.from(menu.querySelectorAll('input[type="checkbox"]'));
}

function openDataviewMulti(multi, focusFirst = false, focusIndex = 0) {
  const button = multi.querySelector('.dataview-multi-btn');
  const menu = multi.querySelector('.dataview-multi-menu');
  if (!button || !menu || button.disabled) return;
  closeDataviewMultiMenus(document, multi);
  menu.hidden = false;
  button.setAttribute('aria-expanded', 'true');
  positionDataviewMultiMenu(button, menu);
  const inputs = getDataviewMultiInputs(menu);
  inputs.forEach((input, index) => { input.tabIndex = index === Math.max(0, focusIndex) ? 0 : -1; });
  if (focusFirst && inputs.length) inputs[Math.max(0, Math.min(focusIndex, inputs.length - 1))].focus();
}

function closeDataviewMulti(multi, focusButton = false) {
  const button = multi.querySelector('.dataview-multi-btn');
  const menu = multi.querySelector('.dataview-multi-menu');
  if (!button || !menu) return;
  menu.hidden = true;
  menu.style.removeProperty('left');
  menu.style.removeProperty('top');
  button.setAttribute('aria-expanded', 'false');
  if (focusButton) button.focus();
}

function closeDataviewMultiMenus(root = document, exceptMulti = null) {
  const scope = root || document;
  scope.querySelectorAll?.('.dataview-multi-filter .dataview-multi-btn[aria-expanded="true"]').forEach((button) => {
    const multi = button.closest('.dataview-multi-filter');
    if (multi && multi !== exceptMulti) closeDataviewMulti(multi, false);
  });
}

function positionDataviewMultiMenu(button, menu) {
  const rect = button.getBoundingClientRect();
  menu.style.position = 'fixed';
  menu.style.left = '0px';
  menu.style.top = '0px';
  const menuRect = menu.getBoundingClientRect();
  const left = Math.max(8, Math.min(window.innerWidth - menuRect.width - 8, rect.left));
  const opensDown = rect.bottom + 6 + menuRect.height <= window.innerHeight - 8;
  const top = opensDown ? Math.min(window.innerHeight - menuRect.height - 8, rect.bottom + 6) : Math.max(8, rect.top - menuRect.height - 6);
  menu.style.left = left + 'px';
  menu.style.top = top + 'px';
}

function handleDataviewMultiKeydown(ev, multi, state) {
  const menu = multi.querySelector('.dataview-multi-menu');
  const inputs = getDataviewMultiInputs(menu);
  const current = inputs.indexOf(ev.target);
  if (ev.key === 'Escape') {
    ev.preventDefault();
    closeDataviewMulti(multi, true);
  } else if (ev.key === 'Tab') {
    closeDataviewMulti(multi, false);
  } else if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(ev.key)) {
    ev.preventDefault();
    let next = current;
    if (ev.key === 'ArrowDown') next = current + 1;
    else if (ev.key === 'ArrowUp') next = current - 1;
    else if (ev.key === 'Home') next = 0;
    else if (ev.key === 'End') next = inputs.length - 1;
    focusDataviewMultiInput(inputs, next);
  } else if (ev.key === 'Enter' || ev.key === ' ') {
    ev.preventDefault();
    if (current < 0 || state.loading) return;
    inputs[current].checked = !inputs[current].checked;
    syncDataviewMultiAfterToggle(multi, inputs[current]);
    state.page = 1;
    requestDataviewTable(multi.closest('.dataview-table-wrap'), state);
  }
}

function focusDataviewMultiInput(inputs, index) {
  if (!inputs.length) return;
  const next = (index + inputs.length) % inputs.length;
  inputs.forEach((input, i) => { input.tabIndex = i === next ? 0 : -1; });
  inputs[next].focus();
}

function syncDataviewMultiFromSelect(multi) {
  const select = multi.querySelector('select[multiple][data-dataview-filter-multi]');
  const selected = new Set(Array.from(select?.selectedOptions || []).map((option) => option.value));
  const inputs = getDataviewMultiInputs(multi.querySelector('.dataview-multi-menu'));
  inputs.forEach((input) => {
    input.checked = input.value ? selected.has(input.value) : selected.size === 0;
    input.setAttribute('aria-checked', String(input.checked));
    input.tabIndex = -1;
  });
  updateDataviewMultiButton(multi);
}

function syncDataviewMultiAfterToggle(multi, changedInput) {
  const select = multi.querySelector('select[multiple][data-dataview-filter-multi]');
  const inputs = getDataviewMultiInputs(multi.querySelector('.dataview-multi-menu'));
  const allInput = inputs.find((input) => input.value === '');
  if (changedInput.value === '') {
    inputs.forEach((input) => { input.checked = input.value === ''; });
    Array.from(select.options).forEach((option) => { option.selected = false; });
  } else {
    const selectedValues = new Set(inputs.filter((input) => input.value && input.checked).map((input) => input.value));
    if (allInput) allInput.checked = selectedValues.size === 0;
    Array.from(select.options).forEach((option) => { option.selected = selectedValues.has(option.value); });
  }
  inputs.forEach((input) => { input.setAttribute('aria-checked', String(input.checked)); });
  updateDataviewMultiButton(multi);
}

function updateDataviewMultiButton(multi) {
  const button = multi.querySelector('.dataview-multi-btn');
  const select = multi.querySelector('select[multiple][data-dataview-filter-multi]');
  if (!button || !select) return;
  const count = Array.from(select.selectedOptions).filter((option) => option.value).length;
  const label = button.dataset.dataviewFilterLabel || dataviewMultiLabel(button);
  button.textContent = label + ': ' + (count > 0 ? count + ' selected' : 'All');
}

function ensureDataviewGlobalHandlers() {
  if (dataviewGlobalHandlersReady) return;
  dataviewGlobalHandlersReady = true;
  document.addEventListener('click', (ev) => {
    document.querySelectorAll('.dataview-multi-filter .dataview-multi-btn[aria-expanded="true"]').forEach((button) => {
      const multi = button.closest('.dataview-multi-filter');
      if (multi && !multi.contains(ev.target)) closeDataviewMulti(multi, false);
    });
  });
  window.addEventListener('resize', () => closeDataviewMultiMenus());
  window.addEventListener('scroll', () => closeDataviewMultiMenus(), true);
}
