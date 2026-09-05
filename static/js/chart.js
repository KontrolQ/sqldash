(function () {
  const holders = document.querySelectorAll('[data-chart]');

  if (holders.length === 0) {
    return;
  }

  const SVG = 'http://www.w3.org/2000/svg';
  const PLOT_HEIGHT = 180;
  const AXIS_WIDTH = 52;
  const AXIS_HEIGHT = 22;

  function make(name, attributes) {
    const node = document.createElementNS(SVG, name);

    Object.keys(attributes || {}).forEach(function (key) {
      node.setAttribute(key, attributes[key]);
    });

    return node;
  }

  function readable(value) {
    if (value >= 1000000) {
      return (value / 1000000).toFixed(1) + 'M';
    }

    if (value >= 1000) {
      return (value / 1000).toFixed(1) + 'k';
    }

    return String(Math.round(value * 100) / 100);
  }

  function duration(value) {
    return value >= 1000 ? (value / 1000).toFixed(2) + ' s' : value.toFixed(2) + ' ms';
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
    const stacked = chart.kind === 'stacked';
    const series = chart.series || [];

    function draw() {
      const width = Math.max(canvas.clientWidth, 320);
      const height = PLOT_HEIGHT + AXIS_HEIGHT;
      const plotWidth = width - AXIS_WIDTH;

      let tallest = 0;

      chart.points.forEach(function (point) {
        series.forEach(function (one, index) {
          const value = stacked && index > 0 ? point.values[0] : point.values[index];
          tallest = Math.max(tallest, stacked ? point.values.reduce(function (a, b) { return a + b; }, 0) : value);
        });
      });

      if (tallest <= 0) {
        tallest = 1;
      }

      const svg = make('svg', {
        class: 'chart-canvas',
        viewBox: '0 0 ' + width + ' ' + height,
        width: width,
        height: height,
      });

      const scale = function (value) {
        return PLOT_HEIGHT - (value / tallest) * PLOT_HEIGHT;
      };

      for (let step = 0; step <= 4; step += 1) {
        const value = (tallest / 4) * step;
        const y = scale(value);

        svg.appendChild(make('line', {
          class: 'chart-rule', x1: AXIS_WIDTH, y1: y, x2: width, y2: y,
        }));

        const text = make('text', { class: 'chart-tick', x: AXIS_WIDTH - 8, y: y + 4, 'text-anchor': 'end' });
        text.textContent = chart.unit === 'ms' ? duration(value) : readable(value);
        svg.appendChild(text);
      }

      const slot = plotWidth / chart.points.length;

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
              x: AXIS_WIDTH + index * slot + slot * 0.15,
              y: scale(top),
              width: Math.max(slot * 0.7, 1),
              height: Math.max(scale(bottom) - scale(top), 0),
            }));

            bottom = top;
          });
        });
      } else {
        series.forEach(function (one, band) {
          let path = '';

          chart.points.forEach(function (point, index) {
            const x = AXIS_WIDTH + index * slot + slot / 2;
            path += (index === 0 ? 'M' : 'L') + x + ' ' + scale(point.values[band]) + ' ';
          });

          svg.appendChild(make('path', { class: 'chart-line is-' + band, d: path }));
        });
      }

      const marks = [0, Math.floor(chart.points.length / 2), chart.points.length - 1];

      marks.forEach(function (index) {
        const text = make('text', {
          class: 'chart-tick',
          x: AXIS_WIDTH + index * slot + slot / 2,
          y: PLOT_HEIGHT + 16,
          'text-anchor': index === 0 ? 'start' : index === chart.points.length - 1 ? 'end' : 'middle',
        });

        text.textContent = chart.points[index].label;
        svg.appendChild(text);
      });

      const crosshair = make('line', {
        class: 'chart-crosshair', x1: 0, y1: 0, x2: 0, y2: PLOT_HEIGHT,
      });
      crosshair.style.opacity = '0';
      svg.appendChild(crosshair);

      svg.addEventListener('mousemove', function (event) {
        const box = svg.getBoundingClientRect();
        const inside = (event.clientX - box.left) * (width / box.width) - AXIS_WIDTH;
        const index = Math.min(Math.max(Math.floor(inside / slot), 0), chart.points.length - 1);
        const point = chart.points[index];
        const x = AXIS_WIDTH + index * slot + slot / 2;

        crosshair.setAttribute('x1', x);
        crosshair.setAttribute('x2', x);
        crosshair.style.opacity = '1';

        let rows = '';

        series.forEach(function (one, band) {
          const value = point.values[band];
          rows += '<span class="chart-tip-row"><span class="chart-swatch is-' + band + '"></span>' +
            one + '<b>' + (chart.unit === 'ms' ? duration(value) : readable(value)) + '</b></span>';
        });

        tip.innerHTML = '<span class="chart-tip-when">' + point.label + '</span>' + rows;
        tip.hidden = false;
        tip.style.left = Math.min(Math.max((x / width) * box.width - 60, 0), box.width - 140) + 'px';
      });

      svg.addEventListener('mouseleave', function () {
        crosshair.style.opacity = '0';
        tip.hidden = true;
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
