(function () {
  const field = document.querySelector('[data-statement]');

  if (!field) {
    return;
  }

  const form = field.form;

  field.addEventListener('keydown', function (event) {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      form.requestSubmit();
    }
  });

  const clauses = [
    'SELECT', 'FROM', 'WHERE', 'GROUP BY', 'HAVING', 'ORDER BY', 'LIMIT', 'OFFSET',
    'INSERT INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE FROM', 'RETURNING',
    'INNER JOIN', 'LEFT OUTER JOIN', 'LEFT JOIN', 'RIGHT JOIN', 'CROSS JOIN', 'JOIN',
    'UNION ALL', 'UNION', 'EXCEPT', 'INTERSECT',
  ];

  const words = [
    'AND', 'OR', 'NOT', 'ON', 'AS', 'IN', 'IS', 'NULL', 'LIKE', 'BETWEEN', 'EXISTS',
    'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'DISTINCT', 'ASC', 'DESC', 'COUNT', 'SUM',
    'AVG', 'MIN', 'MAX', 'CAST', 'COALESCE', 'CREATE', 'TABLE', 'INDEX', 'VIEW',
    'PRIMARY', 'KEY', 'REFERENCES', 'DEFAULT', 'UNIQUE', 'IF', 'BEGIN', 'COMMIT',
  ];

  function pattern(phrase) {
    return new RegExp('\\s*\\b' + phrase.split(' ').join('\\s+') + '\\b\\s*', 'gi');
  }

  function tidy(sql) {
    let text = sql.replace(/\s+/g, ' ').trim();

    if (text === '') {
      return text;
    }

    words.forEach(function (word) {
      text = text.replace(new RegExp('\\b' + word + '\\b', 'gi'), word);
    });

    clauses.forEach(function (clause) {
      text = text.replace(pattern(clause), '\n' + clause + ' ');
    });

    text = text.replace(/,\s*/g, ',\n  ');
    text = text.replace(/\s*\bAND\b\s*/g, '\n  AND ');
    text = text.replace(/\s*\bOR\b\s*/g, '\n  OR ');

    return text.split('\n').map(function (line) {
      return line.replace(/\s+$/, '');
    }).filter(function (line, index, lines) {
      return line !== '' || index === lines.length - 1;
    }).join('\n').trim();
  }

  const button = document.querySelector('[data-tidy]');

  if (button) {
    button.addEventListener('click', function () {
      field.value = tidy(field.value);
      field.focus();
    });
  }
})();
