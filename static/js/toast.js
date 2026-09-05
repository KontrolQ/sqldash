(function () {
  const flashes = document.querySelectorAll('[data-toast]');

  if (flashes.length === 0) {
    return;
  }

  const stack = document.createElement('div');
  stack.className = 'toasts';
  stack.setAttribute('aria-live', 'polite');
  document.body.appendChild(stack);

  flashes.forEach(function (flash, position) {
    const toast = document.createElement('div');
    toast.className = 'toast ' + (flash.dataset.toast === 'bad' ? 'is-bad' : 'is-good');
    toast.setAttribute('role', flash.dataset.toast === 'bad' ? 'alert' : 'status');
    toast.innerHTML = flash.innerHTML;

    const close = document.createElement('button');
    close.className = 'toast-close';
    close.type = 'button';
    close.setAttribute('aria-label', 'Dismiss');
    close.textContent = '\u00d7';
    toast.appendChild(close);

    flash.remove();
    stack.appendChild(toast);

    function dismiss() {
      toast.classList.add('is-going');
      toast.addEventListener('animationend', function () {
        toast.remove();
      });
    }

    close.addEventListener('click', dismiss);
    setTimeout(dismiss, 5000 + position * 400);
  });
})();
