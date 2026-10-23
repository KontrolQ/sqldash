(function () {
  document.querySelectorAll('[data-extension-toggle]').forEach(function (toggle) {
    const form = toggle.closest('form');
    if (!form || toggle.dataset.manageable !== 'true') { return; }
    toggle.disabled = false;
    toggle.addEventListener('change', function () {
      toggle.checked = toggle.dataset.enabled === 'true';
      form.requestSubmit();
    });
  });

  const catalog = document.querySelector('[data-extension-catalog]');
  if (!catalog) { return; }
  const search = catalog.querySelector('[data-extension-search]');
  if (!search) { return; }
  const table = catalog.querySelector('[data-extension-table]');
  const empty = catalog.querySelector('[data-extension-search-empty]');
  const status = catalog.querySelector('[data-extension-search-status]');
  const extensions = Array.from(catalog.querySelectorAll('[data-extension-row]')).map(function (row) {
    return {
      row: row,
      text: normalizeExtensionSearch(row.querySelector('[data-extension-search-text]').textContent)
    };
  });

  function normalizeExtensionSearch(text) {
    return text.toLowerCase().replace(/\s+/g, ' ').trim();
  }

  function filterExtensions() {
    const query = normalizeExtensionSearch(search.value);
    let shown = 0;
    extensions.forEach(function (extension) {
      extension.row.hidden = !extension.text.includes(query);
      if (!extension.row.hidden) { shown++; }
    });
    table.hidden = shown === 0;
    empty.hidden = shown !== 0;
    status.textContent = 'Showing ' + shown + ' of ' + extensions.length;
  }

  search.disabled = false;
  search.addEventListener('input', filterExtensions);
  filterExtensions();
})();
