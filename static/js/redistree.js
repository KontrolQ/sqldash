(() => {
  'use strict';
  const paths = {
    folder: ['M3 7V5a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z'],
    openFolder: ['M3 8V5a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v2', 'M3 8h18l-3 12H3Z'],
    string: ['M4 5h16M12 5v14M8 19h8M4 5v3M20 5v3'],
    hash: ['M4 8h16M4 16h16M10 3 8 21M16 3l-2 18'],
    list: ['M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01'],
    set: ['M9 5a7 7 0 1 0 0 14 7 7 0 0 0 0-14ZM15 5a7 7 0 1 0 0 14 7 7 0 0 0 0-14Z'],
    zset: ['M3 5h7M3 12h12M3 19h17M18 3v9m-3-3 3 3 3-3'],
    stream: ['M3 6h13m-3-3 3 3-3 3M8 12h13m-3-3 3 3-3 3M3 18h13m-3-3 3 3-3 3'],
    json: ['M8 3H6v6l-3 3 3 3v6h2M16 3h2v6l3 3-3 3v6h-2'],
    vector: ['M5 5h.01M19 5h.01M12 19h.01M5 5l7 14 7-14H5'],
    timeseries: ['M3 3v18h18M5 15l4-6 4 4 5-8'],
    bloom: ['M3 4h18l-7 8v7l-4 2v-9Z', 'M8 4v3M12 4v3M16 4v3'],
    cuckoo: ['M3 4h18l-7 8v7l-4 2v-9Z', 'M9 4l6 4M15 4 9 8'],
    cms: ['M3 4h18v16H3Z', 'M3 9h18M3 14h18M9 4v16M15 4v16'],
    topk: ['M4 20V10h4v10M10 20V4h4v16M16 20v-7h4v7'],
    tdigest: ['M3 19h18M4 16c4 0 3-12 8-12s4 12 8 12M8 16v3M12 12v7M16 16v3'],
    unknown: ['M4 5h16v14H4ZM4 10h16M9 10v9']
  };

  function icon(kind) {
    const result = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    result.setAttribute('viewBox', '0 0 24 24');
    result.setAttribute('class', 'redis-key-icon');
    result.setAttribute('fill', 'none');
    result.setAttribute('stroke', 'currentColor');
    result.setAttribute('stroke-width', '1.6');
    result.setAttribute('stroke-linecap', 'round');
    result.setAttribute('stroke-linejoin', 'round');
    result.setAttribute('aria-hidden', 'true');
    result.setAttribute('focusable', 'false');
    result.setAttribute('data-redis-icon', kind);
    for (const shape of paths[kind] || paths.unknown) {
      const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
      path.setAttribute('d', shape);
      result.append(path);
    }
    return result;
  }

  function typeIcon(type) {
    const kind = String(type).toLowerCase();
    if (Object.hasOwn(paths, kind)) return kind;
    if (kind.includes('json')) return 'json';
    if (kind.includes('vset') || kind.includes('vector')) return 'vector';
    if (kind.includes('tsdb') || kind.includes('timeseries')) return 'timeseries';
    if (kind.includes('cuckoo') || kind === 'mbbloomcf') return 'cuckoo';
    if (kind.includes('bloom')) return 'bloom';
    if (kind.includes('cms')) return 'cms';
    if (kind.includes('topk')) return 'topk';
    if (kind.includes('tdis')) return 'tdigest';
    return 'unknown';
  }

  function render(target, summaries, options) {
    const root = { folders: new Map(), leaves: [], count: 0 };
    const { element, label, selected, expanded, safely, onSelect } = options;
    const ordered = [...summaries].sort((left, right) => label(left.key).localeCompare(label(right.key)));
    for (const summary of ordered) {
      const shown = label(summary.key);
      let parts = summary.key.text != null ? shown.split(':') : [shown];
      if (parts.length > 12) parts = [...parts.slice(0, 11), parts.slice(11).join(':')];
      let parent = root;
      let prefix = '';
      for (const part of parts.slice(0, -1)) {
        prefix += ':' + part;
        if (!parent.folders.has(part)) parent.folders.set(part, { prefix, name: part, folders: new Map(), leaves: [], count: 0 });
        parent = parent.folders.get(part);
        parent.count++;
      }
      parent.leaves.push({ summary, shown, name: parts.at(-1) });
      root.count++;
    }

    function appendContents(parent, container) {
      for (const folder of parent.folders.values()) {
        const group = element('details');
        const heading = element('summary', null, 'rail-group');
        let folderIcon = icon(expanded.has(folder.prefix) ? 'openFolder' : 'folder');
        heading.append(folderIcon, element('span', folder.name || '(empty prefix)', 'rail-name'), element('span', String(folder.count), 'tally'));
        group.append(heading);
        let populated = false;
        const populate = () => {
          if (!populated) { populated = true; appendContents(folder, group); }
        };
        group.open = expanded.has(folder.prefix);
        if (group.open) populate();
        group.addEventListener('toggle', () => {
          if (group.open) { expanded.add(folder.prefix); populate(); }
          else expanded.delete(folder.prefix);
          const replacement = icon(group.open ? 'openFolder' : 'folder');
          folderIcon.replaceWith(replacement);
          folderIcon = replacement;
        });
        container.append(group);
      }
      for (const leaf of parent.leaves) {
        const button = element('button', null, 'rail-item' + (selected === leaf.summary.key.data ? ' is-current' : ''));
        button.type = 'button';
        button.title = leaf.shown;
        button.setAttribute('aria-pressed', String(selected === leaf.summary.key.data));
        button.append(icon(typeIcon(leaf.summary.type)), element('span', leaf.name, 'rail-name'), element('span', leaf.summary.type === 'ReJSON-RL' ? 'JSON' : leaf.summary.type, 'badge is-outline'));
        button.addEventListener('click', safely(() => onSelect(leaf.summary)));
        container.append(button);
      }
    }

    target.replaceChildren();
    appendContents(root, target);
    if (!root.count) target.append(element('div', 'No keys in this scan window. Continue scanning if the cursor is not complete.', 'rail-empty'));
  }

  window.SqlDashRedisTree = { render };
})();
