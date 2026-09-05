(function () {
  const field = document.querySelector('[data-statement]');

  if (!field) {
    return;
  }

  const shell = field.closest('.editor');
  const NEWLINE = String.fromCharCode(10);
  const paint = shell.querySelector('.editor-paint');
  const gutter = shell.querySelector('[data-editor-lines]');
  const hint = shell.querySelector('.editor-hint');

  let schema = [];

  try {
    schema = JSON.parse(shell.dataset.schema || '[]');
  } catch (readError) {
    schema = [];
  }

  const keywords = [
    'ABORT', 'ACTION', 'ADD', 'AFTER', 'ALL', 'ALTER', 'AND', 'AS', 'ASC', 'ATTACH',
    'AUTOINCREMENT', 'BEFORE', 'BEGIN', 'BETWEEN', 'BY', 'CASCADE', 'CASE', 'CAST',
    'CHECK', 'COLLATE', 'COLUMN', 'COMMIT', 'CONFLICT', 'CONSTRAINT', 'CREATE',
    'CROSS', 'CURRENT_DATE', 'CURRENT_TIME', 'CURRENT_TIMESTAMP', 'DATABASE',
    'DEFAULT', 'DEFERRABLE', 'DELETE', 'DESC', 'DETACH', 'DISTINCT', 'DROP', 'EACH',
    'ELSE', 'END', 'ESCAPE', 'EXCEPT', 'EXCLUSIVE', 'EXISTS', 'EXPLAIN', 'FAIL',
    'FOR', 'FOREIGN', 'FROM', 'FULL', 'GLOB', 'GROUP', 'HAVING', 'IF', 'IGNORE',
    'IMMEDIATE', 'IN', 'INDEX', 'INDEXED', 'INITIALLY', 'INNER', 'INSERT', 'INSTEAD',
    'INTERSECT', 'INTO', 'IS', 'ISNULL', 'JOIN', 'KEY', 'LEFT', 'LIKE', 'LIMIT',
    'MATCH', 'NATURAL', 'NO', 'NOT', 'NOTNULL', 'NULL', 'OF', 'OFFSET', 'ON', 'OR',
    'ORDER', 'OUTER', 'PLAN', 'PRAGMA', 'PRIMARY', 'QUERY', 'RAISE', 'RECURSIVE',
    'REFERENCES', 'REGEXP', 'REINDEX', 'RELEASE', 'RENAME', 'REPLACE', 'RESTRICT',
    'RETURNING', 'RIGHT', 'ROLLBACK', 'ROW', 'SAVEPOINT', 'SELECT', 'SET', 'TABLE',
    'TEMP', 'TEMPORARY', 'THEN', 'TO', 'TRANSACTION', 'TRIGGER', 'UNION', 'UNIQUE',
    'UPDATE', 'USING', 'VACUUM', 'VALUES', 'VIEW', 'VIRTUAL', 'WHEN', 'WHERE', 'WITH',
  ];

  const builtins = [
    'ABS', 'AVG', 'CHANGES', 'CHAR', 'COALESCE', 'CONCAT', 'COUNT', 'DATE',
    'DATETIME', 'GROUP_CONCAT', 'HEX', 'IFNULL', 'INSTR', 'JSON', 'JSON_EXTRACT',
    'JULIANDAY', 'LAST_INSERT_ROWID', 'LENGTH', 'LOWER', 'LTRIM', 'MAX', 'MIN',
    'NULLIF', 'PRINTF', 'QUOTE', 'RANDOM', 'REPLACE', 'ROUND', 'RTRIM', 'STRFTIME',
    'SUBSTR', 'SUM', 'TIME', 'TOTAL', 'TRIM', 'TYPEOF', 'UNICODE', 'UPPER', 'ZEROBLOB',
  ];

  const keywordSet = new Set(keywords);
  const builtinSet = new Set(builtins);

  function escaped(text) {
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }

  function classOf(token) {
    if (token.startsWith('--') || token.startsWith('/*')) {
      return 'is-note';
    }

    if (token.startsWith("'") || token.startsWith('"') || token.startsWith('`') || token.startsWith('[')) {
      return 'is-text';
    }

    if (/^[0-9]/.test(token)) {
      return 'is-number';
    }

    if (/^[A-Za-z_][A-Za-z0-9_$]*$/.test(token)) {
      const upper = token.toUpperCase();

      if (keywordSet.has(upper)) {
        return 'is-word';
      }

      if (builtinSet.has(upper)) {
        return 'is-call';
      }

      return '';
    }

    if (/^[?:@$]/.test(token)) {
      return 'is-bind';
    }

    if (/^[-+*/%<>=!|&~^]+$/.test(token)) {
      return 'is-mark';
    }

    return '';
  }

  const grammar = /(--[^\n]*|\/\*[\s\S]*?\*\/|'(?:''|[^'])*'?|"(?:""|[^"])*"?|`(?:[^`])*`?|\[[^\]]*\]?|\b\d+(?:\.\d+)?\b|[A-Za-z_][A-Za-z0-9_$]*|[?:@$][A-Za-z0-9_]*|[-+*/%<>=!|&~^]+|[^\s])/g;

  function repaint() {
    const source = field.value;
    let html = '';
    let at = 0;

    source.replace(grammar, function (token, _piece, index) {
      html += escaped(source.slice(at, index));

      const kind = classOf(token);
      html += kind === '' ? escaped(token) : '<span class="' + kind + '">' + escaped(token) + '</span>';

      at = index + token.length;
      return token;
    });

    paint.innerHTML = html + escaped(source.slice(at)) + '\n';
    paint.scrollTop = field.scrollTop;
    paint.scrollLeft = field.scrollLeft;
    number();
  }

  let offered = [];
  let chosen = 0;
  let anchor = 0;

  function number() {
    if (!gutter) {
      return;
    }

    const count = field.value.split(NEWLINE).length;

    if (gutter.childElementCount === count) {
      return;
    }

    const marks = [];

    for (let line = 1; line <= count; line += 1) {
      marks.push('<span>' + line + '</span>');
    }

    gutter.innerHTML = marks.join('');
    gutter.scrollTop = field.scrollTop;
  }

  function columnsOf(name) {
    const held = schema.find(function (table) {
      return table.Name.toLowerCase() === name.toLowerCase();
    });

    return held ? held.Columns : [];
  }

  function everyColumn() {
    const seen = new Set();
    const all = [];

    schema.forEach(function (table) {
      table.Columns.forEach(function (column) {
        if (!seen.has(column)) {
          seen.add(column);
          all.push({ label: column, note: table.Name, kind: 'column' });
        }
      });
    });

    return all;
  }

  function suggestions() {
    const upto = field.value.slice(0, field.selectionStart);
    const qualified = /([A-Za-z_][A-Za-z0-9_$]*)\.([A-Za-z0-9_$]*)$/.exec(upto);

    if (qualified) {
      anchor = field.selectionStart - qualified[2].length;

      return columnsOf(qualified[1])
        .filter(function (column) {
          return column.toLowerCase().startsWith(qualified[2].toLowerCase());
        })
        .map(function (column) {
          return { label: column, note: qualified[1], kind: 'column' };
        });
    }

    const bare = /([A-Za-z_][A-Za-z0-9_$]*)$/.exec(upto);

    if (!bare) {
      return [];
    }

    anchor = field.selectionStart - bare[1].length;

    const needle = bare[1].toLowerCase();

    if (needle.length < 2) {
      return [];
    }

    const tables = schema.map(function (table) {
      return { label: table.Name, note: 'table', kind: 'table' };
    });

    const terms = keywords.concat(builtins).map(function (word) {
      return { label: word, note: builtinSet.has(word) ? 'function' : 'keyword', kind: 'word' };
    });

    return tables
      .concat(everyColumn())
      .concat(terms)
      .filter(function (item) {
        return item.label.toLowerCase().startsWith(needle);
      })
      .slice(0, 10);
  }

  function place() {
    const before = field.value.slice(0, field.selectionStart);
    const lines = before.split('\n');
    const style = getComputedStyle(field);
    const height = parseFloat(style.lineHeight);
    const width = 8.4;

    hint.style.top = (lines.length * height + 18 - field.scrollTop) + 'px';
    const gutterWidth = gutter ? gutter.offsetWidth : 0;

    hint.style.left = Math.min(
      lines[lines.length - 1].length * width + 16 + gutterWidth - field.scrollLeft,
      field.clientWidth - 240
    ) + 'px';
  }

  function draw() {
    if (offered.length === 0) {
      hint.hidden = true;
      return;
    }

    hint.innerHTML = offered.map(function (item, index) {
      return '<button class="hint-item' + (index === chosen ? ' is-active' : '') +
        '" type="button" data-hint="' + index + '">' +
        '<span class="hint-label">' + escaped(item.label) + '</span>' +
        '<span class="hint-note">' + escaped(item.note) + '</span></button>';
    }).join('');

    hint.hidden = false;
    place();
  }

  function offer() {
    offered = suggestions();
    chosen = 0;
    draw();
  }

  function accept() {
    const item = offered[chosen];

    if (!item) {
      return;
    }

    const after = field.value.slice(field.selectionStart);
    field.value = field.value.slice(0, anchor) + item.label + after;

    const caret = anchor + item.label.length;
    field.setSelectionRange(caret, caret);

    offered = [];
    hint.hidden = true;
    repaint();
  }

  hint.addEventListener('mousedown', function (event) {
    const button = event.target.closest('[data-hint]');

    if (!button) {
      return;
    }

    event.preventDefault();
    chosen = Number(button.dataset.hint);
    accept();
  });

  field.addEventListener('input', function () {
    repaint();
    offer();
  });

  field.addEventListener('scroll', function () {
    paint.scrollTop = field.scrollTop;
    paint.scrollLeft = field.scrollLeft;

    if (gutter) {
      gutter.scrollTop = field.scrollTop;
    }
  });

  field.addEventListener('blur', function () {
    hint.hidden = true;
  });

  field.addEventListener('keydown', function (event) {
    if (hint.hidden) {
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      chosen = (chosen + 1) % offered.length;
      draw();
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      chosen = (chosen - 1 + offered.length) % offered.length;
      draw();
    } else if (event.key === 'Enter' || event.key === 'Tab') {
      event.preventDefault();
      accept();
    } else if (event.key === 'Escape') {
      event.preventDefault();
      offered = [];
      hint.hidden = true;
    }
  });

  repaint();
})();
