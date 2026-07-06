/* Fatture */
(function () {
  const M = MIC;
  M.registerView('invoices', 'Fatture', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/invoices',
      filters: [{ name: 'status', label: 'Stato', type: 'select', options: [
        { value: 'bozza', label: 'Bozza' },
        { value: 'inviato', label: 'Inviata' },
        { value: 'pagato', label: 'Pagata' },
        { value: 'scaduto', label: 'Scaduta' },
      ]}],
      newButton: { label: 'Nuova fattura', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openDetail(row, () => list.reload()),
      columns: [
        { key: 'numero', label: 'Numero' },
        { key: 'data',   label: 'Data', render: r => M.fmtDate(r.data) },
        { key: 'scadenza', label: 'Scadenza', render: r => M.fmtDate(r.scadenza) },
        { key: 'cliente_nome', label: 'Cliente' },
        { key: 'totale', label: 'Totale', num: true, render: r => M.fmtMoney(r.totale) },
        { key: 'sdi_status', label: 'SDI' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const customers = (await M.get('/api/customers?limit=200')).data;
    const f = M.buildForm([
      { name: 'numero', label: 'Numero' },
      { name: 'data', label: 'Data', type: 'date', default: new Date().toISOString().slice(0,10) },
      { name: 'scadenza', label: 'Scadenza', type: 'date' },
      { name: 'cliente_id', label: 'Cliente', type: 'select',
        options: [{ value: '', label: '-- seleziona --' }].concat(customers.map(c => ({ value: c.id, label: c.ragione_sociale }))) },
      { name: 'imponibile', label: 'Imponibile', type: 'number', step: '0.01' },
      { name: 'iva', label: 'IVA', type: 'number', step: '0.01' },
      { name: 'totale', label: 'Totale', type: 'number', step: '0.01' },
      { name: 'sezionale', label: 'Sezionale', default: 'VEN' },
      { name: 'status', label: 'Stato', type: 'select', default: 'bozza',
        options: ['bozza','inviato','pagato','scaduto'].map(s => ({ value: s, label: s })) },
    ], initial);
    const m = M.modal({
      title: isEdit ? `Fattura ${initial.numero}` : 'Nuova fattura',
      body: f.form, size: 'lg',
      footer: [
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.cliente_id) return f.setError('cliente_id', 'Obbligatorio');
          if (isEdit) await M.put('/api/invoices/' + initial.id, v);
          else await M.post('/api/invoices', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }

  async function openDetail(row, onSaved) {
    const head = M.el('div', { class: 'card' });
    head.innerHTML = `
      <div class="grid-2">
        <div><b>Numero:</b> ${M.escape(row.numero)}<br>
             <b>Data emissione:</b> ${M.fmtDate(row.data)}<br>
             <b>Scadenza:</b> ${M.fmtDate(row.scadenza)}<br>
             <b>Sezionale:</b> ${M.escape(row.sezionale || '-')}<br>
             <b>SDI:</b> ${M.escape(row.sdi_status || '-')}</div>
        <div><b>Cliente:</b> ${M.escape(row.cliente_nome || '-')}<br>
             <b>Ordine:</b> ${M.escape(row.ordine_numero || '-')}<br>
             <b>Imponibile:</b> ${M.fmtMoney(row.imponibile)}<br>
             <b>IVA:</b> ${M.fmtMoney(row.iva)}<br>
             <b>Totale:</b> <span style="font-size:18px;font-weight:600;color:var(--mic-blue)">${M.fmtMoney(row.totale)}</span><br>
             <b>Stato:</b> ${M.statusBadge(row.status)}</div>
      </div>`;

    const sdiBtn = M.el('button', { class: 'btn btn-success' }, 'Invia a SDI (mock)');
    sdiBtn.addEventListener('click', async () => {
      await M.post('/api/invoices/' + row.id + '/invia-sdi', {});
      M.toast('Inviata allo SDI'); m.close(); onSaved();
    });

    const payBtn = M.el('button', { class: 'btn btn-primary' }, 'Registra pagamento');
    payBtn.addEventListener('click', async () => {
      await M.post('/api/pagamenti', {
        codice: 'PAG-' + Date.now(),
        descrizione: 'Pagamento ' + row.numero,
        data: new Date().toISOString().slice(0,10),
        importo: row.totale,
        metodo: 'Bonifico',
        fattura_id: row.id,
      });
      await M.put('/api/invoices/' + row.id, { status: 'pagato' });
      M.toast('Pagamento registrato'); m.close(); onSaved();
    });

    const m = M.modal({
      title: `Fattura ${row.numero}`, size: 'lg',
      body: M.el('div', null, head),
      footer: [
        M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare fattura?', async () => {
            await M.del('/api/invoices/' + row.id);
            M.toast('Eliminata'); m.close(); onSaved();
          });
        }}, 'Elimina'),
        sdiBtn,
        payBtn,
      ],
    });
  }
})();
