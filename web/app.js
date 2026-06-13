const $ = (s, r = document) => r.querySelector(s);

async function api(path, opts) {
  const r = await fetch('/api/' + path, opts);
  return r.json();
}
function esc(s) {
  return String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}
function toast(msg, ok = true) {
  const t = document.createElement('div');
  t.className = 'toast ' + (ok ? 'ok' : 'err');
  t.textContent = msg;
  $('#toasts').appendChild(t);
  setTimeout(() => { t.classList.add('out'); setTimeout(() => t.remove(), 300); }, 2600);
}

const PRESETS = ['balanced', 'fast', 'hd'];

function rowHTML(d, i) {
  const opts = PRESETS.map(p => `<option value="${p}">${p[0].toUpperCase() + p.slice(1)}</option>`).join('');
  return `<div class="row" data-id="${esc(d.id)}" style="animation-delay:${i * 55}ms">
    <span class="led" data-led></span>
    <div class="row-id">
      <span class="row-name">${esc(d.name)}</span>
      <span class="row-serial">${esc(d.ip)}:${esc(d.adbPort)}</span>
    </div>
    <select class="sel" data-preset>${opts}</select>
    <button class="btn ghost" data-act="connect">Connect</button>
    <button class="btn primary" data-act="launch"><span>Launch</span></button>
  </div>`;
}

async function withBusy(btn, led, label, fn) {
  const prev = btn.innerHTML;
  btn.disabled = true; btn.textContent = label;
  if (led) { led.className = 'led busy'; }
  try { return await fn(); }
  finally { btn.disabled = false; btn.innerHTML = prev; }
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
}

async function refresh() {
  const res = await api('devices');
  const list = (res.data && res.data.devices) || [];
  const wrap = $('#devices');
  wrap.innerHTML = list.map(rowHTML).join('');
  $('#count').textContent = list.length;
  $('#empty').style.display = list.length ? 'none' : 'block';
  [...wrap.children].forEach((row, i) => wire(row, list[i]));
  pollStatus();
}

function applyStatus(list) {
  list.forEach(s => {
    const row = document.querySelector(`.row[data-id="${CSS.escape(s.id)}"]`);
    if (!row) return;
    const led = row.querySelector('[data-led]');
    if (!led.classList.contains('busy')) {
      led.className = 'led ' + (s.connected ? 'ok' : s.tsFound ? '' : 'err');
    }
    let tag = row.querySelector('.tag');
    if (!tag) { tag = document.createElement('span'); tag.className = 'tag'; row.querySelector('.row-id').appendChild(tag); }
    tag.dataset.q = s.tsFound ? (s.tsRelay ? 'relay' : 'direct') : 'off';
    tag.textContent = s.tsFound ? (s.tsRelay ? 'relay' : 'direct') : 'offline';
  });
}

async function pollStatus() {
  try {
    const res = await api('status');
    if (res.ok) applyStatus(res.data || []);
  } catch (e) { /* server momentarily unavailable; ignore */ }
}

$('#add').onsubmit = async (e) => {
  e.preventDefault();
  const f = e.target;
  const res = await api('devices', {
    method: 'POST',
    body: JSON.stringify({ id: crypto.randomUUID(), name: f.name.value, ip: f.ip.value, adbPort: Number(f.adbPort.value) }),
  });
  if (res.ok) { f.reset(); refresh(); toast('Device registered'); }
  else toast('Could not save: ' + (res.error || 'unknown'), false);
};

$('#host').textContent = location.host;
refresh();
setInterval(pollStatus, 5000);
