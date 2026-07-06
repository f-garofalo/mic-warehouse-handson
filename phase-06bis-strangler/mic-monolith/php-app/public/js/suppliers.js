/* Fornitori */
(function () {
  const M = MIC;
  M.registerView('suppliers', 'Fornitori', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/suppliers',
      newButton: { label: 'Nuovo fornitore', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openForm(row, () => list.reload()),
      columns: [
        { key: 'piva', label: 'P.IVA' },
        { key: 'ragione_sociale', label: 'Ragione sociale' },
        { key: 'citta', label: 'Citta' },
        { key: 'email', label: 'Email' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'ragione_sociale', label: 'Ragione sociale', full: true },
      { name: 'piva', label: 'P.IVA' },
      { name: 'email', label: 'Email' },
      { name: 'pec', label: 'PEC' },
      { name: 'indirizzo', label: 'Indirizzo', full: true },
      { name: 'citta', label: 'Citta' },
      { name: 'provincia', label: 'Provincia' },
      { name: 'cap', label: 'CAP' },
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo',
        options: [{ value: 'attivo', label: 'Attivo' }, { value: 'sospeso', label: 'Sospeso' }] },
    ], initial);
    const m = M.modal({
      title: isEdit ? `Fornitore ${initial.ragione_sociale}` : 'Nuovo fornitore',
      body: f.form, size: 'lg',
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare fornitore?', async () => {
            await M.del('/api/suppliers/' + initial.id);
            M.toast('Fornitore eliminato'); m.close(); onSaved();
          });
        } }, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.ragione_sociale) return f.setError('ragione_sociale', 'Obbligatorio');
          if (isEdit) await M.put('/api/suppliers/' + initial.id, v);
          else await M.post('/api/suppliers', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }
})();
