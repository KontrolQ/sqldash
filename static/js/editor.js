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

  schema = Array.isArray(schema) ? schema.filter(function (table) {
    return table && typeof table.Name === 'string';
  }).map(function (table) {
    return {
      Schema: typeof table.Schema === 'string' ? table.Schema : '',
      Kind: table.Kind || 'table',
      Name: table.Name,
      Columns: Array.isArray(table.Columns) ? table.Columns.filter(function (column) { return typeof column === 'string'; }) : [],
    };
  }) : [];

  const postgres = shell.dataset.dialect === 'postgres';
  let currentSchema = shell.dataset.currentSchema || 'public';
  const schemaField = shell.closest('form').querySelector('select[name="schema"]');
  if (schemaField) schemaField.addEventListener('change', function () { currentSchema = schemaField.value; });
  let keywords = [
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

  let builtins = [
    'ABS', 'AVG', 'CHANGES', 'CHAR', 'COALESCE', 'CONCAT', 'COUNT', 'DATE',
    'DATETIME', 'GROUP_CONCAT', 'HEX', 'IFNULL', 'INSTR', 'JSON', 'JSON_EXTRACT',
    'JULIANDAY', 'LAST_INSERT_ROWID', 'LENGTH', 'LOWER', 'LTRIM', 'MAX', 'MIN',
    'NULLIF', 'PRINTF', 'QUOTE', 'RANDOM', 'REPLACE', 'ROUND', 'RTRIM', 'STRFTIME',
    'SUBSTR', 'SUM', 'TIME', 'TOTAL', 'TRIM', 'TYPEOF', 'UNICODE', 'UPPER', 'ZEROBLOB',
  ];

  let types = [];

  if (postgres) {
    const sqliteOnly = new Set('ABORT ATTACH AUTOINCREMENT DETACH EXCLUSIVE FAIL GLOB IGNORE IMMEDIATE INDEXED ISNULL NOTNULL PLAN PRAGMA QUERY RAISE REPLACE VIRTUAL'.split(' '));
    keywords = keywords.filter(function (word) { return !sqliteOnly.has(word); }).concat('ALWAYS ANALYZE ANY ARRAY AT AUTHORIZATION BOTH CONCURRENTLY CURRENT_ROLE CURRENT_SCHEMA CURRENT_USER DECLARE DO ENUM FALSE FETCH FILTER FIRST FOLLOWING FUNCTION GENERATED GRANT GROUPING IDENTITY ILIKE INCLUDE INHERITS LAST LANGUAGE LATERAL LEADING LOCAL LOCK MATERIALIZED MERGE NEXT NULLS ONLY OVER OVERRIDING PARTITION PRECEDING PROCEDURE RANGE REFRESH REVOKE ROWS SCHEMA SECURITY SEQUENCE SESSION_USER SIMILAR SOME STORED TABLESPACE TRAILING TRUE TRUNCATE UNBOUNDED USER VARIADIC VARYING WINDOW WITHIN'.split(' '));
    builtins = 'ABS ARRAY_AGG ARRAY_APPEND ARRAY_CAT ARRAY_LENGTH ARRAY_LOWER ARRAY_POSITION ARRAY_POSITIONS ARRAY_PREPEND ARRAY_REMOVE ARRAY_REPLACE ARRAY_TO_STRING ARRAY_UPPER AVG BOOL_AND BOOL_OR BTRIM CARDINALITY CEIL CEILING CHAR_LENGTH CHR CLOCK_TIMESTAMP COALESCE CONCAT CONCAT_WS COUNT CUME_DIST CURRENT_DATABASE CURRENT_SCHEMAS DATE_BIN DATE_PART DATE_TRUNC DENSE_RANK EXTRACT FIRST_VALUE FLOOR FORMAT GENERATE_SERIES GENERATE_SUBSCRIPTS GEN_RANDOM_UUID GREATEST ISFINITE JSON_AGG JSON_ARRAY_ELEMENTS JSON_ARRAY_LENGTH JSON_BUILD_ARRAY JSON_BUILD_OBJECT JSON_EACH JSON_EXTRACT_PATH JSON_OBJECT_AGG JSON_TYPEOF JSONB_AGG JSONB_ARRAY_ELEMENTS JSONB_ARRAY_LENGTH JSONB_BUILD_ARRAY JSONB_BUILD_OBJECT JSONB_EACH JSONB_EXTRACT_PATH JSONB_OBJECT_AGG JSONB_PATH_EXISTS JSONB_PATH_QUERY JSONB_PRETTY JSONB_SET JSONB_STRIP_NULLS JSONB_TYPEOF LAG LAST_VALUE LEAD LEAST LEFT LENGTH LOWER LPAD LTRIM MAKE_DATE MAKE_INTERVAL MAKE_TIME MAKE_TIMESTAMP MAX MIN NOW NTH_VALUE NTILE NULLIF OCTET_LENGTH OVERLAY PERCENT_RANK PERCENTILE_CONT PERCENTILE_DISC POSITION POWER QUOTE_IDENT QUOTE_LITERAL QUOTE_NULLABLE RANDOM RANK REGEXP_MATCH REGEXP_MATCHES REGEXP_REPLACE REGEXP_SPLIT_TO_ARRAY REGEXP_SPLIT_TO_TABLE REPEAT REPLACE REVERSE RIGHT ROUND ROW_NUMBER RPAD RTRIM SPLIT_PART SQRT STATEMENT_TIMESTAMP STRING_AGG STRING_TO_ARRAY STRPOS SUBSTRING SUM TIMEOFDAY TO_CHAR TO_DATE TO_JSON TO_JSONB TO_NUMBER TO_TIMESTAMP TRANSACTION_TIMESTAMP TRANSLATE TRIM TRUNC UNNEST UPPER UUIDV4 UUIDV7'.split(' ');
    types = ['BIGINT', 'BIGSERIAL', 'BIT', 'BIT VARYING', 'BOOLEAN', 'BOX', 'BYTEA', 'CHAR', 'CHARACTER', 'CHARACTER VARYING', 'CIDR', 'CIRCLE', 'DATE', 'DECIMAL', 'DOUBLE PRECISION', 'INET', 'INTEGER', 'INTERVAL', 'JSON', 'JSONB', 'LINE', 'LSEG', 'MACADDR', 'MACADDR8', 'MONEY', 'NUMERIC', 'OID', 'PATH', 'POINT', 'POLYGON', 'REAL', 'SERIAL', 'SMALLINT', 'SMALLSERIAL', 'TEXT', 'TIME', 'TIMESTAMP', 'TIMESTAMPTZ', 'TIMETZ', 'TSQUERY', 'TSVECTOR', 'UUID', 'VARCHAR', 'XML'];
  }
  keywords = Array.from(new Set(keywords));
  const keywordSet = new Set(keywords);
  const builtinSet = new Set(builtins);
  const typeSet = new Set(types);

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

      if (keywordSet.has(upper) || typeSet.has(upper)) {
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

  function columnsOf(name, schemaName) {
    const held = schema.find(function (table) {
      return table.Name === name && (!postgres || table.Schema === (schemaName || currentSchema));
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
          all.push({ label: column, insert: identifier(column), note: (table.Schema ? table.Schema + '.' : '') + table.Name, kind: 'column' });
        }
      });
    });

    return all;
  }

  function identifier(value) {
    return postgres ? '"' + value.replace(/"/g, '""') + '"' : value;
  }

  function decodedIdentifier(value) {
    return value.startsWith('"') ? value.slice(1, -1).replace(/""/g, '"') : (postgres ? value.toLowerCase() : value);
  }

  function schemaNames() {
    if (!postgres) {
      return [];
    }

    const names = schema.map(function (table) { return table.Schema; });
    if (schemaField) {
      Array.from(schemaField.options).forEach(function (option) { names.push(option.value); });
    }
    return Array.from(new Set(names.filter(function (name) { return name !== ''; })));
  }

  function suggestions(manual) {
    const upto = field.value.slice(0, field.selectionStart);
    const identifierPattern = '("(?:""|[^"])*"|[A-Za-z_][A-Za-z0-9_$]*)';
    const qualified = new RegExp(identifierPattern + '\\.' + identifierPattern + '\\.([A-Za-z0-9_$]*)$').exec(upto);
    if (qualified && postgres) {
      anchor = field.selectionStart - qualified[3].length;
      return columnsOf(decodedIdentifier(qualified[2]), decodedIdentifier(qualified[1]))
        .filter(function (column) { return column.toLowerCase().startsWith(qualified[3].toLowerCase()); })
        .map(function (column) { return { label: column, insert: identifier(column), note: qualified[1] + '.' + qualified[2], kind: 'column' }; });
    }
    const pair = new RegExp(identifierPattern + '\\.([A-Za-z0-9_$]*)$').exec(upto);
    if (pair) {
      anchor = field.selectionStart - pair[2].length;
      const name = decodedIdentifier(pair[1]);
      const schemaTables = postgres ? schema.filter(function (table) { return table.Schema === name; }) : [];
      if (postgres && schemaNames().includes(name)) {
        return schemaTables.filter(function (table) { return table.Name.toLowerCase().startsWith(pair[2].toLowerCase()); })
          .map(function (table) { return { label: table.Name, insert: identifier(table.Name), note: table.Kind, kind: 'table' }; });
      }
      return columnsOf(name)
        .filter(function (column) { return column.toLowerCase().startsWith(pair[2].toLowerCase()); })
        .map(function (column) { return { label: column, insert: identifier(column), note: pair[1], kind: 'column' }; });
    }

    const bare = /([A-Za-z_][A-Za-z0-9_$]*)$/.exec(upto);

    if (!bare && !manual) {
      return [];
    }

    const prefix = bare ? bare[1] : '';
    anchor = field.selectionStart - prefix.length;
    const needle = prefix.toLowerCase();

    if (needle.length < 2 && !manual) {
      return [];
    }

    const tables = schema.map(function (table) {
      return { label: table.Name, insert: (postgres ? identifier(table.Schema) + '.' : '') + identifier(table.Name), note: (table.Schema ? table.Schema + ' · ' : '') + (table.Kind || 'table'), kind: 'table' };
    });

    const schemas = schemaNames().map(function (name) {
      return { label: name, insert: identifier(name), note: 'schema', kind: 'schema' };
    });
    const terms = Array.from(new Set(keywords.concat(builtins, types))).map(function (word) {
      return { label: word, note: typeSet.has(word) ? 'type' : (builtinSet.has(word) ? 'function' : 'keyword'), kind: 'word' };
    });

    return tables
      .concat(schemas)
      .concat(everyColumn())
      .concat(terms)
      .filter(function (item) {
        return item.label.toLowerCase().startsWith(needle);
      })
      .slice(0, 10);
  }

  function place() {
    if (hint.hidden) {
      return;
    }

    const before = field.value.slice(0, field.selectionStart);
    const lines = before.split(NEWLINE);
    const style = getComputedStyle(field);
    const lineHeight = parseFloat(style.lineHeight) || 22;
    const measure = document.createElement('span');
    measure.style.position = 'fixed';
    measure.style.left = '-100000px';
    measure.style.visibility = 'hidden';
    measure.style.whiteSpace = 'pre';
    measure.style.fontFamily = style.fontFamily;
    measure.style.fontSize = style.fontSize;
    measure.style.fontWeight = style.fontWeight;
    measure.style.fontStyle = style.fontStyle;
    measure.style.letterSpacing = style.letterSpacing;
    measure.style.tabSize = style.tabSize;
    measure.textContent = lines[lines.length - 1];
    document.body.appendChild(measure);
    const lineWidth = measure.getBoundingClientRect().width;
    measure.remove();

    const shellRectangle = shell.getBoundingClientRect();
    const viewportWidth = document.documentElement.clientWidth;
    const viewportHeight = document.documentElement.clientHeight;
    const gutterWidth = gutter ? gutter.offsetWidth : 0;
    const minimumLeft = Math.max(gutterWidth + 8, 8 - shellRectangle.left);
    const maximumRight = Math.min(shell.clientWidth - 8, viewportWidth - shellRectangle.left - 8);
    const minimumTop = Math.max(8, 8 - shellRectangle.top);
    const maximumBottom = Math.min(shell.clientHeight - 8, viewportHeight - shellRectangle.top - 8);
    const caretLeft = field.offsetLeft + parseFloat(style.paddingLeft) + lineWidth - field.scrollLeft;
    const caretTop = field.offsetTop + parseFloat(style.paddingTop) + (lines.length - 1) * lineHeight - field.scrollTop;
    const caretBottom = caretTop + lineHeight;

    if (maximumRight <= minimumLeft || maximumBottom <= minimumTop || caretBottom <= minimumTop || caretTop >= maximumBottom) {
      hint.hidden = true;
      return;
    }

    hint.style.width = Math.min(240, maximumRight - minimumLeft) + 'px';
    hint.style.maxHeight = Math.min(220, maximumBottom - minimumTop) + 'px';
    const desiredHeight = hint.getBoundingClientRect().height;
    const below = Math.max(maximumBottom - caretBottom - 4, 0);
    const above = Math.max(caretTop - minimumTop - 4, 0);
    const opensAbove = below < desiredHeight && above > below;
    hint.style.maxHeight = Math.max(Math.min(220, opensAbove ? above : below), 1) + 'px';
    const popup = hint.getBoundingClientRect();
    const proposedTop = opensAbove ? caretTop - popup.height - 4 : caretBottom + 4;
    hint.style.left = Math.min(Math.max(caretLeft, minimumLeft), Math.max(minimumLeft, maximumRight - popup.width)) + 'px';
    hint.style.top = Math.min(Math.max(proposedTop, minimumTop), Math.max(minimumTop, maximumBottom - popup.height)) + 'px';
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

  function offer(manual) {
    offered = suggestions(manual);
    chosen = 0;
    draw();
  }

  function accept() {
    const item = offered[chosen];

    if (!item) {
      return;
    }

    const after = field.value.slice(field.selectionStart);
    const inserted = item.insert || item.label;
    field.value = field.value.slice(0, anchor) + inserted + after;

    const caret = anchor + inserted.length;
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
    place();
  });

  field.addEventListener('click', function () { offer(); });
  field.addEventListener('keyup', function (event) {
    if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) {
      offer();
    }
  });
  window.addEventListener('resize', place);
  if (typeof ResizeObserver === 'function') {
    new ResizeObserver(place).observe(field);
  }

  field.addEventListener('blur', function () {
    hint.hidden = true;
  });

  field.addEventListener('keydown', function (event) {
    if (event.ctrlKey && (event.code === 'Space' || event.key === ' ')) {
      event.preventDefault();
      offer(true);
      return;
    }

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
