/* Clienti */
(function () {
  const M = MIC;

  M.registerView('customers', 'Clienti', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/customers',
      newButton: { label: 'Nuovo cliente', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openDetail(row, () => list.reload()),
      columns: [
        { key: 'piva_or_cf', label: 'P.IVA / CF' },
        { key: 'ragione_sociale', label: 'Ragione sociale' },
        { key: 'tipo', label: 'Tipo' },
        { key: 'citta', label: 'Citta' },
        { key: 'email', label: 'Email' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'ragione_sociale', label: 'Ragione sociale / Nome', full: true },
      { name: 'tipo', label: 'Tipo', type: 'select', options: [
        { value: 'B2B', label: 'B2B (Azienda)' },
        { value: 'B2C', label: 'B2C (Privato)' },
      ]},
      { name: 'piva_or_cf', label: 'P.IVA o Codice Fiscale' },
      { name: 'cf', label: 'Codice Fiscale (se diverso)' },
      { name: 'email', label: 'Email' },
      { name: 'pec', label: 'PEC' },
      { name: 'codice_sdi', label: 'Codice SDI' },
      { name: 'indirizzo', label: 'Indirizzo', full: true },
      { name: 'citta', label: 'Citta' },
      { name: 'provincia', label: 'Provincia' },
      { name: 'cap', label: 'CAP' },
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo', options: [
        { value: 'attivo', label: 'Attivo' }, { value: 'sospeso', label: 'Sospeso' },
      ]},
    ], initial);
    const m = M.modal({
      title: isEdit ? `Cliente ${initial.ragione_sociale}` : 'Nuovo cliente',
      body: f.form, size: 'lg',
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare cliente?', async () => {
            await M.del('/api/customers/' + initial.id); M.toast('Cliente eliminato');
            m.close(); onSaved();
          });
        } }, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          f.clearErrors();
          const v = f.getValues();
          if (!v.ragione_sociale) return f.setError('ragione_sociale', 'Obbligatorio');
          if (isEdit) await M.put('/api/customers/' + initial.id, v);
          else await M.post('/api/customers', v);
          M.toast('Cliente salvato'); m.close(); onSaved();
        } }, 'Salva'),
      ],
    });
  }

  async function openDetail(row, onSaved) {
    const tabs = M.el('div', { class: 'tabs' });
    const body = M.el('div');
    const tabsDef = [
      { key: 'anag',     label: 'Anagrafica' },
      { key: 'orders',   label: 'Ordini' },
      { key: 'invoices', label: 'Fatture' },
      { key: 'pay',      label: 'Pagamenti' },
    ];
    for (const t of tabsDef) {
      const a = M.el('div', { class: 'tab', onClick: () => activate(t.key) }, t.label);
      a.dataset.k = t.key;
      tabs.appendChild(a);
    }
    const m = M.modal({ title: row.ragione_sociale, size: 'lg', body: M.el('div', null, tabs, body) });

    async function activate(k) {
      tabs.querySelectorAll('.tab').forEach(x => x.classList.toggle('active', x.dataset.k === k));
      M.clear(body);
      if (k === 'anag') {
        // re-use form inline
        body.appendChild(M.el('p', null, 'Modifica i dati e salva.'));
        const f = M.buildForm([
          { name: 'ragione_sociale', label: 'Ragione sociale', full: true },
          { name: 'piva_or_cf', label: 'P.IVA / CF' },
          { name: 'tipo', label: 'Tipo', type: 'select', options: [
            { value: 'B2B', label: 'B2B' }, { value: 'B2C', label: 'B2C' },
          ]},
          { name: 'email', label: 'Email' },
          { name: 'pec', label: 'PEC' },
          { name: 'codice_sdi', label: 'SDI' },
          { name: 'indirizzo', label: 'Indirizzo', full: true },
          { name: 'citta', label: 'Citta' },
          { name: 'provincia', label: 'Provincia' },
          { name: 'cap', label: 'CAP' },
        ], row);
        body.appendChild(f.form);
        body.appendChild(M.el('div', { style: { marginTop: '12px' } },
          M.el('button', { class: 'btn btn-primary', onClick: async () => {
            await M.put('/api/customers/' + row.id, f.getValues());
            M.toast('Salvato'); m.close(); onSaved();
          } }, 'Salva')));
      } else if (k === 'orders') {
        const data = (await M.get('/api/customers/' + row.id + '/orders')).data;
        body.appendChild(renderRowsTable(data, ['numero','data','totale','status'], {
          totale: M.fmtMoney, data: M.fmtDate, status: M.statusBadge,
        }, 'Nessun ordine.'));
      } else if (k === 'invoices') {
        const data = (await M.get('/api/customers/' + row.id + '/invoices')).data;
        body.appendChild(renderRowsTable(data, ['numero','data','scadenza','totale','status'], {
          totale: M.fmtMoney, data: M.fmtDate, scadenza: M.fmtDate, status: M.statusBadge,
        }, 'Nessuna fattura.'));
      } else if (k === 'pay') {
        const fatt = (await M.get('/api/customers/' + row.id + '/invoices')).data;
        const tot = fatt.reduce((s,f) => s + (Number(f.totale) || 0), 0);
        const pagate = fatt.filter(f => f.status === 'pagato');
        const scadute = fatt.filter(f => f.status === 'scaduto');
        const aperte  = fatt.filter(f => f.status === 'inviato');
        body.appendChild(M.el('div', { class: 'kpis' },
          kpi('Fatturato totale', M.fmtMoney(tot)),
          kpi('Fatture pagate', String(pagate.length), 'success'),
          kpi('Fatture aperte', String(aperte.length)),
          kpi('Fatture scadute', String(scadute.length), 'danger'),
        ));
      }
    }
    activate('anag');
  }

  function kpi(label, value, kind) {
    return M.el('div', { class: 'kpi ' + (kind || '') },
      M.el('div', { class: 'label' }, label),
      M.el('div', { class: 'value' }, value),
    );
  }

  function renderRowsTable(rows, cols, fmts, emptyMsg) {
    if (!rows.length) return M.el('div', { class: 'card' }, emptyMsg);
    const tbl = M.el('table', { class: 'tbl' });
    const trh = M.el('tr');
    for (const c of cols) trh.appendChild(M.el('th', null, c));
    tbl.appendChild(M.el('thead', null, trh));
    const tbody = M.el('tbody');
    for (const r of rows) {
      const tr = M.el('tr');
      for (const c of cols) {
        const f = fmts[c];
        const v = r[c];
        if (f === M.statusBadge) tr.appendChild(M.el('td', { html: f(v) || '' }));
        else tr.appendChild(M.el('td', null, f ? f(v) : (v == null ? '' : String(v))));
      }
      tbody.appendChild(tr);
    }
    tbl.appendChild(tbody);
    return tbl;
  }
})();
