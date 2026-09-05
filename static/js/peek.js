(function () {
  const chips = document.querySelectorAll('[data-peek]');

  if (chips.length === 0) {
    return;
  }

  function close(chip) {
    const row = document.getElementById(chip.dataset.peek);

    if (row) {
      row.hidden = true;
    }

    chip.setAttribute('aria-expanded', 'false');
  }

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      const row = document.getElementById(chip.dataset.peek);

      if (!row) {
        return;
      }

      const opening = row.hidden;

      chips.forEach(close);

      if (opening) {
        row.hidden = false;
        chip.setAttribute('aria-expanded', 'true');
      }
    });
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      chips.forEach(close);
    }
  });
})();
