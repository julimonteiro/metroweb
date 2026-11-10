// Command palette: Ctrl/Cmd+K from anywhere. Type to run an action, open a
// playlist, or find a song, artist or album. Loaded after app.js.

(() => {
  let root = null, input = null, list = null, items = [], active = 0, seq = 0, remote = [], deb = 0, searching = false;

  // what you can do. keys: extra words people might type
  const actions = () => [
    { n: 'Play / pause', k: 'toggle resume stop', i: 'play', h: 'Space', run: () => toggle() },
    { n: 'Next song', k: 'skip', i: 'next', h: 'Shift →', run: () => next() },
    { n: 'Previous song', k: 'back', i: 'prev', h: 'Shift ←', run: () => prev() },
    { n: shuffle ? 'Turn shuffle off' : 'Turn shuffle on', k: 'random smart', i: 'shuf', h: 'S', run: () => setShuffle(!shuffle) },
    { n: 'Change repeat mode', k: 'loop', i: 'rep', h: 'R', run: () => $('#dRep').click() },
    { n: 'Like or unlike the current song', k: 'favourite favorite heart', i: 'heart', h: 'L', run: () => cur() && toggleLike(cur()) },
    { n: 'Show the queue', k: 'up next panel', i: 'queue', h: 'Q', run: () => (window.toggleQueuePanel ? window.toggleQueuePanel() : go('/queue')) },
    { n: 'Play Liked songs, shuffled', k: 'favourites smart', i: 'heart', run: () => liked.length ? playShuffled(liked.map(T)) : snack('No liked songs yet') },
    ...(window.openMiniPlayer ? [{ n: 'Pop out the mini player', k: 'picture in picture floating window pip', i: 'pip', run: () => window.openMiniPlayer() }] : []),
    ...(window.canInstallApp && window.canInstallApp() ? [{ n: 'Install metroweb as an app', k: 'pwa desktop', i: 'dl', run: () => window.installApp() }] : []),
    { n: 'Sleep timer…', k: 'stop later night', i: 'moon', run: () => sleepSheet() },
    { n: 'Playback speed…', k: 'faster slower tempo', i: 'speed', run: () => speedSheet() },
    { n: 'Mute or unmute', k: 'silence volume', i: 'vol', h: 'M', run: () => { const v = $('.volic'); if (v) v.click(); } },
    { n: 'Go to Home', k: 'start', i: 'home', run: () => go('/') },
    { n: 'Go to Search', k: 'find', i: 'search', h: '/', run: () => go('/search') },
    { n: 'Go to Library', k: 'playlists', i: 'lib', run: () => go('/library') },
    { n: 'Go to Stats', k: 'listening history charts', i: 'stats', run: () => go('/stats') },
    { n: 'Go to Offline songs', k: 'downloaded saved cloud', i: 'cloud', run: () => go('/offline') },
    { n: 'Your year in music', k: 'wrapped recap review', i: 'stats', run: () => go('/wrapped') },
    { n: 'Server status', k: 'health yt-dlp backup disk', i: 'stats', run: () => go('/status') },
    { n: 'Settings…', k: 'preferences theme appearance fade leveling', i: 'speed', run: () => settingsSheet() },
    { n: 'Switch theme (dark, light, system)', k: 'appearance colour color', i: 'speed', run: () => { themePref = themePref === 'dark' ? 'light' : themePref === 'light' ? 'system' : 'dark'; store.set('mw.theme', themePref); applyTheme(); snack(`Theme: ${themePref}`); } },
    { n: 'New playlist…', k: 'create', i: 'add', run: () => promptSheet('New playlist', '', 'Create', async name => { await send('POST', '/api/playlists', { name }); await refreshPlaylists(); snack('Playlist created'); if (routeName === 'library') route(); }) },
    { n: 'Import a playlist…', k: 'spotify youtube csv', i: 'dl', run: () => importSheet() },
    { n: 'Keyboard shortcuts', k: 'help keys', i: 'queue', h: '?', run: () => shortcutsSheet() },
    { n: 'Collapse or expand the sidebar', k: 'menu', i: 'menu', run: () => setSide(!document.body.classList.contains('side-open')) },
    { n: 'Account and password…', k: 'sign out logout user', i: 'user', run: () => accountSheet() },
  ];

  const norm = s => (s || '').toLowerCase();
  const matches = (text, q) => { const t = norm(text); return q.split(/\s+/).filter(Boolean).every(w => t.includes(w)); };
  const tag = (it, kind) => ({ kind, it });

  function build() {
    root = document.createElement('div'); root.id = 'cmdk'; root.setAttribute('role', 'dialog'); root.setAttribute('aria-modal', 'true'); root.setAttribute('aria-label', 'Command palette');
    root.innerHTML = `<div class="ck-box"><label class="ck-in">${ic('search', 22)}<input type="text" placeholder="Type a command, or search songs, artists, playlists…" autocomplete="off" spellcheck="false" aria-label="Command palette"></label>
      <div class="ck-list" role="listbox"></div>
      <div class="ck-foot"><span><kbd>↑</kbd><kbd>↓</kbd> move</span><span><kbd>Enter</kbd> run</span><span><kbd>Shift</kbd>+<kbd>Enter</kbd> add to queue</span><span><kbd>Esc</kbd> close</span></div></div>`;
    document.body.appendChild(root);
    input = root.querySelector('input'); list = root.querySelector('.ck-list');
    input.addEventListener('input', () => { active = 0; refresh(); search(); });
    root.addEventListener('mousedown', e => { if (e.target === root) close(); });
    list.addEventListener('mousemove', e => { const b = e.target.closest('.ck-item'); if (b && +b.dataset.i !== active) { active = +b.dataset.i; mark(); } });
    list.addEventListener('click', e => { const b = e.target.closest('.ck-item'); if (b) run(+b.dataset.i, e.shiftKey); });
  }

  // everything that matches right now: actions, playlists, your own songs, then the live search results
  function collect() {
    const q = norm(input.value).trim(), out = [];
    const acts = actions().filter(a => !q || matches(a.n + ' ' + a.k, q));
    const pls = [{ id: 'liked', name: 'Liked songs' }, ...playlists.filter(p => p.id !== 'liked')].filter(p => q && matches(p.name, q));
    const mine = q.length > 1 ? [...liked, ...recent].filter((t, i, a) => a.findIndex(x => x.id === t.id) === i).filter(t => matches(`${t.title} ${t.artist}`, q)).slice(0, 4) : [];
    if (acts.length) out.push({ head: q ? 'Actions' : 'Suggested' }, ...acts.slice(0, q ? 6 : 8).map(a => tag(a, 'act')));
    if (pls.length) out.push({ head: 'Playlists' }, ...pls.slice(0, 4).map(p => tag(p, 'pl')));
    if (mine.length) out.push({ head: 'In your library' }, ...mine.map(t => tag(t, 'song')));
    const songs = remote.filter(x => x.type === 'song' && !mine.some(m => m.id === x.id)).slice(0, 5), arts = remote.filter(x => x.type === 'artist').slice(0, 3), albs = remote.filter(x => x.type === 'album').slice(0, 3);
    if (songs.length) out.push({ head: 'Songs' }, ...songs.map(t => tag(t, 'song')));
    if (arts.length) out.push({ head: 'Artists' }, ...arts.map(a => tag(a, 'artist')));
    if (albs.length) out.push({ head: 'Albums' }, ...albs.map(a => tag(a, 'album')));
    return out;
  }

  function refresh() {
    const rows = collect();
    items = rows.filter(r => r.it);
    let n = -1, html = '';
    for (const r of rows) {
      if (r.head) { html += `<div class="ck-head">${esc(r.head)}</div>`; continue; }
      n++;
      const it = r.it;
      let lead, title, sub = '', hint = '';
      if (r.kind === 'act') { lead = `<span class="ck-ic">${ic(it.i, 20)}</span>`; title = it.n; hint = it.h ? `<kbd>${esc(it.h)}</kbd>` : ''; }
      else if (r.kind === 'pl') { lead = `<span class="ck-ic">${ic(it.id === 'liked' ? 'heart' : 'lib', 20)}</span>`; title = it.name; sub = 'Playlist'; hint = '<kbd>Shift</kbd>+<kbd>Enter</kbd> play'; }
      else if (r.kind === 'song') { lead = `<img src="${img(it.thumb)}" alt="">`; title = it.title; sub = it.artist || ''; hint = 'Play'; }
      else { lead = it.thumb ? `<img class="${r.kind === 'artist' ? 'round' : ''}" src="${img(it.thumb)}" alt="">` : `<span class="ck-ic">${ic(r.kind === 'artist' ? 'user' : 'album', 20)}</span>`; title = it.title; sub = r.kind === 'artist' ? 'Artist' : ['Album', it.subtitle].filter(Boolean).join(' • '); }
      html += `<button class="ck-item${n === active ? ' on' : ''}" role="option" data-i="${n}">${lead}<span class="ck-t"><b>${esc(title)}</b>${sub ? `<i>${esc(sub)}</i>` : ''}</span><span class="ck-h">${hint}</span></button>`;
    }
    if (searching) html += '<div class="ck-note">Searching…</div>';
    else if (!items.length) html = '<div class="ck-note">Nothing matches. Try a song, artist or command.</div>';
    list.innerHTML = html;
    scrollActive();
  }
  const mark = () => { list.querySelectorAll('.ck-item').forEach(b => b.classList.toggle('on', +b.dataset.i === active)); scrollActive(); };
  const scrollActive = () => { const b = list.querySelector('.ck-item.on'); if (b) b.scrollIntoView({ block: 'nearest' }); };

  function search() {
    clearTimeout(deb); const q = input.value.trim(), my = ++seq;
    if (q.length < 2) { remote = []; searching = false; return; }
    searching = true;
    deb = setTimeout(async () => {
      try { const r = await cached(`/api/search?q=${encodeURIComponent(q)}&type=all`, 2 * 60 * 1000); if (my !== seq) return; remote = r.map(x => x.type === 'song' ? T(x) : x).map((x, i) => (r[i].type === 'song' ? { ...x, type: 'song' } : x)); }
      catch { if (my !== seq) return; remote = []; }
      searching = false; if (root) refresh();
    }, 260);
    refresh();
  }

  function run(n, shift) {
    const r = items[n]; if (!r) return;
    const kind = r.kind, it = r.it;
    close();
    if (kind === 'act') it.run();
    else if (kind === 'song') { shift ? (queue.push(T(it)), persist(), renderQueueList(), snack('Added to queue')) : playNow(it); }
    else if (kind === 'pl') {
      if (shift) { it.id === 'liked' ? playShuffled(liked.map(T)) : api('/api/playlists/' + it.id).then(p => p.tracks.length ? playList(p.tracks.map(T), 0) : snack('Empty playlist')).catch(e => snack(e.message)); }
      else go('/playlist/' + it.id);
    }
    else if (kind === 'artist') go('/artist/' + it.id);
    else if (kind === 'album') go('/album/' + it.id);
  }

  function open() {
    if (!root) build();
    remote = []; searching = false; active = 0; input.value = '';
    root.classList.add('on'); refresh(); input.focus();
  }
  function close() { if (root) root.classList.remove('on'); }
  const isOpen = () => root && root.classList.contains('on');

  document.addEventListener('keydown', e => {
    if ((e.ctrlKey || e.metaKey) && !e.altKey && (e.key === 'k' || e.key === 'K')) { e.preventDefault(); e.stopPropagation(); isOpen() ? close() : open(); return; }
    if (!isOpen()) return;
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); close(); }
    else if (e.key === 'ArrowDown') { e.preventDefault(); e.stopPropagation(); active = Math.min(items.length - 1, active + 1); mark(); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); e.stopPropagation(); active = Math.max(0, active - 1); mark(); }
    else if (e.key === 'Enter') { e.preventDefault(); e.stopPropagation(); run(active, e.shiftKey); }
    else if (e.key === 'Tab') e.preventDefault();
    else e.stopPropagation();   // typing here must not trigger the app's shortcuts
  }, true);
  window.openPalette = open;
})();
