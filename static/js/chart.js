(function () {
  const holders = document.querySelectorAll('[data-chart]');

  if (holders.length === 0) {
    return;
  }

  const SVG = 'http://www.w3.org/2000/svg';
  const PLOT_HEIGHT = 190;
  const AXIS_WIDTH = 62;
  const AXIS_HEIGHT = 30;
  const TOP_ROOM = 12;
  const TICK_LENGTH = 5;
  const TICK_GAP = 6;

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
    const text = value.toFixed(value >= 10 || Number.isInteger(value) ? 0 : 1);

    return text.indexOf('.') < 0 ? text : text.replace(/\.?0+$/, '');
  }

  function duration(value) {
    if (value >= 1000) {
      return trim(value / 1000) + ' s';
    }

    return trim(value) + ' ms';
  }

  function label(value, unit) {
    return unit === 'ms' ? duration(value) : readable(value);
  }

  function bare(value, unit) {
    if (unit !== 'ms') {
      return readable(value);
    }

    return value >= 1000 ? trim(value / 1000) : trim(value);
  }

  function unitOf(chart) {
    if (chart.unit === 'ms') {
      return 'ms';
    }

    return chart.axis || '';
  }

  function niceStep(rough) {
    const power = Math.pow(10, Math.floor(Math.log10(rough)));
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
    const step = niceStep(tallest / 4);
    const top = Math.ceil(tallest / step) * step;
    const marks = [];

    for (let value = 0; value <= top + step / 2; value += step) {
      marks.push(value);
    }

    return { top: top || 1, marks: marks };
  }

  holders.forEach(function (holder) {
    const source = holder.querySelector('script[type="application/json"]');

    if (!source) {
      return;
    }

    let chart;

    try {
      chart = JSON.parse(source.textContent);
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

    function summarise(band) {
      if (stacked) {
        return chart.points.reduce(function (total, point) {
          return total + point.values[band];
        }, 0);
      }

      return chart.points.reduce(function (highest, point) {
        return Math.max(highest, point.values[band]);
      }, 0);
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
        value.textContent = label(summarise(band), chart.unit);

        item.appendChild(swatch);
        item.appendChild(title);
        item.appendChild(value);
        key.appendChild(item);

        readings.push({ item: item, value: value, resting: value.textContent });
      });

      const note = document.createElement('span');
      note.className = 'chart-key-note';
      note.textContent = stacked ? 'total' : 'peak';
      key.appendChild(note);
    }

    function rest() {
      readings.forEach(function (reading) {
        reading.value.textContent = reading.resting;
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
      const width = Math.max(canvas.clientWidth, 360);
      const height = PLOT_HEIGHT + AXIS_HEIGHT + TOP_ROOM;
      const plotWidth = width - AXIS_WIDTH;

      let tallest = 0;

      chart.points.forEach(function (point) {
        const value = stacked
          ? point.values.reduce(function (a, b) { return a + b; }, 0)
          : Math.max.apply(null, point.values);

        tallest = Math.max(tallest, value);
      });

      const axis = ticksFor(tallest || 1);

      const svg = make('svg', {
        class: 'chart-canvas',
        viewBox: '0 0 ' + width + ' ' + height,
        width: width,
        height: height,
      });

      const scale = function (value) {
        return TOP_ROOM + PLOT_HEIGHT - (value / axis.top) * PLOT_HEIGHT;
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

          point.values.forEach(function (value, band) {
            if (value <= 0) {
              return;
            }

            const top = bottom + value;

            svg.appendChild(make('rect', {
              class: 'chart-band is-' + band,
              x: AXIS_WIDTH + index * slot + slot * 0.18,
              y: scale(top),
              width: Math.max(slot * 0.64, 1),
              height: Math.max(scale(bottom) - scale(top), 1),
            }));

            bottom = top;
          });
        });
      } else {
        series.forEach(function (one, band) {
          let path = '';

          chart.points.forEach(function (point, index) {
            path += (index === 0 ? 'M' : 'L') + centreOf(index) + ' ' + scale(point.values[band]) + ' ';
          });

          svg.appendChild(make('path', { class: 'chart-line is-' + band, d: path }));
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

      svg.addEventListener('mousemove', function (event) {
        const box = svg.getBoundingClientRect();
        const inside = (event.clientX - box.left) * (width / box.width) - AXIS_WIDTH;
        const index = Math.min(Math.max(Math.floor(inside / slot), 0), chart.points.length - 1);
        const point = chart.points[index];
        const x = centreOf(index);

        crosshair.setAttribute('x1', x);
        crosshair.setAttribute('x2', x);
        crosshair.style.opacity = '1';

        if (!stacked) {
          dots.forEach(function (dot, band) {
            dot.setAttribute('cx', x);
            dot.setAttribute('cy', scale(point.values[band]));
            dot.style.opacity = '1';
          });
        }

        show(point);

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

        tip.hidden = false;
        tip.style.left = Math.min(Math.max((x / width) * box.width - 70, 0), box.width - 170) + 'px';
      });

      svg.addEventListener('mouseleave', function () {
        crosshair.style.opacity = '0';
        dots.forEach(function (dot) {
          dot.style.opacity = '0';
        });
        tip.hidden = true;
        rest();
      });

      canvas.innerHTML = '';
      canvas.appendChild(svg);
    }

    draw();

    let waiting = null;

    window.addEventListener('resize', function () {
      clearTimeout(waiting);
      waiting = setTimeout(draw, 120);
    });
  });
})();
