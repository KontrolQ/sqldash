(function () {
  document.querySelectorAll('[data-postgres-ssl]').forEach(function (choice) {
    const checkbox = choice.querySelector('input[type="checkbox"]');
    const uri = document.getElementById(choice.dataset.postgresSsl);
    const guidance = document.getElementById(choice.dataset.postgresSslHelp);
    if (!checkbox || !uri) return;
    function updateSSLChoice() {
      const current = uri.tagName === 'INPUT' ? uri.value : uri.textContent;
      const updated = current.replace(/([?&]sslmode=)[^&#]*/, function (match, prefix) {
        return prefix + (checkbox.checked ? 'verify-full' : 'disable');
      });
      if (uri.tagName === 'INPUT') uri.value = updated;
      else uri.textContent = updated;
      if (guidance) guidance.hidden = !checkbox.checked;
    }
    checkbox.addEventListener('change', updateSSLChoice);
    updateSSLChoice();
  });
  document.querySelectorAll('[data-postgres-copy-input], [data-postgres-copy-password]').forEach(function (button) {
    button.addEventListener('click', async function () {
      const input = document.getElementById(button.dataset.postgresCopyInput || button.dataset.postgresCopyPassword);
      if (!input) return;
      const label = button.dataset.postgresCopyLabel || (button.dataset.postgresCopyInput ? 'The connection URI' : 'The password');
      try {
        await navigator.clipboard.writeText(input.value);
        document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: label + ' was copied.' } }));
      } catch (error) {
        input.type = 'text';
        input.focus();
        input.select();
        document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: 'It is selected — press Ctrl+C to copy.', tone: 'bad' } }));
      }
    });
  });
  window.addEventListener('pagehide', function () {
    document.querySelectorAll('#postgres-connection-uri, #issued-postgres-password, #postgres-overview-password').forEach(function (input) { input.value = ''; input.removeAttribute('value'); });
  });
})();
