
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
let paletteLoaded = false;
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
      paletteLoaded = true;
      return paletteItems;
    })
    .catch((err) => {
      console.error(err);
      paletteLoadError = true;
      paletteLoaded = true;
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
  const editContext = currentPaletteEditContext();
  const actionItems = editContext ? buildPaletteActions(editContext) : [];
  if (!paletteLoaded && !actionItems.length) {
    paletteMatches = [];
    results.setAttribute('aria-busy', String(!paletteLoadError));
    results.innerHTML = paletteLoadError ? renderPaletteState('error', 'Unable to load search results.', 'Check the server, then reopen search.') : renderPaletteState('loading', 'Loading…', 'Preparing notes, tags, and favorites.');
    return;
  }
  results.setAttribute('aria-busy', 'false');
  const q = (query || '').trim().toLowerCase();
  const allItems = paletteItems.concat(actionItems);
  paletteMatches = allItems.filter((item) => {
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
  if (!item) return;
  if (item.kind === 'action' && item.action) {
    closePalette();
    item.action();
    return;
  }
  if (item.url) location.assign(item.url);
}
function currentPaletteEditContext() {
  return document.querySelector('[data-edit-surface][data-edit-csrf]') || document.querySelector('[data-edit-context][data-edit-csrf]') || document.querySelector('[data-trash-context][data-edit-csrf]') || document.querySelector('[data-global-edit-context][data-edit-csrf]');
}
function confirmDiscardEditBeforeAction() {
  if (!editState?.dirty) return true;
  if (!confirm('Discard unsaved changes?')) return false;
  editNavigationConfirmed = true;
  setTimeout(() => { editNavigationConfirmed = false; }, 1000);
  return true;
}
function buildPaletteActions(context) {
  const isNote = context.matches('[data-edit-surface]');
  const hasLocalContext = isNote || context.matches('[data-edit-context]');
  const currentPath = context?.dataset.editPath || '';
  const csrf = context?.dataset.editCsrf || '';
  const directory = context?.dataset.editDirectory || editDirname(currentPath);
  const canRename = context?.dataset.canRename === 'true';
  const canTrash = context?.dataset.canTrash === 'true';
  const dirtyDraftActive = Boolean(editState?.dirty);
  const actions = [];
  if (isNote && currentPath && csrf && !editState) {
    actions.push({title: 'Edit current', kind: 'action', path: currentPath, action: () => { const surface = document.querySelector('[data-edit-surface][data-edit-csrf]'); if (surface) openEditMode(surface); }});
  }
  if (hasLocalContext && directory && !dirtyDraftActive) {
    actions.push({title: 'New here', kind: 'action', path: directory, action: () => openCreateDialog(context)});
  }
  if (hasLocalContext && currentPath && csrf && !dirtyDraftActive && canRename) {
    actions.push({title: 'Rename current', kind: 'action', path: currentPath, action: () => openRenameDialog(context)});
  }
  if (hasLocalContext && currentPath && csrf && !dirtyDraftActive && canTrash) {
    actions.push({title: 'Move to Trash', kind: 'action', path: currentPath, action: () => moveEditPathToTrash(context)});
  }
  actions.push({title: 'Open Trash', kind: 'action', action: () => { if (confirmDiscardEditBeforeAction()) location.assign('/_trash'); }});
  return actions;
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

function resetNoteActionsMenu(menu) {
  menu.hidden = true;
  menu.style.removeProperty('left');
  menu.style.removeProperty('top');
}
function noteActionsTriggerForMenu(menu) {
  return menu.closest('[data-note-actions]')?.querySelector('[data-note-actions-toggle]');
}
function noteActionsMenuForTrigger(trigger) {
  return trigger.closest('[data-note-actions]')?.querySelector('[data-note-actions-menu]');
}
function closeNoteActionMenus(options = {}) {
  document.querySelectorAll('[data-note-actions-toggle][aria-expanded="true"]').forEach((trigger) => {
    trigger.setAttribute('aria-expanded', 'false');
    const menu = noteActionsMenuForTrigger(trigger);
    if (menu) resetNoteActionsMenu(menu);
    if (options.returnFocus && typeof trigger.focus === 'function') trigger.focus();
  });
}
function positionNoteActionsMenu(trigger, menu) {
  const buttonRect = trigger.getBoundingClientRect();
  menu.style.position = 'fixed';
  menu.style.left = '0px';
  menu.style.top = '0px';
  menu.hidden = false;
  const menuRect = menu.getBoundingClientRect();
  const left = Math.max(8, Math.min(window.innerWidth - menuRect.width - 8, buttonRect.right - menuRect.width));
  const opensDown = buttonRect.bottom + 6 + menuRect.height <= window.innerHeight - 8;
  const top = opensDown ? Math.min(window.innerHeight - menuRect.height - 8, buttonRect.bottom + 6) : Math.min(window.innerHeight - menuRect.height - 8, Math.max(8, buttonRect.top - menuRect.height - 6));
  menu.style.left = left + 'px';
  menu.style.top = top + 'px';
}
function focusFirstNoteAction(menu) {
  const first = menu.querySelector('a[href],button:not([disabled])');
  if (first && typeof first.focus === 'function') first.focus();
}
function moveNoteActionFocus(menu, direction) {
  const items = Array.from(menu.querySelectorAll('a[href],button:not([disabled])'));
  if (!items.length) return;
  const currentIndex = Math.max(0, items.indexOf(document.activeElement));
  const nextIndex = (currentIndex + direction + items.length) % items.length;
  items[nextIndex].focus();
}
function initNoteActionMenus() {
  document.addEventListener('click', (ev) => {
    const trigger = ev.target.closest('[data-note-actions-toggle]');
    if (trigger) {
      ev.preventDefault();
      const menu = noteActionsMenuForTrigger(trigger);
      if (!menu) return;
      const expanded = trigger.getAttribute('aria-expanded') === 'true';
      closeNoteActionMenus();
      trigger.setAttribute('aria-expanded', String(!expanded));
      if (expanded) {
        resetNoteActionsMenu(menu);
      } else {
        positionNoteActionsMenu(trigger, menu);
        setTimeout(() => focusFirstNoteAction(menu), 0);
      }
      return;
    }
    const menu = ev.target.closest('[data-note-actions-menu]');
    if (menu) {
      const action = ev.target.closest('a[href],button');
      if (action && !action.matches('[data-copy],[data-copy-path]')) setTimeout(() => closeNoteActionMenus(), 0);
      return;
    }
    closeNoteActionMenus();
  });
  document.addEventListener('keydown', (ev) => {
    const menu = ev.target.closest?.('[data-note-actions-menu]');
    if (ev.key === 'Escape') {
      if (document.querySelector('[data-note-actions-toggle][aria-expanded="true"]')) {
        ev.preventDefault();
        closeNoteActionMenus({returnFocus: true});
      }
      return;
    }
    if (!menu) return;
    if (ev.key === 'Tab') {
      closeNoteActionMenus();
      return;
    }
    if (ev.key === 'ArrowDown') { ev.preventDefault(); moveNoteActionFocus(menu, 1); }
    else if (ev.key === 'ArrowUp') { ev.preventDefault(); moveNoteActionFocus(menu, -1); }
  });
  window.addEventListener('resize', () => closeNoteActionMenus());
  window.addEventListener('scroll', () => closeNoteActionMenus(), true);
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
let editState = null;
let editNavigationConfirmed = false;

function editPathURL(path) {
  return String(path || '').split('/').map((part) => encodeURIComponent(part)).join('/');
}

function isTypingTarget(target) {
  if (!target) return false;
  if (target.isContentEditable) return true;
  return ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName);
}

async function readEditResponse(response) {
  let data = null;
  try { data = await response.json(); }
  catch {}
  if (!response.ok) {
    const err = new Error(data?.error || 'Request failed.');
    err.status = response.status;
    err.data = data;
    throw err;
  }
  return data || {};
}

function editJSONRequest(state, url, method, body) {
  return fetch(url, {
    method,
    credentials: 'same-origin',
    cache: 'no-store',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      'X-CSRF-Token': state.csrf,
    },
    body: JSON.stringify(body),
  }).then(readEditResponse);
}

function setEditTab(state, tab) {
  state.activeTab = tab === 'preview' ? 'preview' : 'source';
  state.workbench.querySelectorAll('[data-edit-tab]').forEach((button) => {
    const selected = button.dataset.editTab === state.activeTab;
    button.classList.toggle('is-active', selected);
    button.setAttribute('aria-selected', String(selected));
    button.tabIndex = selected ? 0 : -1;
  });
  state.workbench.querySelectorAll('[data-edit-panel]').forEach((panel) => {
    panel.hidden = panel.dataset.editPanel !== state.activeTab;
  });
}

function setEditMessage(state, kind, title, detail, actions) {
  const message = state.workbench.querySelector('[data-edit-message]');
  if (!message) return;
  message.hidden = false;
  message.className = 'edit-message edit-message-' + kind;
  message.replaceChildren();
  const strong = document.createElement('strong');
  strong.textContent = title;
  message.appendChild(strong);
  if (detail) {
    const p = document.createElement('p');
    p.textContent = detail;
    message.appendChild(p);
  }
  if (actions) {
    const row = document.createElement('div');
    row.className = 'edit-message-actions';
    actions.forEach((action) => row.appendChild(action));
    message.appendChild(row);
  }
}

function clearEditMessage(state) {
  const message = state.workbench.querySelector('[data-edit-message]');
  if (!message) return;
  message.hidden = true;
  message.replaceChildren();
  message.className = 'edit-message';
}

function editStatusText(state) {
  if (state.loading) return 'Loading source…';
  if (state.saving) return 'Saving…';
  if (state.previewing) return 'Rendering preview…';
  if (state.conflict) return 'Save conflict';
  return state.dirty ? 'Unsaved changes' : 'Saved';
}

function updateEditControls(state) {
  const textarea = state.workbench.querySelector('[data-edit-textarea]');
  const status = state.workbench.querySelector('[data-edit-status]');
  const previewButton = state.workbench.querySelector('[data-edit-preview]');
  const saveButton = state.workbench.querySelector('[data-edit-save]');
  const staleBadges = state.workbench.querySelectorAll('[data-edit-preview-stale]');
  const busy = state.loading || state.previewing || state.saving;
  state.workbench.setAttribute('aria-busy', String(busy));
  state.workbench.classList.toggle('is-dirty', state.dirty);
  state.workbench.classList.toggle('is-preview-stale', state.previewStale);
  state.workbench.classList.toggle('has-conflict', state.conflict);
  if (textarea) textarea.disabled = state.loading;
  if (status) status.textContent = editStatusText(state);
  if (previewButton) previewButton.disabled = state.loading || state.previewing || !state.baseHash;
  if (saveButton) saveButton.disabled = state.loading || state.saving || !state.baseHash;
  staleBadges.forEach((badge) => { badge.hidden = !state.previewStale; });
}

function markEditDirtyFromTextarea(state) {
  const textarea = state.workbench.querySelector('[data-edit-textarea]');
  if (!textarea) return;
  const value = textarea.value;
  state.dirty = value !== state.cleanContent;
  state.previewStale = state.lastPreviewContent !== null && value !== state.lastPreviewContent;
  updateEditControls(state);
}

function enhanceEditPreview(root) {
  // Dataview AJAX is intentionally not initialized inside edit preview panels.
  // The preview renders static content; interactive Dataview controls would
  // refetch the saved disk note, not the unsaved draft.
  if (window.MathJax?.typesetPromise) window.MathJax.typesetPromise([root]).catch(() => {});
  if (window.mermaid?.run) window.mermaid.run({nodes: root.querySelectorAll('.mermaid')}).catch(() => {});
}

async function fetchEditSource(state, replaceDraft) {
  state.loading = true;
  updateEditControls(state);
  try {
    const response = await fetch('/_api/edit/source/' + editPathURL(state.path), {
      method: 'GET',
      credentials: 'same-origin',
      cache: 'no-store',
      headers: {'Accept': 'application/json', 'X-CSRF-Token': state.csrf},
    });
    const data = await readEditResponse(response);
    if (editState !== state) return;
    state.path = data.path || state.path;
    state.baseHash = data.hash || '';
    state.cleanContent = data.content || '';
    state.lastPreviewContent = null;
    state.previewStale = false;
    state.dirty = false;
    state.conflict = false;
    state.workbench.querySelector('[data-edit-path-label]').textContent = state.path;
    state.workbench.querySelector('[data-edit-textarea]').value = state.cleanContent;
    state.workbench.querySelector('[data-edit-preview-content]').replaceChildren();
    state.workbench.querySelector('[data-edit-preview-empty]').hidden = false;
    clearEditMessage(state);
    if (replaceDraft) setEditTab(state, 'source');
    setTimeout(() => state.workbench.querySelector('[data-edit-textarea]')?.focus(), 0);
  } catch (err) {
    if (editState !== state) return;
    setEditMessage(state, 'error', 'Unable to load source.', err.message || 'Check the server, then try again.');
  } finally {
    if (editState === state) {
      state.loading = false;
      updateEditControls(state);
    }
  }
}

async function renderEditPreview(state) {
  if (!state || state.loading || state.previewing || !state.baseHash) return;
  const textarea = state.workbench.querySelector('[data-edit-textarea]');
  const content = textarea?.value || '';
  state.previewing = true;
  updateEditControls(state);
  try {
    const data = await editJSONRequest(state, '/_api/edit/preview', 'POST', {path: state.path, content});
    if (editState !== state) return;
    const preview = state.workbench.querySelector('[data-edit-preview-content]');
    preview.innerHTML = data.html || '';
    state.workbench.querySelector('[data-edit-preview-empty]').hidden = true;
    state.lastPreviewContent = content;
    state.previewStale = false;
    if (!state.conflict) clearEditMessage(state);
    setEditTab(state, 'preview');
    enhanceEditPreview(preview);
  } catch (err) {
    if (editState !== state) return;
    setEditMessage(state, 'error', 'Preview failed.', err.message || 'Try again after checking the source.');
  } finally {
    if (editState === state) {
      state.previewing = false;
      updateEditControls(state);
    }
  }
}

function showEditConflict(state) {
  state.conflict = true;
  const copy = document.createElement('button');
  copy.type = 'button';
  copy.className = 'btn ghost';
  copy.textContent = 'Copy draft';
  copy.addEventListener('click', async () => {
    await copyText(state.workbench.querySelector('[data-edit-textarea]')?.value || '');
    markCopied(copy, 'Draft copied');
  });
  const reload = document.createElement('button');
  reload.type = 'button';
  reload.className = 'btn ghost';
  reload.textContent = 'Reload disk';
  reload.addEventListener('click', () => {
    if (!confirm('Reload disk and replace this draft?')) return;
    fetchEditSource(state, true);
  });
  setEditMessage(state, 'conflict', 'Save conflict', 'This note changed on disk. Your draft is still here.', [copy, reload]);
  updateEditControls(state);
}

async function saveEditDraft(state) {
  if (!state || state.loading || state.saving || !state.baseHash) return;
  const textarea = state.workbench.querySelector('[data-edit-textarea]');
  const content = textarea?.value || '';
  state.saving = true;
  updateEditControls(state);
  try {
    const data = await editJSONRequest(state, '/_api/edit/save', 'PUT', {path: state.path, content, base_hash: state.baseHash});
    if (editState !== state) return;
    state.cleanContent = content;
    state.baseHash = data.hash || state.baseHash;
    state.dirty = false;
    state.conflict = false;
    state.saving = false;
    updateEditControls(state);
    editState = null;
    window.location.reload();
  } catch (err) {
    if (editState !== state) return;
    if (err.status === 409) showEditConflict(state);
    else setEditMessage(state, 'error', 'Save failed.', err.message || 'Your draft is still here.');
  } finally {
    if (editState === state) {
      state.saving = false;
      updateEditControls(state);
    }
  }
}

// Phase 2 edit CRUD dialogs: create, rename, and missing-link create.
let editModalState = null;

function normalizeEditRel(path) {
  const rel = String(path || '').replace(/^\/+|\/+$/g, '');
  return rel === '.' ? '' : rel;
}

function editURLForRel(path) {
  const rel = normalizeEditRel(path);
  return rel ? '/' + editPathURL(rel) : '/';
}

function editDirname(path) {
  const rel = normalizeEditRel(path);
  const idx = rel.lastIndexOf('/');
  return idx >= 0 ? rel.slice(0, idx) : '';
}

function editBasename(path) {
  const rel = normalizeEditRel(path);
  const idx = rel.lastIndexOf('/');
  return idx >= 0 ? rel.slice(idx + 1) : rel;
}

function stripEditableMarkdownExt(path) {
  return String(path || '').replace(/\.md$/i, '');
}

function joinEditPath(dir, name) {
  const cleanDir = normalizeEditRel(dir);
  const cleanName = String(name || '').replace(/^\/+/, '');
  if (!cleanDir) return cleanName;
  if (!cleanName) return cleanDir + '/';
  return cleanDir + '/' + cleanName;
}

function editSlugifyTitle(title) {
  const transliterated = String(title || '').replace(/[ÀÁÂÃÄÅàáâãäå]/g, 'a').replace(/[Ææ]/g, 'ae').replace(/[Çç]/g, 'c').replace(/[ÈÉÊËèéêë]/g, 'e').replace(/[ÌÍÎÏìíîï]/g, 'i').replace(/[Ðð]/g, 'd').replace(/[Ññ]/g, 'n').replace(/[ÒÓÔÕÖØòóôõöø]/g, 'o').replace(/[Œœ]/g, 'oe').replace(/[Šš]/g, 's').replace(/ß/g, 'ss').replace(/[Þþ]/g, 'th').replace(/[ÙÚÛÜùúûü]/g, 'u').replace(/[ÝŸýÿ]/g, 'y').replace(/[Žž]/g, 'z');
  return transliterated.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
}

function currentEditDirectory(context) {
  if (context?.dataset.editDirectory !== undefined) return normalizeEditRel(context.dataset.editDirectory);
  return editDirname(context?.dataset.editPath || currentCopyPath());
}

function editContextCSRF(context) {
  return context?.dataset.editCsrf || context?.closest('[data-edit-csrf]')?.dataset.editCsrf || '';
}

function editModalFocusable(panel) {
  return Array.from(panel.querySelectorAll('a[href],button:not([disabled]),input:not([disabled]),textarea:not([disabled]),select:not([disabled]),[tabindex]:not([tabindex="-1"])')).filter((el) => el.offsetParent !== null || el === document.activeElement);
}

function trapEditModalFocus(ev, overlay) {
  if (ev.key !== 'Tab') return;
  const panel = overlay.querySelector('.edit-modal-panel');
  const focusable = panel ? editModalFocusable(panel) : [];
  if (!focusable.length) return;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (ev.shiftKey && document.activeElement === first) {
    ev.preventDefault();
    last.focus();
  } else if (!ev.shiftKey && document.activeElement === last) {
    ev.preventDefault();
    first.focus();
  }
}

function openEditModal(title, className, trigger) {
  closeEditModal(true);
  const overlay = document.createElement('div');
  overlay.className = 'edit-modal ' + (className || '');
  overlay.innerHTML = '<div class="edit-modal-backdrop" data-edit-modal-close></div><section class="edit-modal-panel" role="dialog" aria-modal="true" aria-labelledby="edit-modal-title"><header><div><p class="edit-eyebrow">Edit mode</p><h2 id="edit-modal-title"></h2></div><button type="button" class="btn ghost icon-btn" data-edit-modal-close aria-label="Close">×</button></header><div class="edit-modal-body" data-edit-modal-body></div></section>';
  overlay.querySelector('#edit-modal-title').textContent = title;
  overlay.addEventListener('click', (ev) => {
    if (!ev.target.closest('[data-edit-modal-close]')) return;
    ev.preventDefault();
    closeEditModal();
  });
  overlay.addEventListener('keydown', (ev) => trapEditModalFocus(ev, overlay));
  document.body.appendChild(overlay);
  document.body.classList.add('edit-modal-open');
  editModalState = {overlay, busy: false, returnFocus: trigger || null};
  return editModalState;
}

function closeEditModal(force) {
  if (!editModalState) return;
  const returnFocus = editModalState.returnFocus;
  if (editModalState.busy && !force) return;
  editModalState.overlay.remove();
  editModalState = null;
  document.body.classList.remove('edit-modal-open');
  if (returnFocus && typeof returnFocus.focus === 'function') setTimeout(() => returnFocus.focus(), 0);
}

function setEditModalBusy(state, busy) {
  state.busy = Boolean(busy);
  state.overlay.setAttribute('aria-busy', String(state.busy));
  state.overlay.querySelectorAll('button,input,textarea,select').forEach((el) => {
    if (el.matches('[data-edit-modal-close]')) el.disabled = state.busy;
  });
}

function setEditModalMessage(state, kind, title, detail, actions) {
  const message = state.overlay.querySelector('[data-edit-modal-message]');
  if (!message) return;
  message.hidden = false;
  message.className = 'edit-message edit-message-' + kind;
  message.replaceChildren();
  const strong = document.createElement('strong');
  strong.textContent = title;
  message.appendChild(strong);
  if (detail) {
    const p = document.createElement('p');
    p.textContent = detail;
    message.appendChild(p);
  }
  if (actions) {
    const row = document.createElement('div');
    row.className = 'edit-message-actions';
    actions.forEach((action) => row.appendChild(action));
    message.appendChild(row);
  }
}

function clearEditModalMessage(state) {
  const message = state.overlay.querySelector('[data-edit-modal-message]');
  if (!message) return;
  message.hidden = true;
  message.replaceChildren();
  message.className = 'edit-message';
}

function editConfirmationText(data) {
  if (data?.requires_confirmation === 'missing_dirs') {
    const dirs = Array.isArray(data.missing_dirs) ? data.missing_dirs.join(', ') : '';
    return dirs ? 'Missing folders: ' + dirs : 'Missing folders need confirmation.';
  }
  if (data?.requires_confirmation === 'hidden') {
    const message = data.message || 'This path is hidden from normal listings.';
    return /direct URL/i.test(message) ? message : message + ' Hidden paths remain accessible by direct URL.';
  }
  return data?.message || data?.error || 'This action needs confirmation.';
}

function showEditModalConfirmation(state, data, onConfirm) {
  const confirmButton = document.createElement('button');
  confirmButton.type = 'button';
  confirmButton.className = 'btn primary';
  confirmButton.textContent = 'Continue';
  confirmButton.addEventListener('click', onConfirm);
  setEditModalMessage(state, 'confirm', 'Confirmation needed', editConfirmationText(data), [confirmButton]);
}

function editErrorText(err) {
  return err?.data?.message || err?.data?.error || err?.message || 'Request failed.';
}

function renderImpactPreview(container, impact) {
  if (!container) return;
  container.replaceChildren();
  const visible = Array.isArray(impact?.visible) ? impact.visible : [];
  const hidden = Array.isArray(impact?.hidden) ? impact.hidden : [];
  const untouched = Array.isArray(impact?.untouched) ? impact.untouched : [];
  const total = visible.length + hidden.length;
  const title = document.createElement('h3');
  title.textContent = 'Impact preview';
  container.appendChild(title);
  const summary = document.createElement('p');
  summary.className = 'edit-impact-summary';
  summary.textContent = total ? total + ' path' + (total === 1 ? '' : 's') + ' may change.' : 'No link rewrites.';
  container.appendChild(summary);
  const appendGroup = (label, items, hiddenGroup) => {
    const section = document.createElement('section');
    section.className = 'edit-impact-group';
    const heading = document.createElement('h4');
    heading.textContent = label + ' (' + items.length + ')';
    section.appendChild(heading);
    if (hiddenGroup && items.length) {
      const note = document.createElement('p');
      note.className = 'edit-impact-note';
      note.textContent = 'Hidden paths remain accessible by direct URL.';
      section.appendChild(note);
    }
    if (!items.length) {
      const empty = document.createElement('p');
      empty.className = 'empty-state';
      empty.textContent = 'No paths.';
      section.appendChild(empty);
    } else {
      const list = document.createElement('ul');
      list.className = 'edit-impact-list';
      items.forEach((item) => {
        const li = document.createElement('li');
        const strong = document.createElement('strong');
        strong.textContent = item.path || 'Unknown path';
        const details = document.createElement('small');
        const wiki = Number(item.wikilinks || 0);
        const md = Number(item.markdown_links || 0);
        const parts = [];
        if (wiki) parts.push(wiki + ' wikilink' + (wiki === 1 ? '' : 's'));
        if (md) parts.push(md + ' Markdown link' + (md === 1 ? '' : 's'));
        details.textContent = parts.join(', ') || 'Will be checked';
        li.append(strong, details);
        list.appendChild(li);
      });
      section.appendChild(list);
    }
    container.appendChild(section);
  };
  appendGroup('Visible', visible, false);
  appendGroup('Hidden', hidden, true);
  appendGroup('Unchanged', untouched, false);
}

function resetPreviewAction(state) {
  state.expectedHashes = null;
  state.previewData = null;
  const renameButton = state.overlay.querySelector('[data-edit-rename-execute]');
  const createButton = state.overlay.querySelector('[data-edit-missing-execute]');
  if (renameButton) renameButton.disabled = true;
  if (createButton) createButton.disabled = true;
  state.overlay.querySelector('[data-edit-impact]')?.replaceChildren();
  state.overlay.querySelector('[data-edit-template-hint]')?.replaceChildren();
}

function generatedCreatePath(directory, title, kind) {
  const trimmedTitle = String(title || '').trim();
  if (kind === 'folder') return joinEditPath(directory, trimmedTitle) + (trimmedTitle ? '/' : '');
  const slug = editSlugifyTitle(trimmedTitle);
  return slug ? joinEditPath(directory, slug + '.md') : joinEditPath(directory, '');
}

function selectedCreateKind(state) {
  const rawPath = state.overlay.querySelector('[name="edit-create-path"]')?.value.trim() || '';
  if (rawPath.endsWith('/')) return 'folder';
  return state.overlay.querySelector('[name="edit-create-kind"]:checked')?.value || 'note';
}

async function submitCreateDialog(state) {
  if (state.busy) return;
  const titleInput = state.overlay.querySelector('[name="edit-create-title"]');
  const pathInput = state.overlay.querySelector('[name="edit-create-path"]');
  const kind = selectedCreateKind(state);
  const title = titleInput?.value.trim() || '';
  const path = pathInput?.value.trim() || '';
  if (!title && !path) {
    setEditModalMessage(state, 'error', 'Path required', 'Add a title or path before creating.');
    return;
  }
  setEditModalBusy(state, true);
  try {
    const data = await editJSONRequest(state, '/_api/edit/create', 'POST', {kind, directory: state.directory, title, path, confirm_missing_dirs: state.confirmMissingDirs, confirm_hidden: state.confirmHidden});
    location.assign(editURLForRel(data.path));
  } catch (err) {
    if (err.status === 409 && err.data?.requires_confirmation) {
      showEditModalConfirmation(state, err.data, () => {
        if (err.data.requires_confirmation === 'missing_dirs') state.confirmMissingDirs = true;
        if (err.data.requires_confirmation === 'hidden') state.confirmHidden = true;
        submitCreateDialog(state);
      });
    } else {
      setEditModalMessage(state, 'error', 'Create failed', editErrorText(err));
    }
  } finally {
    setEditModalBusy(state, false);
  }
}

function openCreateDialog(trigger) {
  const context = trigger.closest('[data-edit-surface],[data-edit-context]');
  const csrf = editContextCSRF(context);
  if (!csrf) return;
  const state = openEditModal('New', 'edit-create-modal', trigger);
  Object.assign(state, {csrf, directory: currentEditDirectory(context), confirmMissingDirs: false, confirmHidden: false, pathFrozen: false});
  state.overlay.querySelector('[data-edit-modal-body]').innerHTML = '<form class="edit-form" data-edit-create-form><div class="edit-choice-row" role="radiogroup" aria-label="Create kind"><label><input type="radio" name="edit-create-kind" value="note" checked> Create note</label><label><input type="radio" name="edit-create-kind" value="folder"> Create folder</label></div><label>Title<input name="edit-create-title" autocomplete="off"></label><label>Path<input name="edit-create-path" autocomplete="off"></label><p class="edit-field-hint">Path ending in / creates a folder. Folder names are preserved.</p><div class="edit-message" data-edit-modal-message hidden></div><footer class="edit-modal-actions"><button type="submit" class="btn primary" data-edit-create-submit>Create note</button><button type="button" class="btn ghost" data-edit-modal-close>Cancel</button></footer></form>';
  const titleInput = state.overlay.querySelector('[name="edit-create-title"]');
  const pathInput = state.overlay.querySelector('[name="edit-create-path"]');
  const submitButton = state.overlay.querySelector('[data-edit-create-submit]');
  const updateGeneratedPath = () => {
    if (!state.pathFrozen) pathInput.value = generatedCreatePath(state.directory, titleInput.value, state.overlay.querySelector('[name="edit-create-kind"]:checked')?.value || 'note');
    submitButton.textContent = selectedCreateKind(state) === 'folder' ? 'Create folder' : 'Create note';
  };
  titleInput.addEventListener('input', () => { clearEditModalMessage(state); updateGeneratedPath(); });
  pathInput.addEventListener('input', () => { state.pathFrozen = true; clearEditModalMessage(state); updateGeneratedPath(); });
  state.overlay.querySelectorAll('[name="edit-create-kind"]').forEach((input) => input.addEventListener('change', () => { clearEditModalMessage(state); updateGeneratedPath(); }));
  state.overlay.querySelector('[data-edit-create-form]').addEventListener('submit', (ev) => { ev.preventDefault(); submitCreateDialog(state); });
  updateGeneratedPath();
  setTimeout(() => titleInput.focus(), 0);
}

function editContextKind(context) {
  if (context?.dataset.editKind) return context.dataset.editKind;
  if (context?.matches('[data-edit-context]')) return 'folder';
  return 'note';
}

function generatedRenamePath(currentPath, title, kind) {
  if (kind === 'folder') {
    const name = String(title || '').trim();
    return name ? joinEditPath(editDirname(currentPath), name) : currentPath;
  }
  const slug = editSlugifyTitle(title);
  if (!slug) return currentPath;
  return joinEditPath(editDirname(currentPath), slug + '.md');
}

function renameRequestBody(state, dryRun) {
  return {
    path: state.path,
    title: state.overlay.querySelector('[name="edit-rename-title"]')?.value.trim() || '',
    new_path: state.overlay.querySelector('[name="edit-rename-path"]')?.value.trim() || '',
    dry_run: dryRun,
    confirm_hidden: state.confirmHidden,
    expected_hashes: dryRun ? undefined : (state.expectedHashes || {}),
  };
}

async function previewRenameDialog(state) {
  if (state.busy) return;
  setEditModalBusy(state, true);
  try {
    const data = await editJSONRequest(state, '/_api/edit/rename', 'POST', renameRequestBody(state, true));
    state.expectedHashes = data.expected_hashes || {};
    state.previewData = data;
    clearEditModalMessage(state);
    renderImpactPreview(state.overlay.querySelector('[data-edit-impact]'), data.impact);
    state.overlay.querySelector('[data-edit-rename-execute]').disabled = false;
  } catch (err) {
    if (err.status === 409 && err.data?.requires_confirmation === 'hidden') {
      showEditModalConfirmation(state, err.data, () => { state.confirmHidden = true; previewRenameDialog(state); });
    } else {
      setEditModalMessage(state, 'error', 'Preview failed', editErrorText(err));
    }
  } finally {
    setEditModalBusy(state, false);
  }
}

async function executeRenameDialog(state) {
  if (state.busy) return;
  if (!state.expectedHashes) return;
  setEditModalBusy(state, true);
  try {
    const data = await editJSONRequest(state, '/_api/edit/rename', 'POST', renameRequestBody(state, false));
    location.assign(editURLForRel(data.new_path));
  } catch (err) {
    if (err.status === 409 && err.data?.requires_confirmation === 'hidden') {
      showEditModalConfirmation(state, err.data, () => { state.confirmHidden = true; executeRenameDialog(state); });
    } else {
      setEditModalMessage(state, 'error', 'Rename failed', editErrorText(err));
    }
  } finally {
    setEditModalBusy(state, false);
  }
}

function openRenameDialog(trigger) {
  const context = trigger.closest('[data-edit-surface]') || trigger.closest('[data-edit-context]');
  const csrf = editContextCSRF(context);
  const path = context?.dataset.editPath || '';
  if (!csrf || !path) return;
  const kind = editContextKind(context);
  const currentTitle = context.querySelector('h1')?.textContent?.trim() || stripEditableMarkdownExt(editBasename(path));
  const state = openEditModal('Rename', 'edit-rename-modal', trigger);
  Object.assign(state, {csrf, kind, path, confirmHidden: false, pathFrozen: false, expectedHashes: null, previewData: null});
  state.overlay.querySelector('[data-edit-modal-body]').innerHTML = '<form class="edit-form" data-edit-rename-form><dl class="edit-path-facts"><div><dt>Current path</dt><dd data-edit-current-path></dd></div></dl><label>Title<input name="edit-rename-title" autocomplete="off"></label><label>Raw path<input name="edit-rename-path" autocomplete="off"></label><div class="edit-message" data-edit-modal-message hidden></div><div class="edit-impact" data-edit-impact></div><footer class="edit-modal-actions"><button type="button" class="btn" data-edit-rename-preview>Preview impact</button><button type="button" class="btn primary" data-edit-rename-execute disabled>Rename</button><button type="button" class="btn ghost" data-edit-modal-close>Cancel</button></footer></form>';
  state.overlay.querySelector('[data-edit-current-path]').textContent = path;
  const titleInput = state.overlay.querySelector('[name="edit-rename-title"]');
  const pathInput = state.overlay.querySelector('[name="edit-rename-path"]');
  titleInput.value = currentTitle;
  pathInput.value = path;
  titleInput.addEventListener('input', () => {
    clearEditModalMessage(state);
    if (!state.pathFrozen) pathInput.value = generatedRenamePath(path, titleInput.value, state.kind);
    resetPreviewAction(state);
  });
  pathInput.addEventListener('input', () => { state.pathFrozen = true; clearEditModalMessage(state); resetPreviewAction(state); });
  state.overlay.querySelector('[data-edit-rename-preview]').addEventListener('click', () => previewRenameDialog(state));
  state.overlay.querySelector('[data-edit-rename-execute]').addEventListener('click', () => executeRenameDialog(state));
  setTimeout(() => titleInput.focus(), 0);
}

function renderMissingTemplateHint(container, data) {
  if (!container) return;
  container.replaceChildren();
  if (!data?.template_path && !data?.content) return;
  const title = document.createElement('h3');
  title.textContent = 'New note content';
  container.appendChild(title);
  if (data.template_path) {
    const p = document.createElement('p');
    p.textContent = 'Template: ' + data.template_path;
    container.appendChild(p);
  }
  if (data.content) {
    const pre = document.createElement('pre');
    pre.className = 'edit-template-snippet';
    pre.textContent = String(data.content).slice(0, 320);
    container.appendChild(pre);
  }
}

function missingCreateRequestBody(state, dryRun) {
  return {
    target: state.target,
    source_path: state.sourcePath,
    dry_run: dryRun,
    confirm_missing_dirs: state.confirmMissingDirs,
    confirm_hidden: state.confirmHidden,
    expected_hashes: dryRun ? undefined : (state.expectedHashes || {}),
  };
}

async function previewMissingCreateDialog(state) {
  if (state.busy) return;
  setEditModalBusy(state, true);
  try {
    const data = await editJSONRequest(state, '/_api/edit/missing-link-create', 'POST', missingCreateRequestBody(state, true));
    state.expectedHashes = data.expected_hashes || {};
    state.previewData = data;
    clearEditModalMessage(state);
    state.overlay.querySelector('[data-edit-created-path]').textContent = data.path || '';
    renderImpactPreview(state.overlay.querySelector('[data-edit-impact]'), data.impact);
    renderMissingTemplateHint(state.overlay.querySelector('[data-edit-template-hint]'), data);
    state.overlay.querySelector('[data-edit-missing-execute]').disabled = false;
  } catch (err) {
    if (err.status === 409 && err.data?.requires_confirmation) {
      showEditModalConfirmation(state, err.data, () => {
        if (err.data.requires_confirmation === 'missing_dirs') state.confirmMissingDirs = true;
        if (err.data.requires_confirmation === 'hidden') state.confirmHidden = true;
        previewMissingCreateDialog(state);
      });
    } else {
      setEditModalMessage(state, 'error', 'Preview failed', editErrorText(err));
    }
  } finally {
    setEditModalBusy(state, false);
  }
}

async function executeMissingCreateDialog(state) {
  if (state.busy) return;
  if (!state.expectedHashes) return;
  setEditModalBusy(state, true);
  try {
    const data = await editJSONRequest(state, '/_api/edit/missing-link-create', 'POST', missingCreateRequestBody(state, false));
    location.assign(editURLForRel(data.path));
  } catch (err) {
    if (err.status === 409 && err.data?.requires_confirmation) {
      showEditModalConfirmation(state, err.data, () => {
        if (err.data.requires_confirmation === 'missing_dirs') state.confirmMissingDirs = true;
        if (err.data.requires_confirmation === 'hidden') state.confirmHidden = true;
        executeMissingCreateDialog(state);
      });
    } else {
      setEditModalMessage(state, 'error', 'Create failed', editErrorText(err));
    }
  } finally {
    setEditModalBusy(state, false);
  }
}

function openMissingCreateDialog(trigger) {
  const context = trigger.closest('[data-missing-create-context]');
  const csrf = editContextCSRF(context);
  const target = context?.dataset.missingTarget || '';
  const sourcePath = context?.dataset.missingSource || '';
  if (!csrf || !target || !sourcePath) return;
  const state = openEditModal('Create this note', 'edit-missing-create-modal', trigger);
  Object.assign(state, {csrf, target, sourcePath, confirmMissingDirs: false, confirmHidden: false, expectedHashes: null, previewData: null});
  state.overlay.querySelector('[data-edit-modal-body]').innerHTML = '<div class="edit-form"><dl class="edit-path-facts"><div><dt>Target</dt><dd data-edit-missing-target></dd></div><div><dt>Source</dt><dd data-edit-missing-source></dd></div><div><dt>Path</dt><dd data-edit-created-path>Preview required</dd></div></dl><div class="edit-message" data-edit-modal-message hidden></div><div class="edit-template-hint" data-edit-template-hint></div><div class="edit-impact" data-edit-impact></div><footer class="edit-modal-actions"><button type="button" class="btn" data-edit-missing-preview>Preview impact</button><button type="button" class="btn primary" data-edit-missing-execute disabled>Create note</button><button type="button" class="btn ghost" data-edit-modal-close>Cancel</button></footer></div>';
  state.overlay.querySelector('[data-edit-missing-target]').textContent = target;
  state.overlay.querySelector('[data-edit-missing-source]').textContent = sourcePath;
  state.overlay.querySelector('[data-edit-missing-preview]').addEventListener('click', () => previewMissingCreateDialog(state));
  state.overlay.querySelector('[data-edit-missing-execute]').addEventListener('click', () => executeMissingCreateDialog(state));
  setTimeout(() => state.overlay.querySelector('[data-edit-missing-preview]')?.focus(), 0);
}

function editActionMessageNode(scope) {
  if (!scope) return null;
  let message = scope.querySelector('[data-edit-action-message],[data-trash-message]');
  if (message) return message;
  message = document.createElement('div');
  message.className = 'edit-message edit-action-message';
  message.dataset.editActionMessage = '';
  message.hidden = true;
  const header = scope.querySelector(':scope > header') || scope.querySelector('header');
  if (header) header.after(message);
  else scope.prepend(message);
  return message;
}

function clearEditActionMessage(scope) {
  const message = scope?.querySelector('[data-edit-action-message],[data-trash-message]');
  if (!message) return;
  message.hidden = true;
  message.replaceChildren();
  message.className = message.dataset.trashMessage !== undefined ? 'edit-message trash-message' : 'edit-message edit-action-message';
}

function setEditActionMessage(scope, kind, title, detail) {
  const message = editActionMessageNode(scope);
  if (!message) return;
  message.hidden = false;
  const baseClass = message.dataset.trashMessage !== undefined ? 'edit-message trash-message' : 'edit-message edit-action-message';
  message.className = baseClass + ' edit-message-' + kind;
  message.replaceChildren();
  const strong = document.createElement('strong');
  strong.textContent = title;
  message.appendChild(strong);
  if (detail) {
    const p = document.createElement('p');
    p.textContent = detail;
    message.appendChild(p);
  }
}

function editTrashActionContext(button) {
  return button.closest('[data-edit-surface],[data-edit-context],[data-trash-context]');
}

function editTrashActionPath(button, context) {
  return normalizeEditRel(button.dataset.editTrashPath || context?.dataset.editPath || context?.dataset.editDirectory || currentCopyPath());
}

async function moveEditPathToTrash(button) {
  const context = editTrashActionContext(button);
  const csrf = editContextCSRF(context);
  const path = editTrashActionPath(button, context);
  const kind = button.dataset.editTrashKind || editContextKind(context);
  if (!csrf || !path) return;
  if (!confirm('Move this ' + kind + ' to Trash?')) return;
  clearEditActionMessage(context);
  if ('disabled' in button) button.disabled = true;
  button.setAttribute('aria-busy', 'true');
  try {
    await editJSONRequest({csrf}, '/_api/edit/trash', 'POST', {path});
    location.assign('/_trash');
  } catch (err) {
    const detail = kind === 'folder' && err.status === 409 ? 'This folder is not empty. Empty it before moving it to Trash.' : editErrorText(err);
    setEditActionMessage(context, 'error', 'Move to Trash failed', detail);
  } finally {
    if ('disabled' in button) button.disabled = false;
    button.removeAttribute('aria-busy');
  }
}

function trashEntryState(button) {
  const entry = button.closest('[data-trash-entry]');
  const context = button.closest('[data-trash-context]');
  return {
    context,
    csrf: editContextCSRF(context),
    entry,
    originalPath: entry?.dataset.trashOriginalPath || '',
    snapshot: entry?.dataset.trashSnapshot || '',
  };
}

function setTrashEntryBusy(entry, busy) {
  if (!entry) return;
  entry.setAttribute('aria-busy', String(Boolean(busy)));
  entry.querySelectorAll('button').forEach((button) => { button.disabled = Boolean(busy); });
}

function promptTrashRestorePath(defaultPath, message) {
  const value = prompt((message ? message + '\n' : '') + 'Restore as relative path:', defaultPath || '');
  if (value === null) return null;
  return normalizeEditRel(value.trim());
}

async function restoreTrashEntry(button, restorePath) {
  const state = trashEntryState(button);
  if (!state.csrf || !state.snapshot) return;
  clearEditActionMessage(state.context);
  setTrashEntryBusy(state.entry, true);
  try {
    const body = {snapshot: state.snapshot};
    const targetPath = restorePath || state.originalPath;
    if (targetPath) body.restore_path = targetPath;
    await editJSONRequest({csrf: state.csrf}, '/_api/edit/trash/restore', 'POST', body);
    location.reload();
  } catch (err) {
    if (err.status === 409 && err.data?.requires_confirmation === 'restore_as') {
      const existsAt = err.data.exists_at || state.originalPath;
      const nextPath = promptTrashRestorePath(err.data.original_path || state.originalPath, 'A path already exists at ' + existsAt + '.');
      if (nextPath) await restoreTrashEntry(button, nextPath);
    } else {
      setEditActionMessage(state.context, 'error', 'Restore failed', editErrorText(err));
    }
  } finally {
    setTrashEntryBusy(state.entry, false);
  }
}

function restoreTrashEntryAs(button) {
  const state = trashEntryState(button);
  const nextPath = promptTrashRestorePath(state.originalPath, 'Choose a new path for this snapshot.');
  if (!nextPath) return;
  restoreTrashEntry(button, nextPath);
}

function createEditWorkbench(state) {
  const workbench = document.createElement('section');
  workbench.className = 'edit-workbench';
  workbench.dataset.editWorkbench = '';
  workbench.setAttribute('aria-label', 'Edit note');
  workbench.innerHTML = '<header class="edit-workbench-header"><div><p class="edit-eyebrow">Editing</p><h1 data-edit-path-label></h1></div><span class="edit-status" data-edit-status role="status">Loading source…</span></header><div class="edit-tabs" role="tablist" aria-label="Edit panels"><button type="button" role="tab" class="is-active" aria-selected="true" data-edit-tab="source">Source</button><button type="button" role="tab" aria-selected="false" tabindex="-1" data-edit-tab="preview">Preview <span class="edit-stale-badge" data-edit-preview-stale hidden>Preview stale</span></button></div><div class="edit-message" data-edit-message hidden></div><div class="edit-panel edit-source-panel" data-edit-panel="source"><textarea data-edit-textarea aria-label="Markdown source" spellcheck="false" disabled></textarea></div><div class="edit-panel edit-preview-panel content" data-edit-panel="preview" hidden><p class="edit-preview-empty" data-edit-preview-empty>Press Preview to render the current source.</p><div data-edit-preview-content></div></div><footer class="edit-toolbar"><button type="button" class="btn" data-edit-preview>Preview <span class="edit-stale-badge" data-edit-preview-stale hidden>Preview stale</span></button><button type="button" class="btn primary" data-edit-save>Save</button><button type="button" class="btn ghost" data-edit-cancel>Cancel</button></footer>';
  workbench.querySelector('[data-edit-path-label]').textContent = state.path;
  workbench.querySelectorAll('[data-edit-tab]').forEach((tabButton) => tabButton.addEventListener('click', () => setEditTab(state, tabButton.dataset.editTab)));
  workbench.querySelector('[data-edit-preview]').addEventListener('click', () => renderEditPreview(state));
  workbench.querySelector('[data-edit-save]').addEventListener('click', () => saveEditDraft(state));
  workbench.querySelector('[data-edit-cancel]').addEventListener('click', () => cancelEditMode());
  workbench.querySelector('[data-edit-textarea]').addEventListener('input', () => markEditDirtyFromTextarea(state));
  return workbench;
}

function openEditMode(surface) {
  if (editState || !surface?.dataset.editCsrf) return;
  const readShell = document.createElement('div');
  readShell.dataset.editReadShell = '';
  readShell.hidden = true;
  while (surface.firstChild) readShell.appendChild(surface.firstChild);
  const state = {
    activeTab: 'source',
    baseHash: '',
    cleanContent: '',
    conflict: false,
    csrf: surface.dataset.editCsrf,
    dirty: false,
    lastPreviewContent: null,
    loading: false,
    path: surface.dataset.editPath || currentCopyPath(),
    previewing: false,
    previewStale: false,
    readShell,
    saving: false,
    surface,
    workbench: null,
  };
  state.workbench = createEditWorkbench(state);
  editState = state;
  surface.classList.add('is-editing');
  document.body.classList.add('edit-mode-active');
  surface.append(readShell, state.workbench);
  updateEditControls(state);
  fetchEditSource(state, false);
}

function cancelEditMode() {
  const state = editState;
  if (!state) return;
  if (state.dirty && !confirm('Discard unsaved changes?')) return;
  editState = null;
  state.workbench.remove();
  while (state.readShell.firstChild) state.surface.appendChild(state.readShell.firstChild);
  state.readShell.remove();
  state.surface.classList.remove('is-editing');
  document.body.classList.remove('edit-mode-active');
  setTimeout(() => state.surface.querySelector('[data-edit-open]')?.focus(), 0);
}

function shouldGuardEditNavigation(link, ev) {
  if (!link || !editState?.dirty) return false;
  if (link.target && link.target !== '_self') return false;
  if (link.hasAttribute('download')) return false;
  // Modified clicks (Ctrl/Cmd/Shift/Alt or middle mouse) open new tabs;
  // they should not trigger the same-tab dirty guard.
  if (ev && (ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey)) return false;
  const raw = link.getAttribute('href') || '';
  if (!raw || raw.startsWith('#') || raw.startsWith('javascript:') || raw.startsWith('mailto:')) return false;
  const url = new URL(raw, document.baseURI);
  return url.origin === location.origin;
}

function initEditMode() {
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-edit-open]');
    if (!button) return;
    ev.preventDefault();
    openEditMode(button.closest('[data-edit-surface]'));
  });
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-edit-new]');
    if (!button) return;
    ev.preventDefault();
    openCreateDialog(button);
  });
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-edit-rename]');
    if (!button) return;
    ev.preventDefault();
    openRenameDialog(button);
  });
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-edit-missing-create]');
    if (!button) return;
    ev.preventDefault();
    openMissingCreateDialog(button);
  });
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-edit-trash]');
    if (!button) return;
    ev.preventDefault();
    moveEditPathToTrash(button);
  });
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-trash-restore]');
    if (!button) return;
    ev.preventDefault();
    restoreTrashEntry(button);
  });
  document.addEventListener('click', (ev) => {
    const button = ev.target.closest('[data-trash-restore-as]');
    if (!button) return;
    ev.preventDefault();
    restoreTrashEntryAs(button);
  });
  document.addEventListener('click', (ev) => {
    const link = ev.target.closest('a[href]');
    if (!shouldGuardEditNavigation(link, ev)) return;
    if (confirm('Leave this note and discard unsaved changes?')) {
      editNavigationConfirmed = true;
      setTimeout(() => { editNavigationConfirmed = false; }, 1000);
      return;
    }
    ev.preventDefault();
    ev.stopPropagation();
  }, true);
  document.addEventListener('keydown', (ev) => {
    if (editModalState && ev.key === 'Escape') { ev.preventDefault(); closeEditModal(); return; }
    if (editState) {
      if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 's') { ev.preventDefault(); saveEditDraft(editState); }
      else if ((ev.metaKey || ev.ctrlKey) && ev.key === 'Enter') { ev.preventDefault(); renderEditPreview(editState); }
      else if (ev.key === 'Escape') { ev.preventDefault(); cancelEditMode(); }
      return;
    }
    if (ev.key.toLowerCase() !== 'e' || ev.metaKey || ev.ctrlKey || ev.altKey || ev.repeat || isTypingTarget(ev.target)) return;
    const surface = document.querySelector('[data-edit-surface][data-edit-csrf]');
    if (!surface) return;
    ev.preventDefault();
    openEditMode(surface);
  });
  window.addEventListener('beforeunload', (ev) => {
    if (editNavigationConfirmed) return;
    if (!editState?.dirty) return;
    ev.preventDefault();
    ev.returnValue = '';
  });
}

document.addEventListener('DOMContentLoaded', () => { applyInitialPreferences(); initThemePicker(); initReadingControls(); initSettingsModal(); initCommandPalette(); restoreSidebarState(); restorePanelState(); initMobileSidebar(); initNoteActionMenus(); initListFilters(); initHomepageProjectFilter(); initTodoActions(); initTodoFilters(); initEditMode();
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
    setTimeout(() => closeNoteActionMenus({returnFocus: true}), 150);
    return;
  }
  const pathCopy = ev.target.closest('[data-copy-path]');
  if (pathCopy) {
    ev.preventDefault();
    await copyText(currentCopyPath());
    markCopied(pathCopy, 'Path copied');
    setTimeout(() => closeNoteActionMenus({returnFocus: true}), 150);
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
