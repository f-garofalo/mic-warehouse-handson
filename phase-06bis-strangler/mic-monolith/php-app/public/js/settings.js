/* Categorie / Aliquote IVA / Utenti - simple settings views. */
(function () {
  const M = MIC;

  // ---------- Categorie ----------
  M.registerView('categories', 'Categorie articoli', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/categories',
      newButton: { label: 'Nuova categoria', onClick: () => openCat(null, () => list.reload()) },
      onRowClick: (row) => openCat(row, () => list.reload()),
      columns: [
        { key: 'codice', label: 'Codice' },
        { key: 'nome', label: 'Nome' },
        { key: 'descrizione', label: 'Descrizione' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });
  async function openCat(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'codice', label: 'Codice' },
      { name: 'nome', label: 'Nome', full: true },
      { name: 'descrizione', label: 'Descrizione', type: 'textarea', full: true },
    ], initial);
    const m = M.modal({
      title: isEdit ? 'Modifica categoria' : 'Nuova categoria',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare categoria?', async () => {
            await M.del('/api/categories/' + initial.id); M.toast('Eliminata'); m.close(); onSaved();
          });
        }}, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.nome) return f.setError('nome', 'Obbligatorio');
          if (isEdit) await M.put('/api/categories/' + initial.id, v);
          else await M.post('/api/categories', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }

  // ---------- IVA ----------
  M.registerView('iva', 'Aliquote IVA', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/iva-rates',
      newButton: { label: 'Nuova aliquota', onClick: () => openIva(null, () => list.reload()) },
      onRowClick: (row) => openIva(row, () => list.reload()),
      columns: [
        { key: 'codice', label: 'Codice' },
        { key: 'nome', label: 'Nome' },
        { key: 'percentuale', label: '%', num: true, render: r => r.percentuale + '%' },
        { key: 'descrizione', label: 'Descrizione' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });
  async function openIva(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'codice', label: 'Codice' },
      { name: 'nome', label: 'Nome', full: true },
      { name: 'percentuale', label: 'Percentuale (%)', type: 'number', step: '0.01' },
      { name: 'descrizione', label: 'Descrizione', type: 'textarea', full: true },
    ], initial);
    const m = M.modal({
      title: isEdit ? 'Modifica aliquota' : 'Nuova aliquota',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare aliquota?', async () => {
            await M.del('/api/iva-rates/' + initial.id); M.toast('Eliminata'); m.close(); onSaved();
          });
        }}, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.nome) return f.setError('nome', 'Obbligatorio');
          if (isEdit) await M.put('/api/iva-rates/' + initial.id, v);
          else await M.post('/api/iva-rates', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }

  // ---------- Utenti ----------
  M.registerView('users', 'Utenti', async function (view) {
    M.clear(view);
    const list = M.listView(view, {
      url: '/api/users',
      newButton: { label: 'Nuovo utente', onClick: () => openUser(null, () => list.reload()) },
      onRowClick: (row) => openUser(row, () => list.reload()),
      columns: [
        { key: 'email', label: 'Email' },
        { key: 'nome', label: 'Nome' },
        { key: 'ruolo', label: 'Ruolo' },
        { key: 'status', label: 'Stato', render: r => M.statusBadge(r.status) },
      ],
    });
  });
  async function openUser(initial, onSaved) {
    const isEdit = !!(initial && initial.id);
    const f = M.buildForm([
      { name: 'email', label: 'Email' },
      { name: 'nome', label: 'Nome', full: true },
      { name: 'ruolo', label: 'Ruolo', type: 'select', options: [
        { value: 'admin', label: 'Admin' },
        { value: 'operatore', label: 'Operatore' },
        { value: 'contabile', label: 'Contabile' },
        { value: 'agente', label: 'Agente' },
      ]},
      { name: 'status', label: 'Stato', type: 'select', default: 'attivo',
        options: [{ value: 'attivo', label: 'Attivo' }, { value: 'sospeso', label: 'Sospeso' }] },
    ], initial);
    const m = M.modal({
      title: isEdit ? `Utente ${initial.email}` : 'Nuovo utente',
      body: f.form,
      footer: [
        isEdit ? M.el('button', { class: 'btn btn-danger', onClick: () => {
          M.confirmDialog('Eliminare utente?', async () => {
            await M.del('/api/users/' + initial.id); M.toast('Eliminato'); m.close(); onSaved();
          });
        }}, 'Elimina') : null,
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.email) return f.setError('email', 'Obbligatoria');
          if (isEdit) await M.put('/api/users/' + initial.id, v);
          else await M.post('/api/users', v);
          M.toast('Salvato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }
})();
