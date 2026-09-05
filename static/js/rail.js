(function () {
  const field = document.querySelector('[data-rail-search]');

  if (!field) {
    return;
  }

  const rows = [...document.querySelectorAll('[data-rail-row]')];

  field.addEventListener('input', function () {
    const needle = field.value.trim().toLowerCase();

    rows.forEach(function (row) {
      row.hidden = needle !== '' && !row.dataset.railRow.toLowerCase().includes(needle);
    });
  });
})();
