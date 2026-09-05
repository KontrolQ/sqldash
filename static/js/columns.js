(function () {
  const menu = document.querySelector('[data-columns]');

  if (!menu) {
    return;
  }

  const store = 'sqldash.columns.' + menu.dataset.columns;
  const buttons = [...menu.querySelectorAll('[data-column]')];

  function hidden() {
    try {
      return JSON.parse(localStorage.getItem(store)) || [];
    } catch (readError) {
      return [];
    }
  }

  function remember(names) {
    try {
      localStorage.setItem(store, JSON.stringify(names));
    } catch (writeError) {
      return;
    }
  }

  function apply() {
    const away = hidden();

    buttons.forEach(function (button) {
      const name = button.dataset.column;
      const off = away.includes(name);

      button.setAttribute('aria-checked', String(!off));

      document.querySelectorAll('[data-cell="' + CSS.escape(name) + '"]').forEach(function (cell) {
        cell.hidden = off;
      });
    });
  }

  buttons.forEach(function (button) {
    button.addEventListener('click', function () {
      const name = button.dataset.column;
      const away = hidden();
      const at = away.indexOf(name);

      if (at >= 0) {
        away.splice(at, 1);
      } else if (away.length < buttons.length - 1) {
        away.push(name);
      }

      remember(away);
      apply();
    });
  });

  apply();
})();
