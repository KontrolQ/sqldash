(function () {
  const menus = [...document.querySelectorAll('[data-menu]')];

  if (menus.length === 0) {
    return;
  }

  function closeAll(except) {
    menus.forEach(function (menu) {
      if (menu !== except) {
        menu.querySelector('[data-menu-content]').hidden = true;
        menu.querySelector('[data-menu-trigger]').setAttribute('aria-expanded', 'false');
      }
    });
  }

  menus.forEach(function (menu) {
    const trigger = menu.querySelector('[data-menu-trigger]');
    const content = menu.querySelector('[data-menu-content]');

    trigger.addEventListener('click', function (event) {
      event.stopPropagation();
      const opening = content.hidden;
      closeAll(menu);
      content.hidden = !opening;
      trigger.setAttribute('aria-expanded', String(opening));
    });

    content.addEventListener('click', function (event) {
      event.stopPropagation();
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
})();
