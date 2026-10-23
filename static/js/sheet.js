(function () {
  const sheets = [...document.querySelectorAll('[data-sheet]')];

  if (sheets.length === 0) {
    return;
  }

  function close(target) {
    (target ? [target] : sheets).forEach(function (sheet) {
      if (sheet.dataset.cloneBusy !== 'true') sheet.hidden = true;
    });
  }

  document.querySelectorAll('[data-sheet-open]').forEach(function (button) {
    button.addEventListener('click', function () {
      const sheet = document.querySelector('[data-sheet="' + button.dataset.sheetOpen + '"]');

      if (!sheet || sheets.some(function (one) { return !one.hidden && one.dataset.cloneBusy === 'true'; })) {
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
      button.addEventListener('click', function () { close(sheet); });
    });
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      close();
    }
  });
})();
