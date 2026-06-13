async function api(path, opts) {
  const r = await fetch('/api/' + path, opts);
  return r.json();
}
function presetSelect(id) {
  return `<select data-preset="${id}">
    <option value="balanced">Balanced</option>
    <option value="fast">Fast</option>
    <option value="hd">HD</option></select>`;
}
async function refresh() {
  const res = await api('devices');
  const wrap = document.getElementById('devices');
  wrap.innerHTML = '';
  (res.data.devices || []).forEach(d => {
    const el = document.createElement('div');
    el.className = 'device';
    el.innerHTML = `<b>${d.name}</b> <code>${d.ip}:${d.adbPort}</code>
      ${presetSelect(d.id)}
      <button data-act="connect">Connect</button>
      <button data-act="launch">Launch</button>`;
    el.querySelector('[data-act="connect"]').onclick = () =>
      api('connect', { method: 'POST', body: JSON.stringify(d) });
    el.querySelector('[data-act="launch"]').onclick = () =>
      api('launch', { method: 'POST', body: JSON.stringify({ ip: d.ip, adbPort: d.adbPort, preset: el.querySelector('select').value }) });
    wrap.appendChild(el);
  });
}
document.getElementById('add').onsubmit = async (e) => {
  e.preventDefault();
  const f = e.target;
  await api('devices', { method: 'POST', body: JSON.stringify({
    id: crypto.randomUUID(), name: f.name.value, ip: f.ip.value, adbPort: Number(f.adbPort.value) }) });
  f.reset(); refresh();
};
refresh();
