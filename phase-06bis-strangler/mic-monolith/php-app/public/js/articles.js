/* Articoli - listing + create/edit modal */
(function () {
  const M = MIC;

  M.registerView('articles', 'Articoli', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/articles',
      filters: [{ name: 'status', label: 'Stato', type: 'select', options: [
        { value: 'attivo', label: 'Attivo' },
        { value: 'fuori_produzione', label: 'Fuori produzione' },
      ]}],
      newButton: { label: 'Nuovo articolo', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openForm(row, () => list.reload()),
      columns: [
        { key: 'sku',  label: 'SKU' },
        { key: 'nome', label: 'Nome' },
        { key: 'categoria', label: 'Categoria' },
        { key: 'iva_default', label: 'IVA' },
        { key: 'prezzo_listino', label: 'Prezzo', num: true,
          render: (r) => M.fmtMoney(r.prezzo_listino) },
        { key: 'qta_minima', label: 'Q. min', num: true },
        { key: 'status', label: 'Stato', render: (r) => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const cats = (await M.get('/api/categories?limit=200')).data;
    const ivas = (await M.get('/api/iva-rates')).data;

    const f = M.buildForm([
      { name: 'sku', label: 'SKU' },
      { name: 'nome', label: 'Nome', full: true },
      { name: 'categoria', label: 'Categoria', type: 'select',
        options: cats.map(c => ({ value: c.codice, label: `${c.codice} - ${c.nome}` })) },
      { name: 'iva_default', label: 'IVA', type: 'select',
        options: ivas.map(i => ({ value: i.codice, label: `${i.codice} (${i.percentuale}%)` })) },
      { name: 'prezzo_listino', label: 'Prezzo (EUR)', type: 'number', step: '0.01' },
      { name: 'qta_minima',     label: 'Quantita minima', type: 'number' },
      { name: 'unita', label: 'Unita di misura', default: 'pz' },
      { name: 'descrizione', label: 'Descrizione', type: 'textarea', full: true },
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo',
        options: [{ value: 'attivo', label: 'Attivo' }, { value: 'fuori_produzione', label: 'Fuori produzione' }] },
    ], initial);

    const m = M.modal({
      title: isEdit ? `Articolo ${initial.sku}` : 'Nuovo articolo',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog(`Eliminare articolo ${initial.sku}?`, async () => {
            await M.del('/api/articles/' + initial.id);
            M.toast('Articolo eliminato'); m.close(); onSaved();
          });
        } }, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          f.clearErrors();
          const v = f.getValues();
          if (!v.sku) return f.setError('sku', 'Obbligatorio');
          if (!v.nome) return f.setError('nome', 'Obbligatorio');
          if (!v.prezzo_listino || v.prezzo_listino <= 0) return f.setError('prezzo_listino', 'Prezzo non valido');
          if (isEdit) await M.put('/api/articles/' + initial.id, v);
          else await M.post('/api/articles', v);
          M.toast('Articolo salvato'); m.close(); onSaved();
        } }, 'Salva'),
      ],
    });
  }
})();
