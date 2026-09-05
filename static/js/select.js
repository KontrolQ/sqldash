(function () {
  const fields = document.querySelectorAll('[data-select]');

  if (fields.length === 0) {
    return;
  }

  fields.forEach(function (field) {
    const native = field.querySelector('select');
    const trigger = field.querySelector('[data-select-trigger]');
    const label = field.querySelector('[data-select-label]');
    const content = field.querySelector('[data-select-content]');

    if (!native || !trigger || !content || !label) {
      return;
    }

    native.classList.add('select-native');
    content.innerHTML = '';

    Array.from(native.options).forEach(function (option) {
      const item = document.createElement('div');

      item.className = 'select-item';
      item.setAttribute('role', 'option');
      item.dataset.value = option.value;
      item.textContent = option.textContent;

      const mark = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      mark.setAttribute('class', 'select-mark');
      mark.setAttribute('viewBox', '0 0 24 24');
      mark.innerHTML = '<path d="M20 6 9 17l-5-5"/>';
      item.appendChild(mark);

      if (option.selected) {
        item.classList.add('is-chosen');
      }

      item.addEventListener('click', function () {
        native.value = option.value;
        native.dispatchEvent(new Event('change', { bubbles: true }));
        show();
        close();
      });

      content.appendChild(item);
    });

    function show() {
      const chosen = native.options[native.selectedIndex];

      label.textContent = chosen ? chosen.textContent : '';

      content.querySelectorAll('.select-item').forEach(function (item) {
        item.classList.toggle('is-chosen', item.dataset.value === native.value);
      });
    }

    function open() {
      content.hidden = false;
      trigger.setAttribute('aria-expanded', 'true');
    }

    function close() {
      content.hidden = true;
      trigger.setAttribute('aria-expanded', 'false');
    }

    trigger.addEventListener('click', function (event) {
      event.preventDefault();
      content.hidden ? open() : close();
    });

    document.addEventListener('click', function (event) {
      if (!field.contains(event.target)) {
        close();
      }
    });

    document.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') {
        close();
      }
    });

    native.addEventListener('change', show);
    show();
    close();
  });
})();
