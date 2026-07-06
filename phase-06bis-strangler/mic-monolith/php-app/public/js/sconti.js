/* Sconti */
(function () {
  const M = MIC;
  M.registerView('sconti', 'Sconti', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/sconti',
      newButton: { label: 'Nuovo sconto', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openForm(row, () => list.reload()),
      columns: [
        { key: 'codice', label: 'Codice' },
        { key: 'nome', label: 'Nome' },
        { key: 'percentuale', label: '%', num: true, render: r => r.percentuale + '%' },
        { key: 'tipo', label: 'Tipo' },
        { key: 'valido_da', label: 'Valido da', render: r => M.fmtDate(r.valido_da) },
        { key: 'valido_a',  label: 'Valido a',  render: r => M.fmtDate(r.valido_a) },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'codice', label: 'Codice promo' },
      { name: 'nome', label: 'Nome', full: true },
      { name: 'descrizione', label: 'Descrizione', type: 'textarea', full: true },
      { name: 'percentuale', label: 'Percentuale (%)', type: 'number', step: '0.01' },
      { name: 'tipo', label: 'Tipo', type: 'select', options: [
        { value: 'percentuale', label: 'Percentuale' },
        { value: 'volume', label: 'Volume' },
        { value: 'fedelta', label: 'Fedelta' },
      ]},
      { name: 'valido_da', label: 'Valido da', type: 'date' },
      { name: 'valido_a',  label: 'Valido a',  type: 'date' },
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo',
        options: [{ value: 'attivo', label: 'Attivo' }, { value: 'archiviato', label: 'Archiviato' }] },
    ], initial);
    const m = M.modal({
      title: isEdit ? `Sconto ${initial.codice}` : 'Nuovo sconto',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare sconto?', async () => {
            await M.del('/api/sconti/' + initial.id); M.toast('Eliminato'); m.close(); onSaved();
          });
        } }, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.codice) return f.setError('codice', 'Obbligatorio');
          if (isEdit) await M.put('/api/sconti/' + initial.id, v);
          else await M.post('/api/sconti', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }
})();
