(function () {
  const save = document.querySelector('[data-snippet-save]');
  const fresh = document.querySelector('[data-snippet-new]');
  const field = document.querySelector('[data-statement]');
  const form = document.getElementById('snippet-form');
  const veil = document.querySelector('[data-sheet="name-snippet"]');

  if (fresh) {
    fresh.addEventListener('click', function () {
      window.location.href = window.location.pathname;
    });
  }

  if (!save || !field || !form || !veil) {
    return;
  }

  function close() {
    veil.hidden = true;
    save.focus();
  }

  save.addEventListener('click', function () {
    if (field.value.trim() === '') {
      document.dispatchEvent(new CustomEvent('sqldash:toast', {
        detail: { message: 'Write a statement before saving it.', tone: 'bad' },
      }));

      return;
    }

    form.elements.statement.value = field.value;
    veil.hidden = false;
    form.elements.name.focus();
    form.elements.name.select();
  });

  veil.querySelectorAll('[data-sheet-close]').forEach(function (button) {
    button.addEventListener('click', close);
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && !veil.hidden) {
      close();
    }
  });
})();
