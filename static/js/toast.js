(function () {
  let stack = null;

  function shelf() {
    if (stack) {
      return stack;
    }

    stack = document.createElement('div');
    stack.className = 'toasts';
    stack.setAttribute('aria-live', 'polite');
    document.body.appendChild(stack);

    return stack;
  }

  function raise(inner, tone, delay) {
    const toast = document.createElement('div');
    toast.className = 'toast ' + (tone === 'bad' ? 'is-bad' : 'is-good');
    toast.setAttribute('role', tone === 'bad' ? 'alert' : 'status');
    toast.innerHTML = inner;

    const close = document.createElement('button');
    close.className = 'toast-close';
    close.type = 'button';
    close.setAttribute('aria-label', 'Dismiss');
    close.textContent = '×';
    toast.appendChild(close);

    shelf().appendChild(toast);

    function dismiss() {
      toast.classList.add('is-going');
      toast.addEventListener('animationend', function () {
        toast.remove();
      });
    }

    close.addEventListener('click', dismiss);
    setTimeout(dismiss, 5000 + (delay || 0));
  }

  function escaped(text) {
    const holder = document.createElement('span');
    holder.textContent = text;

    return holder.innerHTML;
  }

  document.addEventListener('sqldash:toast', function (event) {
    raise(escaped(event.detail.message), event.detail.tone);
  });

  document.querySelectorAll('[data-toast]').forEach(function (flash, position) {
    raise(flash.innerHTML, flash.dataset.toast, position * 400);
    flash.remove();
  });
})();
