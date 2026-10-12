(function () {
  const menus = [...document.querySelectorAll('[data-menu]')];

  if (menus.length === 0) {
    return;
  }

  const EDGE = 8;

  function shut(menu) {
    const content = menu.querySelector('[data-menu-content]');

    content.hidden = true;
    content.style.position = '';
    content.style.top = '';
    content.style.left = '';
    content.style.right = '';
    content.style.maxHeight = '';

    menu.querySelector('[data-menu-trigger]').setAttribute('aria-expanded', 'false');
  }

  function closeAll(except) {
    menus.forEach(function (menu) {
      if (menu !== except) {
        shut(menu);
      }
    });
  }

  function place(trigger, content) {
    const anchor = trigger.getBoundingClientRect();

    content.style.position = 'fixed';
    content.style.right = 'auto';
    content.style.top = '0px';
    content.style.left = '0px';
    content.style.maxHeight = '';

    const size = content.getBoundingClientRect();
    const roomBelow = window.innerHeight - anchor.bottom - EDGE;
    const roomAbove = anchor.top - EDGE;
    const above = size.height > roomBelow && roomAbove > roomBelow;

    let left = anchor.right - size.width;

    if (left < EDGE) {
      left = anchor.left;
    }

    left = Math.min(left, window.innerWidth - size.width - EDGE);

    if (left < EDGE) {
      left = EDGE;
    }

    content.style.left = Math.round(left) + 'px';
    content.style.maxHeight = Math.round(Math.max(roomBelow, roomAbove)) + 'px';

    if (above) {
      content.style.top = Math.round(Math.max(EDGE, anchor.top - size.height - 4)) + 'px';
      return;
    }

    content.style.top = Math.round(anchor.bottom + 4) + 'px';
  }

  menus.forEach(function (menu) {
    const trigger = menu.querySelector('[data-menu-trigger]');
    const content = menu.querySelector('[data-menu-content]');

    trigger.addEventListener('click', function (event) {
      event.stopPropagation();

      const opening = content.hidden;

      closeAll(menu);

      if (!opening) {
        shut(menu);
        return;
      }

      content.hidden = false;
      trigger.setAttribute('aria-expanded', 'true');
      place(trigger, content);
    });

    content.addEventListener('click', function (event) {
      event.stopPropagation();
    });

    trigger.addEventListener('keydown', function (event) {
      if (event.key !== 'ArrowDown') { return; }
      event.preventDefault();
      if (content.hidden) { trigger.click(); }
      const first = content.querySelector('a, button:not(:disabled)');
      if (first) { first.focus(); }
    });

    content.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') { shut(menu); trigger.focus(); return; }
      if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) { return; }
      const items = [...content.querySelectorAll('a, button:not(:disabled)')];
      if (items.length === 0) { return; }
      event.preventDefault();
      const current = items.indexOf(document.activeElement);
      let next = event.key === 'ArrowUp' ? (current - 1 + items.length) % items.length : (current + 1) % items.length;
      if (event.key === 'Home') { next = 0; }
      if (event.key === 'End') { next = items.length - 1; }
      items[next].focus();
    });
  });

  document.addEventListener('click', function () {
    closeAll(null);
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      closeAll(null);
    }
  });

  window.addEventListener('resize', function () {
    closeAll(null);
  });

  document.addEventListener('scroll', function () {
    closeAll(null);
  }, true);
})();
