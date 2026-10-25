(function () {
  const holders = document.querySelectorAll('[data-chart]');

  const cleanups = new WeakMap();

  const SVG = 'http://www.w3.org/2000/svg';
  const PLOT_HEIGHT = 190;
  const AXIS_WIDTH = 62;
  const AXIS_HEIGHT = 30;
  const TOP_ROOM = 12;
  const TICK_LENGTH = 5;
  const TICK_GAP = 6;
  const RESTING = '—';
  const UNAVAILABLE = 'Unavailable';

  function isMeasurement(value) {
    return typeof value === 'number' && Number.isFinite(value);
  }

  function make(name, attributes) {
    const node = document.createElementNS(SVG, name);

    Object.keys(attributes || {}).forEach(function (key) {
      node.setAttribute(key, attributes[key]);
    });

    return node;
  }

  function readable(value) {
    if (value >= 1000000000) {
      return trim(value / 1000000000) + 'B';
    }

    if (value >= 1000000) {
      return trim(value / 1000000) + 'M';
    }

    if (value >= 1000) {
      return trim(value / 1000) + 'k';
    }

    return trim(value);
  }

  function trim(value) {
    if (!isMeasurement(value)) {
      return UNAVAILABLE;
    }

    const text = value.toFixed(value >= 10 || Number.isInteger(value) ? 0 : 1);

    return text.indexOf('.') < 0 ? text : text.replace(/\.?0+$/, '');
  }

  function duration(value) {
    if (value >= 1000) {
      return trim(value / 1000) + ' s';
    }

    return trim(value) + ' ms';
  }

  function humanBytes(value) {
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB'];
    let amount = value;
    let unitIndex = 0;

    while (Math.abs(amount) >= 1024 && unitIndex < units.length - 1) {
      amount /= 1024;
      unitIndex += 1;
    }

    return trim(amount) + ' ' + units[unitIndex];
  }

  function label(value, unit) {
    if (!isMeasurement(value)) {
      return UNAVAILABLE;
    }

    if (unit === 'bytes') {
      return humanBytes(value);
    }

    return unit === 'ms' ? duration(value) : readable(value);
  }

  function bare(value, unit) {
    if (!isMeasurement(value)) {
      return UNAVAILABLE;
    }

    if (unit === 'bytes') {
      return humanBytes(value);
    }

    if (unit !== 'ms') {
      return readable(value);
    }

    return value >= 1000 ? trim(value / 1000) : trim(value);
  }

  function unitOf(chart) {
    if (chart.unit === 'ms') {
      return 'ms';
    }

    return chart.axis || (chart.unit === 'bytes' ? 'bytes' : '');
  }

  function niceStep(rough) {
    if (!isMeasurement(rough) || rough <= 0) {
      return 1;
    }

    const power = Math.pow(10, Math.floor(Math.log10(rough)));

    if (power === 0) {
      return rough;
    }

    const share = rough / power;

    if (share <= 1) {
      return power;
    }

    if (share <= 2) {
      return 2 * power;
    }

    if (share <= 2.5) {
      return 2.5 * power;
    }

    if (share <= 5) {
      return 5 * power;
    }

    return 10 * power;
  }

  function ticksFor(tallest) {
    const maximum = isMeasurement(tallest) && tallest > 0 ? tallest : 1;
    const step = niceStep(maximum / 4);
    const roundedTop = Math.ceil(maximum / step) * step;
    const top = isMeasurement(roundedTop) && roundedTop > 0 ? roundedTop : maximum;
    const tickCount = Math.floor(top / step + 0.5);
    const marks = [];

    for (let tickIndex = 0; tickIndex <= tickCount; tickIndex += 1) {
      const value = tickIndex * step;

      if (isMeasurement(value)) {
        marks.push(value);
      }
    }

    return { top: top, marks: marks };
  }

  function appendMeasurementCell(row, name, text, scope) {
    const cell = document.createElement(name);
    cell.textContent = text;

    if (scope) {
      cell.scope = scope;
    }

    row.appendChild(cell);
  }

  function measurementsTable(chart, series) {
    const details = document.createElement('details');
    details.className = 'chart-measurements';

    const summary = document.createElement('summary');
    summary.textContent = 'Measurements';
    details.appendChild(summary);

    const table = document.createElement('table');
    table.className = 'table';
    table.setAttribute('aria-label', 'Measurements');

    const heading = table.createTHead().insertRow();
    const measurementUnit = chart.unit || chart.axis || '';
    const hasNotes = chart.points.some(function (point) { return Boolean(point.note); });
    appendMeasurementCell(heading, 'th', 'Label', 'col');

    series.forEach(function (name) {
      appendMeasurementCell(heading, 'th', name + (measurementUnit ? ' (' + measurementUnit + ')' : ''), 'col');
    });

    if (hasNotes) {
      appendMeasurementCell(heading, 'th', 'Coverage', 'col');
    }

    const body = table.createTBody();

    chart.points.forEach(function (point) {
      const row = body.insertRow();
      appendMeasurementCell(row, 'th', point.label, 'row');

      series.forEach(function (name, band) {
        const value = point.values[band];
        appendMeasurementCell(row, 'td', isMeasurement(value) ? String(value) : UNAVAILABLE);
      });

      if (hasNotes) {
        appendMeasurementCell(row, 'td', point.note || (point.values.some(isMeasurement) ? 'Recorded statement coverage' : UNAVAILABLE));
      }
    });

    const wrapper = document.createElement('div');
    wrapper.className = 'table-wrap';
    wrapper.tabIndex = 0;
    wrapper.setAttribute('role', 'region');
    wrapper.setAttribute('aria-label', series.join(', ') + ' measurements');
    wrapper.appendChild(table);
    details.appendChild(wrapper);

    return details;
  }

  function initialize(holder, supplied) {
    cleanups.get(holder)?.();
    holder.querySelector('.chart-measurements')?.remove();
    const source = holder.querySelector('script[type="application/json"]');

    if (!source && !supplied) {
      return;
    }

    let chart;

    try {
      chart = supplied || JSON.parse(source.textContent);
    } catch (parseError) {
      return;
    }

    if (!chart.points || chart.points.length === 0) {
      return;
    }

    const canvas = holder.querySelector('[data-chart-canvas]');
    const tip = holder.querySelector('[data-chart-tip]');
    const key = holder.querySelector('[data-chart-key]');
    const stacked = chart.kind === 'stacked';
    const series = chart.series || [];
    const readings = [];
    const interactive = chart.table === true;
    let selectedIndex = null;

    if (!canvas) {
      return;
    }

    if (interactive) {
      holder.appendChild(measurementsTable(chart, series));

      if (tip) {
        tip.setAttribute('role', 'status');
        tip.setAttribute('aria-live', 'polite');
        tip.setAttribute('aria-atomic', 'true');
      }
    }

    if (key) {
      key.innerHTML = '';
      readings.length = 0;

      series.forEach(function (name, band) {
        const item = document.createElement('span');
        item.className = 'chart-key-item is-' + band;

        const swatch = document.createElement('span');
        swatch.className = 'chart-swatch is-' + band;

        const title = document.createElement('span');
        title.className = 'chart-key-name';
        title.textContent = name;

        const value = document.createElement('b');
        value.textContent = RESTING;

        item.appendChild(swatch);
        item.appendChild(title);
        item.appendChild(value);
        key.appendChild(item);

        readings.push({ item: item, value: value });
      });
    }

    function rest() {
      readings.forEach(function (reading) {
        reading.value.textContent = RESTING;
        reading.item.classList.remove('is-live');
      });
    }

    function show(point) {
      readings.forEach(function (reading, band) {
        reading.value.textContent = label(point.values[band], chart.unit);
        reading.item.classList.add('is-live');
      });
    }

    function draw() {
      const restoreFocus = interactive && canvas.contains(document.activeElement);
      const width = Math.max(canvas.clientWidth, 360);
      const height = PLOT_HEIGHT + AXIS_HEIGHT + TOP_ROOM;
      const plotWidth = width - AXIS_WIDTH;

      let tallest = 0;
      let hasMeasurements = false;

      chart.points.forEach(function (point) {
        let stackedHeight = 0;

        series.forEach(function (name, band) {
          const value = point.values[band];

          if (!isMeasurement(value)) {
            return;
          }

          hasMeasurements = true;

          if (stacked) {
            stackedHeight += value;
          } else {
            tallest = Math.max(tallest, value);
          }
        });

        if (stacked && isMeasurement(stackedHeight)) {
          tallest = Math.max(tallest, stackedHeight);
        }
      });

      canvas.innerHTML = '';
      rest();

      if (tip) {
        tip.hidden = true;
      }

      if (!hasMeasurements) {
        const empty = document.createElement('div');
        empty.className = 'chart-empty';
        empty.textContent = 'No measurements available.';
        canvas.appendChild(empty);

        return;
      }

      const axis = ticksFor(tallest);

      const svg = make('svg', {
        class: 'chart-canvas',
        viewBox: '0 0 ' + width + ' ' + height,
        width: width,
        height: height,
      });

      const scale = function (value) {
        if (!isMeasurement(value)) {
          return null;
        }

        const position = TOP_ROOM + PLOT_HEIGHT - (value / axis.top) * PLOT_HEIGHT;

        return isMeasurement(position) ? position : null;
      };

      axis.marks.forEach(function (value) {
        const y = scale(value);

        svg.appendChild(make('line', {
          class: 'chart-mark',
          x1: AXIS_WIDTH - TICK_LENGTH - 1,
          y1: y,
          x2: AXIS_WIDTH - 1,
          y2: y,
        }));

        const text = make('text', {
          class: 'chart-tick',
          x: AXIS_WIDTH - TICK_LENGTH - TICK_GAP,
          y: y + 4,
          'text-anchor': 'end',
        });

        text.textContent = bare(value, chart.unit);
        svg.appendChild(text);
      });

      const unit = unitOf(chart);

      if (unit) {
        const side = make('text', {
          class: 'chart-unit',
          x: 12,
          y: TOP_ROOM + PLOT_HEIGHT / 2,
          transform: 'rotate(-90 12 ' + (TOP_ROOM + PLOT_HEIGHT / 2) + ')',
          'text-anchor': 'middle',
        });

        side.textContent = unit;
        svg.appendChild(side);
      }

      svg.appendChild(make('line', {
        class: 'chart-base',
        x1: AXIS_WIDTH - 1,
        y1: scale(0),
        x2: width,
        y2: scale(0),
      }));

      const slot = plotWidth / chart.points.length;
      const centreOf = function (index) {
        return AXIS_WIDTH + index * slot + slot / 2;
      };

      if (stacked) {
        chart.points.forEach(function (point, index) {
          let bottom = 0;

          series.forEach(function (name, band) {
            const value = point.values[band];

            if (!isMeasurement(value) || value <= 0) {
              return;
            }

            const top = bottom + value;

            if (!isMeasurement(top)) {
              return;
            }

            const topPosition = scale(top);
            const bottomPosition = scale(bottom);

            if (!isMeasurement(topPosition) || !isMeasurement(bottomPosition)) {
              return;
            }

            svg.appendChild(make('rect', {
              class: 'chart-band is-' + band,
              x: AXIS_WIDTH + index * slot + slot * 0.18,
              y: topPosition,
              width: Math.max(slot * 0.64, 1),
              height: Math.max(bottomPosition - topPosition, 1),
            }));

            bottom = top;
          });
        });
      } else {
        series.forEach(function (one, band) {
          let path = '';
          let connected = false;

          chart.points.forEach(function (point, index) {
            const value = point.values[band];

            if (!isMeasurement(value)) {
              connected = false;

              return;
            }

            const position = scale(value);

            if (!isMeasurement(position)) {
              connected = false;

              return;
            }

            if (point.break_before === true) {
              connected = false;
            }

            const nextPoint = chart.points[index + 1];
            const connectsAfter = nextPoint && nextPoint.break_before !== true
              && isMeasurement(nextPoint.values[band]) && isMeasurement(scale(nextPoint.values[band]));

            if (!connected && !connectsAfter) {
              svg.appendChild(make('circle', {
                class: 'chart-dot is-' + band,
                r: 3.5,
                cx: centreOf(index),
                cy: position,
              }));
            }

            path += (connected ? 'L' : 'M') + centreOf(index) + ' ' + position + ' ';
            connected = true;
          });

          if (path) {
            svg.appendChild(make('path', { class: 'chart-line is-' + band, d: path }));
          }
        });
      }

      const wanted = Math.max(2, Math.min(6, Math.floor(plotWidth / 110)));
      const last = chart.points.length - 1;
      const stops = [];

      for (let step = 0; step < wanted; step += 1) {
        const index = Math.round((step * last) / (wanted - 1));

        if (stops.indexOf(index) < 0) {
          stops.push(index);
        }
      }

      stops.forEach(function (index) {
        const x = centreOf(index);

        svg.appendChild(make('line', {
          class: 'chart-mark',
          x1: x,
          y1: scale(0),
          x2: x,
          y2: scale(0) + TICK_LENGTH,
        }));

        const first = index === stops[0];
        const final = index === stops[stops.length - 1];

        const text = make('text', {
          class: 'chart-tick',
          x: final ? width : (first ? AXIS_WIDTH : x),
          y: scale(0) + TICK_LENGTH + 14,
          'text-anchor': final ? 'end' : (first ? 'start' : 'middle'),
        });

        text.textContent = chart.points[index].label;
        svg.appendChild(text);
      });

      const crosshair = make('line', {
        class: 'chart-crosshair', x1: 0, y1: TOP_ROOM, x2: 0, y2: TOP_ROOM + PLOT_HEIGHT,
      });
      crosshair.style.opacity = '0';
      svg.appendChild(crosshair);

      const dots = series.map(function (one, band) {
        const dot = make('circle', { class: 'chart-dot is-' + band, r: 3.5, cx: 0, cy: 0 });
        dot.style.opacity = '0';
        svg.appendChild(dot);

        return dot;
      });

      function selectPoint(index) {
        const point = chart.points[index];
        const x = centreOf(index);
        selectedIndex = index;

        crosshair.setAttribute('x1', x);
        crosshair.setAttribute('x2', x);
        crosshair.style.opacity = '1';

        if (!stacked) {
          dots.forEach(function (dot, band) {
            const value = point.values[band];
            const position = isMeasurement(value) ? scale(value) : null;

            if (!isMeasurement(position)) {
              dot.style.opacity = '0';

              return;
            }

            dot.setAttribute('cx', x);
            dot.setAttribute('cy', position);
            dot.style.opacity = '1';
          });
        }

        show(point);

        if (interactive) {
          const description = series.map(function (name, band) {
            return name + ': ' + label(point.values[band], chart.unit);
          }).join('. ');
          svg.setAttribute('aria-label', point.label + '. ' + description + '. ' + (point.note ? point.note + ' ' : '') + 'Use Left and Right Arrow keys to read measurements.');
        }

        if (!tip) {
          return;
        }

        const when = document.createElement('span');
        when.className = 'chart-tip-when';
        when.textContent = point.label;

        tip.innerHTML = '';
        tip.appendChild(when);

        series.forEach(function (one, band) {
          const row = document.createElement('span');
          row.className = 'chart-tip-row';

          const swatch = document.createElement('span');
          swatch.className = 'chart-swatch is-' + band;

          const name = document.createElement('span');
          name.textContent = one;

          const value = document.createElement('b');
          value.textContent = label(point.values[band], chart.unit);

          row.appendChild(swatch);
          row.appendChild(name);
          row.appendChild(value);
          tip.appendChild(row);
        });

        if (point.note) {
          const note = document.createElement('span');
          note.textContent = point.note;
          tip.appendChild(note);
        }

        const box = svg.getBoundingClientRect();
        tip.hidden = false;
        tip.style.left = Math.min(Math.max((x / width) * box.width - 70, 0), Math.max(box.width - 170, 0)) + 'px';
      }

      function clearSelection() {
        crosshair.style.opacity = '0';
        dots.forEach(function (dot) {
          dot.style.opacity = '0';
        });

        if (tip) {
          tip.hidden = true;
        }

        rest();
      }

      let hovering = false;

      svg.addEventListener('mousemove', function (event) {
        const box = svg.getBoundingClientRect();

        if (box.width <= 0) {
          return;
        }

        const inside = (event.clientX - box.left) * (width / box.width) - AXIS_WIDTH;
        const index = Math.min(Math.max(Math.floor(inside / slot), 0), chart.points.length - 1);
        hovering = true;
        selectPoint(index);
      });

      svg.addEventListener('mouseleave', function () {
        hovering = false;

        if (!interactive || document.activeElement !== svg) {
          clearSelection();
        }
      });

      if (interactive) {
        svg.setAttribute('tabindex', '0');
        svg.setAttribute('role', 'group');
        svg.setAttribute('aria-label', 'Measurements chart. Use Left and Right Arrow keys to read measurements.');

        svg.addEventListener('focus', function () {
          selectPoint(selectedIndex === null ? 0 : selectedIndex);
        });

        svg.addEventListener('blur', function () {
          if (!hovering) {
            clearSelection();
          }
        });

        svg.addEventListener('keydown', function (event) {
          let index = selectedIndex === null ? 0 : selectedIndex;

          if (event.key === 'ArrowLeft') {
            index = Math.max(index - 1, 0);
          } else if (event.key === 'ArrowRight') {
            index = Math.min(index + 1, chart.points.length - 1);
          } else if (event.key === 'Home') {
            index = 0;
          } else if (event.key === 'End') {
            index = chart.points.length - 1;
          } else {
            return;
          }

          event.preventDefault();
          selectPoint(index);
        });
      }

      canvas.appendChild(svg);

      if (restoreFocus) {
        svg.focus();
      }
    }

    draw();

    let waiting = null;

    function resize() {
      clearTimeout(waiting);
      waiting = setTimeout(draw, 120);
    }
    window.addEventListener('resize', resize);
    cleanups.set(holder, function () { clearTimeout(waiting); window.removeEventListener('resize', resize); });
  }
  window.SqlDashCharts = {render: initialize, format: label};
  holders.forEach(function (holder) { initialize(holder); });
})();
