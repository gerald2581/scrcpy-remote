const $ = (s, r = document) => r.querySelector(s);

async function api(path, opts) {
  const r = await fetch('/api/' + path, opts);
  return r.json();
}
function uuid() {
  try { if (window.crypto && crypto.randomUUID) return crypto.randomUUID(); } catch (e) {}
  return 'd-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 10);
}
function esc(s) {
  return String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}
function toast(msg, ok = true) {
  const t = document.createElement('div');
  t.className = 'toast ' + (ok ? 'ok' : 'err');
  t.textContent = msg;
  $('#toasts').appendChild(t);
  setTimeout(() => { t.classList.add('out'); setTimeout(() => t.remove(), 300); }, 3200);
}

let DEVS = [];
const PRESETS = ['balanced', 'fast', 'hd', 'lagfree'];
const PRESET_LABELS = { balanced: 'Balanced', fast: 'Fast', hd: 'HD', lagfree: 'Lag-free' };
const FIXED = 5555;

function presetSelect() {
  return `<select class="sel" data-preset>${PRESETS.map(p =>
    `<option value="${p}">${PRESET_LABELS[p] || p}</option>`).join('')}</select>`;
}

function rowHTML(d, i) {
  const persistent = Number(d.adbPort) === FIXED;
  const serial = `${esc(d.ip)}:${esc(d.adbPort || '—')}`;
  return `<div class="row" data-id="${esc(d.id)}" style="animation-delay:${i * 55}ms">
    <span class="led" data-led></span>
    <div class="row-id">
      <span class="row-name">${esc(d.name)}</span>
      <span class="row-serial">${serial} <span class="ping" data-ping></span></span>
      ${persistent ? '<span class="badge-persist">PERSISTENT :5555</span>' : ''}
    </div>
    ${presetSelect()}
    <button class="btn ghost" data-act="connect">Connect</button>
    <button class="btn primary" data-act="launch"><span>Launch</span></button>
    ${persistent ? '' : '<button class="btn warn" data-act="bootstrap">Bootstrap</button>'}
    <button class="btn del" data-act="del" title="Remove device">✕</button>
  </div>`;
}

async function withBusy(btn, led, label, fn) {
  const prev = btn.innerHTML;
  btn.disabled = true; btn.textContent = label;
  if (led) led.className = 'led busy';
  try { return await fn(); } finally { btn.disabled = false; btn.innerHTML = prev; }
}

function showBootstrapForm(row, d) {
  if ($('.bs-form', row.parentNode) && $('.bs-form', row.parentNode).previousElementSibling === row) {
    $('.bs-form', row.parentNode).remove(); return;
  }
  const form = document.createElement('div');
  form.className = 'bs-form';
  form.innerHTML = `
    <input data-f="wdPort" type="number" placeholder="WD connect port (e.g. 34171)" required>
    <input data-f="pairPort" type="number" placeholder="pairing port (first time only)">
    <input data-f="pairCode" placeholder="pairing code (first time only)">
    <button class="btn primary" data-go>Make persistent</button>
    <span class="bs-hint">From the phone's Wireless Debugging screen. Pairing only needed the first time.</span>`;
  row.after(form);
  $('[data-go]', form).onclick = async (e) => {
    const wdPort = Number($('[data-f="wdPort"]', form).value);
    if (!wdPort) { toast('WD connect port is required', false); return; }
    const payload = {
      id: d.id, name: d.name, ip: d.ip, wdPort,
      pairPort: Number($('[data-f="pairPort"]', form).value) || 0,
      pairCode: $('[data-f="pairCode"]', form).value || '',
    };
    const res = await withBusy(e.currentTarget, null, 'Bootstrapping…', () =>
      api('bootstrap', { method: 'POST', body: JSON.stringify(payload) }));
    if (res.ok) { toast('Persistent on :5555 — survives network changes'); refresh(); }
    else toast('Bootstrap failed: ' + (res.error || 'unknown'), false);
  };
}

function wire(row, d) {
  const led = $('[data-led]', row);
  $('[data-act="connect"]', row).onclick = async (e) => {
    const res = await withBusy(e.currentTarget, led, 'Connecting…', () =>
      api('connect', { method: 'POST', body: JSON.stringify(d) }));
    led.className = 'led ' + (res.ok ? 'ok' : 'err');
    toast(res.ok ? `Connected · ${d.ip}` : `Connect failed: ${res.error || res.data || 'unknown'}`, res.ok);
  };
  $('[data-act="launch"]', row).onclick = async (e) => {
    const preset = $('[data-preset]', row).value;
    const res = await withBusy(e.currentTarget, null, 'Launching…', () =>
      api('launch', { method: 'POST', body: JSON.stringify({ ip: d.ip, adbPort: d.adbPort, preset }) }));
    toast(res.ok ? `scrcpy launched · ${preset}` : `Launch failed: ${res.error || 'unknown'}`, res.ok);
  };
  const bs = $('[data-act="bootstrap"]', row);
  if (bs) bs.onclick = () => showBootstrapForm(row, d);
  const del = $('[data-act="del"]', row);
  if (del) del.onclick = async () => {
    if (!confirm(`Remove "${d.name}"?`)) return;
    const res = await api('devices?id=' + encodeURIComponent(d.id), { method: 'DELETE' });
    if (res.ok) { refresh(); toast('Device removed'); }
    else toast('Remove failed: ' + (res.error || 'unknown'), false);
  };
}

function applyStatus(list) {
  list.forEach(s => {
    const row = document.querySelector(`.row[data-id="${CSS.escape(s.id)}"]`);
    if (!row) return;
    const led = $('[data-led]', row);
    if (!led.classList.contains('busy')) led.className = 'led ' + (s.connected ? 'ok' : s.tsFound ? '' : 'err');
    let tag = $('.tag', row);
    if (!tag) { tag = document.createElement('span'); tag.className = 'tag'; $('.row-id', row).appendChild(tag); }
    tag.dataset.q = s.tsFound ? (s.tsRelay ? 'relay' : 'direct') : 'off';
    tag.textContent = s.tsFound ? (s.tsRelay ? 'relay' : 'direct') : 'offline';
  });
}
async function pollStatus() {
  try { const res = await api('status'); if (res.ok) applyStatus(res.data || []); } catch (e) {}
}

async function pingAll() {
  for (const d of DEVS) {
    try {
      const res = await api('ping?ip=' + encodeURIComponent(d.ip));
      const row = document.querySelector(`.row[data-id="${CSS.escape(d.id)}"]`);
      const el = row && row.querySelector('[data-ping]');
      if (!el) continue;
      if (res.ok) {
        el.textContent = res.data.ms + 'ms';
        el.dataset.q = res.data.ms < 80 ? 'good' : res.data.ms < 160 ? 'ok' : 'bad';
        el.title = 'Tailscale ' + res.data.via;
      } else { el.textContent = '—'; el.dataset.q = 'bad'; }
    } catch (e) {}
  }
}

async function refresh() {
  const res = await api('devices');
  const list = (res.data && res.data.devices) || [];
  DEVS = list;
  const wrap = $('#devices');
  wrap.innerHTML = list.map(rowHTML).join('');
  $('#count').textContent = list.length;
  $('#empty').style.display = list.length ? 'none' : 'block';
  [...wrap.querySelectorAll('.row')].forEach((row, i) => wire(row, list[i]));
  pollStatus();
}

$('#add').onsubmit = async (e) => {
  e.preventDefault();
  const f = e.target;
  const res = await api('devices', {
    method: 'POST',
    body: JSON.stringify({ id: uuid(), name: f.name.value, ip: f.ip.value, adbPort: Number(f.adbPort.value) || 0 }),
  });
  if (res.ok) { f.reset(); refresh(); toast('Device registered — click Bootstrap to make it persistent'); }
  else toast('Could not save: ' + (res.error || 'unknown'), false);
};

$('#host').textContent = location.host;
refresh();
setInterval(pollStatus, 5000);
setInterval(pingAll, 8000);
setTimeout(pingAll, 1500);
