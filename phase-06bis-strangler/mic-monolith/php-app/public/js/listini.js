/* Listini + voci listino. */
(function () {
  const M = MIC;
  M.registerView('listini', 'Listini', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/listini',
      newButton: { label: 'Nuovo listino', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openDetail(row, () => list.reload()),
      columns: [
        { key: 'codice', label: 'Codice' },
        { key: 'nome',   label: 'Nome' },
        { key: 'valido_da', label: 'Valido da', render: r => M.fmtDate(r.valido_da) },
        { key: 'valido_a',  label: 'Valido a',  render: r => M.fmtDate(r.valido_a) },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'codice', label: 'Codice' },
      { name: 'nome',   label: 'Nome', full: true },
      { name: 'descrizione', label: 'Descrizione', type: 'textarea', full: true },
      { name: 'valido_da', label: 'Valido da', type: 'date' },
      { name: 'valido_a',  label: 'Valido a',  type: 'date' },
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo',
        options: [{ value: 'attivo', label: 'Attivo' }, { value: 'archiviato', label: 'Archiviato' }] },
    ], initial);

    const m = M.modal({
      title: isEdit ? `Listino ${initial.codice}` : 'Nuovo listino',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare listino e tutte le voci?', async () => {
            await M.del('/api/listini/' + initial.id);
            M.toast('Listino eliminato'); m.close(); onSaved();
          });
        } }, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.codice) return f.setError('codice', 'Obbligatorio');
          if (!v.nome)   return f.setError('nome', 'Obbligatorio');
          if (isEdit) await M.put('/api/listini/' + initial.id, v);
          else await M.post('/api/listini', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }

  async function openDetail(row, onSaved) {
    const voci = (await M.get('/api/listini/' + row.id + '/voci')).data;
    const articles = (await M.get('/api/articles?limit=500')).data;

    const tbl = M.el('table', { class: 'tbl' });
    tbl.innerHTML = '<thead><tr><th>Codice voce</th><th>Articolo</th><th class="num">Prezzo</th></tr></thead>';
    const tbody = M.el('tbody');
    for (const v of voci.slice(0, 200)) {
      tbody.appendChild(M.el('tr', null,
        M.el('td', null, v.codice),
        M.el('td', null, v.nome),
        M.el('td', { class: 'num' }, M.fmtMoney(v.prezzo))));
    }
    tbl.appendChild(tbody);

    const addArt = M.el('select');
    addArt.appendChild(M.el('option', { value: '' }, '-- scegli articolo --'));
    for (const a of articles) addArt.appendChild(M.el('option', { value: a.id }, `${a.sku} - ${a.nome}`));
    const addPrezzo = M.el('input', { type: 'number', step: '0.01', placeholder: 'Prezzo EUR' });
    const addBtn = M.el('button', { class: 'btn btn-primary' }, '+ Aggiungi voce');

    addBtn.addEventListener('click', async () => {
      const articolo_id = Number(addArt.value);
      const prezzo = Number(addPrezzo.value);
      if (!articolo_id || !prezzo) return M.toast('Articolo e prezzo richiesti', 'warn');
      await M.post('/api/listini/' + row.id + '/voci', { articolo_id, prezzo });
      M.toast('Voce aggiunta'); m.close(); onSaved();
    });

    const m = M.modal({
      title: `Listino ${row.codice} - ${row.nome}`,
      size: 'lg',
      body: M.el('div', null,
        M.el('div', { class: 'card' },
          M.el('h4', null, 'Aggiungi voce'),
          M.el('div', { class: 'toolbar' }, addArt, addPrezzo, addBtn),
        ),
        M.el('h4', null, `Voci (${voci.length})`),
        tbl,
        voci.length > 200 ? M.el('p', null, `Mostrate prime 200 di ${voci.length}.`) : null,
      ),
    });
  }
})();
