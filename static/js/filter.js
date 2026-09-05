(function () {
  const rows = document.querySelector('[data-filter-rows]');

  if (!rows) {
    return;
  }

  const add = document.querySelector('[data-filter-add]');
  const template = document.querySelector('[data-filter-template]');

  function watch(row) {
    row.querySelector('[data-filter-drop]').addEventListener('click', function () {
      if (rows.querySelectorAll('[data-filter-row]').length === 1) {
        row.querySelector('input[name="value"]').value = '';
        return;
      }

      row.remove();
    });
  }

  rows.querySelectorAll('[data-filter-row]').forEach(watch);

  add.addEventListener('click', function () {
    const row = template.content.firstElementChild.cloneNode(true);

    rows.appendChild(row);
    watch(row);
    document.dispatchEvent(new Event('sqldash:scan'));
    row.querySelector('input[name="value"]').focus();
  });
})();
