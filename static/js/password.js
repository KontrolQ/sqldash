(function () {
  const fields = document.querySelectorAll('[data-password]');

  if (fields.length === 0) {
    return;
  }

  fields.forEach(function (field) {
    const input = field.querySelector('input');
    const toggle = field.querySelector('[data-password-toggle]');

    if (!input || !toggle) {
      return;
    }

    toggle.addEventListener('click', function () {
      const hidden = input.getAttribute('type') === 'password';

      input.setAttribute('type', hidden ? 'text' : 'password');
      toggle.setAttribute('aria-label', hidden ? 'Hide password' : 'Show password');
      toggle.setAttribute('aria-pressed', hidden ? 'true' : 'false');
      field.classList.toggle('is-showing', hidden);
      input.focus();
    });
  });
})();
