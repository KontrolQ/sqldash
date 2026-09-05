(function () {
  const search = document.querySelector('[data-rail-search]');
  const toggle = document.querySelector('[data-rail-toggle]');
  const workspace = document.querySelector('.workspace');

  if (search) {
    const rows = [...document.querySelectorAll('[data-rail-row]')];
    const groups = [...document.querySelectorAll('.rail-group')];

    search.addEventListener('input', function () {
      const wanted = search.value.trim().toLowerCase();

      rows.forEach(function (row) {
        row.hidden = wanted !== '' && row.dataset.railRow.toLowerCase().indexOf(wanted) < 0;
      });

      groups.forEach(function (group) {
        let following = group.nextElementSibling;
        let any = false;

        while (following && following.dataset.railRow) {
          if (!following.hidden) {
            any = true;
          }

          following = following.nextElementSibling;
        }

        group.hidden = !any;
      });
    });
  }

  if (!toggle || !workspace) {
    return;
  }

  const remembered = 'sqldash-rail';

  function apply(shut) {
    workspace.classList.toggle('is-shut', shut);
    toggle.setAttribute('aria-expanded', shut ? 'false' : 'true');
    toggle.setAttribute('aria-label', shut ? 'Show the table list' : 'Hide the table list');
    toggle.title = shut ? 'Show the table list' : 'Hide the table list';
  }

  let shut = false;

  try {
    shut = window.localStorage.getItem(remembered) === 'shut';
  } catch (blocked) {
    shut = false;
  }

  apply(shut);

  toggle.addEventListener('click', function () {
    shut = !shut;
    apply(shut);

    try {
      window.localStorage.setItem(remembered, shut ? 'shut' : 'open');
    } catch (blocked) {
      return;
    }
  });
})();
