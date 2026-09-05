(function () {
  document.querySelectorAll('[data-swap]').forEach(function (group) {
    const tabs = [...group.querySelectorAll('[data-swap-to]')];
    const root = group.parentElement;

    tabs.forEach(function (tab) {
      tab.addEventListener('click', function () {
        tabs.forEach(function (one) {
          one.classList.toggle('is-current', one === tab);
        });

        root.querySelectorAll('[data-swap-panel]').forEach(function (panel) {
          panel.hidden = panel.dataset.swapPanel !== tab.dataset.swapTo;
        });
      });
    });
  });
})();
