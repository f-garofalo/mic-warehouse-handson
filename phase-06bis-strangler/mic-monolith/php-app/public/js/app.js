/* =====================================================================
   MIC SPA shell.
   - hash-based router (#/dashboard, #/customers, ...)
   - fetch wrapper with toast on error
   - modal + toast helpers
   - shared table renderer
   ===================================================================== */

window.MIC = (function () {
  const routes = {};        // name -> render(view, params)
  const titles = {};        // name -> string

  function registerView(name, title, renderer) {
    routes[name] = renderer;
    titles[name] = title;
  }

  // ---------- API ----------
  async function api(method, url, body) {
    const opts = { method, headers: { 'Accept': 'application/json' } };
    if (body !== undefined) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    let res;
    try {
      res = await fetch(url, opts);
    } catch (e) {
      toast('Errore di rete: ' + e.message, 'error');
      throw e;
    }
    let data = null;
    const text = await res.text();
    try { data = text ? JSON.parse(text) : null; } catch (_) {}
    if (!res.ok) {
      const msg = (data && data.error) ? data.error : ('HTTP ' + res.status);
      toast('Errore: ' + msg, 'error');
      throw new Error(msg);
    }
    return data;
  }

  const get = (url) => api('GET', url);
  const post = (url, body) => api('POST', url, body || {});
  const put = (url, body) => api('PUT', url, body || {});
  const del = (url) => api('DELETE', url);

  // ---------- DOM helpers ----------
  function el(tag, attrs, ...children) {
    const e = document.createElement(tag);
    if (attrs) {
      for (const k of Object.keys(attrs)) {
        const v = attrs[k];
        if (v == null) continue;
        if (k === 'class') e.className = v;
        else if (k === 'html') e.innerHTML = v;
        else if (k.startsWith('on') && typeof v === 'function') e.addEventListener(k.slice(2).toLowerCase(), v);
        else if (k === 'style' && typeof v === 'object') Object.assign(e.style, v);
        else if (k === 'dataset' && typeof v === 'object') Object.assign(e.dataset, v);
        else e.setAttribute(k, v);
      }
    }
    for (const c of children.flat()) {
      if (c == null || c === false) continue;
      e.appendChild(typeof c === 'string' || typeof c === 'number' ? document.createTextNode(String(c)) : c);
    }
    return e;
  }
  function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); }

  function fmtMoney(n) {
    if (n == null || n === '' || isNaN(n)) return '-';
    return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format(Number(n));
  }
  function fmtNum(n, digits = 2) {
    if (n == null || n === '' || isNaN(n)) return '-';
    return new Intl.NumberFormat('it-IT', { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(Number(n));
  }
  function fmtDate(s) {
    if (!s) return '-';
    const d = new Date(s);
    if (isNaN(d)) return s;
    return d.toLocaleDateString('it-IT');
  }
  function statusBadge(s) {
    if (!s) return '';
    const cls = 'badge badge-' + (s.replace(/\s+/g, '-'));
    return `<span class="${cls}">${escape(s)}</span>`;
  }
  function escape(s) {
    if (s == null) return '';
    return String(s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  // ---------- toast ----------
  function toast(msg, kind) {
    const root = document.getElementById('toast-root');
    const t = el('div', { class: 'toast' + (kind ? ' ' + kind : '') }, msg);
    root.appendChild(t);
    setTimeout(() => t.remove(), 3500);
  }

  // ---------- modal ----------
  function modal({ title, body, footer, size }) {
    const root = document.getElementById('modal-root');
    clear(root);
    const close = () => clear(root);
    const back = el('div', { class: 'modal-backdrop', onClick: (e) => { if (e.target === back) close(); } },
      el('div', { class: 'modal' + (size === 'lg' ? ' lg' : '') },
        el('div', { class: 'modal-head' },
          el('h3', null, title || ''),
          el('button', { class: 'close', onClick: close }, '×')
        ),
        el('div', { class: 'modal-body' }, body || ''),
        footer ? el('div', { class: 'modal-foot' }, footer) : null,
      )
    );
    root.appendChild(back);
    return { close };
  }

  function confirmDialog(message, onYes) {
    const m = modal({
      title: 'Conferma',
      body: el('div', null, message),
      footer: [
        el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        el('button', { class: 'btn btn-danger', onClick: () => { onYes(); m.close(); } }, 'Conferma'),
      ],
    });
  }

  // ---------- router ----------
  async function navigate() {
    const hash = location.hash || '#/dashboard';
    const [_, route, ...rest] = hash.split('/');
    const name = route || 'dashboard';
    const params = rest;

    document.querySelectorAll('.nav a').forEach(a =>
      a.classList.toggle('active', a.dataset.route === name)
    );
    document.getElementById('page-title').textContent = titles[name] || name;

    const view = document.getElementById('view');
    clear(view);
    view.appendChild(el('div', { class: 'loading' }, 'Caricamento...'));

    try {
      const renderer = routes[name];
      if (!renderer) {
        clear(view);
        view.appendChild(el('div', { class: 'card' }, 'Pagina non trovata: ' + name));
        return;
      }
      await renderer(view, params);
    } catch (e) {
      console.error(e);
      clear(view);
      view.appendChild(el('div', { class: 'card' }, 'Errore: ' + e.message));
    }
  }

  function boot() {
    window.addEventListener('hashchange', navigate);
    if (!location.hash) location.hash = '#/dashboard';
    navigate();

    // Global search: simple navigation hint
    const gs = document.getElementById('global-search');
    gs.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter') return;
      const q = gs.value.trim();
      if (!q) return;
      const candidates = [
        ['articolo', 'articles'], ['cliente', 'customers'],
        ['fornitore', 'suppliers'], ['ordine', 'orders'],
        ['fattura', 'invoices'], ['listino', 'listini'],
      ];
      const lc = q.toLowerCase();
      const hit = candidates.find(([k]) => lc.includes(k));
      if (hit) location.hash = `#/${hit[1]}?q=${encodeURIComponent(q)}`;
      else location.hash = `#/articles?q=${encodeURIComponent(q)}`;
    });
  }

  // ---------- table renderer ----------
  /**
   * Render a paginated table from an API list endpoint.
   *
   * config = {
   *   url: string,
   *   columns: [{ key, label, render?, num?, sortable? }],
   *   onRowClick: row => void,
   *   filters: [{ name, label, type, options? }]    // for status select etc.
   *   newButton: { label, onClick }
   *   pageSize?: number
   * }
   */
  function listView(host, config) {
    let state = { q: '', status: '', limit: config.pageSize || 25, offset: 0, total: 0 };

    const toolbar = el('div', { class: 'toolbar' });
    const search = el('input', {
      type: 'search', placeholder: 'Cerca...',
      onInput: (e) => { state.q = e.target.value; state.offset = 0; debouncedReload(); }
    });
    toolbar.appendChild(search);
    if ((config.filters || []).length) {
      for (const f of config.filters) {
        if (f.type === 'select') {
          const sel = el('select', { onChange: (e) => { state[f.name] = e.target.value; state.offset = 0; reload(); } });
          sel.appendChild(el('option', { value: '' }, f.label));
          for (const o of f.options) sel.appendChild(el('option', { value: o.value }, o.label));
          toolbar.appendChild(sel);
        }
      }
    }
    toolbar.appendChild(el('div', { class: 'grow' }));
    if (config.newButton) {
      toolbar.appendChild(el('button', {
        class: 'btn btn-primary', onClick: config.newButton.onClick
      }, '+ ' + config.newButton.label));
    }
    host.appendChild(toolbar);

    const tableHost = el('div', { class: 'card', style: { padding: 0, overflow: 'hidden' } });
    host.appendChild(tableHost);

    const pag = el('div', { class: 'pagination' });
    host.appendChild(pag);

    let reloadTimer = null;
    function debouncedReload() {
      clearTimeout(reloadTimer);
      reloadTimer = setTimeout(reload, 250);
    }

    async function reload() {
      const params = new URLSearchParams();
      if (state.q) params.set('q', state.q);
      if (state.status) params.set('status', state.status);
      params.set('limit', state.limit);
      params.set('offset', state.offset);
      const url = config.url + '?' + params.toString();
      const res = await get(url);
      state.total = res.meta.total;
      const tbl = el('table', { class: 'tbl' });
      const thead = el('thead');
      const trh = el('tr');
      for (const c of config.columns) trh.appendChild(el('th', { class: c.num ? 'num' : null }, c.label));
      thead.appendChild(trh);
      tbl.appendChild(thead);
      const tbody = el('tbody');
      for (const row of res.data) {
        const tr = el('tr');
        if (config.onRowClick) tr.addEventListener('click', () => config.onRowClick(row));
        for (const c of config.columns) {
          const v = c.render ? c.render(row) : row[c.key];
          if (typeof v === 'string' && v.startsWith('<')) {
            tr.appendChild(el('td', { class: c.num ? 'num' : null, html: v }));
          } else {
            tr.appendChild(el('td', { class: c.num ? 'num' : null }, v == null ? '' : String(v)));
          }
        }
        tbody.appendChild(tr);
      }
      tbl.appendChild(tbody);
      clear(tableHost); tableHost.appendChild(tbl);

      // pagination
      clear(pag);
      const from = state.total === 0 ? 0 : state.offset + 1;
      const to = Math.min(state.offset + state.limit, state.total);
      pag.appendChild(el('span', null, `${from}-${to} di ${state.total}`));
      pag.appendChild(el('button', {
        class: 'btn btn-sm', onClick: () => { state.offset = Math.max(0, state.offset - state.limit); reload(); },
      }, '<'));
      pag.appendChild(el('button', {
        class: 'btn btn-sm', onClick: () => { if (state.offset + state.limit < state.total) { state.offset += state.limit; reload(); } },
      }, '>'));
    }

    reload();
    return { reload };
  }

  // ---------- form helpers ----------
  function buildForm(fields, initial) {
    const form = el('form', { class: 'form-grid' });
    const refs = {};
    for (const f of fields) {
      const wrap = el('div', { class: 'field' + (f.full ? ' field-full' : '') });
      wrap.appendChild(el('label', null, f.label));
      let input;
      const v = initial && initial[f.name] != null ? initial[f.name] : (f.default ?? '');
      if (f.type === 'select') {
        input = el('select');
        for (const o of f.options) input.appendChild(el('option', { value: o.value, selected: String(o.value) === String(v) ? 'selected' : null }, o.label));
      } else if (f.type === 'textarea') {
        input = el('textarea', { rows: 3 }); input.value = v;
      } else {
        input = el('input', { type: f.type || 'text', step: f.step });
        input.value = v;
      }
      input.name = f.name;
      wrap.appendChild(input);
      const err = el('div', { class: 'err' });
      wrap.appendChild(err);
      form.appendChild(wrap);
      refs[f.name] = { input, err, def: f };
    }
    function getValues() {
      const out = {};
      for (const k of Object.keys(refs)) {
        let v = refs[k].input.value;
        const def = refs[k].def;
        if (def.type === 'number') v = v === '' ? null : Number(v);
        out[k] = v;
      }
      return out;
    }
    function setError(field, msg) {
      if (refs[field]) refs[field].err.textContent = msg || '';
    }
    function clearErrors() {
      Object.keys(refs).forEach(k => refs[k].err.textContent = '');
    }
    return { form, getValues, setError, clearErrors };
  }

  return {
    boot, registerView, api, get, post, put, del,
    el, clear, fmtMoney, fmtNum, fmtDate, escape, statusBadge,
    toast, modal, confirmDialog, listView, buildForm,
  };
})();
