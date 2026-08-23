async function refresh() {
  const status = await fetch('/api/status').then(r => r.json());
  const rigs = document.getElementById('rigs');
  rigs.innerHTML = '';
  for (const c of status.cranes || []) {
    const d = document.createElement('div');
    d.className = 'rig-card';
    d.textContent = `${c.RigID} mode=${c.Mode} az=${c.Azimuth} moment=${c.MomentPct}%`;
    rigs.appendChild(d);
  }
  const alarms = await fetch('/api/alarms').then(r => r.json());
  const ul = document.getElementById('alarms');
  ul.innerHTML = '';
  for (const a of alarms.active || []) {
    const li = document.createElement('li');
    li.textContent = `${a.RigID}: ${a.Code}`;
    ul.appendChild(li);
  }
}
document.getElementById('btn-refresh').onclick = refresh;
refresh();
