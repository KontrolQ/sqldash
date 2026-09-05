(function () {
  const triggers = document.querySelectorAll('[data-reveal]');

  if (triggers.length === 0) {
    return;
  }

  triggers.forEach(function (trigger) {
    trigger.addEventListener('click', function () {
      const target = document.getElementById(trigger.dataset.reveal);

      if (!target) {
        return;
      }

      target.hidden = !target.hidden;

      if (!target.hidden) {
        const first = target.querySelector('input, select, textarea');

        if (first) {
          first.focus();
        }
      }
    });
  });
})();
