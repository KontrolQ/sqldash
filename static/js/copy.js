(function () {
  const buttons = document.querySelectorAll('[data-copy]');

  if (buttons.length === 0) {
    return;
  }

  buttons.forEach(function (button) {
    button.addEventListener('click', async function () {
      const target = document.getElementById(button.dataset.copy);

      if (!target) {
        return;
      }

      const was = button.textContent;

      try {
        await navigator.clipboard.writeText(target.textContent.trim());
        button.textContent = 'Copied';
      } catch (copyError) {
        button.textContent = 'Press Ctrl+C';

        const range = document.createRange();
        range.selectNodeContents(target);
        window.getSelection().removeAllRanges();
        window.getSelection().addRange(range);
      }

      setTimeout(function () {
        button.textContent = was;
      }, 1600);
    });
  });
})();
