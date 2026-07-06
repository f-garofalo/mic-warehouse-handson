/* Ordini - lista + dettaglio con righe inline. */
(function () {
  const M = MIC;

  M.registerView('orders', 'Ordini', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/orders',
      filters: [{ name: 'status', label: 'Stato', type: 'select', options: [
        { value: 'bozza', label: 'Bozza' },
        { value: 'confermato', label: 'Confermato' },
        { value: 'spedito', label: 'Spedito' },
        { value: 'fatturato', label: 'Fatturato' },
        { value: 'annullato', label: 'Annullato' },
      ]}],
      newButton: { label: 'Nuovo ordine', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openDetail(row, () => list.reload()),
      columns: [
        { key: 'numero', label: 'Numero' },
        { key: 'data',   label: 'Data', render: r => M.fmtDate(r.data) },
        { key: 'cliente_nome', label: 'Cliente' },
        { key: 'agente_nome',  label: 'Agente' },
        { key: 'totale', label: 'Totale', num: true, render: r => M.fmtMoney(r.totale) },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const customers = (await M.get('/api/customers?limit=200')).data;
    const listini = (await M.get('/api/listini')).data;
    const agenti = (await M.get('/api/agenti')).data;

    const f = M.buildForm([
      { name: 'numero', label: 'Numero ordine' },
      { name: 'data', label: 'Data', type: 'date', default: new Date().toISOString().slice(0,10) },
      { name: 'cliente_id', label: 'Cliente', type: 'select',
        options: [{ value: '', label: '-- seleziona --' }].concat(customers.map(c => ({ value: c.id, label: `${c.ragione_sociale} (${c.piva_or_cf || '-'})` }))) },
      { name: 'listino_id', label: 'Listino', type: 'select',
        options: [{ value: '', label: '-- nessuno --' }].concat(listini.map(l => ({ value: l.id, label: l.nome }))) },
      { name: 'agente_id', label: 'Agente', type: 'select',
        options: [{ value: '', label: '-- nessuno --' }].concat(agenti.map(a => ({ value: a.id, label: a.nome }))) },
      { name: 'sconto_globale', label: 'Sconto globale %', type: 'number', step: '0.01', default: 0 },
      { name: 'status', label: 'Stato', type: 'select', default: 'bozza',
        options: ['bozza','confermato','spedito','fatturato','annullato'].map(s => ({ value: s, label: s })) },
      { name: 'note', label: 'Note', type: 'textarea', full: true },
    ], initial);

    const m = M.modal({
      title: isEdit ? `Ordine ${initial.numero}` : 'Nuovo ordine',
      body: f.form, size: 'lg',
      footer: [
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.cliente_id) return f.setError('cliente_id', 'Obbligatorio');
          if (isEdit) await M.put('/api/orders/' + initial.id, v);
          else await M.post('/api/orders', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }

  async function openDetail(row, onSaved) {
    const articoli = (await M.get('/api/articles?limit=500')).data;

    const head = M.el('div', { class: 'card' });
    head.innerHTML = `
      <div class="grid-2">
        <div><b>Numero:</b> ${M.escape(row.numero)}<br>
             <b>Data:</b> ${M.fmtDate(row.data)}<br>
             <b>Stato:</b> ${M.statusBadge(row.status)}<br>
             <b>Sconto globale:</b> ${row.sconto_globale || 0}%</div>
        <div><b>Cliente:</b> ${M.escape(row.cliente_nome || '-')}<br>
             <b>Listino:</b> ${M.escape(row.listino_nome || '-')}<br>
             <b>Agente:</b> ${M.escape(row.agente_nome || '-')}<br>
             <b>Totale:</b> <span style="font-size:18px;font-weight:600;color:var(--mic-blue)">${M.fmtMoney(row.totale)}</span></div>
      </div>`;

    const righeHost = M.el('div');
    async function reloadRighe() {
      M.clear(righeHost);
      const righe = (await M.get('/api/orders/' + row.id + '/righe')).data;
      const tbl = M.el('table', { class: 'tbl' });
      tbl.innerHTML = `<thead><tr>
        <th>SKU</th><th>Articolo</th>
        <th class="num">Qta</th><th class="num">Prezzo</th>
        <th class="num">Sconto %</th><th class="num">Imponibile</th>
        <th class="num">IVA</th><th class="num">Totale</th></tr></thead>`;
      const tbody = M.el('tbody');
      let imp = 0, iva = 0, tot = 0;
      for (const r of righe) {
        imp += Number(r.imponibile || 0); iva += Number(r.iva || 0); tot += Number(r.totale || 0);
        const tr = M.el('tr');
        tr.innerHTML = `<td>${M.escape(r.sku || '-')}</td>
                        <td>${M.escape(r.articolo_nome || '-')}</td>
                        <td class="num">${r.qta}</td>
                        <td class="num">${M.fmtMoney(r.prezzo_unitario)}</td>
                        <td class="num">${r.sconto_riga || 0}</td>
                        <td class="num">${M.fmtMoney(r.imponibile)}</td>
                        <td class="num">${M.fmtMoney(r.iva)} (${r.iva_perc}%)</td>
                        <td class="num">${M.fmtMoney(r.totale)}</td>`;
        tbody.appendChild(tr);
      }
      tbl.appendChild(tbody);
      tbl.appendChild(M.el('tfoot', null, M.el('tr', null,
        M.el('td', { colspan: 5 }, 'Totali:'),
        M.el('td', { class: 'num' }, M.fmtMoney(imp)),
        M.el('td', { class: 'num' }, M.fmtMoney(iva)),
        M.el('td', { class: 'num' }, M.fmtMoney(tot)),
      )));
      righeHost.appendChild(tbl);
    }
    reloadRighe();

    const artSel = M.el('select');
    artSel.appendChild(M.el('option', { value: '' }, '-- articolo --'));
    for (const a of articoli) artSel.appendChild(M.el('option', { value: a.id }, `${a.sku} - ${a.nome} (${M.fmtMoney(a.prezzo_listino)})`));
    const qtaIn = M.el('input', { type: 'number', step: '1', placeholder: 'Qta', value: '1' });
    const prezzoIn = M.el('input', { type: 'number', step: '0.01', placeholder: 'Prezzo (auto)' });
    const scontoIn = M.el('input', { type: 'number', step: '0.01', placeholder: 'Sconto %', value: '0' });
    const addBtn = M.el('button', { class: 'btn btn-primary' }, '+ Aggiungi riga');
    addBtn.addEventListener('click', async () => {
      const articolo_id = Number(artSel.value);
      const qta = Number(qtaIn.value);
      const prezzo_unitario = Number(prezzoIn.value || 0);
      const sconto_riga = Number(scontoIn.value || 0);
      if (!articolo_id || !qta) return M.toast('Articolo e qta richiesti', 'warn');
      await M.post('/api/orders/' + row.id + '/righe',
        { articolo_id, qta, prezzo_unitario, sconto_riga });
      qtaIn.value = '1'; prezzoIn.value = ''; scontoIn.value = '0';
      M.toast('Riga aggiunta');
      await reloadRighe();
      onSaved();
    });

    const genFat = M.el('button', { class: 'btn btn-success' }, 'Genera fattura');
    genFat.addEventListener('click', async () => {
      await M.post('/api/invoices', {
        cliente_id: row.cliente_id,
        ordine_id: row.id,
        numero: 'FAT-' + Date.now(),
        data: new Date().toISOString().slice(0,10),
        scadenza: new Date(Date.now() + 30*86400000).toISOString().slice(0,10),
        imponibile: row.totale ? row.totale * 0.82 : 0,
        iva: row.totale ? row.totale * 0.18 : 0,
        totale: row.totale,
        status: 'bozza',
      });
      M.toast('Fattura generata'); m.close(); onSaved();
    });

    M.modal({
      title: `Ordine ${row.numero}`, size: 'lg',
      body: M.el('div', null,
        head,
        M.el('div', { class: 'card' },
          M.el('h4', null, 'Aggiungi riga'),
          M.el('div', { class: 'toolbar' }, artSel, qtaIn, prezzoIn, scontoIn, addBtn)),
        M.el('h4', null, 'Righe ordine'),
        righeHost,
      ),
      footer: [
        M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare ordine?', async () => {
            await M.del('/api/orders/' + row.id);
            M.toast('Eliminato'); onSaved(); document.getElementById('modal-root').innerHTML = '';
          });
        }}, 'Elimina'),
        genFat,
      ],
    });
    var m = { close: () => document.getElementById('modal-root').innerHTML = '' };
  }
})();
