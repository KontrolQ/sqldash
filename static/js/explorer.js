(function () {
  const form = document.querySelector('[data-object-form]');
  if (!form) return;
  const kind = form.elements.kind;
  const action = form.elements.action;
  const constraint = form.elements.constraint_kind;
  const operations = {
    table: ['create', 'rename', 'drop'],
    view: ['create', 'replace', 'rename', 'drop'],
    'materialized view': ['create', 'refresh', 'rename', 'drop'],
    column: ['add', 'rename', 'drop', 'type', 'default', 'drop_default', 'not_null', 'nullable'],
    constraint: ['add', 'rename', 'drop'],
    index: ['create', 'rename', 'drop'],
    function: ['create', 'replace', 'rename', 'drop'],
    sequence: ['create', 'alter', 'rename', 'drop'],
  };

  function update() {
    const allowed = operations[kind.value] || [];
    Array.from(action.options).forEach(function (option) {
      option.disabled = !allowed.includes(option.value);
      option.hidden = option.disabled;
    });
    if (!allowed.includes(action.value)) action.value = allowed[0];
    const actionShell = action.closest('[data-select]');
    actionShell.querySelectorAll('[data-value]').forEach(function (item) {
      item.hidden = !allowed.includes(item.dataset.value);
    });
    actionShell.querySelector('[data-select-label]').textContent = action.selectedOptions[0].textContent;
    const currentKind = kind.value;
    const operation = action.value;
    const createFunction = currentKind === 'function' && ['create', 'replace'].includes(operation);
    const visible = new Set();
    if (operation === 'create' || createFunction || ['index', 'constraint', 'sequence'].includes(currentKind)) visible.add('name');
    if (['table', 'view', 'materialized view'].includes(currentKind) && operation !== 'create') visible.add('table');
    if (['column', 'constraint', 'index'].includes(currentKind)) visible.add('table');
    if (operation === 'rename') visible.add('new_name');
    if (currentKind === 'column') visible.add('column');
    if (currentKind === 'column' && ['add', 'type'].includes(operation)) visible.add('type');
    if (currentKind === 'column' && operation === 'add') {
      visible.add('not_null'); visible.add('column_options');
      if (form.elements.mode.value === 'generated') visible.add('expression');
      if (form.elements.default_state.value === 'value') visible.add('value');
    }
    if (currentKind === 'column' && operation === 'default') visible.add('value');
    if (currentKind === 'table' && operation === 'create') {
      visible.add('column_definitions'); visible.add('target_schema');
    }
    if (['view', 'materialized view'].includes(currentKind) && ['create', 'replace'].includes(operation)) visible.add('expression');
    if (currentKind === 'index' && operation === 'create') { visible.add('columns'); visible.add('unique'); }
    if (currentKind === 'constraint' && operation === 'add') {
      visible.add('constraint_kind');
      if (constraint.value === 'check') visible.add('expression'); else visible.add('columns');
      if (constraint.value === 'foreign') visible.add('references');
    }
    if (createFunction) { visible.add('function'); visible.add('expression'); }
    if (currentKind === 'function' && ['drop', 'rename'].includes(operation)) visible.add('signature');
    if (currentKind === 'sequence' && ['create', 'alter'].includes(operation)) visible.add('sequence');
    if (operation === 'drop') visible.add('destructive');
    form.querySelectorAll('[data-object-field]').forEach(function (section) {
      section.hidden = !visible.has(section.dataset.objectField);
      section.querySelectorAll('input, select, textarea').forEach(function (field) { field.disabled = section.hidden; });
    });
  }

  kind.addEventListener('change', update);
  action.addEventListener('change', update);
  constraint.addEventListener('change', update);
  form.elements.mode.addEventListener('change', update);
  form.elements.default_state.addEventListener('change', update);
  const rows = form.querySelector('[data-object-column-rows]');
  const prototype = rows.firstElementChild.cloneNode(true);
  prototype.querySelectorAll('[data-select]').forEach(function (select) { delete select.dataset.selectReady; });
  form.querySelector('[data-object-column-add]').addEventListener('click', function () {
    const row = prototype.cloneNode(true);
    row.querySelector('input').value = '';
    row.querySelectorAll('input, select').forEach(function (field) { field.disabled = false; });
    rows.appendChild(row);
    document.dispatchEvent(new Event('sqldash:scan'));
  });
  rows.addEventListener('click', function (event) {
    if (event.target.closest('[data-object-column-remove]') && rows.children.length > 1) event.target.closest('.explorer-column-definition').remove();
  });
  function identifierList(value) {
    const values = []; let held = ''; let quoted = false; let wasQuoted = false;
    for (let index = 0; index < value.length; index += 1) {
      const character = value[index];
      if (character === '"') {
        if (quoted && value[index + 1] === '"') { held += '"'; index += 1; }
        else { quoted = !quoted; wasQuoted = true; }
      } else if (character === ',' && !quoted) {
        values.push(wasQuoted ? held : held.trim()); held = ''; wasQuoted = false;
      } else if (quoted || !wasQuoted || character.trim() !== '') held += character;
    }
    if (quoted) throw new Error('Close every quoted column name.');
    values.push(wasQuoted ? held : held.trim());
    return values.filter(function (name) { return name !== ''; });
  }

  form.addEventListener('submit', function (event) {
    form.querySelectorAll('[data-object-generated]').forEach(function (field) { field.remove(); });
    [['[data-object-column-list]', 'columns'], ['[data-object-reference-list]', 'reference_columns'], ['[data-object-argument-list]', 'argument_types']].forEach(function (entry) {
      const source = form.querySelector(entry[0]);
      if (source.disabled) return;
      let values;
      try { values = identifierList(source.value); }
      catch (error) { event.preventDefault(); source.setCustomValidity(error.message); source.reportValidity(); source.addEventListener('input', function () { source.setCustomValidity(''); }, { once: true }); return; }
      values.forEach(function (value) {
        const field = document.createElement('input');
        field.type = 'hidden'; field.name = entry[1]; field.value = value; field.dataset.objectGenerated = 'yes'; form.appendChild(field);
      });
    });
  });
  update();
})();
