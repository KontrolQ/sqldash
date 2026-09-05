(function () {
  const cells = [...document.querySelectorAll('[data-edit]')];

  if (cells.length === 0) {
    return;
  }

  const table = cells[0].closest('table');
  const rows = [...table.querySelectorAll('tbody tr')];

  let editing = null;

  function textOf(cell) {
    const held = cell.querySelector('.cell-text');

    return {
      node: held,
      value: held.classList.contains('null') ? '' : held.textContent,
      wasNull: held.classList.contains('null'),
    };
  }

  function commit(cell, value, clear) {
    const form = document.getElementById(cell.dataset.edit);

    form.elements.value.value = value;
    form.elements.clear.value = clear ? 'on' : '';
    form.submit();
  }

  function stop() {
    if (!editing) {
      return;
    }

    const held = editing;
    editing = null;

    held.cell.classList.remove('is-editing');
    held.editor.remove();
    held.text.node.hidden = false;
    held.cell.focus();
  }

  function edit(cell) {
    if (editing && editing.cell === cell) {
      return;
    }

    stop();

    const text = textOf(cell);
    const editor = document.createElement('input');

    editor.className = 'cell-editor';
    editor.value = text.value;
    editor.setAttribute('aria-label', 'Edit ' + cell.dataset.cell);

    text.node.hidden = true;
    cell.classList.add('is-editing');
    cell.appendChild(editor);

    editing = { cell: cell, editor: editor, text: text };

    editor.focus();
    editor.select();

    editor.addEventListener('keydown', function (event) {
      if (event.key === 'Enter') {
        event.preventDefault();

        if (editor.value === text.value && !text.wasNull) {
          stop();
          return;
        }

        commit(cell, editor.value, false);
      } else if (event.key === 'Escape') {
        event.preventDefault();
        stop();
      }
    });

    editor.addEventListener('blur', stop);
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

      if (target.dataset.edit) {
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
    cell.addEventListener('click', function () {
      cell.focus();
    });

    cell.addEventListener('dblclick', function () {
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
        commit(cell, '', true);
      } else if (event.key === 'ArrowDown') {
        event.preventDefault();
        move(cell, 1, 0);
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        move(cell, -1, 0);
      } else if (event.key === 'ArrowRight' || event.key === 'Tab') {
        event.preventDefault();
        move(cell, 0, 1);
      } else if (event.key === 'ArrowLeft') {
        event.preventDefault();
        move(cell, 0, -1);
      } else if (event.key.length === 1 && !event.metaKey && !event.ctrlKey) {
        edit(cell);
        editing.editor.value = event.key;
        event.preventDefault();
      }
    });
  });

  if (cells.length > 0) {
    cells[0].tabIndex = 0;
  }

  table.addEventListener('focusin', function (event) {
    const cell = event.target.closest('[data-edit]');

    if (!cell) {
      return;
    }

    cells.forEach(function (one) {
      one.tabIndex = one === cell ? 0 : -1;
    });
  });
})();
