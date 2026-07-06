/* Phase 07 teaching overlay.
   The facade injects this file before MIC's app.js. It is intentionally a
   starter: complete Step B4 to add a simplified user login and propagate the
   Bearer token to /api/* requests.
*/
(function () {
  const TOKEN_KEY = 'phase07.accessToken';
  const WORKSPACE_KEY = 'phase07.workspaceId';
  const DEFAULT_WORKSPACE = 'ws-acme';
  const USER_CLIENT_ID = '11111111-2222-4333-8444-555555555555';

  const originalFetch = window.fetch.bind(window);

  window.fetch = async function phase07Fetch(input, init) {
    // TODO Phase 07 Parte B:
    // - clone init without mutating the caller's object;
    // - for same-origin /api/* calls, add:
    //   Authorization: Bearer <sessionStorage token>
    //   X-Workspace-ID: <sessionStorage workspace or ws-acme>
    // - leave /iam/* and external calls untouched.
    return originalFetch(input, init);
  };

  async function loginDemoUser(username) {
    // TODO Phase 07 Parte B:
    // POST application/x-www-form-urlencoded to /iam/oauth/token with:
    // grant_type=password, username, password=demo, client_id, scope.
    // Store access_token in sessionStorage under TOKEN_KEY.
    window.alert('TODO Phase 07: implement demo login before retrying Articles.');
  }

  function logoutDemoUser() {
    sessionStorage.removeItem(TOKEN_KEY);
    window.location.reload();
  }

  function installControls() {
    const topRight = document.querySelector('.top-right');
    if (!topRight || document.getElementById('phase07-auth-controls')) return;

    if (!sessionStorage.getItem(WORKSPACE_KEY)) {
      sessionStorage.setItem(WORKSPACE_KEY, DEFAULT_WORKSPACE);
    }

    const box = document.createElement('span');
    box.id = 'phase07-auth-controls';
    box.style.display = 'inline-flex';
    box.style.gap = '8px';
    box.style.alignItems = 'center';
    box.style.marginRight = '12px';

    const status = document.createElement('small');
    status.textContent = sessionStorage.getItem(TOKEN_KEY) ? 'IAM demo: autenticato' : 'IAM demo: non autenticato';

    const mkLogin = function (username, label) {
      const btn = document.createElement('button');
      btn.className = 'btn btn-sm';
      btn.textContent = label;
      btn.addEventListener('click', function () {
        loginDemoUser(username).catch(function (err) {
          window.alert('Login demo fallito: ' + err.message);
        });
      });
      return btn;
    };

    const logout = document.createElement('button');
    logout.className = 'btn btn-sm';
    logout.textContent = 'Logout';
    logout.addEventListener('click', logoutDemoUser);

    box.appendChild(status);
    box.appendChild(mkLogin('alice', 'Login Alice'));
    box.appendChild(mkLogin('bob', 'Login Bob'));
    box.appendChild(logout);
    topRight.prepend(box);
  }

  document.addEventListener('DOMContentLoaded', installControls);
})();
