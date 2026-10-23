(function () {
  function attach(field) {
    if (field.dataset.selectReady === 'yes') {
      return;
    }

    const native = field.querySelector('select');
    const trigger = field.querySelector('[data-select-trigger]');
    const label = field.querySelector('[data-select-label]');
    const content = field.querySelector('[data-select-content]');

    if (!native || !trigger || !content || !label) {
      return;
    }

    const originalParent = content.parentNode;
    const originalSibling = content.nextSibling;
    const usesPopover = typeof content.showPopover === 'function' && typeof content.hidePopover === 'function';

    if (usesPopover) {
      content.setAttribute('popover', 'manual');
    }

    field.dataset.selectReady = 'yes';
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

    function position() {
      if (content.hidden) {
        return;
      }

      const viewport = window.visualViewport;
      const viewportLeft = viewport ? viewport.offsetLeft : 0;
      const viewportTop = viewport ? viewport.offsetTop : 0;
      const viewportWidth = viewport ? viewport.width : document.documentElement.clientWidth;
      const viewportHeight = viewport ? viewport.height : document.documentElement.clientHeight;
      const viewportRight = viewportLeft + viewportWidth;
      const viewportBottom = viewportTop + viewportHeight;
      const gutter = 8;
      const gap = 4;
      const rectangle = trigger.getBoundingClientRect();

      if (!field.isConnected || rectangle.bottom <= viewportTop || rectangle.top >= viewportBottom || rectangle.right <= viewportLeft || rectangle.left >= viewportRight) {
        close();
        return;
      }

      const availableWidth = Math.max(viewportWidth - 2 * gutter, 1);
      content.style.minWidth = Math.min(128, availableWidth) + 'px';
      content.style.maxWidth = availableWidth + 'px';
      content.style.width = field.classList.contains('is-wide') ? Math.min(rectangle.width, availableWidth) + 'px' : 'max-content';
      content.style.maxHeight = Math.min(320, Math.max(viewportHeight - 2 * gutter, 1)) + 'px';

      const desiredHeight = content.getBoundingClientRect().height;
      const below = Math.max(viewportBottom - Math.max(rectangle.bottom, viewportTop) - gap - gutter, 0);
      const above = Math.max(Math.min(rectangle.top, viewportBottom) - viewportTop - gap - gutter, 0);
      const opensAbove = below < desiredHeight && above > below;
      const availableHeight = opensAbove ? above : below;
      content.style.maxHeight = Math.max(Math.min(320, availableHeight), 1) + 'px';

      const popup = content.getBoundingClientRect();
      const left = Math.min(Math.max(rectangle.right - popup.width, viewportLeft + gutter), Math.max(viewportLeft + gutter, viewportRight - popup.width - gutter));
      const proposedTop = opensAbove ? rectangle.top - popup.height - gap : rectangle.bottom + gap;
      const top = Math.min(Math.max(proposedTop, viewportTop + gutter), Math.max(viewportTop + gutter, viewportBottom - popup.height - gutter));
      content.style.left = left + 'px';
      content.style.top = top + 'px';
    }

    function open() {
      if (!usesPopover) {
        document.body.appendChild(content);
      }

      content.hidden = false;
      if (usesPopover) {
        content.showPopover();
      }
      trigger.setAttribute('aria-expanded', 'true');
      position();
    }

    function close() {
      if (usesPopover && content.matches(':popover-open')) {
        content.hidePopover();
      }
      content.hidden = true;
      if (!usesPopover && content.parentNode !== originalParent) {
        originalParent.insertBefore(content, originalSibling && originalSibling.parentNode === originalParent ? originalSibling : null);
      }
      trigger.setAttribute('aria-expanded', 'false');
    }

    trigger.addEventListener('click', function (event) {
      event.preventDefault();
      content.hidden ? open() : close();
    });

    document.addEventListener('click', function (event) {
      if (!field.contains(event.target) && !content.contains(event.target)) {
        close();
      }
    });

    window.addEventListener('resize', position);
    window.addEventListener('scroll', function (event) {
      if (!content.contains(event.target)) {
        position();
      }
    }, { capture: true, passive: true });

    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', position);
      window.visualViewport.addEventListener('scroll', position);
    }

    document.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') {
        close();
      }
    });

    native.addEventListener('change', show);
    show();
    close();
  }

  function scan() {
    document.querySelectorAll('[data-select]').forEach(attach);
  }

  document.addEventListener('sqldash:scan', scan);
  scan();
})();
