// Desktop extras: sortable columns, multi-select, keyboard moves, hover helpers,
// the queue side panel, and the sidebar playlists. Loaded after app.js; it uses
// app.js globals at run time and talks to it through "mw:*" events.

const DESKTOP = matchMedia('(min-width: 900px)');
const isTyping = t => !!(t && t.closest && t.closest('input, textarea, select, [contenteditable]'));

// ---- sortable columns: click a header to sort, again for the other way, a third time to restore the order
(() => {
  const state = new WeakMap(), by = {
    title: (a, b) => (a.title || '').localeCompare(b.title || ''),
    artist: (a, b) => (a.artist || '').localeCompare(b.artist || '') || (a.title || '').localeCompare(b.title || ''),
    album: (a, b) => (a.album || '').localeCompare(b.album || '') || (a.title || '').localeCompare(b.title || ''),
    duration: (a, b) => secs(a.duration) - secs(b.duration),
  };
  document.addEventListener('click', e => {
    const b = e.target.closest('.thead [data-sortk]'); if (!b) return;
    const table = b.closest('.ttable'), L = V.lists[table.dataset.l]; if (!L) return;
    const cur = state.get(table) || {}, key = b.dataset.sortk;
    const dir = cur.k === key ? (cur.dir === 'asc' ? 'desc' : null) : 'asc';
    if (!L.orig) L.orig = L.items.slice();
    let items = L.orig.slice();
    if (dir) { items.sort(by[key]); if (dir === 'desc') items.reverse(); }
    L.items.length = 0; L.items.push(...items);
    state.set(table, dir ? { k: key, dir } : {});
    table.dataset.sorted = dir ? '1' : '';
    table.querySelectorAll('.thead button').forEach(x => { x.classList.toggle('is-asc', !!dir && x === b && dir === 'asc'); x.classList.toggle('is-desc', !!dir && x === b && dir === 'desc'); });
    table.querySelector('.trows').innerHTML = songRows(L.items, table.dataset.l, { cols: true, noAlbum: table.classList.contains('no-alb'), reorder: table.classList.contains('can-reorder') });
    clearSelection();
  });
})();

// ---- selecting several songs: Ctrl/Cmd+click toggles, Shift+click selects a range, Ctrl+A all
const sel = { l: null, anchor: -1, idx: new Set() };
function selectedTracks() { const L = V.lists[sel.l]; return L ? [...sel.idx].sort((a, b) => a - b).map(i => L.items[i]).filter(Boolean) : []; }
function clearSelection() {
  sel.idx.clear(); sel.anchor = -1; sel.l = null;
  $$('#view .row.is-sel').forEach(r => r.classList.remove('is-sel'));
  paintSelBar();
}
function paintSelRows() { $$('#view .row[data-id]').forEach(r => r.classList.toggle('is-sel', r.dataset.l === sel.l && sel.idx.has(+r.dataset.i))); }
function paintSelBar() {
  let bar = $('#selbar');
  if (!sel.idx.size) { if (bar) bar.remove(); return; }
  if (!bar) { bar = document.createElement('div'); bar.id = 'selbar'; bar.setAttribute('role', 'toolbar'); bar.setAttribute('aria-label', 'Selected songs'); document.body.appendChild(bar); }
  const inPl = routeName === 'playlist' && V.playlistId && V.playlistId !== 'liked';
  bar.innerHTML = `<b>${sel.idx.size} selected</b>
    <button data-sa="play">${ic('play', 18)}Play</button><button data-sa="next">${ic('playnext', 18)}Play next</button><button data-sa="queue">${ic('addq', 18)}Queue</button>
    <button data-sa="pl">${ic('lib', 18)}Playlist</button><button data-sa="dl">${ic('dl', 18)}Download</button><button data-sa="off">${ic('cloud', 18)}Offline</button>
    ${inPl ? `<button data-sa="rm">${ic('del', 18)}Remove</button>` : ''}<button class="x" data-sa="clear" aria-label="Clear selection">${ic('close', 18)}</button>`;
}
document.addEventListener('click', e => {
  const sa = e.target.closest('#selbar [data-sa]');
  if (sa) {
    const tracks = selectedTracks().map(T), a = sa.dataset.sa;
    if (a === 'clear') { clearSelection(); return; }
    if (a === 'play') playList(tracks, 0, true);
    else if (a === 'next') { queue.splice(idx + 1, 0, ...tracks); persist(); renderQueueList(); snack(`${tracks.length} songs playing next`); }
    else if (a === 'queue') { queue.push(...tracks); persist(); renderQueueList(); snack(`${tracks.length} songs added to the queue`); }
    else if (a === 'pl') playlistPicker(tracks, `${tracks.length} songs`);
    else if (a === 'dl') downloadAll(tracks);
    else if (a === 'off') saveOffline(tracks);
    else if (a === 'rm') {
      confirmSheet(`Remove ${tracks.length} songs?`, 'They are removed from this playlist only.', 'Remove', async () => {
        for (const t of tracks) await send('DELETE', `/api/playlists/${V.playlistId}/tracks/${t.id}`);
        await refreshPlaylists(); route();
      });
      return;
    }
    clearSelection();
    return;
  }
  const row = e.target.closest('#view .row[data-id]');
  if (!row || e.target.closest('[data-more], [data-grip]')) return;
  const mod = e.ctrlKey || e.metaKey, range = e.shiftKey;
  if (!mod && !range && !sel.idx.size) return;          // a plain click plays, as before
  e.preventDefault(); e.stopPropagation();
  const l = row.dataset.l, i = +row.dataset.i;
  if (sel.l !== l) { sel.idx.clear(); sel.anchor = -1; sel.l = l; }
  if (range && sel.anchor >= 0) { sel.idx.clear(); for (let k = Math.min(sel.anchor, i); k <= Math.max(sel.anchor, i); k++) sel.idx.add(k); }
  else { sel.idx.has(i) ? sel.idx.delete(i) : sel.idx.add(i); sel.anchor = i; }
  if (!sel.idx.size) sel.l = null;
  paintSelRows(); paintSelBar();
}, true);
document.addEventListener('mw:route', clearSelection);
document.addEventListener('input', e => { if (e.target.id === 'tf') clearSelection(); });

// ---- keyboard: move between songs, Ctrl+F, Ctrl+A, Delete, Esc
document.addEventListener('keydown', e => {
  if (e.key === 'Escape' && sel.idx.size) { e.stopPropagation(); clearSelection(); return; }
  const row = e.target.closest && e.target.closest('.row[data-id]');
  if (row && ['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(e.key) && !e.metaKey && !e.ctrlKey && !e.altKey) {
    const rows = [...row.parentElement.querySelectorAll('.row[data-id]')], i = rows.indexOf(row);
    const to = e.key === 'ArrowDown' ? rows[i + 1] : e.key === 'ArrowUp' ? rows[i - 1] : e.key === 'Home' ? rows[0] : rows[rows.length - 1];
    e.preventDefault(); e.stopPropagation();
    if (to) { to.focus(); to.scrollIntoView({ block: 'nearest' }); }
    return;
  }
  if (row && (e.key === 'Delete' || e.key === 'Backspace') && row.dataset.l === 'queue') {
    e.preventDefault(); removeFromQueue(+row.dataset.i);
    return;
  }
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey) {
    if (e.key === 'f' || e.key === 'F') {
      const f = $('#tf') || $('#q');
      e.preventDefault(); if (f) { f.focus(); f.select(); } else { go('/search'); setTimeout(() => { const q = $('#q'); if (q) q.focus(); }, 120); }
    } else if ((e.key === 'a' || e.key === 'A') && !isTyping(e.target)) {
      const first = $('#view .row[data-id]'); if (!first) return;
      e.preventDefault(); const l = first.dataset.l, L = V.lists[l]; if (!L) return;
      sel.l = l; sel.idx = new Set(L.items.map((_, i) => i)); sel.anchor = 0; paintSelRows(); paintSelBar();
    }
  }
}, true);

// ---- the seek bar tells you the time under the pointer; the wheel changes the volume
(() => {
  const bar = $('#dBar'), vol = $('#vol'), volic = $('.volic');
  if (!bar) return;
  const tip = document.createElement('div'); tip.id = 'bartip'; tip.hidden = true; bar.parentElement.appendChild(tip);
  bar.addEventListener('mousemove', e => {
    const r = bar.getBoundingClientRect(), f = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width));
    tip.textContent = time((audio.duration || 0) * f); tip.style.left = `${bar.offsetLeft + f * r.width}px`; tip.hidden = !isFinite(audio.duration);
  });
  bar.addEventListener('mouseleave', () => { tip.hidden = true; });
  const wheel = e => { e.preventDefault(); vol.value = Math.max(0, Math.min(100, +vol.value + (e.deltaY < 0 ? 5 : -5))); vol.dispatchEvent(new Event('input')); };
  [vol, volic].forEach(el => el && el.addEventListener('wheel', wheel, { passive: false }));
})();

// ---- hover play button on cards: starts Liked songs, a mix, a playlist or a song without opening it
document.addEventListener('click', async e => {
  const b = e.target.closest('[data-hpcard]'); if (!b) return;
  e.preventDefault(); e.stopPropagation();
  const r = itemOf(b.closest('[data-card]')); if (!r || !r.it) return;
  const it = r.it;
  try {
    if (it.type === 'song') playNow(it);
    else if (it.mixId) { const m = (await cached(`/api/mixes?tz=${-new Date().getTimezoneOffset()}`, 10 * 60 * 1000)).find(x => x.id === it.mixId); if (m) playList(m.tracks.map(T), 0); }
    else if (it.local === 'liked') playShuffled(liked.map(T));
    else if (it.local) { const p = await api('/api/playlists/' + it.local); if (p.tracks.length) playList(p.tracks.map(T), 0); }
  } catch (err) { snack(err.message); }
}, true);

// ---- the queue as a side panel (it stays open while you browse)
(() => {
  let panel = null;
  function build() {
    panel = document.createElement('aside'); panel.id = 'qpanel'; panel.setAttribute('aria-label', 'Queue');
    panel.innerHTML = `<header><h2>Queue</h2><div class="qp-actions">${queueButtons()}<button class="iconbtn" data-qpclose aria-label="Close queue">${ic('close')}</button></div></header>
      <div class="qp-body"><div class="qsum"></div><div class="qlist"></div></div>`;
    $('#app').appendChild(panel);
    panel.addEventListener('click', e => { if (e.target.closest('[data-qpclose]')) toggleQueuePanel(false); });
  }
  window.toggleQueuePanel = force => {
    if (!DESKTOP.matches) { go('/queue'); return; }
    const open = force === undefined ? !document.body.classList.contains('q-open') : force;
    if (open && !panel) build();
    document.body.classList.toggle('q-open', open);
    store.set('mw.qOpen', open);
    const btn = $('#dQueue'); if (btn) { btn.classList.toggle('on', open); btn.setAttribute('aria-pressed', String(open)); }
    if (open) { showPlayed = false; renderQueueList(true); }
  };
  document.addEventListener('keydown', e => { if ((e.key === 'q' || e.key === 'Q') && !isTyping(e.target) && !e.ctrlKey && !e.metaKey && !e.altKey && DESKTOP.matches) window.toggleQueuePanel(); });
  if (store.get('mw.qOpen', false) && DESKTOP.matches) window.toggleQueuePanel(true);
  DESKTOP.addEventListener('change', () => { if (!DESKTOP.matches) document.body.classList.remove('q-open'); else if (store.get('mw.qOpen', false)) window.toggleQueuePanel(true); });
})();

// ---- sidebar: your playlists (drop songs on them), and drag the edge to resize
(() => {
  const side = $('#side'); if (!side) return;
  const box = document.createElement('div'); box.id = 'sidePl';
  side.insertBefore(box, side.querySelector('.grow'));
  function paint() {
    const items = [{ id: 'liked', name: 'Liked songs', count: liked.length, icon: 'heart' }, ...playlists.filter(p => p.id !== 'liked').map(p => ({ id: p.id, name: p.name, count: p.count, icon: 'music' }))];
    box.innerHTML = `<h4>Playlists</h4>` + items.map(p => `<a class="sp" href="#/playlist/${esc(p.id)}" data-pid="${esc(p.id)}" data-pn="${esc(p.name)}" title="${esc(p.name)}">${ic(p.icon, 18)}<span class="lbl">${esc(p.name)}</span><small>${p.count}</small></a>`).join('') +
      `<button class="sp new" data-newpl title="New playlist">${ic('add', 18)}<span class="lbl">New playlist</span></button>`;
    markActive();
  }
  const markActive = () => $$('#sidePl .sp[data-pid]').forEach(a => a.classList.toggle('on', routeName === 'playlist' && location.hash === `#/playlist/${a.dataset.pid}`));
  document.addEventListener('mw:playlists', paint); document.addEventListener('mw:route', markActive);
  document.addEventListener('mw:now', () => { if (liked && $('#sidePl .sp[data-pid="liked"] small')) $('#sidePl .sp[data-pid="liked"] small').textContent = liked.length; });
  setTimeout(paint, 0);

  // songs dragged from a list land on a playlist
  let dragged = [];
  document.addEventListener('dragstart', e => {
    const row = e.target.closest && e.target.closest('#view .row[data-id]'); if (!row) return;
    const L = V.lists[row.dataset.l], it = L && L.items[+row.dataset.i];
    dragged = row.classList.contains('is-sel') && sel.idx.size ? selectedTracks() : it ? [it] : [];
    e.dataTransfer.setData('text/plain', dragged.map(t => t.title).join(', '));
  });
  document.addEventListener('dragend', () => { dragged = []; $$('#sidePl .sp.over').forEach(x => x.classList.remove('over')); });
  box.addEventListener('dragover', e => { const a = e.target.closest('.sp[data-pid]'); if (a && dragged.length) { e.preventDefault(); $$('#sidePl .sp.over').forEach(x => x.classList.toggle('over', x === a)); } });
  box.addEventListener('dragleave', e => { if (!box.contains(e.relatedTarget)) $$('#sidePl .sp.over').forEach(x => x.classList.remove('over')); });
  box.addEventListener('drop', e => {
    const a = e.target.closest('.sp[data-pid]'); if (!a || !dragged.length) return;
    e.preventDefault(); e.stopPropagation(); addToPlaylist(a.dataset.pid, dragged, a.dataset.pn); clearSelection(); a.classList.remove('over');
  });

  // resize the expanded sidebar by dragging its right edge
  const grip = document.createElement('div'); grip.id = 'sideResize'; grip.setAttribute('role', 'separator'); grip.setAttribute('aria-orientation', 'vertical'); grip.title = 'Drag to resize';
  side.appendChild(grip);
  const setW = w => document.documentElement.style.setProperty('--side-open-w', Math.max(200, Math.min(420, Math.round(w))) + 'px');
  const saved = +store.get('mw.sideW', 0); if (saved) setW(saved);
  grip.addEventListener('pointerdown', e => {
    if (!document.body.classList.contains('side-open')) return;
    e.preventDefault(); grip.setPointerCapture(e.pointerId); document.body.classList.add('side-resizing');
    const move = ev => setW(ev.clientX);
    const up = () => { grip.removeEventListener('pointermove', move); grip.removeEventListener('pointerup', up); document.body.classList.remove('side-resizing'); store.set('mw.sideW', parseInt(getComputedStyle(document.documentElement).getPropertyValue('--side-open-w')) || 240); };
    grip.addEventListener('pointermove', move); grip.addEventListener('pointerup', up);
  });
})();

// ---- reorder a playlist by dragging a row's grip (only while the list is in its own order)
document.addEventListener('pointerdown', e => {
  const grip = e.target.closest && e.target.closest('.ttable [data-rgrip]'); if (!grip) return;
  const table = grip.closest('.ttable'), f = $('#tf');
  if (table.dataset.sorted || (f && f.value.trim()) || !V.onReorder || !V.playlistId) { snack('Clear the sort and search to reorder'); e.preventDefault(); return; }
  dragRows(e, grip, async (from, to) => {
    if (from === to) return;
    const L = V.lists[table.dataset.l], t = L && L.items[from]; if (!t) return;
    try { await send('POST', `/api/playlists/${V.playlistId}/move`, { trackId: t.id, to }); V.onReorder(from, to); }
    catch (err) { snack(err.message); }
  });
});
document.addEventListener('click', e => { if (e.target.closest && e.target.closest('[data-rgrip]')) { e.stopPropagation(); e.preventDefault(); } }, true);

// ---- install as an app (Chrome, Edge...)
(() => {
  let evt = null;
  window.addEventListener('beforeinstallprompt', e => { e.preventDefault(); evt = e; document.dispatchEvent(new Event('mw:install')); });
  window.addEventListener('appinstalled', () => { evt = null; snack('metroweb installed'); });
  window.canInstallApp = () => !!evt;
  window.installApp = async () => { if (!evt) return; evt.prompt(); try { await evt.userChoice; } catch {} evt = null; };
})();

// ---- pop out a small player that stays on top of other windows (Document Picture-in-Picture)
(() => {
  if (!('documentPictureInPicture' in window)) return;
  IC.pip = 'M19 11h-8v6h8v-6zm4 8V4.98C23 3.88 22.1 3 21 3H3c-1.1 0-2 .88-2 1.98V19c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2zm-2 .02H3V4.97h18v14.05z';
  let pw = null;
  const btn = document.createElement('button');
  btn.className = 'iconbtn'; btn.id = 'dPip'; btn.title = 'Pop out mini player'; btn.setAttribute('aria-label', 'Pop out mini player'); btn.innerHTML = ic('pip');
  const anchor = $('#dSleep'); if (anchor) anchor.parentElement.insertBefore(btn, anchor);

  function paint() {
    if (!pw) return;
    const d = pw.document, t = cur();
    d.querySelector('.pip-art').src = t ? img(t.thumb) : '';
    d.querySelector('.pip-t').textContent = t ? t.title : 'Nothing playing';
    d.querySelector('.pip-a').textContent = t ? t.artist || '' : '';
    d.querySelector('[data-pp="play"]').innerHTML = ic(playing() ? 'pause' : 'play', 28);
    const fav = t && likedIds.has(t.id), h = d.querySelector('[data-pp="like"]');
    h.innerHTML = ic(fav ? 'heart' : 'heart0', 22); h.style.color = fav ? 'var(--accent)' : '';
  }
  const tick = () => { if (!pw || !audio.duration) return; const b = pw.document.querySelector('.pip-bar'); if (b && pw.document.activeElement !== b) b.value = Math.round(audio.currentTime / audio.duration * 1000); };
  document.addEventListener('mw:now', paint);
  audio.addEventListener('timeupdate', tick);

  async function open() {
    if (pw) { pw.close(); return; }
    try { pw = await documentPictureInPicture.requestWindow({ width: 380, height: 150 }); } catch (e) { snack('Could not open the mini player'); return; }
    const d = pw.document;
    for (const ss of document.styleSheets) {           // the pop-out looks like the app
      try { const s = d.createElement('style'); s.textContent = [...ss.cssRules].map(r => r.cssText).join('\n'); d.head.appendChild(s); }
      catch { if (ss.href) { const l = d.createElement('link'); l.rel = 'stylesheet'; l.href = ss.href; d.head.appendChild(l); } }
    }
    d.documentElement.dataset.theme = document.documentElement.dataset.theme; d.documentElement.style.cssText = document.documentElement.style.cssText;
    d.body.className = 'pip';
    d.body.innerHTML = `<div class="pip-wrap"><img class="pip-art" alt=""><div class="pip-main"><div class="pip-t"></div><div class="pip-a"></div>
      <input type="range" class="pip-bar" min="0" max="1000" value="0" aria-label="Seek">
      <div class="pip-ctl"><button data-pp="like" aria-label="Like"></button><button data-pp="prev" aria-label="Previous">${ic('prev', 24)}</button><button data-pp="play" class="main" aria-label="Play or pause"></button><button data-pp="next" aria-label="Next">${ic('next', 24)}</button><button data-pp="back" aria-label="Back to metroweb">${ic('expand', 20)}</button></div></div></div>`;
    d.body.addEventListener('click', e => {
      const b = e.target.closest('[data-pp]'); if (!b) return;
      const a = b.dataset.pp;
      if (a === 'play') toggle(); else if (a === 'next') next(); else if (a === 'prev') prev();
      else if (a === 'like') { const t = cur(); if (t) toggleLike(t); } else if (a === 'back') { window.focus(); pw.close(); }
    });
    d.querySelector('.pip-bar').addEventListener('input', e => { if (audio.duration) audio.currentTime = audio.duration * (+e.target.value / 1000); });
    pw.addEventListener('pagehide', () => { pw = null; btn.classList.remove('on'); });
    btn.classList.add('on'); paint(); tick();
  }
  btn.addEventListener('click', open);
  window.openMiniPlayer = open;
})();
