(function () {
  const fields = document.querySelectorAll('[data-moment]');

  if (fields.length === 0) {
    return;
  }

  const weekdays = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];
  const months = ['January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'];

  function twoDigits(value) {
    return String(value).padStart(2, '0');
  }

  fields.forEach(function (field) {
    const hidden = field.querySelector('input[type="hidden"]');
    const trigger = field.querySelector('[data-moment-trigger]');
    const label = field.querySelector('[data-moment-label]');
    const content = field.querySelector('[data-moment-content]');
    const title = field.querySelector('[data-moment-title]');
    const grid = field.querySelector('[data-moment-grid]');
    const time = field.querySelector('[data-moment-time]');
    const clear = field.querySelector('[data-moment-clear]');
    const back = field.querySelector('[data-moment-back]');
    const forward = field.querySelector('[data-moment-forward]');

    if (!hidden || !trigger || !content || !grid) {
      return;
    }

    let chosen = null;
    let shown = new Date();

    function endOfToday() {
      const now = new Date();

      return new Date(now.getFullYear(), now.getMonth(), now.getDate());
    }

    function clockLimit() {
      if (!chosen || chosen.toDateString() !== new Date().toDateString()) {
        return '';
      }

      const now = new Date();

      return twoDigits(now.getHours()) + ':' + twoDigits(now.getMinutes());
    }

    function write() {
      if (!chosen) {
        hidden.value = '';
        label.textContent = trigger.dataset.placeholder || 'Pick a moment';
        label.classList.add('is-empty');
        return;
      }

      const limit = clockLimit();

      if (limit !== '') {
        time.max = limit;

        if (time.value > limit) {
          time.value = limit;
        }
      } else {
        time.removeAttribute('max');
      }

      const clock = (time.value || '00:00').split(':');
      const stamp = chosen.getFullYear() + '-' + twoDigits(chosen.getMonth() + 1) + '-' +
        twoDigits(chosen.getDate()) + 'T' + twoDigits(clock[0]) + ':' + twoDigits(clock[1]);

      hidden.value = stamp;
      label.textContent = months[chosen.getMonth()] + ' ' + chosen.getDate() + ', ' +
        chosen.getFullYear() + ' at ' + twoDigits(clock[0]) + ':' + twoDigits(clock[1]);
      label.classList.remove('is-empty');
    }

    function draw() {
      const now = new Date();
      forward.disabled = shown.getFullYear() === now.getFullYear() && shown.getMonth() === now.getMonth();

      title.textContent = months[shown.getMonth()] + ' ' + shown.getFullYear();
      grid.innerHTML = '';

      weekdays.forEach(function (name) {
        const cell = document.createElement('div');
        cell.className = 'calendar-weekday';
        cell.textContent = name;
        grid.appendChild(cell);
      });

      const first = new Date(shown.getFullYear(), shown.getMonth(), 1);
      const start = new Date(first);
      start.setDate(1 - first.getDay());

      const today = new Date();
      const latest = endOfToday();

      for (let index = 0; index < 42; index += 1) {
        const day = new Date(start);
        day.setDate(start.getDate() + index);

        const cell = document.createElement('button');
        cell.type = 'button';
        cell.className = 'calendar-day';
        cell.textContent = day.getDate();

        if (day.getMonth() !== shown.getMonth()) {
          cell.classList.add('is-outside');
        }

        if (day.toDateString() === today.toDateString()) {
          cell.classList.add('is-today');
        }

        if (chosen && day.toDateString() === chosen.toDateString()) {
          cell.classList.add('is-chosen');
        }

        if (day > latest) {
          cell.disabled = true;
          cell.classList.add('is-later');
        }

        cell.addEventListener('click', function () {
          chosen = day;

          if (day.toDateString() === today.toDateString() && time.value === '00:00') {
            time.value = twoDigits(today.getHours()) + ':' + twoDigits(today.getMinutes());
          }

          write();
          draw();
        });

        grid.appendChild(cell);
      }
    }

    trigger.addEventListener('click', function (event) {
      event.preventDefault();
      content.hidden = !content.hidden;

      if (!content.hidden) {
        draw();
      }
    });

    document.addEventListener('click', function (event) {
      if (!field.contains(event.target)) {
        content.hidden = true;
      }
    });

    back.addEventListener('click', function () {
      shown = new Date(shown.getFullYear(), shown.getMonth() - 1, 1);
      draw();
    });

    forward.addEventListener('click', function () {
      const next = new Date(shown.getFullYear(), shown.getMonth() + 1, 1);

      if (next > new Date()) {
        return;
      }

      shown = next;
      draw();
    });

    time.addEventListener('change', write);

    clear.addEventListener('click', function () {
      chosen = null;
      time.value = '00:00';
      write();
      draw();
    });

    write();
  });
})();
