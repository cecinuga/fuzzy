// Explorer for the HTML report: search, filters, sorting and insights over the
// page's own tables. It reads the figures the page carries in data-* attributes,
// writes text only through text nodes, and makes no requests: the page's policy
// allows this script by its hash, and nothing else at all. Without it, the page
// is the plain report.
(() => {
  'use strict';

  const summaryTable = document.querySelector('table.summary');
  const summaryRows = summaryTable ? [...summaryTable.querySelectorAll('tr[data-ep]')] : [];
  if (summaryRows.length === 0) return;

  const PAGE = 100;
  const CLASSES = ['1xx', '2xx', '3xx', '4xx', '5xx', 'err'];
  const CLASS_NAMES = {
    '1xx': '1xx informational', '2xx': '2xx success', '3xx': '3xx redirect',
    '4xx': '4xx client error', '5xx': '5xx server error', err: 'ERR no response',
  };
  const METHOD_ORDER = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS', 'TRACE'];
  const METRICS = ['mean', 'p50', 'p95', 'p99', 'max'];
  const runMs = Number(document.body.dataset.durationMs) || 0;
  const skipped = Number(document.body.dataset.skipped) || 0;

  // ---- Helpers -------------------------------------------------------------

  // h builds an element. Strings among the children become text nodes, so
  // nothing passed in is ever parsed as markup.
  function h(tag, props, ...children) {
    const el = document.createElement(tag);
    for (const [key, value] of Object.entries(props || {})) {
      if (value == null || value === false) continue;
      if (key === 'class') el.className = value;
      else if (key === 'text') el.textContent = value;
      else if (key === 'style') Object.assign(el.style, value);
      else if (key.startsWith('on')) el.addEventListener(key.slice(2), value);
      else el.setAttribute(key, value === true ? '' : String(value));
    }
    for (const child of children.flat(Infinity)) {
      if (child != null && child !== false) el.append(child);
    }
    return el;
  }

  const nf = new Intl.NumberFormat('en-US');
  const fmtInt = (n) => nf.format(n);
  function fmtPct(x) {
    if (!Number.isFinite(x)) return '–';
    const p = x * 100;
    if (p === 0) return '0%';
    if (p < 0.1) return '<0.1%';
    if (p < 10 || (p > 99.5 && p < 100)) return p.toFixed(1) + '%';
    return Math.round(p) + '%';
  }
  function fmtMs(ms) {
    if (ms >= 10000) return (ms / 1000).toFixed(1) + ' s';
    if (ms >= 1000) return (ms / 1000).toFixed(2) + ' s';
    if (ms >= 100) return Math.round(ms) + ' ms';
    if (ms >= 10) return ms.toFixed(1) + ' ms';
    return ms.toFixed(2) + ' ms';
  }
  function fmtDuration(ms) {
    if (ms < 1000) return Math.round(ms) + ' ms';
    if (ms < 60000) return (ms / 1000).toFixed(1) + ' s';
    const m = Math.floor(ms / 60000);
    return m + 'm ' + Math.round((ms % 60000) / 1000) + 's';
  }
  const fmtRate = (r) => (r >= 100 ? fmtInt(Math.round(r)) : r.toFixed(1));
  const statusLabel = (s) => (s === 0 ? 'ERR' : String(s));
  // classOf groups statuses the way the report colors them: 0 is no response.
  function classOf(s) {
    if (s === 0) return 'err';
    if (s >= 500) return '5xx';
    if (s >= 400) return '4xx';
    if (s >= 300) return '3xx';
    if (s >= 200) return '2xx';
    return '1xx';
  }
  const zeroClasses = () => Object.fromEntries(CLASSES.map((c) => [c, 0]));
  const setHidden = (el, hidden) => { if (el && el.hidden !== hidden) el.hidden = hidden; };

  // ---- Model, read from the page -------------------------------------------

  const endpoints = summaryRows.map((row) => {
    const d = row.dataset;
    const i = Number(d.ep);
    const codes = new Map();
    for (const pair of (d.codes || '').split(' ')) {
      if (!pair) continue;
      const [status, count] = pair.split(':').map(Number);
      codes.set(status, count);
    }
    const classes = zeroClasses();
    for (const [status, count] of codes) classes[classOf(status)] += count;
    const section = document.getElementById('ep-' + i);
    const ep = {
      i, row, section, codes, classes,
      name: row.cells[0].textContent,
      method: d.method || '',
      sent: Number(d.sent) || 0,
      rt: Object.fromEntries(['count', 'min', ...METRICS].map((k) => [k, Number(d[k]) || 0])),
      rows: [], text: null, visible: true, matched: 0, limit: PAGE, more: null,
      sortCol: -1, sortDir: 0,
    };
    const table = section && section.querySelector('table.results');
    if (table) {
      ep.table = table;
      ep.rows = [...table.querySelectorAll('tr[data-s]')].map((tr, k) => {
        const s = Number(tr.dataset.s);
        return { tr, k, ep, s, cls: classOf(s), ms: Number(tr.dataset.ms) || 0, match: true, shown: true, text: null, msg: null };
      });
    }
    return ep;
  });
  const totalRows = endpoints.reduce((n, ep) => n + ep.rows.length, 0);
  const tbody = summaryRows[0].parentNode;
  const container = document.getElementById('endpoints');

  // Searchable text of an endpoint: its name, description and parameters.
  function epText(ep) {
    if (ep.text == null) {
      const parts = [ep.name];
      if (ep.section) {
        for (const el of ep.section.querySelectorAll(':scope > h2, :scope > p:not(.stats), :scope > ul, table.params')) {
          parts.push(el.textContent);
        }
      }
      ep.text = parts.join('\n').toLowerCase();
    }
    return ep.text;
  }
  function rowText(r) {
    if (r.text == null) r.text = r.tr.textContent.toLowerCase();
    return r.text;
  }
  const rowMessage = (r) => (r.tr.cells[2] ? r.tr.cells[2].textContent : '');
  // messageKey folds what changes between otherwise identical messages: ids,
  // numbers and UUIDs.
  function messageKey(r) {
    if (r.msg == null) {
      r.msg = rowMessage(r)
        .replace(/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi, '<uuid>')
        .replace(/0x[0-9a-f]+/gi, '#')
        .replace(/\d+(\.\d+)?/g, '#');
    }
    return r.msg;
  }

  // ---- State, kept in the URL fragment so a reload keeps the view ----------

  const SORTS = {
    spec: { label: 'Spec order', key: (ep) => ep.i, asc: true },
    name: { label: 'Endpoint', key: (ep) => ep.name, asc: true },
    sent: { label: 'Requests sent', key: (ep) => ep.sent },
    reported: { label: 'Results reported', key: (ep) => ep.rows.length },
    failures: { label: 'Failure rate (5xx + ERR)', key: (ep) => (ep.sent ? (ep.classes['5xx'] + ep.classes.err) / ep.sent : null) },
    '5xx': { label: '5xx count', key: (ep) => ep.classes['5xx'] },
    err: { label: 'ERR count', key: (ep) => ep.classes.err },
    mean: { label: 'Mean time', key: (ep) => (ep.rt.count ? ep.rt.mean : null) },
    p50: { label: 'p50 time', key: (ep) => (ep.rt.count ? ep.rt.p50 : null) },
    p95: { label: 'p95 time', key: (ep) => (ep.rt.count ? ep.rt.p95 : null) },
    p99: { label: 'p99 time', key: (ep) => (ep.rt.count ? ep.rt.p99 : null) },
    max: { label: 'Max time', key: (ep) => (ep.rt.count ? ep.rt.max : null) },
    tail: { label: 'Tail ratio (p99 / p50)', key: (ep) => (ep.rt.count && ep.rt.p50 > 0 ? ep.rt.p99 / ep.rt.p50 : null) },
  };
  const defaults = () => ({ q: '', re: false, methods: new Set(), classes: new Set(), minMs: 0, sort: 'spec', asc: true, metric: 'p95' });
  let state = defaults();

  const presentMethods = [...new Set(endpoints.map((ep) => ep.method))].sort((a, b) => {
    const ia = METHOD_ORDER.indexOf(a), ib = METHOD_ORDER.indexOf(b);
    return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib) || a.localeCompare(b);
  });
  const allClasses = zeroClasses();
  for (const ep of endpoints) for (const c of CLASSES) allClasses[c] += ep.classes[c];
  const presentClasses = CLASSES.filter((c) => allClasses[c] > 0);

  function readHash() {
    const hash = location.hash;
    if (!hash.startsWith('#!')) return false;
    const p = new URLSearchParams(hash.slice(2));
    const list = (key) => (p.get(key) || '').split(',').filter(Boolean);
    const s = defaults();
    s.q = p.get('q') || '';
    s.re = p.get('re') === '1';
    s.methods = new Set(list('m').filter((m) => presentMethods.includes(m)));
    s.classes = new Set(list('s').filter((c) => presentClasses.includes(c)));
    s.minMs = Math.max(0, Number(p.get('min')) || 0);
    if (SORTS[p.get('sort')]) s.sort = p.get('sort');
    s.asc = p.has('dir') ? p.get('dir') === 'asc' : !!SORTS[s.sort].asc;
    if (METRICS.includes(p.get('metric'))) s.metric = p.get('metric');
    state = s;
    return true;
  }
  function writeHash() {
    const p = new URLSearchParams();
    if (state.q) p.set('q', state.q);
    if (state.re) p.set('re', '1');
    if (state.methods.size) p.set('m', [...state.methods].join(','));
    if (state.classes.size) p.set('s', [...state.classes].join(','));
    if (state.minMs) p.set('min', String(state.minMs));
    if (state.sort !== 'spec') p.set('sort', state.sort);
    if (state.asc !== !!SORTS[state.sort].asc) p.set('dir', state.asc ? 'asc' : 'desc');
    if (state.metric !== 'p95') p.set('metric', state.metric);
    const query = p.toString();
    const ours = location.hash.startsWith('#!');
    if (!query && !ours) return; // leave a jump to an endpoint alone
    const want = query ? '#!' + query : location.href.split('#')[0];
    if (query && location.hash === '#!' + query) return;
    try { history.replaceState(null, '', want); } catch (_) { /* some browsers refuse it on file: pages */ }
  }

  // matcher returns null without a query, else a test of an endpoint's text and
  // a row's. Plain queries need every word somewhere in either; a regular
  // expression needs to match one of them.
  let queryError = false;
  function matcher() {
    queryError = false;
    const q = state.q.trim();
    if (!q) return null;
    if (state.re) {
      let re;
      try { re = new RegExp(q, 'i'); } catch (_) { queryError = true; return null; }
      return (a, b) => re.test(a) || (b !== '' && re.test(b));
    }
    const terms = q.toLowerCase().split(/\s+/);
    return (a, b) => terms.every((t) => a.includes(t) || b.includes(t));
  }
  const filtering = () => !!(state.q.trim() || state.methods.size || state.classes.size || state.minMs);

  // ---- Toolbar -------------------------------------------------------------

  const ui = {};
  function chip(label, onclick, swatchClass) {
    return h('button', { type: 'button', class: 'chip', 'aria-pressed': 'false', onclick },
      swatchClass ? h('span', { class: 'dot ' + swatchClass, 'aria-hidden': 'true' }) : null, label);
  }
  function toggle(set, value) {
    if (set.has(value)) set.delete(value); else set.add(value);
  }
  function buildToolbar() {
    ui.search = h('input', {
      type: 'search', placeholder: 'Search endpoints, parameters, messages, inputs…',
      'aria-label': 'Search', autocomplete: 'off', spellcheck: 'false',
      oninput: () => { state.q = ui.search.value; schedule(); },
      onkeydown: (e) => {
        if (e.key !== 'Escape') return;
        if (ui.search.value) { ui.search.value = ''; state.q = ''; apply(); } else ui.search.blur();
      },
    });
    ui.regex = h('input', { type: 'checkbox', onchange: () => { state.re = ui.regex.checked; apply(); } });
    ui.searchBox = h('div', { class: 'fz-search' },
      h('span', { class: 'muted', 'aria-hidden': 'true' }, '⌕'), ui.search,
      h('label', { title: 'Treat the search as a regular expression' }, ui.regex, h('code', null, '.*')),
      h('kbd', { title: 'Press / to search' }, '/'));

    ui.chips = [];
    const chips = (values, set, label, swatch) => h('span', { class: 'fz-group', role: 'group', 'aria-label': label },
      values.map((v) => {
        const button = chip(v === 'err' ? 'ERR' : v, () => { toggle(state[set], v); apply(); }, swatch ? 'k-' + v : null);
        ui.chips.push([button, set, v]);
        return button;
      }));
    ui.methods = chips(presentMethods, 'methods', 'HTTP method', false);
    ui.statuses = chips(presentClasses, 'classes', 'Status class', true);
    ui.minMs = h('input', {
      type: 'number', min: '0', step: 'any', placeholder: '0', 'aria-label': 'Minimum time in milliseconds',
      oninput: () => { state.minMs = Math.max(0, Number(ui.minMs.value) || 0); schedule(); },
    });
    ui.sort = h('select', { 'aria-label': 'Sort endpoints by', onchange: () => setSort(ui.sort.value) },
      Object.entries(SORTS).map(([key, s]) => h('option', { value: key }, s.label)));
    ui.dir = h('button', { type: 'button', class: 'btn', onclick: () => { state.asc = !state.asc; sortEndpoints(); apply(); } });
    ui.count = h('span', { class: 'fz-count', 'aria-live': 'polite' });

    ui.bar = h('div', { class: 'fz-bar', role: 'search' },
      h('div', { class: 'fz-row' }, ui.searchBox),
      h('div', { class: 'fz-row' },
        presentMethods.length > 1 ? h('span', { class: 'fz-group' }, h('span', { class: 'fz-label' }, 'Method'), ui.methods) : null,
        presentClasses.length > 1 ? h('span', { class: 'fz-group' }, h('span', { class: 'fz-label' }, 'Status'), ui.statuses) : null,
        h('label', { class: 'fz-group' }, h('span', { class: 'fz-label' }, 'Slower than'), ui.minMs, h('span', { class: 'fz-label' }, 'ms')),
        h('span', { class: 'fz-group' }, h('span', { class: 'fz-label' }, 'Sort'), ui.sort, ui.dir),
        h('button', { type: 'button', class: 'btn', onclick: reset }, 'Reset')),
      h('div', { class: 'fz-row' }, ui.count));
    return ui.bar;
  }
  // syncToolbar makes the controls show the state, after a reset or a reload.
  function syncToolbar() {
    if (ui.search.value !== state.q) ui.search.value = state.q;
    ui.regex.checked = state.re;
    if ((Number(ui.minMs.value) || 0) !== state.minMs) ui.minMs.value = state.minMs || '';
    for (const [button, set, value] of ui.chips) button.setAttribute('aria-pressed', String(state[set].has(value)));
    ui.sort.value = state.sort;
    ui.dir.textContent = state.asc ? '↑ Asc' : '↓ Desc';
    ui.dir.setAttribute('aria-label', state.asc ? 'Ascending, switch to descending' : 'Descending, switch to ascending');
    if (queryError) ui.searchBox.setAttribute('data-invalid', ''); else ui.searchBox.removeAttribute('data-invalid');
    ui.search.setAttribute('aria-invalid', String(queryError));
    for (const [th, key] of ui.summaryHeads) {
      th.setAttribute('aria-sort', state.sort === key ? (state.asc ? 'ascending' : 'descending') : 'none');
    }
  }
  function reset() {
    const { sort, asc, metric } = state;
    state = Object.assign(defaults(), { sort, asc, metric });
    apply();
    ui.search.focus();
  }
  let timer = 0;
  function schedule() {
    clearTimeout(timer);
    timer = setTimeout(apply, 120);
  }

  // ---- Sorting -------------------------------------------------------------

  function setSort(key) {
    if (state.sort === key) state.asc = !state.asc;
    else { state.sort = key; state.asc = !!SORTS[key].asc; }
    sortEndpoints();
    apply();
  }
  // compareKeys orders by key, and puts endpoints without a value last either way.
  function compareKeys(a, b, asc) {
    if (a == null || b == null) return (a == null) - (b == null);
    const c = typeof a === 'string' ? a.localeCompare(b) : a - b;
    return asc ? c : -c;
  }
  function sortEndpoints() {
    const key = SORTS[state.sort].key;
    const order = [...endpoints].sort((a, b) => compareKeys(key(a), key(b), state.asc) || a.i - b.i);
    for (const ep of order) {
      tbody.append(ep.row);
      if (container && ep.section) container.append(ep.section);
    }
  }
  // The summary's headings sort the endpoints too.
  function enhanceSummaryHeads() {
    const keys = ['name', 'sent', 'reported', 'failures', 'p95'];
    const heads = summaryTable.querySelectorAll('tr:first-child > th');
    ui.summaryHeads = [];
    heads.forEach((th, col) => {
      const key = keys[col];
      if (!key) return;
      const label = th.textContent;
      th.replaceChildren(h('button', { type: 'button', class: 'sort', title: 'Sort by ' + SORTS[key].label.toLowerCase(), onclick: () => setSort(key) }, label));
      ui.summaryHeads.push([th, key]);
    });
  }
  // Each results table sorts by any column: descending, ascending, then as run.
  function enhanceResultHeads(ep) {
    if (!ep.table) return;
    const heads = [...ep.table.querySelectorAll('tr:first-child > th')];
    ep.heads = heads;
    heads.forEach((th, col) => {
      const label = th.textContent;
      th.setAttribute('aria-sort', 'none');
      th.replaceChildren(h('button', { type: 'button', class: 'sort', onclick: () => sortRows(ep, col) }, label));
    });
  }
  function sortRows(ep, col) {
    ep.sortDir = ep.sortCol === col ? (ep.sortDir === -1 ? 1 : ep.sortDir === 1 ? 0 : -1) : -1;
    ep.sortCol = col;
    const numeric = { 1: (r) => r.s, 4: (r) => r.ms }[col];
    const key = numeric || ((r) => (r.tr.cells[col] ? r.tr.cells[col].textContent : ''));
    ep.rows.sort((a, b) => {
      if (ep.sortDir === 0) return a.k - b.k;
      const ka = key(a), kb = key(b);
      const c = numeric ? ka - kb : ka < kb ? -1 : ka > kb ? 1 : 0;
      return c * ep.sortDir || a.k - b.k;
    });
    const parent = ep.rows.length ? ep.rows[0].tr.parentNode : null;
    for (const r of ep.rows) parent.append(r.tr);
    ep.heads.forEach((th, i) => th.setAttribute('aria-sort',
      i !== col || ep.sortDir === 0 ? 'none' : ep.sortDir < 0 ? 'descending' : 'ascending'));
    renderRows(ep);
  }

  // ---- Filtering -----------------------------------------------------------

  function apply() {
    clearTimeout(timer);
    const match = matcher();
    let shownEndpoints = 0, shownRows = 0;
    for (const ep of endpoints) {
      let ok = (!state.methods.size || state.methods.has(ep.method))
        && (!state.classes.size || [...state.classes].some((c) => ep.classes[c] > 0))
        && (!state.minMs || ep.rt.max >= state.minMs || ep.rows.some((r) => r.ms >= state.minMs));
      const text = match ? epText(ep) : '';
      const nameHit = !match || match(text, '');
      let matched = 0;
      for (const r of ep.rows) {
        r.match = ok
          && (!state.classes.size || state.classes.has(r.cls))
          && r.ms >= state.minMs
          && (nameHit || match(text, rowText(r)));
        if (r.match) matched++;
      }
      // A search that only matches results hides the endpoints without any.
      if (ok && !nameHit && matched === 0) ok = false;
      ep.visible = ok;
      ep.matched = matched;
      setHidden(ep.row, !ok);
      setHidden(ep.section, !ok);
      if (ok) { shownEndpoints++; shownRows += matched; }
      renderRows(ep);
    }
    syncToolbar();
    ui.count.replaceChildren(
      h('strong', null, fmtInt(shownEndpoints)), ` of ${fmtInt(endpoints.length)} endpoints · `,
      h('strong', null, fmtInt(shownRows)), ` of ${fmtInt(totalRows)} reported results`,
      queryError ? ' · invalid regular expression, ignored' : '');
    renderInsights(endpoints.filter((ep) => ep.visible));
    writeHash();
  }

  // renderRows shows the first rows that match, a page at a time, and touches
  // only the rows that change, so big tables stay quick to filter.
  function renderRows(ep) {
    if (!ep.table) return;
    let n = 0;
    for (const r of ep.rows) {
      const show = r.match && n < ep.limit;
      if (r.match) n++;
      if (r.shown !== show) { r.tr.hidden = !show; r.shown = show; }
    }
    if (!ep.more) {
      ep.more = h('div', { class: 'fz-more' });
      ep.table.parentNode.after(ep.more);
    }
    const parts = [];
    if (ep.rows.length && n === 0) parts.push('No results match the filters.');
    else if (n > ep.limit) {
      parts.push(`Showing ${fmtInt(ep.limit)} of ${fmtInt(n)} matching results.`,
        h('button', { type: 'button', class: 'btn', onclick: () => { ep.limit += 500; renderRows(ep); } }, `Show ${fmtInt(Math.min(500, n - ep.limit))} more`),
        h('button', { type: 'button', class: 'btn', onclick: () => { ep.limit = Infinity; renderRows(ep); } }, 'Show all'));
    }
    ep.more.replaceChildren(...parts);
    ep.more.hidden = parts.length === 0;
  }

  // reveal scrolls to a result row, paging the table as far as needed.
  function reveal(r) {
    const ep = r.ep;
    let index = 0;
    for (const other of ep.rows) {
      if (other === r) break;
      if (other.match) index++;
    }
    if (index >= ep.limit) { ep.limit = Math.ceil((index + 1) / PAGE) * PAGE; renderRows(ep); }
    r.tr.scrollIntoView({ block: 'center', behavior: 'smooth' });
    r.tr.classList.remove('flash');
    void r.tr.offsetWidth; // restart the animation
    r.tr.classList.add('flash');
  }

  // ---- Insights ------------------------------------------------------------

  function tile(label, value, sub, swatch) {
    return h('div', { class: 'tile' },
      h('div', { class: 'l' }, swatch ? h('span', { class: 'dot ' + swatch, 'aria-hidden': 'true' }) : null, label),
      h('div', { class: 'v' }, value),
      sub ? h('div', { class: 's' }, sub) : null);
  }
  const epLink = (ep) => h('a', { href: '#ep-' + ep.i, class: 'name' }, ep.name);

  function card(title, hint, body, extra) {
    return h('div', { class: 'card' + (extra && extra.wide ? ' wide' : '') },
      h('h3', null, h('span', null, title), extra && extra.control),
      hint ? h('p', { class: 'hint' }, hint) : null, body);
  }
  // rankList renders the top entries of a ranking, each with its value and an
  // optional bar scaled to the largest.
  function rankList(items, empty) {
    if (items.length === 0) return h('p', { class: 'muted' }, empty);
    return h('ol', { class: 'rank' }, items.map((it) => h('li', null,
      it.label, h('span', { class: 'v' }, it.value),
      it.sub ? h('span', { class: 'sub' }, it.sub) : null,
      it.bar != null ? h('span', { class: 'bar', 'aria-hidden': 'true' },
        h('i', { class: it.barClass || '', style: { width: Math.max(1, Math.min(100, it.bar * 100)) + '%' } })) : null)));
  }
  function more(list, top) {
    return list.length > top ? h('p', { class: 'hint' }, `and ${fmtInt(list.length - top)} more`) : null;
  }

  function renderInsights(eps) {
    const sent = eps.reduce((n, ep) => n + ep.sent, 0);
    const cls = zeroClasses();
    const codes = new Map();
    let responses = 0, weighted = 0, worst = null;
    for (const ep of eps) {
      for (const c of CLASSES) cls[c] += ep.classes[c];
      for (const [s, n] of ep.codes) codes.set(s, (codes.get(s) || 0) + n);
      if (ep.rt.count) {
        responses += ep.rt.count;
        weighted += ep.rt.mean * ep.rt.count;
        if (!worst || ep.rt.p95 > worst.rt.p95) worst = ep;
      }
    }

    ui.scope.textContent = filtering()
      ? `filtered: ${fmtInt(eps.length)} of ${fmtInt(endpoints.length)} endpoints`
      : `all ${fmtInt(endpoints.length)} endpoints`;

    const tiles = h('div', { class: 'tiles' },
      tile('Endpoints', fmtInt(eps.length),
        filtering() ? `of ${fmtInt(endpoints.length)} tested` : skipped ? `${fmtInt(skipped)} more skipped` : 'all tested'),
      tile('Requests sent', fmtInt(sent),
        runMs > 0 ? `${fmtRate(sent / (runMs / 1000))} req/s over ${fmtDuration(runMs)}` : null),
      tile('Server errors', fmtInt(cls['5xx']), `${fmtPct(sent ? cls['5xx'] / sent : 0)} of requests`, cls['5xx'] ? 'k-5xx' : null),
      tile('No response', fmtInt(cls.err), `${fmtPct(sent ? cls.err / sent : 0)} of requests`, cls.err ? 'k-err' : null),
      tile('Mean time', responses ? fmtMs(weighted / responses) : '–', `over ${fmtInt(responses)} responses`),
      tile('Worst p95', worst ? fmtMs(worst.rt.p95) : '–', worst ? epLink(worst) : 'no responses'));

    const focused = document.activeElement && document.activeElement.dataset.keep;
    ui.insightsBody.replaceChildren(tiles, statusMix(cls, codes, sent), h('div', { class: 'cards' }, rankings(eps)));
    if (focused) { const el = ui.insightsBody.querySelector(`[data-keep="${focused}"]`); if (el) el.focus(); }
    fitSegmentLabels();
  }

  // statusMix is one bar split by status class, with its legend and the exact
  // codes as the table view.
  function statusMix(cls, codes, total) {
    const parts = CLASSES.filter((c) => cls[c] > 0);
    const wrap = h('div', { class: 'mix' }, h('h3', null, 'Status mix'));
    if (total === 0) {
      wrap.append(h('p', { class: 'muted' }, 'No requests.'));
      return wrap;
    }
    const describe = (c) => `${CLASS_NAMES[c]}: ${fmtInt(cls[c])} (${fmtPct(cls[c] / total)})`;
    wrap.append(
      h('div', { class: 'segbar', role: 'img', 'aria-label': 'Status mix. ' + parts.map(describe).join('; ') },
        parts.map((c) => h('div', {
          class: 'seg k-' + c, tabindex: '0', title: describe(c), 'aria-label': describe(c),
          style: { flex: `${cls[c]} 1 0px` },
        }, fmtPct(cls[c] / total)))),
      h('ul', { class: 'legend' }, parts.map((c) => h('li', null, h('span', { class: 'sw k-' + c, 'aria-hidden': 'true' }), CLASS_NAMES[c]))),
      h('p', { class: 'codes' }, 'Codes: ', [...codes.keys()].sort((a, b) => (a || 1000) - (b || 1000))
        .map((s) => `${statusLabel(s)} ×${fmtInt(codes.get(s))}`).join(' · ')));
    return wrap;
  }
  // A segment's label shows only where it fits; hover, focus and the legend
  // carry the rest.
  function fitSegmentLabels() {
    requestAnimationFrame(() => {
      for (const seg of ui.insightsBody.querySelectorAll('.seg')) {
        if (seg.scrollWidth > seg.clientWidth) seg.textContent = '';
      }
    });
  }

  function rankings(eps) {
    const TOP = 5;
    const timed = eps.filter((ep) => ep.rt.count > 0);
    const m = state.metric;
    const slowest = [...timed].sort((a, b) => b.rt[m] - a.rt[m]);
    const peak = slowest.length ? slowest[0].rt[m] || 1 : 1;
    const latencyItem = (ep) => ({
      label: epLink(ep), value: fmtMs(ep.rt[m]), bar: ep.rt[m] / peak,
      sub: `p50 ${fmtMs(ep.rt.p50)} · p95 ${fmtMs(ep.rt.p95)} · max ${fmtMs(ep.rt.max)} · ${fmtInt(ep.rt.count)} responses`,
    });
    const metricSelect = h('select', {
      'aria-label': 'Response time figure', 'data-keep': 'metric',
      onchange: (e) => { state.metric = e.target.value; apply(); },
    }, METRICS.map((k) => h('option', { value: k, selected: k === m }, k)));

    const failing = eps.filter((ep) => ep.classes['5xx'] > 0)
      .sort((a, b) => b.classes['5xx'] / b.sent - a.classes['5xx'] / a.sent || b.classes['5xx'] - a.classes['5xx']);
    const unreachable = eps.filter((ep) => ep.classes.err > 0)
      .sort((a, b) => b.classes.err - a.classes.err || b.classes.err / b.sent - a.classes.err / a.sent);
    const blocked = eps.filter((ep) => ep.sent > 0 && ep.classes['2xx'] + ep.classes['3xx'] === 0);
    const jittery = timed.filter((ep) => ep.rt.count >= 20 && ep.rt.p50 > 0 && ep.rt.p99 / ep.rt.p50 >= 2)
      .sort((a, b) => b.rt.p99 / b.rt.p50 - a.rt.p99 / a.rt.p50);
    const peakTail = jittery.length ? jittery[0].rt.p99 / jittery[0].rt.p50 : 1;
    const topCodes = (ep) => [...ep.codes].sort((a, b) => b[1] - a[1]).slice(0, 3)
      .map(([s, n]) => `${statusLabel(s)} ×${fmtInt(n)}`).join(', ');

    return [
      card('Slowest endpoints', `By ${m} response time; failed requests are not timed.`,
        rankList(slowest.slice(0, TOP).map(latencyItem), 'No responses to time.'), { control: metricSelect }),
      card('Fastest endpoints', `By ${m} response time.`,
        rankList([...slowest].reverse().slice(0, TOP).map(latencyItem), 'No responses to time.')),
      card('Server errors', 'Highest share of 5xx responses: the first places to look for bugs.',
        [rankList(failing.slice(0, TOP).map((ep) => ({
          label: epLink(ep), value: fmtPct(ep.classes['5xx'] / ep.sent), bar: ep.classes['5xx'] / ep.sent, barClass: 'k-5xx',
          sub: `${fmtInt(ep.classes['5xx'])} of ${fmtInt(ep.sent)} requests`,
        })), 'No 5xx responses.'), more(failing, TOP)]),
      card('No response', 'Requests that got no answer: timeouts, refused or reset connections.',
        [rankList(unreachable.slice(0, TOP).map((ep) => ({
          label: epLink(ep), value: fmtInt(ep.classes.err), bar: ep.classes.err / ep.sent, barClass: 'k-err',
          sub: `${fmtPct(ep.classes.err / ep.sent)} of ${fmtInt(ep.sent)} requests`,
        })), 'Every request got a response.'), more(unreachable, TOP)]),
      card('Never succeeded', 'No 2xx or 3xx at all: auth or validation stopped every request, so the logic behind them went untested.',
        [rankList(blocked.slice(0, TOP).map((ep) => ({ label: epLink(ep), value: '', sub: topCodes(ep) })),
          'Every endpoint succeeded at least once.'), more(blocked, TOP)]),
      card('Least predictable', 'Highest p99 / p50 ratio, with 20 responses or more: a long tail of slow calls.',
        [rankList(jittery.slice(0, TOP).map((ep) => ({
          label: epLink(ep), value: '×' + (ep.rt.p99 / ep.rt.p50).toFixed(1), bar: ep.rt.p99 / ep.rt.p50 / peakTail,
          sub: `p50 ${fmtMs(ep.rt.p50)} → p99 ${fmtMs(ep.rt.p99)}`,
        })), 'No endpoint has a tail of 2× its median.'), more(jittery, TOP)]),
      slowestRequests(eps),
      messageGroups(eps),
    ];
  }

  // slowestRequests lists single responses with their input, so a slow call can
  // be replayed. Requests without a response only took as long as the timeout.
  function slowestRequests(eps) {
    const TOP = 10;
    const rows = [];
    for (const ep of eps) for (const r of ep.rows) if (r.match && r.s !== 0) rows.push(r);
    rows.sort((a, b) => b.ms - a.ms);
    const title = 'Slowest requests';
    if (rows.length === 0) {
      return card(title, null, h('p', { class: 'muted' }, 'No reported result got a response.'), { wide: true });
    }
    const cell = (r, i) => (r.tr.cells[i] ? r.tr.cells[i].textContent : '');
    const clip = (text, n) => (text.length > n ? text.slice(0, n) + '…' : text);
    const table = h('div', { class: 'scroll' }, h('table', null,
      h('tr', null, ['Time', 'Status', 'Endpoint', 'Message', 'Input'].map((name) => h('th', null, name))),
      rows.slice(0, TOP).map((r) => h('tr', null,
        h('td', { class: 'n' }, fmtMs(r.ms)),
        h('td', { class: { '5xx': 'server-error', '4xx': 'client-error' }[r.cls] || 'success' }, statusLabel(r.s)),
        h('td', null, h('button', { type: 'button', class: 'linkish', title: 'Show this result in its endpoint', onclick: () => reveal(r) }, r.ep.name)),
        h('td', null, clip(rowMessage(r), 120) || h('span', { class: 'muted' }, '(empty body)')),
        h('td', null, h('code', null, clip(cell(r, 3), 160)))))));
    return card(title, 'Single responses among the reported results, slowest first. Requests without a response are in “No response”.',
      table, { wide: true });
  }

  // messageGroups counts the reported results by status and message, with ids
  // and numbers folded, so a thousand identical failures read as one line.
  function messageGroups(eps) {
    const groups = new Map();
    let total = 0;
    for (const ep of eps) {
      for (const r of ep.rows) {
        if (!r.match) continue;
        total++;
        const key = r.s + '\u0000' + messageKey(r);
        let g = groups.get(key);
        if (!g) groups.set(key, g = { s: r.s, cls: r.cls, example: rowMessage(r), count: 0, eps: new Map(), variants: new Set() });
        g.count++;
        g.eps.set(ep.i, ep);
        if (g.variants.size < 100) g.variants.add(rowMessage(r));
      }
    }
    const list = [...groups.values()].sort((a, b) => b.count - a.count || a.s - b.s);
    const TOP = 10;
    const hint = total
      ? `${fmtInt(list.length)} distinct messages in ${fmtInt(total)} reported results. Numbers and ids are folded together; pick a message to search for it.`
      : null;
    if (!total) {
      return card('Most frequent messages', null,
        h('p', { class: 'muted' }, 'No results reported: the status filter left them out. The counts above still cover every request.'), { wide: true });
    }
    const table = h('div', { class: 'scroll' }, h('table', { class: 'msgs' },
      h('tr', null, h('th', null, 'Status'), h('th', null, 'Message'), h('th', null, 'Results'), h('th', null, 'Endpoints')),
      list.slice(0, TOP).map((g) => {
        const eplist = [...g.eps.values()];
        return h('tr', null,
          h('td', { class: { '5xx': 'server-error', err: 'failed', '4xx': 'client-error' }[g.cls] || 'success' }, statusLabel(g.s)),
          h('td', null,
            h('button', {
              type: 'button', class: 'linkish', title: 'Search for this message',
              onclick: () => { state.q = g.example; state.re = false; apply(); ui.search.focus(); },
            }, g.example || '(empty body)'),
            g.variants.size > 1 ? h('span', { class: 'muted' }, ` (${g.variants.size >= 100 ? '100+' : g.variants.size} variants)`) : null),
          h('td', { class: 'n' }, `${fmtInt(g.count)} · ${fmtPct(g.count / total)}`),
          h('td', null, eplist.slice(0, 3).map((ep, k) => [k ? ', ' : '', epLink(ep)]),
            eplist.length > 3 ? h('span', { class: 'muted' }, ` +${eplist.length - 3}`) : null));
      })));
    return card('Most frequent messages', hint, [table, more(list, TOP)], { wide: true });
  }

  // ---- Summary decorations: a status bar and a latency bar per endpoint -----

  function decorateSummary() {
    const peakP95 = Math.max(0, ...endpoints.map((ep) => ep.rt.p95)) || 1;
    for (const ep of endpoints) {
      const cells = ep.row.cells;
      if (ep.sent > 0 && cells[3]) {
        cells[3].append(h('span', { class: 'minibar', 'aria-hidden': 'true' },
          CLASSES.filter((c) => ep.classes[c] > 0).map((c) =>
            h('i', { class: 'k-' + c, style: { flex: `${ep.classes[c]} 1 0px` } }))));
      }
      if (ep.rt.count > 0 && cells[4]) {
        cells[4].append(h('span', { class: 'latbar', 'aria-hidden': 'true', title: 'p95 against the slowest endpoint' },
          h('i', { style: { width: Math.max(1, (ep.rt.p95 / peakP95) * 100) + '%' } })));
      }
    }
  }

  // ---- Start ---------------------------------------------------------------

  const anchor = document.getElementById('summary');
  ui.scope = h('span', { class: 'muted' });
  ui.insightsBody = h('div');
  const insights = h('details', { class: 'fz-insights', open: true },
    h('summary', null, h('h2', null, 'Insights'), ui.scope), ui.insightsBody);
  insights.addEventListener('toggle', fitSegmentLabels);
  anchor.before(buildToolbar(), insights);

  enhanceSummaryHeads();
  for (const ep of endpoints) enhanceResultHeads(ep);
  decorateSummary();

  // Keep anchored sections clear of the sticky toolbar.
  const setBarHeight = () => document.documentElement.style.setProperty('--bar-h', ui.bar.offsetHeight + 'px');
  if (window.ResizeObserver) new ResizeObserver(setBarHeight).observe(ui.bar);
  setBarHeight();

  document.addEventListener('keydown', (e) => {
    if (e.key !== '/' || e.ctrlKey || e.metaKey || e.altKey) return;
    const t = e.target;
    if (t instanceof HTMLInputElement || t instanceof HTMLTextAreaElement || t instanceof HTMLSelectElement || (t && t.isContentEditable)) return;
    e.preventDefault();
    ui.search.focus();
    ui.search.select();
  });
  window.addEventListener('hashchange', () => {
    if (readHash()) { sortEndpoints(); apply(); }
  });

  readHash();
  if (state.sort !== 'spec' || !state.asc) sortEndpoints();
  apply();
})();
