(function () {
  const picker = document.querySelector('[data-colour-picker]');
  if (!picker) { return; }
  const colour = picker.querySelector('[data-role-colour]');
  const plane = picker.querySelector('[data-colour-plane]');
  const cursor = picker.querySelector('[data-colour-cursor]');
  const hue = picker.querySelector('[data-colour-hue]');
  const hex = picker.querySelector('[data-colour-hex]');
  const preview = picker.querySelector('[data-colour-preview]');
  let currentHue = 0;
  let saturation = 0;
  let brightness = 1;

  function readColour(value) {
    const channels = [1, 3, 5].map(function (offset) { return parseInt(value.slice(offset, offset + 2), 16) / 255; });
    const highest = Math.max(...channels);
    const lowest = Math.min(...channels);
    const difference = highest - lowest;
    if (difference) {
      if (highest === channels[0]) { currentHue = 60 * (((channels[1] - channels[2]) / difference) % 6); }
      else if (highest === channels[1]) { currentHue = 60 * ((channels[2] - channels[0]) / difference + 2); }
      else { currentHue = 60 * ((channels[0] - channels[1]) / difference + 4); }
      currentHue = (currentHue + 360) % 360;
    }
    saturation = highest === 0 ? 0 : difference / highest;
    brightness = highest;
  }

  function encodedColour() {
    const chroma = brightness * saturation;
    const intermediate = chroma * (1 - Math.abs((currentHue / 60) % 2 - 1));
    const base = brightness - chroma;
    const segments = [[chroma, intermediate, 0], [intermediate, chroma, 0], [0, chroma, intermediate], [0, intermediate, chroma], [intermediate, 0, chroma], [chroma, 0, intermediate]];
    const channels = segments[Math.floor(currentHue / 60) % 6];
    return '#' + channels.map(function (channel) { return Math.round((channel + base) * 255).toString(16).padStart(2, '0'); }).join('');
  }

  function paint() {
    plane.style.setProperty('--picker-hue', 'hsl(' + currentHue + ' 100% 50%)');
    cursor.style.left = saturation * 100 + '%';
    cursor.style.top = (1 - brightness) * 100 + '%';
    plane.setAttribute('aria-valuetext', 'Saturation ' + Math.round(saturation * 100) + '%, brightness ' + Math.round(brightness * 100) + '%');
    hue.value = Math.round(currentHue);
    hex.value = colour.value;
    preview.style.background = colour.value;
  }

  function change() {
    colour.value = encodedColour();
    paint();
    colour.dispatchEvent(new Event('input', { bubbles: true }));
    colour.dispatchEvent(new Event('change', { bubbles: true }));
  }

  function locate(event) {
    const bounds = plane.getBoundingClientRect();
    saturation = Math.min(1, Math.max(0, (event.clientX - bounds.left) / bounds.width));
    brightness = 1 - Math.min(1, Math.max(0, (event.clientY - bounds.top) / bounds.height));
    change();
  }

  plane.addEventListener('pointerdown', function (event) {
    event.preventDefault();
    plane.setPointerCapture(event.pointerId);
    locate(event);
  });
  plane.addEventListener('pointermove', function (event) { if (plane.hasPointerCapture(event.pointerId)) { locate(event); } });
  plane.addEventListener('keydown', function (event) {
    if (!['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(event.key)) { return; }
    event.preventDefault();
    const amount = event.shiftKey ? 0.1 : 0.01;
    if (event.key === 'ArrowLeft') { saturation = Math.max(0, saturation - amount); }
    if (event.key === 'ArrowRight') { saturation = Math.min(1, saturation + amount); }
    if (event.key === 'ArrowUp') { brightness = Math.min(1, brightness + amount); }
    if (event.key === 'ArrowDown') { brightness = Math.max(0, brightness - amount); }
    change();
  });
  hue.addEventListener('input', function () { currentHue = Number(hue.value) % 360; change(); });
  hex.addEventListener('input', function () {
    if (!/^#[0-9a-fA-F]{6}$/.test(hex.value)) { return; }
    colour.value = hex.value.toLowerCase();
    readColour(colour.value);
    paint();
    colour.dispatchEvent(new Event('change', { bubbles: true }));
  });
  colour.addEventListener('change', function () { readColour(colour.value); paint(); });
  readColour(colour.value);
  paint();
})();
