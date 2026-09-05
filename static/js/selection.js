(function () {
  const form = document.querySelector('[data-selection]');

  if (!form) {
    return;
  }

  const held = [...form.elements];
  const every = document.querySelector('[data-selection-all]');
  const boxes = held.filter(function (field) {
    return field.dataset.selectionOne !== undefined;
  });
  const bar = document.querySelector('[data-selection-bar]');
  const count = document.querySelector('[data-selection-count]');
  const clear = document.querySelector('[data-selection-clear]');

  if (!every || boxes.length === 0) {
    return;
  }

  function picked() {
    return boxes.filter(function (box) {
      return box.checked;
    });
  }

  function refresh() {
    const chosen = picked();

    bar.hidden = chosen.length === 0;
    count.textContent = chosen.length === 1 ? '1 row picked' : chosen.length + ' rows picked';

    every.checked = chosen.length === boxes.length;
    every.indeterminate = chosen.length > 0 && chosen.length < boxes.length;

    boxes.forEach(function (box) {
      box.closest('tr').classList.toggle('is-picked', box.checked);
    });
  }

  every.addEventListener('change', function () {
    boxes.forEach(function (box) {
      box.checked = every.checked;
    });

    refresh();
  });

  boxes.forEach(function (box) {
    box.addEventListener('change', refresh);
  });

  clear.addEventListener('click', function () {
    boxes.forEach(function (box) {
      box.checked = false;
    });

    refresh();
  });

  document.querySelectorAll('[data-row-delete]').forEach(function (button) {
    button.addEventListener('click', function () {
      boxes.forEach(function (box) {
        box.checked = false;
      });
    });
  });

  refresh();
})();
