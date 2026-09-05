(function () {
  const cells = [...document.querySelectorAll('[data-edit]')];

  if (cells.length === 0) {
    return;
  }

  const table = cells[0].closest('table');
  const rows = [...table.querySelectorAll('tbody tr')];
  const form = document.getElementById('grid-form');
  const save = document.querySelector('[data-grid-save]');
  const discard = document.querySelector('[data-grid-discard]');
  const counter = document.querySelector('[data-grid-count]');
  const staged = new Map();

  const WAS_NULL = 'null';

  let editing = null;

  function held(cell) {
    return cell.querySelector('.cell-text');
  }

  function readingOf(cell) {
    const node = held(cell);

    return {
      node: node,
      value: node.classList.contains('null') ? '' : node.textContent,
      wasNull: node.classList.contains('null'),
    };
  }

  function tally() {
    counter.textContent = String(staged.size);
    save.disabled = staged.size === 0;
    discard.disabled = staged.size === 0;
  }

  function keyOf(cell) {
    return cell.parentElement.dataset.key + '|' + cell.dataset.cell;
  }

  function show(cell, value, clear) {
    const node = held(cell);

    if (clear) {
      node.textContent = 'NULL';
      node.classList.add('null');
    } else {
      node.textContent = value;
      node.classList.remove('null');
    }
  }

  function remember(cell) {
    if (cell.dataset.was === undefined) {
      const reading = readingOf(cell);
      cell.dataset.was = reading.wasNull ? WAS_NULL : reading.value;
    }

    return cell.dataset.was;
  }

  function stage(cell, value, clear) {
    const was = remember(cell);
    const unchanged = clear ? was === WAS_NULL : was === value;

    if (unchanged) {
      revert(cell);
      return;
    }

    staged.set(keyOf(cell), {
      column: cell.dataset.cell,
      key: cell.parentElement.dataset.key,
      value: value,
      clear: clear,
    });

    show(cell, value, clear);
    cell.classList.add('is-changed');
    tally();
  }

  function revert(cell) {
    if (cell.dataset.was === undefined) {
      return;
    }

    const was = cell.dataset.was;

    show(cell, was === WAS_NULL ? '' : was, was === WAS_NULL);

    delete cell.dataset.was;
    staged.delete(keyOf(cell));
    cell.classList.remove('is-changed');
    tally();
  }

  function stop(keep) {
    if (!editing) {
      return;
    }

    const open = editing;
    editing = null;

    open.cell.classList.remove('is-editing');
    open.editor.remove();
    open.reading.node.style.visibility = '';

    if (keep) {
      stage(open.cell, open.typed, false);
    }
  }

  function edit(cell) {
    if (editing && editing.cell === cell) {
      return;
    }

    stop(true);

    const reading = readingOf(cell);
    const editor = document.createElement('input');

    editor.className = 'cell-editor';
    editor.value = reading.value;
    editor.setAttribute('aria-label', 'Edit ' + cell.dataset.cell);

    reading.node.style.visibility = 'hidden';
    cell.classList.add('is-editing');
    cell.appendChild(editor);

    editing = { cell: cell, editor: editor, reading: reading, typed: reading.value };

    editor.focus();
    editor.select();

    editor.addEventListener('input', function () {
      if (editing) {
        editing.typed = editor.value;
      }
    });

    editor.addEventListener('keydown', function (event) {
      event.stopPropagation();

      if (event.key === 'Enter') {
        event.preventDefault();
        editing.typed = editor.value;
        stop(true);
        cell.focus();
      } else if (event.key === 'Escape') {
        event.preventDefault();
        stop(false);
        cell.focus();
      } else if (event.key === 'Tab') {
        event.preventDefault();
        editing.typed = editor.value;
        stop(true);
        move(cell, 0, event.shiftKey ? -1 : 1);
      }
    });

    editor.addEventListener('blur', function () {
      stop(true);
    });
  }

  function place(cell) {
    const row = cell.parentElement;

    return { row: rows.indexOf(row), column: [...row.children].indexOf(cell) };
  }

  function move(cell, downBy, acrossBy) {
    const at = place(cell);
    const row = rows[at.row + downBy];

    if (!row) {
      return;
    }

    let index = at.column + acrossBy;

    while (index >= 0 && index < row.children.length) {
      const target = row.children[index];

      if (target.dataset.edit !== undefined) {
        target.focus();
        return;
      }

      if (acrossBy === 0) {
        return;
      }

      index += acrossBy;
    }
  }

  cells.forEach(function (cell) {
    cell.addEventListener('click', function (event) {
      if (event.target.closest('.cell-jump')) {
        return;
      }

      if (editing && editing.cell === cell) {
        return;
      }

      cell.focus();
    });

    cell.addEventListener('dblclick', function (event) {
      if (event.target.closest('.cell-jump')) {
        return;
      }

      edit(cell);
    });

    cell.addEventListener('keydown', function (event) {
      if (editing) {
        return;
      }

      if (event.key === 'Enter' || event.key === 'F2') {
        event.preventDefault();
        edit(cell);
      } else if (event.key === 'Delete' || event.key === 'Backspace') {
        event.preventDefault();
        stage(cell, '', true);
      } else if (event.key === 'Escape') {
        event.preventDefault();
        revert(cell);
      } else if (event.key === 'ArrowDown') {
        event.preventDefault();
        move(cell, 1, 0);
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        move(cell, -1, 0);
      } else if (event.key === 'ArrowRight' || event.key === 'Tab') {
        event.preventDefault();
        move(cell, 0, event.shiftKey ? -1 : 1);
      } else if (event.key === 'ArrowLeft') {
        event.preventDefault();
        move(cell, 0, -1);
      } else if (event.key.length === 1 && !event.metaKey && !event.ctrlKey) {
        edit(cell);
        editing.editor.value = event.key;
        editing.typed = event.key;
        event.preventDefault();
      }
    });
  });

  cells[0].tabIndex = 0;

  table.addEventListener('focusin', function (event) {
    const cell = event.target.closest('[data-edit]');

    if (!cell) {
      return;
    }

    cells.forEach(function (one) {
      one.tabIndex = one === cell ? 0 : -1;
    });
  });

  discard.addEventListener('click', function () {
    cells.forEach(revert);
  });

  save.addEventListener('click', function (event) {
    stop(true);

    if (staged.size === 0) {
      event.preventDefault();
      return;
    }

    form.querySelectorAll('[data-grid-field]').forEach(function (node) {
      node.remove();
    });

    staged.forEach(function (change) {
      ['column', 'key', 'value', 'clear'].forEach(function (name) {
        const field = document.createElement('input');

        field.type = 'hidden';
        field.name = name;
        field.dataset.gridField = 'yes';
        field.value = name === 'clear' ? (change.clear ? 'on' : '') : change[name];
        form.appendChild(field);
      });
    });

    staged.clear();
  });

  window.addEventListener('beforeunload', function (event) {
    if (staged.size > 0) {
      event.preventDefault();
      event.returnValue = '';
    }
  });

  tally();
})();
