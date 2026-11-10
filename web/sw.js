// metroweb service worker: songs saved for offline play, and the app shell so the
// app still opens without a connection. Everything else goes straight to the network.
const OFFLINE = 'mw-offline-v1', SHELL = 'mw-shell-v1';
const STREAM = /^\/api\/stream\/([A-Za-z0-9_-]{11})$/;

self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', e => e.waitUntil((async () => {
  for (const k of await caches.keys()) if (k !== OFFLINE && k !== SHELL) await caches.delete(k);
  await self.clients.claim();
})()));

// answer a (possibly ranged) request from a stored song
async function fromStored(req, stored) {
  const blob = await stored.blob(), size = blob.size;
  const type = stored.headers.get('Content-Type') || 'audio/mp4';
  const range = req.headers.get('range');
  const base = { 'Content-Type': type, 'Accept-Ranges': 'bytes' };
  if (!range) return new Response(blob, { status: 200, headers: { ...base, 'Content-Length': String(size) } });
  const m = /bytes=(\d*)-(\d*)/.exec(range);
  if (!m) return new Response(blob, { status: 200, headers: { ...base, 'Content-Length': String(size) } });
  let start, end;
  if (m[1] === '') { start = Math.max(0, size - Number(m[2])); end = size - 1; }
  else { start = Number(m[1]); end = m[2] === '' ? size - 1 : Math.min(Number(m[2]), size - 1); }
  if (start >= size || start > end) return new Response(null, { status: 416, headers: { 'Content-Range': `bytes */${size}` } });
  return new Response(blob.slice(start, end + 1), { status: 206, headers: { ...base, 'Content-Range': `bytes ${start}-${end}/${size}`, 'Content-Length': String(end - start + 1) } });
}

self.addEventListener('fetch', e => {
  const req = e.request;
  if (req.method !== 'GET') return;
  const url = new URL(req.url);
  if (url.origin !== location.origin) return;
  const m = STREAM.exec(url.pathname);
  if (m) {
    e.respondWith((async () => {
      const stored = await (await caches.open(OFFLINE)).match('/offline/' + m[1]);
      if (stored) return fromStored(req, stored);
      return fetch(req);
    })());
    return;
  }
  if (url.pathname.startsWith('/api/') || url.pathname === '/login' || url.pathname === '/logout') return;
  // app shell: network first, the stored copy when offline
  e.respondWith((async () => {
    try {
      const resp = await fetch(req);
      if (resp.ok && !resp.redirected && resp.type === 'basic') (await caches.open(SHELL)).put(req, resp.clone());
      return resp;
    } catch (err) {
      const hit = await (await caches.open(SHELL)).match(req, { ignoreSearch: true });
      if (hit) return hit;
      if (req.mode === 'navigate') { const idx = await (await caches.open(SHELL)).match('/'); if (idx) return idx; }
      throw err;
    }
  })());
});
