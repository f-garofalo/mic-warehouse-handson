/* Agenti */
(function () {
  const M = MIC;
  M.registerView('agenti', 'Agenti', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/agenti',
      newButton: { label: 'Nuovo agente', onClick: () => openForm(null, () => list.reload()) },
      onRowClick: (row) => openForm(row, () => list.reload()),
      columns: [
        { key: 'codice', label: 'Codice' },
        { key: 'nome', label: 'Nome' },
        { key: 'email', label: 'Email' },
        { key: 'zona', label: 'Zona' },
        { key: 'provvigione', label: 'Provv. %', num: true, render: r => r.provvigione + '%' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });

  async function openForm(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'codice', label: 'Codice' },
      { name: 'nome', label: 'Nome completo', full: true },
      { name: 'email', label: 'Email' },
      { name: 'zona', label: 'Zona' },
      { name: 'provvigione', label: 'Provvigione (%)', type: 'number', step: '0.01' },
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo',
        options: [{ value: 'attivo', label: 'Attivo' }, { value: 'inattivo', label: 'Inattivo' }] },
    ], initial);
    const m = M.modal({
      title: isEdit ? `Agente ${initial.nome}` : 'Nuovo agente',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare agente?', async () => {
            await M.del('/api/agenti/' + initial.id); M.toast('Eliminato'); m.close(); onSaved();
          });
        } }, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.nome) return f.setError('nome', 'Obbligatorio');
          if (isEdit) await M.put('/api/agenti/' + initial.id, v);
          else await M.post('/api/agenti', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }
})();
