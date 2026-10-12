(function () {
  document.querySelectorAll('[data-picture-upload]').forEach(function (form) {
    const trigger = form.querySelector('[data-picture-trigger]');
    const input = form.querySelector('[data-picture-input]');
    const problem = form.querySelector('[data-picture-error]');
    const label = form.querySelector('[data-picture-label]');
    const maximum = Number(form.dataset.pictureLimit);
    const idleLabel = label.textContent;

    window.addEventListener('pageshow', function () {
      trigger.disabled = false;
      trigger.removeAttribute('aria-busy');
      label.textContent = idleLabel;
    });

    trigger.addEventListener('click', function () {
      input.click();
    });

    input.addEventListener('change', function () {
      const file = input.files[0];
      if (!file) {
        return;
      }
      if (file.size > maximum) {
        problem.textContent = form.dataset.pictureTooLarge;
        problem.hidden = false;
        input.value = '';
        return;
      }
      problem.hidden = true;
      trigger.disabled = true;
      trigger.setAttribute('aria-busy', 'true');
      label.textContent = form.dataset.pictureUploading;
      form.requestSubmit();
    });
  });
})();
