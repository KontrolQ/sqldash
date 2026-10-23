(function () {
  let chosenEngine = '';
  const unconfirmedClone = 'The clone result could not be confirmed. Check the database list before retrying; your inputs are unchanged.';

  function focusSheet(sheet) {
    if (!sheet || sheet.hidden || sheet.dataset.cloneBusy === 'true') return;
    const first = [...sheet.querySelectorAll('input:not([type="hidden"]),select,textarea')].find(function (input) {
      return !input.disabled && input.getClientRects().length > 0;
    });
    if (first) first.focus();
  }

  document.querySelectorAll('[data-sheet-open]').forEach(function (button) {
    button.addEventListener('click', function () {
      if (document.querySelector('[data-sheet][data-clone-busy="true"]:not([hidden])')) return;
      if (button.dataset.engineChoice) chosenEngine = button.dataset.engineChoice;
      const menu = button.closest('[data-menu]');
      if (menu) {
        const content = menu.querySelector('[data-menu-content]');
        const trigger = menu.querySelector('[data-menu-trigger]');
        if (content) { content.hidden = true; ['position', 'top', 'left', 'right', 'maxHeight'].forEach(function (property) { content.style[property] = ''; }); }
        if (trigger) trigger.setAttribute('aria-expanded', 'false');
      }
      requestAnimationFrame(function () {
        const name = button.dataset.sheetOpen;
        const sheet = document.querySelector('[data-sheet="' + name + '"]');
        if (!sheet || sheet.hidden || sheet.dataset.cloneBusy === 'true') return;
        if (name === 'new-database') {
          const choice = sheet.querySelector('[data-engine-choice="' + chosenEngine + '"]') || sheet.querySelector('[data-engine-choice]');
          if (choice) choice.focus();
          return;
        }
        focusSheet(sheet);
      });
    });
  });

  document.querySelectorAll('[data-sheet] [data-swap-to]').forEach(function (tab) {
    tab.addEventListener('click', function () {
      focusSheet(tab.closest('[data-sheet]'));
    });
  });

  function createCloneBusyState(form) {
    const sheet = form.closest('[data-sheet]');
    const dialog = form.closest('[role="dialog"]');
    const submit = form.querySelector('button[type="submit"]');
    const status = form.querySelector('[data-clone-status]');
    let busy = false;
    let buttons = [];
    let busyAttributes = [];
    let sheetBusy = null;
    let submitLabel = '';
    let statusHidden = true;

    function start() {
      busy = true;
      busyAttributes = [form, dialog].filter(Boolean).map(function (element) {
        const value = element.getAttribute('aria-busy');
        element.setAttribute('aria-busy', 'true');
        return { element: element, value: value };
      });
      const controls = new Set((sheet || form).querySelectorAll('button'));
      if (sheet) {
        sheetBusy = sheet.getAttribute('data-clone-busy');
        sheet.dataset.cloneBusy = 'true';
        document.querySelectorAll('[data-sheet-open]').forEach(function (button) {
          if (button.dataset.sheetOpen === sheet.dataset.sheet) controls.add(button);
        });
      }
      buttons = [...controls].map(function (button) {
        const disabled = button.disabled;
        button.disabled = true;
        return { button: button, disabled: disabled };
      });
      submitLabel = submit.textContent;
      submit.textContent = 'Cloning…';
      statusHidden = status.hidden;
      status.hidden = false;
      status.focus();
    }

    function restore() {
      if (!busy) return;
      busy = false;
      buttons.forEach(function (control) { control.button.disabled = control.disabled; });
      submit.textContent = submitLabel;
      status.hidden = statusHidden;
      busyAttributes.forEach(function (attribute) {
        if (attribute.value === null) attribute.element.removeAttribute('aria-busy');
        else attribute.element.setAttribute('aria-busy', attribute.value);
      });
      if (sheet) {
        if (sheetBusy === null) sheet.removeAttribute('data-clone-busy');
        else sheet.setAttribute('data-clone-busy', sheetBusy);
      }
    }

    window.addEventListener('pageshow', function (event) {
      if (event.persisted) restore();
    });

    return {
      start: start,
      restore: restore,
      isBusy: function () { return busy || Boolean(sheet && sheet.dataset.cloneBusy === 'true'); }
    };
  }

  document.querySelectorAll('[data-clone-native]').forEach(function (form) {
    const clone = createCloneBusyState(form);
    form.addEventListener('submit', function (event) {
      if (clone.isBusy()) { event.preventDefault(); return; }
      if (!event.defaultPrevented) clone.start();
    });
  });

  document.querySelectorAll('[data-remote-clone]').forEach(function (remote) {
    const error = remote.querySelector('[data-remote-clone-error]');
    const heading = remote.querySelector('[data-remote-clone-heading]');
    const message = remote.querySelector('[data-remote-clone-message]');
    const clone = createCloneBusyState(remote);

    function showError(text, title) {
      heading.textContent = title || 'Clone failed';
      message.textContent = text.charAt(0).toUpperCase() + text.slice(1);
      error.hidden = false;
    }

    remote.addEventListener('submit', async function (event) {
      event.preventDefault();
      if (clone.isBusy()) return;
      let navigating = false;
      try {
        const request = new FormData(remote);
        clone.start();
        error.hidden = true;
        const response = await fetch(remote.action, {
          method: 'POST',
          body: request,
          headers: { 'Accept': 'application/json', 'X-Remote-Clone': '1' },
          credentials: 'same-origin',
          mode: 'same-origin',
          cache: 'no-store',
          referrerPolicy: 'no-referrer',
          redirect: 'error'
        });
        if (!(response.headers.get('Content-Type') || '').includes('application/json')) {
          showError(response.ok ? unconfirmedClone : 'The clone request was refused (HTTP ' + response.status + '). Your inputs are unchanged. Check your session and permissions before trying again.', response.ok ? 'Clone not confirmed' : 'Clone failed');
          return;
        }
        const result = await response.json();
        if (!response.ok) {
          showError(result && typeof result.error === 'string' ? result.error : 'The clone request failed. Your inputs are unchanged.');
          return;
        }
        if (!result || result.ok !== true) {
          showError(unconfirmedClone, 'Clone not confirmed');
          return;
        }
        window.location.assign(remote.dataset.remoteCloneSuccess);
        navigating = true;
        const password = remote.querySelector('input[name="source_password"]');
        if (password) { password.value = ''; password.removeAttribute('value'); }
      } catch (_) {
        showError(unconfirmedClone, 'Clone not confirmed');
      } finally {
        if (!navigating) {
          clone.restore();
          error.focus();
        }
      }
    }, true);

    const sheet = remote.closest('[data-sheet]');
    if (!sheet) return;
    const observer = new MutationObserver(function () {
      if (clone.isBusy() || (!sheet.hidden && !remote.hidden)) return;
      const password = remote.querySelector('input[name="source_password"]');
      if (password) { password.value = ''; password.removeAttribute('value'); }
    });
    observer.observe(sheet, { attributes: true, attributeFilter: ['hidden'] });
    observer.observe(remote, { attributes: true, attributeFilter: ['hidden'] });
  });
})();
