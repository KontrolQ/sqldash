(function () {
  const forms = document.querySelectorAll('form[data-confirm]');

  if (forms.length === 0) {
    return;
  }

  let open = null;

  function close() {
    if (!open) {
      return;
    }

    open.veil.remove();
    open.dialog.remove();
    open = null;
  }

  function ask(form) {
    close();

    const veil = document.createElement('div');
    veil.className = 'dialog-veil';

    const dialog = document.createElement('div');
    dialog.className = 'dialog';
    dialog.setAttribute('role', 'alertdialog');
    dialog.setAttribute('aria-modal', 'true');

    const heading = form.dataset.confirmTitle || 'Are you sure?';
    const note = form.dataset.confirm;
    const proceed = form.dataset.confirmAction || 'Continue';

    dialog.innerHTML =
      '<div class="dialog-head">' +
      '<div class="dialog-title"></div>' +
      '<div class="dialog-note"></div>' +
      '</div>' +
      '<div class="dialog-foot">' +
      '<button class="button is-outline" type="button" data-dialog-cancel>Cancel</button>' +
      '<button class="button is-destructive" type="button" data-dialog-go></button>' +
      '</div>';

    dialog.querySelector('.dialog-title').textContent = heading;
    dialog.querySelector('.dialog-note').textContent = note;
    dialog.querySelector('[data-dialog-go]').textContent = proceed;

    document.body.appendChild(veil);
    document.body.appendChild(dialog);

    open = { veil: veil, dialog: dialog };

    dialog.querySelector('[data-dialog-go]').addEventListener('click', function () {
      close();
      form.dataset.confirmed = 'yes';
      form.submit();
    });

    dialog.querySelector('[data-dialog-cancel]').addEventListener('click', close);
    veil.addEventListener('click', close);
    dialog.querySelector('[data-dialog-cancel]').focus();
  }

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      close();
    }
  });

  forms.forEach(function (form) {
    form.addEventListener('submit', function (event) {
      if (form.dataset.confirmed === 'yes') {
        return;
      }

      event.preventDefault();
      ask(form);
    });
  });
})();
