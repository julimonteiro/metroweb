const f = document.getElementById('f'), err = document.getElementById('err'), go = document.getElementById('go');
document.getElementById('eye').onclick = () => {
  const p = document.getElementById('pw'); p.type = p.type === 'password' ? 'text' : 'password';
};
f.addEventListener('submit', async e => {
  e.preventDefault(); err.textContent = ''; go.disabled = true; go.textContent = 'Signing in…';
  try {
    const r = await fetch('/login', { method: 'POST', body: new URLSearchParams(new FormData(f)) });
    if (r.ok) { location.replace('/'); return; }
    const d = await r.json().catch(() => ({}));
    err.textContent = d.error || 'Could not sign in.';
  } catch { err.textContent = 'Cannot reach the server.'; }
  go.disabled = false; go.textContent = 'Sign in';
});
