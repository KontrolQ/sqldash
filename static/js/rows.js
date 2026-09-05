(function () {
  document.querySelectorAll('[data-open]').forEach(function (button) {
    button.addEventListener('click', function () {
      const target = document.getElementById(button.dataset.open);

      if (!target) {
        return;
      }

      target.hidden = !target.hidden;
      button.classList.toggle('is-open', !target.hidden);
    });
  });
})();
