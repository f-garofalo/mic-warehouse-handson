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
    const url = typeof input === 'string' ? input : (input && input.url) || '';
    const isApi = url.startsWith('/api/') || url.startsWith(window.location.origin + '/api/');
    if (!isApi) {
      // /iam/* and external calls: leave the request exactly as the caller made it.
      return originalFetch(input, init);
    }
    // Clone init without mutating the caller's object, then stamp the identity headers.
    const nextInit = Object.assign({}, init);
    const headers = new Headers((init && init.headers) || {});
    const token = sessionStorage.getItem(TOKEN_KEY);
    if (token) {
      headers.set('Authorization', 'Bearer ' + token);
    }
    headers.set('X-Workspace-ID', sessionStorage.getItem(WORKSPACE_KEY) || DEFAULT_WORKSPACE);
    nextInit.headers = headers;
    return originalFetch(input, nextInit);
  };

  async function loginDemoUser(username) {
    const body = new URLSearchParams({
      grant_type: 'password',
      username: username,
      password: 'demo',
      client_id: USER_CLIENT_ID,
      scope: 'openid profile offline_access',
    });
    // Use the original fetch: this is an /iam/* call and must not carry a Bearer.
    const res = await originalFetch('/iam/oauth/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });
    if (!res.ok) {
      throw new Error('token endpoint returned ' + res.status);
    }
    const data = await res.json();
    sessionStorage.setItem(TOKEN_KEY, data.access_token);
    window.location.reload();
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
