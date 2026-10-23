(function () {
  const buttons = document.querySelectorAll('[data-copy]');

  if (buttons.length === 0) {
    return;
  }

  function announce(message, tone) {
    document.dispatchEvent(new CustomEvent('sqldash:toast', { detail: { message: message, tone: tone } }));
  }

  buttons.forEach(function (button) {
    button.addEventListener('click', async function () {
      const target = document.getElementById(button.dataset.copy);

      if (!target) {
        return;
      }

      const what = button.dataset.copyLabel || 'That';
      const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA';
      const value = isInput ? target.value : target.textContent.trim();

      try {
        await navigator.clipboard.writeText(value);
        announce(what + ' was copied.');
      } catch (copyError) {
        if (isInput) {
          target.focus();
          target.select();
        } else {
          const range = document.createRange();
          range.selectNodeContents(target);
          window.getSelection().removeAllRanges();
          window.getSelection().addRange(range);
        }
        announce('It is selected — press Ctrl+C to copy.', 'bad');
      }
    });
  });
})();
