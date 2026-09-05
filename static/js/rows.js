(function () {
  const veil = document.querySelector('[data-statement-veil]');

  if (!veil) {
    return;
  }

  const body = veil.querySelector('[data-statement-body]');
  const copy = veil.querySelector('[data-statement-copy]');

  let opener = null;

  function close() {
    veil.hidden = true;

    if (opener) {
      opener.focus();
      opener = null;
    }
  }

  function open(button) {
    const held = document.getElementById(button.dataset.statementOpen);

    if (!held) {
      return;
    }

    opener = button;
    body.textContent = held.content.textContent.trim();
    veil.hidden = false;

    const first = veil.querySelector('button');

    if (first) {
      first.focus();
    }
  }

  document.querySelectorAll('[data-statement-open]').forEach(function (button) {
    button.addEventListener('click', function () {
      open(button);
    });
  });

  veil.querySelectorAll('[data-statement-close]').forEach(function (button) {
    button.addEventListener('click', close);
  });

  if (copy) {
    copy.addEventListener('click', function () {
      navigator.clipboard.writeText(body.textContent).then(function () {
        document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: 'Statement copied.' } }));
      });
    });
  }

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && !veil.hidden) {
      close();
    }
  });
})();
