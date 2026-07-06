/* Dashboard view: KPIs + charts + recent audit. */
(function () {
  const { registerView, get, el, clear, fmtMoney, fmtDate, escape } = MIC;

  registerView('dashboard', 'Dashboard', async function render(view) {
    clear(view);
    const data = (await get('/api/dashboard/kpi')).data;
    const k = data.kpi;

    const kpis = el('div', { class: 'kpis' },
      kpi('Fatturato del mese', fmtMoney(k.fatturato_mese), `${k.fatture_mese} fatture`, 'success'),
      kpi('Ordini in corso', String(k.ordini_in_corso), 'bozza, confermati, spediti', 'info'),
      kpi('Articoli sotto scorta', String(k.articoli_sotto_scorta), 'attenzione magazzino', 'warn'),
      kpi('Pagamenti scaduti', String(k.pagamenti_scaduti), fmtMoney(k.pagamenti_scaduti_importo), 'danger'),
    );
    view.appendChild(kpis);

    // Two-up charts
    const grid = el('div', { class: 'grid-2' });
    const c1 = el('div', { class: 'card' });
    c1.appendChild(el('h3', null, 'Fatturato ultimi 6 mesi'));
    const cv1 = el('div', { class: 'chart-wrap' }, el('canvas'));
    c1.appendChild(cv1);
    grid.appendChild(c1);

    const c2 = el('div', { class: 'card' });
    c2.appendChild(el('h3', null, 'Top 5 clienti per fatturato'));
    const cv2 = el('div', { class: 'chart-wrap' }, el('canvas'));
    c2.appendChild(cv2);
    grid.appendChild(c2);

    view.appendChild(grid);

    // Pie chart + last activity
    const grid2 = el('div', { class: 'grid-2' });
    const c3 = el('div', { class: 'card' });
    c3.appendChild(el('h3', null, 'Distribuzione ordini per stato'));
    const cv3 = el('div', { class: 'chart-wrap' }, el('canvas'));
    c3.appendChild(cv3);
    grid2.appendChild(c3);

    const c4 = el('div', { class: 'card' });
    c4.appendChild(el('h3', null, 'Ultime 5 attivita'));
    const tbl = el('table', { class: 'tbl' });
    tbl.innerHTML = `<thead><tr>
      <th>Data</th><th>Utente</th><th>Azione</th><th>Target</th></tr></thead>`;
    const tbody = el('tbody');
    for (const r of data.ultimi_log) {
      const tr = el('tr');
      tr.innerHTML = `<td>${fmtDate(r.data)}</td>
                      <td>${escape(r.utente || '-')}</td>
                      <td>${escape(r.azione || '-')}</td>
                      <td>${escape(r.target || '-')}</td>`;
      tbody.appendChild(tr);
    }
    tbl.appendChild(tbody);
    c4.appendChild(tbl);
    grid2.appendChild(c4);
    view.appendChild(grid2);

    // ---- Charts ----
    new Chart(cv1.querySelector('canvas'), {
      type: 'line',
      data: {
        labels: data.fatturato_6m.map(p => p.mese),
        datasets: [{
          label: 'Fatturato (EUR)',
          data: data.fatturato_6m.map(p => p.importo),
          borderColor: '#1a73e8',
          backgroundColor: 'rgba(26,115,232,0.18)',
          fill: true, tension: 0.3,
          pointBackgroundColor: '#1a73e8',
        }],
      },
      options: chartOpts(),
    });

    new Chart(cv2.querySelector('canvas'), {
      type: 'bar',
      data: {
        labels: data.top_clienti.map(c => truncate(c.cliente, 22)),
        datasets: [{
          label: 'Fatturato (EUR)',
          data: data.top_clienti.map(c => c.totale),
          backgroundColor: '#00c853',
        }],
      },
      options: chartOpts({ indexAxis: 'y' }),
    });

    new Chart(cv3.querySelector('canvas'), {
      type: 'doughnut',
      data: {
        labels: data.ordini_per_stato.map(s => s.stato),
        datasets: [{
          data: data.ordini_per_stato.map(s => s.count),
          backgroundColor: ['#1a73e8','#00c853','#f5a623','#5035b3','#e53935','#90a4ae'],
        }],
      },
      options: { responsive: true, maintainAspectRatio: false },
    });
  });

  function kpi(label, value, meta, kind) {
    return el('div', { class: 'kpi ' + (kind || '') },
      el('div', { class: 'label' }, label),
      el('div', { class: 'value' }, value),
      el('div', { class: 'meta' }, meta),
    );
  }
  function chartOpts(extra) {
    return Object.assign({
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: { y: { beginAtZero: true } },
    }, extra || {});
  }
  function truncate(s, n) { return s && s.length > n ? s.slice(0, n - 1) + '…' : s; }
})();
