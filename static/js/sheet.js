(function () {
  const sheets = [...document.querySelectorAll('[data-sheet]')];

  if (sheets.length === 0) {
    return;
  }

  function close() {
    sheets.forEach(function (sheet) {
      sheet.hidden = true;
    });
  }

  document.querySelectorAll('[data-sheet-open]').forEach(function (button) {
    button.addEventListener('click', function () {
      const sheet = document.querySelector('[data-sheet="' + button.dataset.sheetOpen + '"]');

      if (!sheet) {
        return;
      }

      close();
      sheet.hidden = false;

      const first = sheet.querySelector('input:not([type=hidden])');

      if (first) {
        first.focus();
      }
    });
  });

  sheets.forEach(function (sheet) {
    sheet.querySelectorAll('[data-sheet-close]').forEach(function (button) {
      button.addEventListener('click', close);
    });
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      close();
    }
  });
})();
