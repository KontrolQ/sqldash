(() => {
  const form = document.querySelector('[data-role-editor]');
  const panel = form?.querySelector('[data-swap-panel="role-permissions"]');
  if (!panel) return;
  const search = panel.querySelector('[data-permission-search]');
  const groups = [...panel.querySelectorAll('[data-permission-group]')];
  const sections = new Map();
  const holder = document.createElement('div');
  holder.className = 'roles-permission-sections';
  panel.append(holder);
  for (const [engine, name] of [['administration', 'Administration'], ['libsql', 'libSQL'], ['postgres', 'PostgreSQL'], ['redis', 'Redis']]) {
    const section = document.createElement('details');
    section.className = 'card roles-permission-engine';
    section.dataset.permissionEngine = engine;
    const heading = document.createElement('summary');
    heading.className = 'roles-engine-heading';
    const title = document.createElement('span'); title.textContent = name;
    const count = document.createElement('span'); count.className = 'roles-engine-count';
    heading.append(title, count);
    const body = document.createElement('div'); body.className = 'roles-engine-body';
    section.append(heading, body); holder.append(section);
    sections.set(engine, {section, body, count, name, groups: new Map()});
  }
  for (const group of groups) {
    const key = group.querySelector('input[name="permissions"]')?.value || '';
    const prefix = key.split('.')[0];
    const engine = sections.get(prefix) || sections.get('administration');
    const title = group.querySelector('h2');
    const category = title.textContent.split(' · ').at(-1);
    title.textContent = category;
    if (engine.groups.has(category)) {
      const existing = engine.groups.get(category);
      for (const row of [...group.querySelectorAll('[data-permission-row]')]) existing.append(row);
      group.remove();
    } else {
      engine.groups.set(category, group); engine.body.append(group);
    }
  }
  function updateCounts() {
    for (const engine of sections.values()) {
      const inputs = [...engine.body.querySelectorAll('input[name="permissions"]')];
      engine.count.textContent = inputs.filter(input => input.checked).length + ' / ' + inputs.length + ' enabled';
    }
  }
  updateCounts();
  for (const [key, engine] of sections) engine.section.open = key === 'administration' || !!engine.body.querySelector('input:checked');
  let searching = false;
  const previouslyOpen = new Map();
  search.addEventListener('input', () => {
    const query = search.value.trim().toLowerCase();
    if (query && !searching) for (const engine of sections.values()) previouslyOpen.set(engine, engine.section.open);
    for (const engine of sections.values()) {
      for (const [category, group] of engine.groups) {
        const rows = [...group.querySelectorAll('[data-permission-row]')];
        for (const row of rows) row.hidden = !(engine.name + ' ' + category + ' ' + row.textContent).toLowerCase().includes(query);
        group.hidden = rows.every(row => row.hidden);
      }
      engine.section.hidden = [...engine.groups.values()].every(group => group.hidden);
      if (query && !engine.section.hidden) engine.section.open = true;
      else if (!query && searching) engine.section.open = previouslyOpen.get(engine);
    }
    searching = !!query;
  });
  form.addEventListener('change', updateCounts);
  form.addEventListener('click', event => { if (event.target.closest('[data-permissions-clear]')) updateCounts(); });
})();
