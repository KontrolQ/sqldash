(function () {
  const sheet = document.querySelector('[data-result]');

  if (!sheet) {
    return;
  }

  const table = sheet.querySelector('[data-result-table]');
  const text = sheet.querySelector('[data-result-text]');
  const views = document.querySelectorAll('[data-result-view]');
  const copy = document.querySelector('[data-result-copy]');
  const download = document.querySelector('[data-result-download]');

  let shown = 'table';

  function announce(message, tone) {
    document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: message, tone: tone } }));
  }

  function headings() {
    return [...table.querySelectorAll('thead th')].map(function (one) {
      return one.textContent.trim();
    });
  }

  function records() {
    return [...table.querySelectorAll('tbody tr')]
      .filter(function (row) {
        return !row.querySelector('.cell-empty');
      })
      .map(function (row) {
        return [...row.querySelectorAll('td')].map(function (cell) {
          return cell.textContent;
        });
      });
  }

  function asJSON() {
    const names = headings();

    return JSON.stringify(records().map(function (row) {
      const held = {};

      names.forEach(function (name, index) {
        held[name] = row[index];
      });

      return held;
    }), null, 2);
  }

  function quoted(value) {
    if (/[",\n]/.test(value)) {
      return '"' + value.split('"').join('""') + '"';
    }

    return value;
  }

  function asCSV() {
    const lines = [headings().map(quoted).join(',')];

    records().forEach(function (row) {
      lines.push(row.map(quoted).join(','));
    });

    return lines.join(String.fromCharCode(10));
  }

  function body() {
    if (shown === 'json') {
      return asJSON();
    }

    if (shown === 'csv') {
      return asCSV();
    }

    const lines = [headings().join('\t')];

    records().forEach(function (row) {
      lines.push(row.join('\t'));
    });

    return lines.join(String.fromCharCode(10));
  }

  function pick(wanted) {
    shown = wanted;

    views.forEach(function (button) {
      button.classList.toggle('is-current', button.dataset.resultView === wanted);
    });

    if (!table) {
      return;
    }

    if (wanted === 'table') {
      table.hidden = false;
      text.hidden = true;
      return;
    }

    text.textContent = body();
    table.hidden = true;
    text.hidden = false;
  }

  views.forEach(function (button) {
    button.addEventListener('click', function () {
      pick(button.dataset.resultView);
    });
  });

  if (copy) {
    copy.addEventListener('click', async function () {
      if (!table) {
        announce('There is nothing to copy.', 'bad');
        return;
      }

      try {
        await navigator.clipboard.writeText(body());
        announce('The results were copied.');
      } catch (blocked) {
        announce('The results could not be copied.', 'bad');
      }
    });
  }

  if (download) {
    download.addEventListener('click', function () {
      if (!table) {
        announce('There is nothing to download.', 'bad');
        return;
      }

      const kinds = { table: ['text/tab-separated-values', 'tsv'], json: ['application/json', 'json'], csv: ['text/csv', 'csv'] };
      const kind = kinds[shown];
      const blob = new Blob([body()], { type: kind[0] });
      const link = document.createElement('a');

      link.href = URL.createObjectURL(blob);
      link.download = 'results.' + kind[1];
      document.body.appendChild(link);
      link.click();
      link.remove();

      setTimeout(function () {
        URL.revokeObjectURL(link.href);
      }, 2000);
    });
  }
})();
