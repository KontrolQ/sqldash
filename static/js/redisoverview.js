(function () {
  document.querySelectorAll('[data-redis-ssl]').forEach(function (choice) {
    const checkbox = choice.querySelector('input[type="checkbox"]');
    const uri = document.getElementById(choice.dataset.redisSsl);
    const guidance = document.getElementById(choice.dataset.redisSslHelp);
    if (!checkbox || !uri) return;
    function updateSSLChoice() {
      uri.value = uri.value.replace(/^rediss?:\/\//, checkbox.checked ? 'rediss://' : 'redis://');
      if (guidance) guidance.hidden = !checkbox.checked;
    }
    checkbox.addEventListener('change', updateSSLChoice);
    updateSSLChoice();
  });
  window.addEventListener('pagehide', function () {
    document.querySelectorAll('#redis-connection-uri, #redis-overview-password').forEach(function (input) { input.value = ''; input.removeAttribute('value'); });
  });
})();
