(function () {
  const field = document.querySelector('[data-statement]');
  if (!field) return;
  const protectedToken = /^(?:'(?:''|\\[\s\S]|[^'])*'|"(?:""|[^"])*"|`[^`]*`|\[[^\]]*\])/;
  const words = new Set('SELECT FROM WHERE GROUP BY HAVING ORDER LIMIT OFFSET INSERT INTO VALUES UPDATE SET DELETE RETURNING INNER LEFT OUTER RIGHT CROSS JOIN UNION ALL EXCEPT INTERSECT AND OR NOT ON AS IN IS NULL LIKE BETWEEN EXISTS CASE WHEN THEN ELSE END DISTINCT ASC DESC COUNT SUM AVG MIN MAX CAST COALESCE CREATE TABLE INDEX VIEW PRIMARY KEY REFERENCES DEFAULT UNIQUE IF BEGIN COMMIT'.split(' '));

  function tidy(source) {
    const chunks = [];
    let ordinary = '';
    function flush() {
      chunks.push(ordinary.replace(/\b[A-Za-z_][A-Za-z0-9_$]*\b/g, function (word) { return words.has(word.toUpperCase()) ? word.toUpperCase() : word; }));
      ordinary = '';
    }
    for (let index = 0; index < source.length;) {
      const remainder = source.slice(index);
      let token = protectedToken.exec(remainder);
      if (token) { flush(); chunks.push(token[0]); index += token[0].length; continue; }
      const dollar = /^\$(?:[A-Za-z_][A-Za-z0-9_]*)?\$/.exec(remainder);
      if (dollar) {
        const end = source.indexOf(dollar[0], index + dollar[0].length);
        if (end === -1) return source;
        flush(); chunks.push(source.slice(index, end + dollar[0].length)); index = end + dollar[0].length; continue;
      }
      if (remainder.startsWith('--')) {
        const end = source.indexOf('\n', index);
        flush(); chunks.push(source.slice(index, end === -1 ? source.length : end + 1)); index = end === -1 ? source.length : end + 1; continue;
      }
      if (remainder.startsWith('/*')) {
        let depth = 1; let end = index + 2;
        while (end < source.length && depth > 0) {
          if (source.slice(end, end + 2) === '/*') { depth += 1; end += 2; }
          else if (source.slice(end, end + 2) === '*/') { depth -= 1; end += 2; }
          else end += 1;
        }
        if (depth > 0) return source;
        flush(); chunks.push(source.slice(index, end)); index = end; continue;
      }
      if (source[index] === "'" || source[index] === '"') return source;
      ordinary += source[index]; index += 1;
    }
    flush();
    return chunks.join('');
  }

  const tidyButton = document.querySelector('[data-tidy]');
  if (tidyButton) tidyButton.addEventListener('click', function () {
    field.value = tidy(field.value);
    field.dispatchEvent(new Event('input', { bubbles: true }));
    field.focus();
  });

  const form = document.querySelector('[data-postgres-console]');
  if (!form) return;
  const cancelButton = form.querySelector('[data-console-cancel]');
  const runButton = form.querySelector('button[type="submit"]');
  const output = document.querySelector('[data-console-output]');
  let running = false;

  function toast(message, tone) {
    document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: message, tone: tone } }));
  }

  form.addEventListener('submit', async function (event) {
    event.preventDefault();
    if (running) return;
    const identifier = Array.from(crypto.getRandomValues(new Uint8Array(24)), function (byte) { return byte.toString(16).padStart(2, '0'); }).join('');
    form.elements.run_id.value = identifier;
    const body = new FormData(form);
    running = true; runButton.disabled = true; cancelButton.hidden = false; cancelButton.disabled = false;
    try {
      const response = await fetch(form.action, { method: 'POST', body: body, credentials: 'same-origin' });
      const documentResult = new DOMParser().parseFromString(await response.text(), 'text/html');
      const result = documentResult.querySelector('[data-console-output]');
      if (!response.ok || !result) throw new Error('The console submission could not be completed.');
      output.replaceChildren(...Array.from(result.childNodes));
    } catch (error) {
      toast(error.message || 'The query connection was lost.', 'bad');
    } finally {
      running = false; runButton.disabled = false; cancelButton.hidden = true;
    }
  });

  cancelButton.addEventListener('click', async function () {
    cancelButton.disabled = true;
    try {
      const body = new URLSearchParams({ run_id: form.elements.run_id.value });
      const response = await fetch(form.dataset.cancelPath, { method: 'POST', body: body, credentials: 'same-origin' });
      if (!response.ok) throw new Error('The query could not be cancelled.');
      toast('Cancellation requested.');
    } catch (error) { toast(error.message, 'bad'); cancelButton.disabled = false; }
  });

  output.addEventListener('click', async function (event) {
    const tab = event.target.closest('[data-console-result]');
    if (tab) {
      output.querySelectorAll('[data-console-result]').forEach(function (button) {
        const active = button === tab;
        button.classList.toggle('is-current', active); button.setAttribute('aria-selected', String(active));
      });
      output.querySelectorAll('[data-console-panel]').forEach(function (panel) { panel.hidden = panel.dataset.consolePanel !== tab.dataset.consoleResult; });
      return;
    }
    const exportButton = event.target.closest('[data-console-export]');
    if (!exportButton) return;
    const table = exportButton.closest('[data-console-panel]').querySelector('[data-console-table]');
    if (!table) { toast('There is nothing to export.', 'bad'); return; }
    const columns = Array.from(table.querySelectorAll('thead th'), function (column) { return column.textContent; });
    const rows = Array.from(table.querySelectorAll('tbody tr')).filter(function (row) { return !row.querySelector('.cell-empty'); }).map(function (row) {
      return Array.from(row.querySelectorAll('td'), function (cell) { return cell.dataset.null === 'true' ? null : cell.textContent; });
    });
    if (exportButton.dataset.consoleExport === 'json') {
      try { await navigator.clipboard.writeText(JSON.stringify({ columns: columns, rows: rows }, null, 2)); toast('The results were copied.'); }
      catch (error) { toast('The results could not be copied.', 'bad'); }
      return;
    }
    function quoted(value) { const text = value === null ? '' : String(value); return '"' + text.replace(/"/g, '""') + '"'; }
    const csv = [columns].concat(rows).map(function (row) { return row.map(quoted).join(','); }).join('\r\n');
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
    const link = document.createElement('a'); link.href = url; link.download = 'results.csv'; link.click();
    setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
  });
})();
