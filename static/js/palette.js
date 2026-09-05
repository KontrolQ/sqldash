(function () {
  const palette = document.querySelector('.palette');

  if (!palette) {
    return;
  }

  const field = palette.querySelector('.command-input');
  const list = palette.querySelector('.command-list');
  const nothing = palette.querySelector('.command-empty');
  const groups = [...palette.querySelectorAll('[data-group]')];
  const items = [...palette.querySelectorAll('[data-item]')];

  let active = null;

  function shown() {
    return items.filter(function (item) {
      return !item.hidden;
    });
  }

  function highlight(item) {
    if (active) {
      active.classList.remove('is-active');
      active.removeAttribute('aria-selected');
    }

    active = item;

    if (!active) {
      return;
    }

    active.classList.add('is-active');
    active.setAttribute('aria-selected', 'true');
    active.scrollIntoView({ block: 'nearest' });
  }

  function filter() {
    const needle = field.value.trim().toLowerCase();

    items.forEach(function (item) {
      item.hidden = needle !== '' && !item.textContent.toLowerCase().includes(needle);
    });

    groups.forEach(function (group) {
      group.hidden = group.querySelectorAll('[data-item]:not([hidden])').length === 0;
    });

    const left = shown();
    nothing.hidden = left.length > 0;
    highlight(left[0] || null);
  }

  function open() {
    palette.hidden = false;
    field.value = '';
    filter();
    field.focus();
    list.scrollTop = 0;
  }

  function close() {
    palette.hidden = true;
  }

  function step(by) {
    const left = shown();

    if (left.length === 0) {
      return;
    }

    const at = left.indexOf(active);
    highlight(left[(at + by + left.length) % left.length]);
  }

  document.querySelectorAll('[data-palette-open]').forEach(function (button) {
    button.addEventListener('click', open);
  });

  palette.querySelectorAll('[data-palette-close]').forEach(function (veil) {
    veil.addEventListener('click', close);
  });

  items.forEach(function (item) {
    item.addEventListener('mousemove', function () {
      highlight(item);
    });
  });

  field.addEventListener('input', filter);

  field.addEventListener('keydown', function (event) {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      step(1);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      step(-1);
    } else if (event.key === 'Enter' && active) {
      event.preventDefault();
      active.click();
    }
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && !palette.hidden) {
      close();
    }
  });
})();
