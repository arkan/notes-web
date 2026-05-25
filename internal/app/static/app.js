
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
  const tagList = shell.querySelector('[data-todo-tag-list]');
  const rows = Array.from(shell.querySelectorAll('.task-row'));
  populateTodoSelect(tag, uniqueTodoValues(rows.flatMap((row) => (row.dataset.tags || '').trim().split(/\s+/).filter(Boolean))), 'All tags', (value) => '#' + value);
  restoreTodoFilterState({ tag, priority, date, group, hideNoDate, hideDone });
  function persistTodoFilters() {
    writeTodoFilterState({ tag: tag?.value || '', priority: priority?.value || '', date: date?.value || '', group: group?.value || 'Due date', hideNoDate: Boolean(hideNoDate?.checked), hideDone: Boolean(hideDone?.checked) });
  }
  function syncTodoTagList(selectedTag) {
    tagList?.querySelectorAll('[data-todo-tag-value]').forEach((button) => {
      const active = button.dataset.todoTagValue === selectedTag;
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    });
  }
  function apply() {
    const q = (search?.value || '').trim().toLowerCase();
    const selectedTag = tag?.value || '';
    syncTodoTagList(selectedTag);
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
  tagList?.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-todo-tag-value]');
    if (!button || !tagList.contains(button)) return;
    if (tag) tag.value = button.dataset.todoTagValue || '';
    persistTodoFilters();
    apply();
  });
  [tag, priority, date, group, hideNoDate, hideDone].forEach((el) => {
    el?.addEventListener('input', () => { persistTodoFilters(); apply(); });
    el?.addEventListener('change', () => { persistTodoFilters(); apply(); });
  });
  apply();
}
function uniqueTodoValues(values) {
  return [...new Set(values.filter(Boolean))].sort((a, b) => a.localeCompare(b));
}
function populateTodoSelect(select, values, emptyLabel, renderLabel = (value) => value) {
  if (!select || select.dataset.populated === 'true') return;
  if (select.options.length > 1) {
    select.dataset.populated = 'true';
    return;
  }
  select.innerHTML = '<option value="">' + escapeHTML(emptyLabel) + '</option>' + values.map((value) => '<option value="' + escapeHTML(value) + '">' + escapeHTML(renderLabel(value)) + '</option>').join('');
  select.dataset.populated = 'true';
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
  const total = rows.filter((row) => todoRowMatchesFilters(row, filters, false)).length;
  shell.querySelectorAll('[data-todo-tag-value]').forEach((button) => {
    const value = button.dataset.todoTagValue || '';
    if (!value) {
      button.hidden = false;
      button.querySelector('span').textContent = String(total);
      return;
    }
    const count = counts.get(value) || 0;
    button.hidden = count === 0;
    button.querySelector('span').textContent = String(counts.get(value) || 0);
  });
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
document.addEventListener('DOMContentLoaded', () => { applyInitialPreferences(); initThemePicker(); initReadingControls(); initSettingsModal(); initCommandPalette(); restoreSidebarState(); restorePanelState(); initMobileSidebar(); initListFilters(); initTodoActions(); initTodoFilters(); });
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
  const link = ev.target.closest('[data-copy-link]');
  if (link) {
    ev.preventDefault();
    await copyText(location.href);
    markCopied(link, 'Link copied');
  }
});
