/* Magazzino - lista magazzini + giacenze + nuovo movimento. */
(function () {
  const M = MIC;

  M.registerView('magazzino', 'Magazzino', async function (view) {
    M.clear(view);

    // Two cards side-by-side: list of warehouses + recent movimenti
    const wrap = M.el('div', { class: 'grid-2' });

    const magCard = M.el('div', { class: 'card' });
    magCard.appendChild(M.el('h3', null, 'Magazzini'));
    const magBody = M.el('div');
    magCard.appendChild(magBody);
    wrap.appendChild(magCard);

    const movCard = M.el('div', { class: 'card' });
    const movHead = M.el('div', { class: 'toolbar' },
      M.el('h3', { style: { margin: 0 } }, 'Movimenti recenti'),
      M.el('div', { class: 'grow' }),
      M.el('button', { class: 'btn btn-primary', onClick: () => openMov(() => reload()) }, '+ Nuovo movimento'),
    );
    movCard.appendChild(movHead);
    const movBody = M.el('div');
    movCard.appendChild(movBody);
    wrap.appendChild(movCard);
    view.appendChild(wrap);

    async function reload() {
      const mags = (await M.get('/api/magazzini')).data;
      M.clear(magBody);
      const tbl = M.el('table', { class: 'tbl' });
      tbl.innerHTML = '<thead><tr><th>Codice</th><th>Nome</th><th>Citta</th><th></th></tr></thead>';
      const tb = M.el('tbody');
      for (const mg of mags) {
        const tr = M.el('tr');
        tr.innerHTML = `<td>${M.escape(mg.codice)}</td><td>${M.escape(mg.nome)}</td><td>${M.escape(mg.citta || '-')}</td>`;
        const td = M.el('td', { class: 'num' });
        td.appendChild(M.el('button', { class: 'btn btn-sm', onClick: () => openGiacenze(mg) }, 'Giacenze'));
        tr.appendChild(td);
        tb.appendChild(tr);
      }
      tbl.appendChild(tb);
      magBody.appendChild(tbl);

      // movimenti recenti
      M.clear(movBody);
      const movs = (await M.get('/api/movimenti?limit=20&order=date_1&dir=DESC')).data;
      const t2 = M.el('table', { class: 'tbl' });
      t2.innerHTML = '<thead><tr><th>Data</th><th>Tipo</th><th>SKU</th><th class="num">Qta</th><th>Mag.</th></tr></thead>';
      const tb2 = M.el('tbody');
      for (const mv of movs) {
        const tr = M.el('tr');
        tr.innerHTML = `<td>${M.fmtDate(mv.data)}</td>
                        <td>${M.escape(mv.tipo || '-')}</td>
                        <td>${M.escape(mv.articolo_sku || '-')}</td>
                        <td class="num">${mv.qta}</td>
                        <td>${M.escape(mv.magazzino_nome || '-')}</td>`;
        tb2.appendChild(tr);
      }
      t2.appendChild(tb2);
      movBody.appendChild(t2);
    }

    reload();
  });

  async function openGiacenze(mag) {
    const data = (await M.get('/api/magazzini/' + mag.id + '/giacenze')).data;
    const tbl = M.el('table', { class: 'tbl' });
    tbl.innerHTML = '<thead><tr><th>SKU</th><th>Articolo</th><th class="num">Giacenza</th></tr></thead>';
    const tb = M.el('tbody');
    if (!data.length) tb.appendChild(M.el('tr', null, M.el('td', { colspan: 3 }, 'Nessun movimento.')));
    for (const r of data) {
      const tr = M.el('tr');
      if (r.giacenza < 0) tr.classList.add('danger-row');
      tr.innerHTML = `<td>${M.escape(r.sku)}</td><td>${M.escape(r.articolo_nome)}</td><td class="num">${M.fmtNum(r.giacenza, 0)}</td>`;
      tb.appendChild(tr);
    }
    tbl.appendChild(tb);
    M.modal({ title: `Giacenze ${mag.nome}`, size: 'lg', body: tbl });
  }

  async function openMov(onSaved) {
    const articoli = (await M.get('/api/articles?limit=500')).data;
    const mags = (await M.get('/api/magazzini')).data;
    const f = M.buildForm([
      { name: 'data', label: 'Data', type: 'date', default: new Date().toISOString().slice(0,10) },
      { name: 'tipo', label: 'Tipo', type: 'select', options: ['entrata','uscita','trasferimento','rettifica'].map(s => ({ value: s, label: s })) },
      { name: 'qta', label: 'Quantita', type: 'number', step: '1' },
      { name: 'articolo_id', label: 'Articolo', type: 'select',
        options: [{ value: '', label: '--' }].concat(articoli.map(a => ({ value: a.id, label: `${a.sku} - ${a.nome}` }))) },
      { name: 'magazzino_id', label: 'Magazzino', type: 'select',
        options: [{ value: '', label: '--' }].concat(mags.map(m => ({ value: m.id, label: m.nome }))) },
      { name: 'causale', label: 'Causale', default: 'inventario' },
    ]);
    const m = M.modal({
      title: 'Nuovo movimento',
      body: f.form,
      footer: [
        M.el('button', { class: 'btn', onClick: () => m.close() }, 'Annulla'),
        M.el('button', { class: 'btn btn-primary', onClick: async () => {
          const v = f.getValues();
          if (!v.articolo_id) return f.setError('articolo_id', 'Obbligatorio');
          if (!v.magazzino_id) return f.setError('magazzino_id', 'Obbligatorio');
          if (!v.qta) return f.setError('qta', 'Obbligatorio');
          await M.post('/api/movimenti', v);
          M.toast('Movimento registrato'); m.close(); onSaved();
        }}, 'Salva'),
      ],
    });
  }
})();
