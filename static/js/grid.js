(function () {
  const cells = [...document.querySelectorAll('[data-edit]')];
  const form = document.getElementById('grid-form');
  const save = document.querySelector('[data-grid-save]');
  const discard = document.querySelector('[data-grid-discard]');
  const counter = document.querySelector('[data-grid-count]');

  if (cells.length === 0 || !form || !save || !discard || !counter) {
    return;
  }

  const table = cells[0].closest('table');
  const rows = [...table.querySelectorAll('tbody > tr[data-key]')];
  const engine = form.dataset.gridEngine || 'libsql';
  const staged = new Map();
  const originals = new Map();
  const columns = new Map();
  let editing = null;
  let submitting = false;

  const adapters = {
    libsql: {
      fields: ['column', 'key', 'value', 'clear'],
      field: function (change, name) {
        return name === 'clear' ? (change.state === 'null' ? 'on' : '') : change[name];
      },
    },
    postgres: {
      fields: ['column', 'key', 'value', 'state'],
      field: function (change, name) { return name === 'value' ? JSON.stringify(change.value) : change[name]; },
    },
  };
  const adapter = adapters[engine];

  function unavailable(message) {
    const notice = document.createElement('div');
    notice.className = 'explorer-key-note field-note';
    notice.setAttribute('role', 'alert');
    notice.textContent = message;
    table.closest('.sheet').before(notice);
    save.disabled = true;
    discard.disabled = true;
    console.error(message);
  }

  try {
    if (!adapter) throw new Error('Unknown grid engine.');
    const payload = document.querySelector('[data-grid-values]');
    const data = payload ? JSON.parse(payload.textContent) : null;
    if (engine === 'postgres' && !data) throw new Error('PostgreSQL grid values are missing.');
    const values = new Map();
    if (data) {
      data.Columns.forEach(function (column) { columns.set(column.Name, column); });
      data.Rows.forEach(function (row) {
        values.set(row.Key, new Map(row.Cells.map(function (cell) { return [cell.Column, cell]; })));
      });
    }
    cells.forEach(function (cell) {
      const node = cell.querySelector('.cell-text');
      const row = values.get(cell.parentElement.dataset.key);
      const value = row && row.get(cell.dataset.cell);
      if (data && (!value || !columns.has(cell.dataset.cell))) throw new Error('Grid values do not match the displayed rows.');
      originals.set(cell, {
        node: node,
        value: value ? value.Value : (node.classList.contains('null') ? '' : node.textContent),
        isNull: value ? value.IsNull : node.classList.contains('null'),
        text: node.textContent,
        title: node.getAttribute('title'),
      });
    });
  } catch (error) {
    unavailable('The grid editor is unavailable. Reload the page. ' + error.message);
    return;
  }

  function unchanged(cell, value, state) {
    const original = originals.get(cell);
    return state === 'keep' || (state === 'null' && original.isNull) ||
      (state === 'value' && !original.isNull && value === original.value);
  }

  function tally() {
    let count = staged.size;
    if (editing) {
      if (staged.has(editing.cell)) count -= 1;
      if (!unchanged(editing.cell, editing.typed, editing.state)) count += 1;
    }
    counter.textContent = String(count);
    save.disabled = count === 0;
    discard.disabled = count === 0 && staged.size === 0;
  }

  function show(cell, value, state) {
    const node = originals.get(cell).node;
    node.textContent = state === 'null' ? 'NULL' : (state === 'default' ? 'DEFAULT' : value);
    node.classList.toggle('null', state === 'null');
    node.classList.toggle('is-default', state === 'default');
    if (engine === 'postgres') node.title = node.textContent;
  }

  function revert(cell) {
    const original = originals.get(cell);
    original.node.textContent = original.text;
    original.node.classList.toggle('null', original.isNull);
    original.node.classList.remove('is-default');
    if (original.title === null) original.node.removeAttribute('title');
    else original.node.setAttribute('title', original.title);
    staged.delete(cell);
    cell.classList.remove('is-changed');
    tally();
  }

  function stage(cell, value, state) {
    if (unchanged(cell, value, state)) {
      revert(cell);
      return;
    }
    staged.set(cell, { column: cell.dataset.cell, key: cell.parentElement.dataset.key, value: value, state: state });
    show(cell, value, state);
    cell.classList.add('is-changed');
    tally();
  }

  function stop(keep) {
    if (!editing) return;
    const open = editing;
    editing = null;
    open.cell.classList.remove('is-editing');
    open.editor.remove();
    originals.get(open.cell).node.style.visibility = '';
    if (keep) stage(open.cell, open.typed, open.state);
    else tally();
  }

  function positionEditor() {
    if (!editing || engine !== 'postgres') return;
    const bounds = editing.cell.getBoundingClientRect();
    const editor = editing.editor;
    const width = Math.min(360, window.innerWidth - 32);
    editor.style.width = width + 'px';
    editor.style.left = Math.max(16, Math.min(bounds.left, window.innerWidth - width - 16)) + 'px';
    const height = editor.getBoundingClientRect().height;
    const below = bounds.bottom + 4;
    editor.style.top = Math.max(16, Math.min(below + height <= window.innerHeight - 16 ? below : bounds.top - height - 4, window.innerHeight - height - 16)) + 'px';
  }

  function edit(cell) {
    if (editing && editing.cell === cell) return;
    stop(true);
    const original = originals.get(cell);
    const change = staged.get(cell);
    const editor = document.createElement(engine === 'postgres' ? 'div' : 'input');
    const input = engine === 'postgres' ? document.createElement('textarea') : editor;
    input.value = change ? change.value : original.value;
    input.setAttribute('aria-label', 'Edit ' + cell.dataset.cell);
    editing = { cell: cell, editor: editor, input: input, typed: change ? change.value : original.value, state: change ? change.state : (engine === 'postgres' ? 'keep' : 'value') };
    const open = editing;
    cell.classList.add('is-editing');

    if (engine === 'postgres') {
      const column = columns.get(cell.dataset.cell);
      editor.className = 'explorer-grid-editor';
      editor.setAttribute('role', 'dialog');
      editor.setAttribute('aria-label', 'Edit ' + cell.dataset.cell);
      const label = document.createElement('strong');
      label.className = 'label';
      label.textContent = cell.dataset.cell + ' · ' + column.Kind;
      editor.appendChild(label);
      const state = document.createElement('select');
      state.className = 'input is-small';
      state.setAttribute('aria-label', cell.dataset.cell + ' field state');
      [['keep', 'Keep current value'], ['value', 'Value (empty text is allowed)'], ['null', 'NULL'], ['default', 'DEFAULT']].forEach(function (choice) {
        if (choice[0] === 'null' && column.NotNull) return;
        const option = document.createElement('option');
        option.value = choice[0];
        option.textContent = choice[1];
        state.appendChild(option);
      });
      state.value = open.state;
      editor.appendChild(state);
      input.className = 'input';
      input.rows = 3;
      editor.appendChild(input);
      const guidance = document.createElement('span');
      guidance.className = 'field-note';
      guidance.textContent = 'Enter adds a new line. Ctrl+Enter applies the edit.';
      editor.appendChild(guidance);
      const actions = document.createElement('div');
      actions.className = 'actions';
      [['Cancel', false], ['Apply', true]].forEach(function (choice) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'button is-small' + (choice[1] ? '' : ' is-ghost');
        button.textContent = choice[0];
        button.addEventListener('click', function () { stop(choice[1]); cell.focus(); });
        actions.appendChild(button);
      });
      editor.appendChild(actions);
      open.stateControl = state;
      state.addEventListener('change', function () {
        open.state = state.value;
        input.disabled = open.state === 'null' || open.state === 'default';
        tally();
      });
      input.disabled = open.state === 'null' || open.state === 'default';
      document.body.appendChild(editor);
      positionEditor();
      editor.addEventListener('focusout', function (event) {
        if (editing === open && !editor.contains(event.relatedTarget)) stop(true);
      });
    } else {
      editor.className = 'cell-editor';
      original.node.style.visibility = 'hidden';
      cell.appendChild(editor);
      editor.addEventListener('blur', function () { if (editing === open) stop(true); });
    }

    input.addEventListener('input', function () {
      if (editing !== open) return;
      open.typed = input.value;
      open.state = 'value';
      if (open.stateControl) open.stateControl.value = 'value';
      tally();
    });
    editor.addEventListener('keydown', function (event) {
      event.stopPropagation();
      if (event.key === 'Escape') {
        event.preventDefault();
        stop(false);
        cell.focus();
      } else if (event.key === 'Enter' && (engine === 'libsql' || event.target === input && (event.ctrlKey || event.metaKey))) {
        event.preventDefault();
        stop(true);
        cell.focus();
      } else if (event.key === 'Tab' && engine === 'libsql') {
        event.preventDefault();
        stop(true);
        move(cell, 0, event.shiftKey ? -1 : 1);
      }
    });
    if (input.disabled) open.stateControl.focus();
    else { input.focus(); input.select(); }
    tally();
  }

  function move(cell, downBy, acrossBy) {
    const currentRow = cell.parentElement;
    const row = rows[rows.indexOf(currentRow) + downBy];
    if (!row) return;
    let index = [...currentRow.children].indexOf(cell) + acrossBy;
    while (index >= 0 && index < row.children.length) {
      const target = row.children[index];
      if (target.hasAttribute('data-edit') && !target.hidden) {
        target.focus();
        return;
      }
      if (acrossBy === 0) return;
      index += acrossBy;
    }
  }

  function interactive(event) { return event.target.closest('a, button, input, select, textarea'); }

  cells.forEach(function (cell) {
    cell.addEventListener('click', function (event) {
      if (!interactive(event) && (!editing || editing.cell !== cell)) cell.focus();
    });
    cell.addEventListener('dblclick', function (event) {
      if (!interactive(event)) edit(cell);
    });
    cell.addEventListener('keydown', function (event) {
      if (editing || interactive(event)) return;
      if (event.key === 'Enter' || event.key === 'F2') {
        event.preventDefault();
        edit(cell);
      } else if (event.key === 'Delete' || event.key === 'Backspace') {
        event.preventDefault();
        if (engine !== 'postgres' || !columns.get(cell.dataset.cell).NotNull) stage(cell, '', 'null');
        else document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: 'This column cannot be NULL.' } }));
      } else if (event.key === 'Escape') {
        event.preventDefault();
        revert(cell);
      } else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault();
        move(cell, event.key === 'ArrowDown' ? 1 : -1, 0);
      } else if (event.key === 'ArrowRight' || event.key === 'Tab' || event.key === 'ArrowLeft') {
        event.preventDefault();
        move(cell, 0, event.key === 'ArrowLeft' || event.shiftKey ? -1 : 1);
      } else if (event.key.length === 1 && !event.metaKey && !event.ctrlKey && !event.altKey) {
        event.preventDefault();
        edit(cell);
        editing.input.disabled = false;
        editing.input.value = event.key;
        editing.typed = event.key;
        editing.state = 'value';
        if (editing.stateControl) editing.stateControl.value = 'value';
        editing.input.focus();
        tally();
      }
    });
  });

  const firstVisible = cells.find(function (cell) { return !cell.hidden; });
  if (firstVisible) firstVisible.tabIndex = 0;
  table.addEventListener('focusin', function (event) {
    const cell = event.target.closest('[data-edit]');
    if (cell) cells.forEach(function (one) { one.tabIndex = one === cell ? 0 : -1; });
  });
  document.addEventListener('pointerdown', function (event) {
    if (editing && !editing.editor.contains(event.target) && !editing.cell.contains(event.target)) stop(true);
  });
  window.addEventListener('resize', positionEditor);
  window.addEventListener('scroll', positionEditor, true);
  discard.addEventListener('click', function () {
    stop(false);
    cells.forEach(revert);
  });
  form.addEventListener('submit', function (event) {
    stop(true);
    if (staged.size === 0) {
      event.preventDefault();
      return;
    }
    form.querySelectorAll('[data-grid-field]').forEach(function (node) { node.remove(); });
    staged.forEach(function (change) {
      adapter.fields.forEach(function (name) {
        const field = document.createElement('input');
        field.type = 'hidden';
        field.name = name;
        field.dataset.gridField = 'yes';
        field.value = adapter.field(change, name);
        form.appendChild(field);
      });
    });
    submitting = true;
  });
  window.addEventListener('beforeunload', function (event) {
    if (!submitting && (staged.size > 0 || editing && !unchanged(editing.cell, editing.typed, editing.state))) {
      event.preventDefault();
      event.returnValue = '';
    }
  });
  tally();
})();
