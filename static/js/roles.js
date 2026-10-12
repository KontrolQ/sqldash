(function () {
  const search = document.querySelector('[data-role-list-search]');
  if (search) {
    search.addEventListener('input', function () {
      const query = search.value.trim().toLowerCase();
      document.querySelectorAll('tbody[data-role-order] [data-role-id]').forEach(function (row) { row.hidden = !row.textContent.toLowerCase().includes(query); });
    });
  }

  document.querySelectorAll('[data-role-order]').forEach(function (list) {
    let dragging = null;
    let saving = false;

    function restore(items) {
      items.forEach(function (item) { list.appendChild(item); });
    }

    function move(item, before) {
      const positions = new Map([...list.children].map(function (child) { return [child, child.getBoundingClientRect().top]; }));
      list.insertBefore(item, before);
      if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) { return; }
      positions.forEach(function (top, child) {
        const distance = top - child.getBoundingClientRect().top;
        if (distance && child !== item) { child.animate([{ transform: 'translateY(' + distance + 'px)' }, { transform: 'translateY(0)' }], { duration: 160, easing: 'ease-out' }); }
      });
    }

    async function finish(cancelled) {
      if (!dragging) { return; }
      const held = dragging;
      dragging = null;
      held.item.classList.remove('is-dragging');
      held.ghost.remove();
      if (cancelled) { restore(held.before); return; }
      const current = [...list.children];
      if (current.every(function (item, index) { return item === held.before[index]; })) { return; }
      saving = true;
      list.setAttribute('aria-busy', 'true');
      try {
        const ordered = [...list.querySelectorAll('[data-role-id][data-role-movable]')].map(function (item) { return Number(item.dataset.roleId); });
        const response = await fetch(list.dataset.roleOrderUrl, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ roles: ordered }) });
        if (!response.ok) { throw new Error('Reorder refused'); }
        document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: 'Role order saved.' } }));
      } catch {
        restore(held.before);
        document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: 'Role order could not be saved.', tone: 'bad' } }));
      } finally {
        saving = false;
        list.removeAttribute('aria-busy');
      }
    }

    list.addEventListener('pointerdown', function (event) {
      const handle = event.target.closest('[data-role-drag]');
      if (!handle || saving || event.button !== 0) { return; }
      const item = handle.closest('[data-role-id][data-role-movable]');
      if (!item) { return; }
      event.preventDefault();
      const bounds = item.getBoundingClientRect();
      const ghost = document.createElement('div');
      ghost.className = 'roles-drag-preview';
      const name = item.querySelector('[data-role-name-label]');
      ghost.textContent = name ? name.textContent.trim() : item.textContent.trim();
      ghost.style.width = bounds.width + 'px';
      ghost.style.left = bounds.left + 'px';
      ghost.style.top = bounds.top + 'px';
      document.body.appendChild(ghost);
      item.classList.add('is-dragging');
      handle.setPointerCapture(event.pointerId);
      dragging = { item: item, before: [...list.children], ghost: ghost, start: event.clientY, top: bounds.top, pointer: event.pointerId };
    });

    list.addEventListener('pointermove', function (event) {
      if (!dragging || event.pointerId !== dragging.pointer) { return; }
      event.preventDefault();
      dragging.ghost.style.transform = 'translateY(' + (event.clientY - dragging.start) + 'px)';
      const candidates = [...list.querySelectorAll('[data-role-id][data-role-movable]')].filter(function (item) { return item !== dragging.item && !item.hidden; });
      const target = candidates.find(function (item) { const bounds = item.getBoundingClientRect(); return event.clientY < bounds.top + bounds.height / 2; });
      const before = target || (candidates.length ? candidates[candidates.length - 1].nextSibling : null);
      if (before !== dragging.item && before !== dragging.item.nextSibling) { move(dragging.item, before); }
    });

    list.addEventListener('pointerup', function () { finish(false); });
    list.addEventListener('pointercancel', function () { finish(true); });
    list.addEventListener('click', function (event) { if (event.target.closest('[data-role-drag]')) { event.preventDefault(); event.stopPropagation(); } });
    document.addEventListener('keydown', function (event) { if (event.key === 'Escape' && dragging) { finish(true); } });
  });
})();
