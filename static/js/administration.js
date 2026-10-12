(function () {
  const roleForm = document.querySelector('[data-role-editor]');

  if (roleForm) {
    const roleName = roleForm.querySelector('[data-role-name]');
    const roleColour = roleForm.querySelector('[data-role-colour]');
    const preview = roleForm.querySelector('[data-role-preview]');
    const search = roleForm.querySelector('[data-permission-search]');
    const writePermission = roleForm.querySelector('[value="data.write"]');
    const dependants = [...roleForm.querySelectorAll('[value="sql.execute"], [value="tokens.manage"]')];

    function updatePreview() {
      preview.style.setProperty('--role-color', roleColour.value);
      preview.querySelector('[data-role-preview-name]').textContent = roleName.value || 'New role';
      roleForm.querySelectorAll('[data-role-swatch]').forEach(function (button) { button.setAttribute('aria-pressed', String(button.dataset.roleSwatch.toLowerCase() === roleColour.value.toLowerCase())); });
    }

    const saveNote = roleForm.querySelector('[data-role-save-note]');
    const initialValues = [...roleForm.querySelectorAll('input[name]')].map(function (input) { return input.type === 'checkbox' ? input.checked : input.value; });
    function updateChanges() {
      const inputs = [...roleForm.querySelectorAll('input[name]')];
      const changed = inputs.some(function (input, index) { return (input.type === 'checkbox' ? input.checked : input.value) !== initialValues[index]; });
      if (saveNote) { saveNote.textContent = changed ? 'Unsaved changes' : ''; }
    }
    roleForm.addEventListener('input', updateChanges);
    roleForm.addEventListener('change', updateChanges);

    roleForm.querySelectorAll('[data-role-swatch]').forEach(function (button) {
      button.addEventListener('click', function () { roleColour.value = button.dataset.roleSwatch; updatePreview(); roleColour.dispatchEvent(new Event('change', { bubbles: true })); });
    });
    roleForm.querySelectorAll('[data-permissions-clear]').forEach(function (button) {
      button.addEventListener('click', function () { button.closest('[data-permission-group]').querySelectorAll('input[type="checkbox"]:not(:disabled)').forEach(function (input) { input.checked = false; }); updateChanges(); });
    });

    roleName.addEventListener('input', updatePreview);
    roleColour.addEventListener('input', updatePreview);
    roleColour.addEventListener('change', updatePreview);
    updatePreview();

    search.addEventListener('input', function () {
      const query = search.value.trim().toLowerCase();
      roleForm.querySelectorAll('[data-permission-group]').forEach(function (group) {
        const rows = [...group.querySelectorAll('[data-permission-row]')];
        rows.forEach(function (row) {
          row.hidden = !row.textContent.toLowerCase().includes(query);
        });
        group.hidden = rows.every(function (row) { return row.hidden; });
      });
    });

    dependants.forEach(function (permission) {
      permission.addEventListener('change', function () {
        if (permission.checked && !writePermission.disabled) {
          writePermission.checked = true;
        }
      });
    });

    writePermission.addEventListener('change', function () {
      if (!writePermission.checked) {
        dependants.forEach(function (permission) {
          if (!permission.disabled) { permission.checked = false; }
        });
      }
    });

    roleForm.addEventListener('invalid', function (event) {
      const panel = event.target.closest('[data-swap-panel]');
      if (panel && panel.hidden) {
        const tab = document.querySelector('[data-swap-to="' + panel.dataset.swapPanel + '"]');
        if (tab) { tab.click(); }
      }
    }, true);
  }

  const roleMembersTab = document.querySelector('[data-swap-to="role-users"]');
  if (roleMembersTab && roleMembersTab.classList.contains('is-current')) { roleMembersTab.click(); }

  document.querySelectorAll('[data-role-member-search], [data-role-picker-search]').forEach(function (input) {
    input.addEventListener('input', function () {
      const selector = input.hasAttribute('data-role-member-search') ? '[data-role-member]' : '[data-role-picker-user]';
      const query = input.value.trim().toLowerCase();
      document.querySelectorAll(selector).forEach(function (row) { row.hidden = !row.textContent.toLowerCase().includes(query); });
    });
  });

  const userAccess = document.querySelector('[data-user-access]');
  if (userAccess) {
    const choices = userAccess.querySelector('[data-database-choices]');
    function updateDatabaseChoices() {
      const scope = userAccess.querySelector('[name="all"]');
      choices.hidden = scope && scope.value === 'true';
    }
    userAccess.querySelectorAll('[name="all"]').forEach(function (input) {
      input.addEventListener('change', updateDatabaseChoices);
    });
    updateDatabaseChoices();
  }
})();
