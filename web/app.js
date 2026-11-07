'use strict';
// ============================================================================
// metroweb frontend: hash router + player. No build step, no dependencies.
// ============================================================================
const $ = s => document.querySelector(s);
const $$ = s => [...document.querySelectorAll(s)];

const IC = {
  play: 'M8 5v14l11-7z', pause: 'M6 19h4V5H6v14zm8-14v14h4V5h-4z',
  next: 'M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z', prev: 'M6 6h2v12H6zm3.5 6l8.5 6V6z',
  search: 'M15.5 14h-.79l-.28-.27A6.471 6.471 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z',
  queue: 'M15 6H3v2h12V6zm0 4H3v2h12v-2zM3 16h8v-2H3v2zM17 6v8.18c-.31-.11-.65-.18-1-.18-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3V8h3V6h-5z',
  heart: 'M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z',
  heart0: 'M16.5 3c-1.74 0-3.41.81-4.5 2.09C10.91 3.81 9.24 3 7.5 3 4.42 3 2 5.42 2 8.5c0 3.78 3.4 6.86 8.55 11.54L12 21.35l1.45-1.32C18.6 15.36 22 12.28 22 8.5 22 5.42 19.58 3 16.5 3zm-4.4 15.55l-.1.1-.1-.1C7.14 14.24 4 11.39 4 8.5 4 6.5 5.5 5 7.5 5c1.54 0 3.04.99 3.57 2.36h1.87C13.46 5.99 14.96 5 16.5 5c2 0 3.5 1.5 3.5 3.5 0 2.89-3.14 5.74-7.9 10.05z',
  dl: 'M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z', check: 'M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z',
  more: 'M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z',
  down: 'M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z', close: 'M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z',
  shuf: 'M10.59 9.17L5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41l-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z',
  rep: 'M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z',
  rep1: 'M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4zm-4-2V9h-1l-2 1v1h1.5v4H13z',
  user: 'M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z',
  out: 'M17 7l-1.41 1.41L18.17 11H8v2h10.17l-2.58 2.58L17 17l5-5zM4 5h8V3H4c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h8v-2H4V5z',
  vol: 'M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z',
  expand: 'M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z',
  warn: 'M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z',
  clock: 'M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z',
  music: 'M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z',
  radio: 'M3.24 6.15C2.51 6.43 2 7.17 2 8v12c0 1.1.89 2 2 2h16c1.11 0 2-.9 2-2V8c0-1.11-.89-2-2-2H8.3l8.26-3.34L15.88 1 3.24 6.15zM7 20c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm13-8h-2v-2h-2v2H4V8h16v4z',
  addq: 'M14 10H3v2h11v-2zm0-4H3v2h11V6zM3 16h7v-2H3v2zm11-1v3h-3v2h3v3h2v-3h3v-2h-3v-3h-2z',
  playnext: 'M4 18l8.5-6L4 6v12zm9-12v12l8.5-6L13 6z',
  home: 'M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z',
  lib: 'M20 2H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-2 5h-3v5.5c0 1.38-1.12 2.5-2.5 2.5S10 13.88 10 12.5s1.12-2.5 2.5-2.5c.57 0 1.08.19 1.5.51V5h4v2zM4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6z',
  share: 'M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92 1.61 0 2.92-1.31 2.92-2.92s-1.31-2.92-2.92-2.92z',
  add: 'M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z',
  del: 'M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z',
  edit: 'M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z',
  back: 'M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z',
  album: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14.5c-2.49 0-4.5-2.01-4.5-4.5S9.51 7.5 12 7.5s4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5zm0-5.5c-.55 0-1 .45-1 1s.45 1 1 1 1-.45 1-1-.45-1-1-1z',
};
const ic = (n, s) => `<svg class="i" viewBox="0 0 24 24" aria-hidden="true"${s ? ` style="width:${s}px;height:${s}px"` : ''}><path d="${IC[n]}"/></svg>`;
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const time = s => !isFinite(s) ? '0:00' : `${Math.floor(s/60)}:${String(Math.floor(s%60)).padStart(2,'0')}`;
const PH = 'data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==';
const img = u => u ? '/api/img?u=' + encodeURIComponent(u) : PH;
const store = {
  get(k, d) { try { const v = localStorage.getItem(k); return v ? JSON.parse(v) : d; } catch { return d; } },
  set(k, v) { try { localStorage.setItem(k, JSON.stringify(v)); } catch {} },
};
const T = x => ({ id: x.id, title: x.title, artist: x.artist || '', artistId: x.artistId || '', album: x.album || '', albumId: x.albumId || '', duration: x.duration || '', thumb: x.thumb || '' });

function snack(msg) {
  const s = $('#snack'); s.textContent = msg; s.classList.add('on');
  clearTimeout(snack.t); snack.t = setTimeout(() => s.classList.remove('on'), 2800);
}

// ---------- network ----------
async function api(path, opt) {
  const r = await fetch(path, opt);
  if (r.status === 401) { location.href = '/login'; throw new Error('Signed out'); }
  let data = null; try { data = await r.json(); } catch {}
  if (!r.ok) throw new Error((data && data.error) || `Request failed (${r.status})`);
  return data;
}
const send = (method, path, body) => api(path, { method, headers: { 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined });
const memo = new Map();
async function cached(path, ttl = 5 * 60 * 1000) {
  const hit = memo.get(path);
  if (hit && Date.now() - hit.t < ttl) return hit.v;
  const v = await api(path); memo.set(path, { t: Date.now(), v }); return v;
}

// ---------- state ----------
const audio = $('#audio');
const fmt = audio.canPlayType('audio/webm; codecs="opus"') ? 'opus' : 'aac';
let queue = store.get('mw.queue', []);
let idx = store.get('mw.idx', -1);
let recent = store.get('mw.history', []);
let repeat = store.get('mw.repeat', 0);        // 0 off, 1 all, 2 one
let shuffle = store.get('mw.shuffle', false);
let liked = [];                                  // Track[] (oldest first)
let likedIds = new Set();
let playlists = [];                              // summaries, incl. "liked"
let dls = {};                                    // id -> queued|running|done|error...
let acc = { user: '', defaultCreds: false };
let fetchingRadio = false;
const cur = () => queue[idx];
const playing = () => !audio.paused && !!audio.src;

// ============================================================================
// Rendering helpers
// ============================================================================
let V = { lists: {} };   // lists rendered in the current view, so clicks can find items

function reg(name, items, mode) { V.lists[name] = { items, mode }; return name; }

function stateOf(id) {
  const s = dls[id] || '';
  if (s === 'queued' || s === 'running') return 'busy';
  if (s === 'done') return 'done';
  if (s.startsWith('error')) return 'err';
  return '';
}
function stIcon(id) {
  const s = stateOf(id);
  if (s === 'busy') return `<span class="st" title="Downloading…">${ic('clock')}</span>`;
  if (s === 'done') return `<span class="st" title="In your library">${ic('check')}</span>`;
  if (s === 'err') return `<span class="st err" title="${esc(dls[id])}">${ic('warn')}</span>`;
  return '';
}

// A song row. o.num shows a track number instead of artwork.
function rowHTML(t, l, i, o = {}) {
  const c = cur(), isCur = c && c.id === t.id && (l !== 'queue' || i === idx);
  const sub = [t.artist, o.noAlbum ? '' : t.album].filter(Boolean).join(' • ');
  return `<div class="row${isCur ? ' cur' : ''}${isCur && playing() ? ' playing' : ''}" data-l="${l}" data-i="${i}" data-id="${esc(t.id)}" role="button" tabindex="0">
    ${o.num ? `<span class="num">${o.num}</span>` : `<img class="art" loading="lazy" src="${img(t.thumb)}" alt="">`}
    <span class="eq"><i></i><i></i><i></i></span>
    <div class="meta"><div class="t">${esc(t.title)}</div><div class="a">${stIcon(t.id)}<span>${esc(sub)}</span></div></div>
    <span class="dur">${esc(t.duration)}</span>
    <button class="iconbtn" data-more aria-label="More options">${ic('more')}</button>
  </div>`;
}
const songRows = (items, l, o) => items.map((t, i) => rowHTML(t, l, i, o)).join('');

function cardHTML(it, l, i) {
  const round = it.type === 'artist';
  const img_ = it.thumb ? `<img class="img" loading="lazy" src="${img(it.thumb)}" alt="">`
    : `<div class="img ph">${ic(it.type === 'artist' ? 'user' : 'music')}</div>`;
  return `<button class="card${round ? ' round' : ''}" data-card data-l="${l}" data-i="${i}">${img_}
    <div class="ct">${esc(it.title)}</div><div class="cs">${esc(it.subtitle || it.artist || '')}</div></button>`;
}
const cards = (items, l) => items.map((it, i) => cardHTML(it, l, i)).join('');
const hscroll = (items, l) => `<div class="hscroll">${cards(items, l)}</div>`;
const section = (title, body, more) => `<h2 class="sec"><span>${esc(title)}</span>${more || ''}</h2>${body}`;
const skeletonRows = n => Array.from({ length: n }, () => `<div class="row skeleton"><div class="art"></div><div><div class="l1"></div><div class="l2"></div></div></div>`).join('');
const skeletonCards = n => `<div class="hscroll">${Array.from({ length: n }, () => `<div class="card skeleton"><div class="img"></div></div>`).join('')}</div>`;
const emptyHTML = (icn, text, btn) => `<div class="empty">${ic(icn)}<div>${text}</div>${btn || ''}</div>`;
const errorHTML = (msg, retry = true) => `<div class="err-box">${ic('warn', 40)}<div>${esc(msg)}</div>${retry ? '<button class="btn tonal" data-retry>Try again</button>' : ''}</div>`;
const backBtn = () => `<button class="iconbtn back" data-back aria-label="Back">${ic('back')}</button>`;
const topBar = (title, right = '') => `<div class="top glass"><button class="iconbtn" data-back aria-label="Back">${ic('back')}</button><h1>${esc(title)}</h1>${right}</div>`;
const avatarBtn = () => `<button class="avatar only-m" data-acc aria-label="Account">${esc((acc.user || 'M')[0].toUpperCase())}</button>`;
const bannerHTML = () => acc.defaultCreds ? `<div class="banner"><div><b>Change the default password</b><span>This server still uses admin / admin.</span></div><button data-acc>Change</button></div>` : '';

// ============================================================================
// Router
// ============================================================================
const routeState = new Map();     // transient data passed between routes (e.g. playlist title)
let token = 0, routeName = 'home', lastTab = 'home';

function parseHash() {
  const h = location.hash.replace(/^#/, '') || '/';
  const [path, qs] = h.split('?');
  return { path: path || '/', params: new URLSearchParams(qs || '') };
}
const go = path => { const h = '#' + path; if (location.hash === h) route(); else location.hash = h; };
const goBack = () => { if (window.history.length > 1) window.history.back(); else go('/'); };

const ROUTES = [
  [/^\/$/, 'home', viewHome],
  [/^\/search$/, 'search', viewSearch],
  [/^\/library$/, 'library', viewLibrary],
  [/^\/queue$/, 'queue', viewQueue],
  [/^\/artist\/([\w-]+)$/, 'artist', viewArtist],
  [/^\/album\/([\w-]+)$/, 'album', viewAlbum],
  [/^\/yt\/([\w-]+)$/, 'yt', viewYT],
  [/^\/playlist\/([\w-]+)$/, 'playlist', viewPlaylist],
];
const TAB_OF = { home: 'home', search: 'search', library: 'library', queue: 'queue', playlist: 'library' };

function route() {
  const { path, params } = parseHash();
  const t = ++token;
  closeSheet();
  let m, fn, name = 'home';
  for (const [re, n, f] of ROUTES) { if ((m = path.match(re))) { fn = f; name = n; break; } }
  if (!fn) { go('/'); return; }
  routeName = name;
  if (TAB_OF[name]) lastTab = TAB_OF[name];
  $$('[data-nav]').forEach(b => b.classList.toggle('on', b.dataset.nav === lastTab));
  V = { lists: {} };
  const view = $('#view');
  view.onclick = null;
  view.style.animation = 'none'; void view.offsetWidth; view.style.animation = '';
  $('#content').scrollTo(0, 0);
  const ctx = { t, params, id: m[1], alive: () => t === token, set: html => { if (t === token) view.innerHTML = html; } };
  Promise.resolve(fn(ctx)).catch(e => { console.error(e); ctx.set(topBar('Error') + errorHTML(e.message)); });
}
window.addEventListener('hashchange', route);

// ============================================================================
// Views
// ============================================================================
async function viewHome(c) {
  const last = recent[0] || liked[liked.length - 1];
  c.set(`<div class="top"><a class="search" href="#/search">${ic('search')}<span>Search songs, artists, albums…</span></a>${avatarBtn()}</div>
    <div id="banner">${bannerHTML()}</div>
    <div id="hQuick"></div><div id="hRecent"></div><div id="hLists"></div><div id="hBrowse">${section('Browse', skeletonCards(6))}</div>`);
  const own = playlists.filter(p => p.id !== 'liked' || p.count > 0);
  if (c.alive()) {
    const ul = own.map(p => ({ type: 'playlist', id: 'local:' + p.id, title: p.name, subtitle: `${p.count} song${p.count === 1 ? '' : 's'}`, thumb: p.thumb, local: p.id }));
    $('#hLists').innerHTML = ul.length ? section('Your library', hscroll(ul, reg('hl', ul))) : '';
    if (recent.length) $('#hRecent').innerHTML = section('Recently played', hscroll(recent.slice(0, 12).map(x => ({ ...x, type: 'song', subtitle: x.artist })), reg('hr', recent.slice(0, 12), 'one')));
  }
  if (last) {
    $('#hQuick').innerHTML = section('Made for you', skeletonRows(5));
    try {
      const radio = await cached('/api/radio/' + last.id);
      if (!c.alive()) return;
      const picks = radio.filter(x => x.id !== last.id).slice(0, 8);
      $('#hQuick').innerHTML = picks.length ? section('Made for you', `<div class="sub" style="color:var(--dim);font-size:13px;padding:0 4px 6px">Because you played ${esc(last.title)}</div>` + songRows(picks, reg('hq', picks, 'one'))) : '';
    } catch { if (c.alive()) $('#hQuick').innerHTML = ''; }
  } else if (c.alive()) {
    $('#hQuick').innerHTML = emptyHTML('music', 'Search for a song to get started', '<a class="btn" href="#/search">Search</a>');
  }
  try {
    const secs = (await cached('/api/home', 15 * 60 * 1000)).filter(s => s.items.length).slice(0, 5);
    if (!c.alive()) return;
    $('#hBrowse').innerHTML = secs.map((s, k) => section(s.title, hscroll(s.items, reg('hb' + k, s.items, 'one')))).join('');
  } catch { if (c.alive()) $('#hBrowse').innerHTML = ''; }
}

// ---------- search ----------
const YT_RE = /(?:youtu\.be\/|[?&]v=|\/shorts\/|\/embed\/)([\w-]{11})/;
function parseYTLink(q) {
  if (!/^https?:\/\//i.test(q) || !/youtu/i.test(q)) return null;
  let m;
  if ((m = q.match(/\/channel\/(UC[\w-]+)/))) return { kind: 'artist', id: m[1] };
  if ((m = q.match(/\/browse\/(MPRE[\w-]+)/))) return { kind: 'album', id: m[1] };
  if ((m = q.match(/[?&]list=([\w-]+)/)) && !YT_RE.test(q)) return { kind: 'yt', id: 'VL' + m[1] };
  if ((m = q.match(YT_RE))) return { kind: 'track', id: m[1] };
  return null;
}
async function viewSearch(c) {
  let q = c.params.get('q') || '', type = c.params.get('t') || 'all', seq = 0, deb;
  const tabs = [['all', 'All'], ['songs', 'Songs'], ['artists', 'Artists'], ['albums', 'Albums']];
  c.set(`<div class="top"><label class="search">${ic('search')}<input id="q" type="search" placeholder="Search songs, artists, albums or paste a link" autocomplete="off" enterkeyhint="search" value="${esc(q)}" aria-label="Search">
      <button class="iconbtn" id="qClear" aria-label="Clear" ${q ? '' : 'hidden'}>${ic('close')}</button></label>${avatarBtn()}</div>
    <div class="chips" id="chips" role="tablist">${tabs.map(([k, n]) => `<button class="chip${k === type ? ' on' : ''}" data-t="${k}" role="tab">${n}</button>`).join('')}</div>
    <div id="sres"></div>`);
  const inp = $('#q'), res = $('#sres');
  const urlFor = () => `#/search?${new URLSearchParams({ ...(q ? { q } : {}), ...(type !== 'all' ? { t: type } : {}) })}`;
  const sync = () => window.history.replaceState(null, '', urlFor());
  const recents = () => store.get('mw.searches', []);
  const remember = s => store.set('mw.searches', [s, ...recents().filter(x => x !== s)].slice(0, 12));

  function showRecents() {
    const r = recents();
    res.innerHTML = r.length ? section('Recent searches', r.map((s, i) => `<div class="row" data-rs="${i}" role="button" tabindex="0" style="grid-template-columns:52px 1fr auto"><span class="num" style="display:grid;place-items:center;color:var(--dim)">${ic('clock')}</span><div class="meta"><div class="t">${esc(s)}</div></div><button class="iconbtn" data-rsx="${i}" aria-label="Remove">${ic('close')}</button></div>`).join(''))
      : emptyHTML('search', 'Find songs, artists and albums');
  }
  async function run() {
    const my = ++seq; sync();
    if (!q.trim()) { showRecents(); return; }
    const link = parseYTLink(q.trim());
    if (link) {
      if (link.kind === 'track') {
        res.innerHTML = skeletonRows(1);
        try { const t = await cached('/api/track/' + link.id); if (my !== seq) return; res.innerHTML = section('From link', songRows([t], reg('s', [t], 'one'))); }
        catch (e) { if (my === seq) res.innerHTML = errorHTML('Could not read that link: ' + e.message, false); }
      } else go('/' + link.kind + '/' + link.id);
      return;
    }
    res.innerHTML = skeletonRows(8);
    try {
      const items = await cached(`/api/search?q=${encodeURIComponent(q.trim())}&type=${type}`, 2 * 60 * 1000);
      if (my !== seq) return;
      remember(q.trim());
      res.innerHTML = searchHTML(items);
    } catch (e) { if (my === seq) res.innerHTML = errorHTML('Search failed: ' + e.message); }
  }
  function searchHTML(items) {
    const songs = items.filter(x => x.type === 'song'), artists = items.filter(x => x.type === 'artist'),
      albums = items.filter(x => x.type === 'album'), pls = items.filter(x => x.type === 'playlist');
    if (!items.length) return emptyHTML('search', 'No results');
    if (type === 'songs') return songRows(songs, reg('s', songs, 'one'));
    if (type === 'artists') return `<div class="grid">${cards(artists, reg('a', artists))}</div>`;
    if (type === 'albums') return `<div class="grid">${cards(albums, reg('b', albums))}</div>`;
    let h = '';
    if (songs.length) h += section('Songs', songRows(songs.slice(0, 6), reg('s', songs, 'one')), songs.length > 6 ? '<small data-t="songs" role="button">Show all</small>' : '');
    if (artists.length) h += section('Artists', hscroll(artists.slice(0, 10), reg('a', artists)), '<small data-t="artists" role="button">Show all</small>');
    if (albums.length) h += section('Albums', hscroll(albums.slice(0, 10), reg('b', albums)), '<small data-t="albums" role="button">Show all</small>');
    if (pls.length) h += section('Playlists', hscroll(pls.slice(0, 10), reg('p', pls)));
    return h;
  }
  inp.addEventListener('input', () => { q = inp.value; $('#qClear').hidden = !q; clearTimeout(deb); deb = setTimeout(run, q.trim().length > 1 ? 380 : 0); });
  inp.addEventListener('keydown', e => { if (e.key === 'Enter') { clearTimeout(deb); q = inp.value; run(); inp.blur(); } });
  $('#qClear').onclick = e => { e.preventDefault(); inp.value = q = ''; $('#qClear').hidden = true; run(); inp.focus(); };
  $('#view').onclick = e => {
    const t = e.target.closest('[data-t]');
    if (t) { type = t.dataset.t; $$('#chips .chip').forEach(b => b.classList.toggle('on', b.dataset.t === type)); run(); return; }
    const rx = e.target.closest('[data-rsx]');
    if (rx) { e.stopPropagation(); store.set('mw.searches', recents().filter((_, i) => i !== +rx.dataset.rsx)); showRecents(); return; }
    const rs = e.target.closest('[data-rs]');
    if (rs) { q = inp.value = recents()[+rs.dataset.rs]; $('#qClear').hidden = false; run(); }
  };
  run();
  if (!q) inp.focus();
}

// ---------- artist ----------
async function viewArtist(c) {
  c.set(`<div class="hero noimg">${backBtn()}<h1>&nbsp;</h1></div><div style="padding-top:12px">${skeletonRows(5)}</div>`);
  const a = await cached('/api/artist/' + c.id);
  if (!c.alive()) return;
  const songs = a.songs.map(T);
  const secs = a.sections.filter(s => s.items.length);
  c.set(`<div class="hero${a.thumb ? '' : ' noimg'}" style="--hero:url('${esc(img(a.thumb))}')">${backBtn()}
      <h1>${esc(a.name)}</h1><div class="sub">${esc(a.subtitle || '')}</div></div>
    <div class="actions">
      <button class="btn" data-act="play">${ic('play')}Play</button>
      <button class="btn tonal" data-act="radio">${ic('radio')}Radio</button>
      <button class="btn tonal" data-act="shuffle">${ic('shuf')}Shuffle</button>
      <button class="iconbtn" data-act="share" aria-label="Share link">${ic('share')}</button>
    </div>
    ${songs.length ? section('Songs', `<div id="aSongs">${songRows(songs, reg('as', songs, 'queue'))}</div>`, a.songsBrowse ? '<small id="aAll" role="button">Show all</small>' : '') : ''}
    ${secs.map((s, k) => section(s.title, hscroll(s.items, reg('sec' + k, s.items, 'one')))).join('')}
    ${a.description ? section('About', `<div class="about clamp" id="aAbout">${esc(a.description)}</div><button class="chip" id="aMore" style="margin:4px">Read more</button>`) : ''}`);
  let all = songs;
  const aAll = $('#aAll');
  if (aAll) aAll.onclick = async () => {
    aAll.textContent = 'Loading…';
    try {
      const list = (await cached(`/api/songs?b=${encodeURIComponent(a.songsBrowse)}&p=${encodeURIComponent(a.songsParams || '')}`)).map(T);
      if (!c.alive()) return;
      all = list; $('#aSongs').innerHTML = songRows(list, reg('as', list, 'queue')); aAll.remove();
    } catch (e) { aAll.textContent = 'Show all'; snack(e.message); }
  };
  const more = $('#aMore'); if (more) more.onclick = () => { $('#aAbout').classList.toggle('clamp'); more.textContent = $('#aAbout').classList.contains('clamp') ? 'Read more' : 'Show less'; };
  c.actions = async act => {
    if (act === 'play') { if (all.length) playList(all, 0); else snack('No songs to play'); }
    if (act === 'shuffle') { if (all.length) playList([...all].sort(() => Math.random() - .5), 0); }
    if (act === 'radio') {
      if (!a.radioId) { if (all.length) startRadio(all[0]); return; }
      snack('Starting radio…');
      try { const r = await api(`/api/mix?id=${encodeURIComponent(a.radioId)}&p=${encodeURIComponent(a.radioParams || '')}&v=${encodeURIComponent(a.radioVideoId || '')}`); playList(r.map(T), 0); }
      catch (e) { snack('Could not start radio: ' + e.message); }
    }
    if (act === 'share') share(ytURL('artist', a.id), a.name);
  };
  bindActions(c);
}

// ---------- album & YouTube playlists ----------
function tracksHeader(o) {
  return `<div class="head2">${backBtn()}
    ${o.thumb ? `<img class="cov" src="${img(o.thumb)}" alt="">` : `<div class="cov ph">${ic(o.icon || 'music')}</div>`}
    <div class="txt"><h1>${esc(o.title)}</h1><div class="sub">${o.sub || ''}</div></div></div>
    <div class="actions">${o.buttons}</div>`;
}
async function viewAlbum(c) {
  c.set(`<div class="head2">${backBtn()}<div class="cov skeleton"></div></div>${skeletonRows(6)}`);
  const al = await cached('/api/album/' + c.id);
  if (!c.alive()) return;
  const tracks = al.tracks.map(T);
  const sub = [al.artistId ? `<button data-artist="${esc(al.artistId)}">${esc(al.artist)}</button>` : esc(al.artist), esc(al.year), `${tracks.length} songs`].filter(Boolean).join(' • ');
  c.set(tracksHeader({ title: al.title, thumb: al.thumb, sub, icon: 'album', buttons: listButtons() }) +
    songRows(tracks, reg('t', tracks, 'queue'), { noAlbum: true }));
  c.actions = act => listAction(act, tracks, al.title, ytURL('album', al.id));
  bindActions(c);
}
async function viewYT(c) {
  const st = routeState.get(c.id) || {};
  c.set(`<div class="head2">${backBtn()}<div class="cov skeleton"></div></div>${skeletonRows(6)}`);
  const tracks = (await cached(`/api/songs?b=${encodeURIComponent(c.id)}`)).map(T);
  if (!c.alive()) return;
  c.set(tracksHeader({ title: st.title || 'Playlist', thumb: st.thumb || (tracks[0] && tracks[0].thumb), sub: `${tracks.length} songs`, buttons: listButtons() }) +
    (tracks.length ? songRows(tracks, reg('t', tracks, 'queue')) : emptyHTML('music', 'This playlist is empty or unavailable')));
  c.actions = act => listAction(act, tracks, st.title || 'Playlist', ytURL('playlist', c.id));
  bindActions(c);
}
const listButtons = (extra = '') => `<button class="btn" data-act="play">${ic('play')}Play</button>
  <button class="btn tonal" data-act="shuffle">${ic('shuf')}Shuffle</button>
  <button class="iconbtn" data-act="addpl" aria-label="Add to playlist">${ic('addq')}</button>
  <button class="iconbtn" data-act="dlall" aria-label="Download all">${ic('dl')}</button>
  <button class="iconbtn" data-act="share" aria-label="Share link">${ic('share')}</button>${extra}`;
function listAction(act, tracks, title, url) {
  if (act === 'play' && tracks.length) playList(tracks, 0);
  if (act === 'shuffle' && tracks.length) playList([...tracks].sort(() => Math.random() - .5), 0);
  if (act === 'addpl') playlistPicker(tracks, title);
  if (act === 'dlall') downloadAll(tracks);
  if (act === 'share' && url) share(url, title);
}
function bindActions(c) {
  $('#view').onclick = e => { const b = e.target.closest('[data-act]'); if (b && c.actions) c.actions(b.dataset.act); };
}

// ---------- library & playlists ----------
async function viewLibrary(c) {
  await refreshPlaylists();
  if (!c.alive()) return;
  const items = playlists.map(p => ({ type: 'playlist', id: 'local:' + p.id, local: p.id, title: p.name, subtitle: `${p.count} song${p.count === 1 ? '' : 's'}`, thumb: p.thumb }));
  c.set(`<div class="top"><h1>Library</h1><button class="btn tonal" data-newpl>${ic('add')}New playlist</button>${avatarBtn()}</div>
    <div id="banner">${bannerHTML()}</div>
    <div class="grid">${cards(items, reg('pl', items))}</div>
    ${recent.length ? section('Recently played', songRows(recent.slice(0, 8), reg('rh', recent.slice(0, 8), 'one'))) : ''}`);
}
async function viewPlaylist(c) {
  c.set(`<div class="head2">${backBtn()}<div class="cov skeleton"></div></div>${skeletonRows(6)}`);
  const p = await api('/api/playlists/' + c.id);
  if (!c.alive()) return;
  const isLiked = p.id === 'liked';
  const tracks = (isLiked ? [...p.tracks].reverse() : p.tracks).map(T);
  const more = isLiked ? '' : `<button class="iconbtn" data-act="rename" aria-label="Rename">${ic('edit')}</button><button class="iconbtn" data-act="delete" aria-label="Delete playlist">${ic('del')}</button>`;
  c.set(tracksHeader({ title: p.name, thumb: tracks[0] && tracks[0].thumb, icon: isLiked ? 'heart' : 'music', sub: `${tracks.length} song${tracks.length === 1 ? '' : 's'}`, buttons: listButtons(more).replace(/<button class="iconbtn" data-act="addpl"[^>]*>.*?<\/button>/, '') }) +
    (tracks.length ? songRows(tracks, reg('t', tracks, 'queue')) : emptyHTML(isLiked ? 'heart' : 'music', isLiked ? 'Songs you like will show up here' : 'No songs yet. Use the ⋮ menu on any song to add it here.')));
  V.playlistId = p.id;
  c.actions = act => {
    if (act === 'rename') promptSheet('Rename playlist', p.name, 'Save', async name => { await send('PATCH', '/api/playlists/' + p.id, { name }); await refreshPlaylists(); route(); });
    else if (act === 'delete') confirmSheet('Delete playlist?', `“${p.name}” will be deleted. The songs themselves are not affected.`, 'Delete', async () => { await send('DELETE', '/api/playlists/' + p.id); await refreshPlaylists(); snack('Playlist deleted'); go('/library'); });
    else listAction(act, tracks, p.name, null);
  };
  bindActions(c);
}

// ---------- queue ----------
function viewQueue(c) {
  c.set(`<div class="top"><h1>Queue</h1>
      <button class="chip" id="qSave" ${queue.length ? '' : 'disabled'}>${ic('addq')}Save</button>
      <button class="chip" id="qClear" ${queue.length ? '' : 'disabled'}>${ic('close')}Clear</button></div>
    <div id="qList"></div>`);
  renderQueueList.done = false; renderQueueList();
  $('#qClear').onclick = () => { queue = cur() ? [cur()] : []; idx = queue.length ? 0 : -1; persist(); renderQueueList(); };
  $('#qSave').onclick = () => playlistPicker(queue.map(T), 'Queue');
}
function renderQueueList() {
  const el = $('#qList'); if (!el) return;
  reg('queue', queue, 'queue');
  el.innerHTML = queue.length ? songRows(queue, 'queue') : emptyHTML('queue', 'The queue is empty', '<a class="btn" href="#/search">Find music</a>');
  const c = el.querySelector('.row.cur'); if (c && !renderQueueList.done) { c.scrollIntoView({ block: 'center' }); renderQueueList.done = true; }
}

// ============================================================================
// Likes, playlists, downloads, sharing
// ============================================================================
async function refreshPlaylists() {
  try { playlists = await api('/api/playlists'); } catch (e) { console.error(e); }
}
async function loadLiked() {
  try {
    const p = await api('/api/playlists/liked');
    liked = p.tracks || []; likedIds = new Set(liked.map(t => t.id));
  } catch (e) { console.error(e); }
}
async function toggleLike(t) {
  if (!t) return;
  const was = likedIds.has(t.id);
  if (was) { likedIds.delete(t.id); liked = liked.filter(x => x.id !== t.id); }
  else { likedIds.add(t.id); liked.push(T(t)); }
  updateNow();
  try {
    if (was) { await send('DELETE', `/api/playlists/liked/tracks/${t.id}`); snack('Removed from Liked songs'); }
    else { await send('POST', '/api/playlists/liked/tracks', { tracks: [T(t)] }); snack('Liked — saving to your library'); download(t.id, true); }
    refreshPlaylists();
    if (routeName === 'playlist' && V.playlistId === 'liked') route();
  } catch (e) {
    if (was) { likedIds.add(t.id); liked.push(T(t)); } else { likedIds.delete(t.id); liked = liked.filter(x => x.id !== t.id); }
    updateNow(); snack(e.message);
  }
}
async function addToPlaylist(id, tracks, name) {
  try {
    const r = await send('POST', `/api/playlists/${id}/tracks`, { tracks: tracks.map(T) });
    snack(r.added ? `Added ${r.added} song${r.added === 1 ? '' : 's'} to ${name}` : `Already in ${name}`);
    if (id === 'liked') { await loadLiked(); updateNow(); }
    refreshPlaylists();
  } catch (e) { snack(e.message); }
}
function playlistPicker(tracks, title) {
  if (!tracks.length) { snack('Nothing to add'); return; }
  const one = tracks.length === 1 ? esc(tracks[0].title) : `${tracks.length} songs`;
  openSheet(`<h3>Add to playlist</h3><div class="sub">${one}</div>
    <button class="act accent" data-new>${ic('add')}New playlist</button>
    ${playlists.map(p => `<button class="act" data-pl="${esc(p.id)}"><img class="pl-thumb" src="${img(p.thumb)}" alt="">${esc(p.name)}<span class="sub">${p.count}</span></button>`).join('')}`);
  $('#sheet').onclick = e => {
    const n = e.target.closest('[data-new]'), p = e.target.closest('[data-pl]');
    if (n) promptSheet('New playlist', '', 'Create', async name => {
      const made = await send('POST', '/api/playlists', { name, tracks: tracks.map(T) });
      await refreshPlaylists(); snack(`Created “${made.name}”`); if (routeName === 'library') route();
    });
    else if (p) { const pl = playlists.find(x => x.id === p.dataset.pl); closeSheet(); addToPlaylist(pl.id, tracks, pl.name); }
  };
}
function promptSheet(title, value, cta, onOk) {
  openSheet(`<h3>${esc(title)}</h3><form id="pf"><input type="text" id="pv" maxlength="100" placeholder="Name" value="${esc(value)}" autocomplete="off" required>
    <div class="msg" id="pm"></div><div class="btns"><button type="button" class="btn ghost" id="pc">Cancel</button><button class="btn" type="submit">${esc(cta)}</button></div></form>`);
  $('#sheet').onclick = null;
  const inp = $('#pv'); setTimeout(() => { inp.focus(); inp.select(); }, 250);
  $('#pc').onclick = closeSheet;
  $('#pf').onsubmit = async e => {
    e.preventDefault(); const name = inp.value.trim(); if (!name) return;
    try { await onOk(name); closeSheet(); } catch (err) { $('#pm').textContent = err.message; }
  };
}
function confirmSheet(title, text, cta, onOk) {
  openSheet(`<h3>${esc(title)}</h3><div class="sub">${esc(text)}</div><div class="btns"><button class="btn ghost" id="cc">Cancel</button><button class="btn" id="co">${esc(cta)}</button></div>`);
  $('#sheet').onclick = null; $('#cc').onclick = closeSheet;
  $('#co').onclick = async () => { closeSheet(); try { await onOk(); } catch (e) { snack(e.message); } };
}

let dlTimer = null;
const dlBusy = () => Object.values(dls).some(s => s === 'queued' || s === 'running');
async function pollDl() {
  try {
    const prev = dls; dls = (await api('/api/downloads')) || {};
    for (const [id, s] of Object.entries(dls)) {
      if (prev[id] && prev[id] !== s && s === 'done') snack('Saved to library: ' + (knownTitle(id) || id));
      if (prev[id] && prev[id] !== s && s.startsWith('error')) snack('Download failed: ' + s.slice(7, 80));
    }
    updateNow();
  } catch (e) { console.error(e); }
  dlTimer = dlBusy() ? setTimeout(pollDl, 2000) : null;
}
const knownTitle = id => (queue.find(t => t.id === id) || recent.find(t => t.id === id) || liked.find(t => t.id === id) || {}).title;
async function download(id, quiet) {
  const s = stateOf(id); if (s === 'busy' || s === 'done') return;
  dls[id] = 'queued'; updateNow();
  try {
    const d = await send('POST', '/api/download/' + encodeURIComponent(id));
    if (d.status) dls[id] = d.status;
    updateNow(); if (!quiet) snack('Downloading…');
    if (!dlTimer) dlTimer = setTimeout(pollDl, 1500);
  } catch (e) { delete dls[id]; updateNow(); snack(e.message); }
}
function downloadAll(tracks) {
  const todo = tracks.filter(t => stateOf(t.id) !== 'done' && stateOf(t.id) !== 'busy');
  todo.forEach(t => download(t.id, true));
  snack(todo.length ? `Downloading ${todo.length} song${todo.length === 1 ? '' : 's'}…` : 'Everything is already in your library');
}

const ytURL = (kind, id) => kind === 'track' ? `https://music.youtube.com/watch?v=${id}`
  : kind === 'artist' ? `https://music.youtube.com/channel/${id}`
  : kind === 'album' ? `https://music.youtube.com/browse/${id}`
  : `https://music.youtube.com/playlist?list=${String(id).replace(/^VL/, '')}`;
async function share(url, title) {
  if (navigator.share) {
    try { await navigator.share({ title, url }); return; } catch (e) { if (e.name === 'AbortError') return; }
  }
  try { await navigator.clipboard.writeText(url); snack('Link copied'); }
  catch { window.prompt('Copy this link', url); }
}

// ============================================================================
// Sheets
// ============================================================================
function openSheet(html) {
  const s = $('#sheet'); s.innerHTML = '<div class="grab"></div>' + html;
  $('#scrim').classList.add('on'); s.classList.add('on');
}
function closeSheet() { $('#sheet').classList.remove('on'); $('#scrim').classList.remove('on'); }

function trackMenu(t, ctx = {}) {
  const liked_ = likedIds.has(t.id), st = stateOf(t.id);
  const act = (a, icon, label, cls = '') => `<button class="act ${cls}" data-a="${a}">${ic(icon)}${label}</button>`;
  openSheet(`<div class="head"><img src="${img(t.thumb)}" alt=""><div><div class="t">${esc(t.title)}</div><div class="a"><span>${esc(t.artist || '')}</span></div></div></div>
    ${act('radio', 'radio', 'Start radio')}
    ${act('next', 'playnext', 'Play next')}
    ${act('queue', 'addq', 'Add to queue')}
    ${act('like', liked_ ? 'heart' : 'heart0', liked_ ? 'Remove from Liked songs' : 'Like', 'accent')}
    ${act('pl', 'lib', 'Add to playlist')}
    ${act('dl', st === 'done' ? 'check' : 'dl', st === 'done' ? 'In your library' : st === 'busy' ? 'Downloading…' : 'Download to library')}
    ${t.artistId ? act('artist', 'user', 'Go to artist') : ''}
    ${t.albumId ? act('album', 'album', 'Go to album') : ''}
    ${act('share', 'share', 'Share link')}
    ${ctx.queueIndex !== undefined ? act('rmq', 'close', 'Remove from queue') : ''}
    ${ctx.playlistId && ctx.playlistId !== 'liked' ? act('rmp', 'del', 'Remove from this playlist') : ''}`);
  $('#sheet').onclick = async e => {
    const b = e.target.closest('[data-a]'); if (!b) return;
    const a = b.dataset.a; closeSheet();
    if (a === 'radio') startRadio(t);
    if (a === 'next') { queue.splice(idx + 1, 0, T(t)); persist(); renderQueueList(); snack('Playing next'); }
    if (a === 'queue') { queue.push(T(t)); persist(); renderQueueList(); snack('Added to queue'); }
    if (a === 'like') toggleLike(t);
    if (a === 'pl') playlistPicker([t], t.title);
    if (a === 'dl') { if (st === 'err') delete dls[t.id]; download(t.id); }
    if (a === 'artist') { closeFp(); go('/artist/' + t.artistId); }
    if (a === 'album') { closeFp(); go('/album/' + t.albumId); }
    if (a === 'share') share(ytURL('track', t.id), t.title);
    if (a === 'rmq') removeFromQueue(ctx.queueIndex);
    if (a === 'rmp') { try { await send('DELETE', `/api/playlists/${ctx.playlistId}/tracks/${t.id}`); snack('Removed'); refreshPlaylists(); route(); } catch (err) { snack(err.message); } }
  };
}
function removeFromQueue(i) {
  queue.splice(i, 1);
  if (i < idx) idx--; else if (i === idx) { audio.pause(); audio.removeAttribute('src'); idx = Math.min(idx, queue.length - 1); }
  persist(); renderQueueList(); updateNow();
}

async function accountSheet() {
  try { acc = await api('/api/account'); } catch {}
  openSheet(`<h3>Account</h3><div class="sub">Signed in as <b>${esc(acc.user)}</b>${acc.defaultCreds ? ' — still using the default password' : ''}</div>
    <form id="accForm" autocomplete="off">
      <input type="password" name="current" placeholder="Current password" autocomplete="current-password" required>
      <input type="text" name="user" placeholder="New username (optional)" value="${esc(acc.user)}" autocapitalize="none" spellcheck="false">
      <input type="password" name="password" placeholder="New password (min. 8 characters)" autocomplete="new-password" minlength="8" required>
      <div class="msg" id="accMsg"></div>
      <div class="btns"><button type="button" class="btn ghost" id="logout">${ic('out')}Sign out</button><button class="btn" type="submit">Save</button></div>
    </form>`);
  $('#sheet').onclick = null;
  $('#logout').onclick = async () => { await fetch('/logout', { method: 'POST' }); location.href = '/login'; };
  $('#accForm').onsubmit = async e => {
    e.preventDefault(); const f = e.target, m = $('#accMsg'); m.className = 'msg'; m.textContent = '';
    try {
      acc = await send('POST', '/api/account', { current: f.current.value, user: f.user.value, password: f.password.value });
      m.className = 'msg ok'; m.textContent = 'Credentials updated.'; paintAccount(); setTimeout(closeSheet, 900);
    } catch (err) { m.textContent = err.message; }
  };
}
function paintAccount() {
  $$('.avatar').forEach(a => a.textContent = (acc.user || 'M')[0].toUpperCase());
  const su = $('#sideUser'); if (su) su.textContent = acc.user || 'Account';
  const b = $('#banner'); if (b) b.innerHTML = bannerHTML();
}

// ============================================================================
// Player
// ============================================================================
function persist() { store.set('mw.queue', queue.slice(0, 300)); store.set('mw.idx', idx); }

function playAt(i) {
  if (i < 0 || i >= queue.length) return;
  idx = i;
  const t = queue[i];
  audio.src = `/api/stream/${t.id}?fmt=${fmt}`;
  audio.play().catch(e => console.warn(e));
  document.title = `${t.title} · metroweb`;
  recent = [T(t), ...recent.filter(x => x.id !== t.id)].slice(0, 40); store.set('mw.history', recent);
  if ('mediaSession' in navigator) {
    navigator.mediaSession.metadata = new MediaMetadata({ title: t.title, artist: t.artist, album: t.album,
      artwork: t.thumb ? [{ src: new URL(img(t.thumb), location.href).href, sizes: '512x512' }] : [] });
  }
  themeFrom(t.thumb);
  persist(); renderQueueList(); updateNow();
  if (idx >= queue.length - 2) topUpRadio();
}
function playList(items, i) { queue = items.map(T); idx = -1; playAt(i); }
function playNow(t) { queue.splice(idx + 1, 0, T(t)); playAt(idx + 1); }
async function topUpRadio() {
  if (fetchingRadio || idx < 0 || !cur()) return;
  fetchingRadio = true;
  try {
    const data = await api('/api/radio/' + cur().id);
    const have = new Set(queue.map(t => t.id));
    queue.push(...data.filter(t => !have.has(t.id)).slice(0, 20).map(T));
    persist(); renderQueueList();
  } catch (e) { console.error(e); }
  fetchingRadio = false;
}
async function startRadio(t) {
  snack('Starting radio…');
  try {
    const data = await api('/api/radio/' + t.id);
    playList([T(t), ...data.filter(x => x.id !== t.id)], 0);
  } catch (e) { snack('Could not start radio: ' + e.message); }
}
function next(auto) {
  if (!queue.length) return;
  if (auto && repeat === 2) { audio.currentTime = 0; audio.play(); return; }
  if (shuffle && queue.length > 1) { let n; do { n = Math.floor(Math.random() * queue.length); } while (n === idx); playAt(n); return; }
  if (idx < queue.length - 1) playAt(idx + 1);
  else if (repeat === 1) playAt(0);
  else if (auto) { audio.pause(); updateNow(); }
}
function prev() { if (audio.currentTime > 3) audio.currentTime = 0; else if (idx > 0) playAt(idx - 1); else audio.currentTime = 0; }
function toggle() { if (!audio.src) { if (queue.length) playAt(Math.max(idx, 0)); return; } audio.paused ? audio.play() : audio.pause(); }

// dynamic accent from artwork
const colorCache = new Map();
async function themeFrom(url) {
  if (!url) return;
  try {
    let c = colorCache.get(url);
    if (!c) {
      const im = new Image(); im.crossOrigin = 'anonymous'; im.src = img(url); await im.decode();
      const cv = document.createElement('canvas'); cv.width = cv.height = 24;
      const x = cv.getContext('2d', { willReadFrequently: true }); x.drawImage(im, 0, 0, 24, 24);
      const d = x.getImageData(0, 0, 24, 24).data;
      let r = 0, g = 0, b = 0, w = 0;
      for (let i = 0; i < d.length; i += 4) {
        const mx = Math.max(d[i], d[i+1], d[i+2]), mn = Math.min(d[i], d[i+1], d[i+2]), wt = 0.15 + (mx - mn) / 255;
        r += d[i] * wt; g += d[i+1] * wt; b += d[i+2] * wt; w += wt;
      }
      c = toHSL(r / w, g / w, b / w); colorCache.set(url, c);
    }
    const [h, s] = c, sat = Math.min(Math.max(s, .25), .75), root = document.documentElement.style;
    root.setProperty('--accent', `hsl(${h} ${sat * 100}% 80%)`);
    root.setProperty('--on-accent', `hsl(${h} ${sat * 100}% 14%)`);
    root.setProperty('--glow', `hsl(${h} ${sat * 55}% 24%)`);
  } catch {}
}
function toHSL(r, g, b) {
  r /= 255; g /= 255; b /= 255;
  const mx = Math.max(r, g, b), mn = Math.min(r, g, b), l = (mx + mn) / 2, d = mx - mn;
  let h = 0, s = 0;
  if (d) { s = d / (1 - Math.abs(2 * l - 1)); h = mx === r ? ((g - b) / d) % 6 : mx === g ? (b - r) / d + 2 : (r - g) / d + 4; h = Math.round(h * 60); if (h < 0) h += 360; }
  return [h, s, l];
}

// keeps the player UI and visible rows in sync
function updateNow() {
  const t = cur(), pl = playing();
  document.body.classList.toggle('has-mini', !!t);
  $('#mini').classList.toggle('show', !!t);
  if (t) {
    $('#miniImg').src = img(t.thumb); $('#miniT').textContent = t.title; $('#miniA').textContent = t.artist || '';
    $('#fpImg').src = img(t.thumb); $('#fpT').textContent = t.title;
    const fa = $('#fpA'); fa.textContent = [t.artist, t.album].filter(Boolean).join(' • '); fa.disabled = !t.artistId;
  }
  const pi = pl ? 'pause' : 'play';
  for (const id of ['#miniPlay', '#play', '#dPlay']) $(id).innerHTML = ic(pi);
  $('#fp').classList.toggle('playing', pl); $('#fp').classList.toggle('paused', !pl);
  const fav = t && likedIds.has(t.id);
  for (const sel of ['#fpFav', '#dFav']) { $(sel).innerHTML = ic(fav ? 'heart' : 'heart0'); $(sel).style.color = fav ? 'var(--accent)' : ''; $(sel).setAttribute('aria-label', fav ? 'Unlike' : 'Like'); }
  const st = t ? stateOf(t.id) : '';
  const dlIcon = st === 'done' ? 'check' : st === 'busy' ? 'clock' : st === 'err' ? 'warn' : 'dl';
  const lbl = st === 'done' ? 'In library' : st === 'busy' ? 'Downloading…' : st === 'err' ? 'Retry' : 'Download';
  $('#fpDl').innerHTML = ic(dlIcon, 20) + lbl; $('#fpDl').className = st === 'done' ? 'done' : '';
  $('#dDl').innerHTML = ic(dlIcon); $('#dDl').style.color = st === 'done' ? 'var(--ok)' : st === 'err' ? 'var(--err)' : '';
  for (const sel of ['#shuf', '#dShuf']) { $(sel).innerHTML = ic('shuf'); $(sel).classList.toggle('on', shuffle); $(sel).style.color = shuffle ? 'var(--accent)' : ''; }
  for (const sel of ['#rep', '#dRep']) { $(sel).innerHTML = ic(repeat === 2 ? 'rep1' : 'rep'); $(sel).classList.toggle('on', repeat > 0); $(sel).style.color = repeat ? 'var(--accent)' : ''; }
  // rows in the current view
  $$('#view .row[data-id]').forEach(r => {
    const id = r.dataset.id, l = r.dataset.l, i = +r.dataset.i;
    const isCur = t && t.id === id && (l !== 'queue' || i === idx);
    r.classList.toggle('cur', !!isCur); r.classList.toggle('playing', !!isCur && pl);
    const a = r.querySelector('.a'); if (!a) return;
    const old = a.querySelector('.st'); if (old) old.remove();
    const h = stIcon(id); if (h) a.insertAdjacentHTML('afterbegin', h);
  });
}

function openFp() { if (!cur()) return; $('#fp').classList.add('open'); $('#fp').setAttribute('aria-hidden', 'false'); }
function closeFp() { const f = $('#fp'); f.classList.remove('open'); f.style.transform = ''; f.setAttribute('aria-hidden', 'true'); }

// ============================================================================
// Events
// ============================================================================
function itemOf(el) {
  const l = el.dataset.l, i = +el.dataset.i, L = V.lists[l];
  return L ? { it: L.items[i], mode: L.mode, items: L.items, i, l } : null;
}
document.addEventListener('click', e => {
  const a = e.target.closest('[data-acc]'); if (a) { accountSheet(); return; }
  if (e.target.closest('[data-back]')) { goBack(); return; }
  if (e.target.closest('[data-retry]')) { route(); return; }
  if (e.target.closest('[data-newpl]')) { promptSheet('New playlist', '', 'Create', async name => { await send('POST', '/api/playlists', { name }); await refreshPlaylists(); snack('Playlist created'); route(); }); return; }
  const ar = e.target.closest('[data-artist]'); if (ar) { go('/artist/' + ar.dataset.artist); return; }

  const card = e.target.closest('[data-card]');
  if (card) {
    const r = itemOf(card); if (!r || !r.it) return;
    const it = r.it;
    if (it.local) { go('/playlist/' + it.local); return; }
    if (it.type === 'artist') go('/artist/' + it.id);
    else if (it.type === 'album') go('/album/' + it.id);
    else if (it.type === 'playlist') { routeState.set(it.id, { title: it.title, thumb: it.thumb }); go('/yt/' + it.id); }
    else if (it.type === 'mix') startMix(it);
    else playNow(it);
    return;
  }
  const row = e.target.closest('#view .row[data-id]');
  if (row) {
    const r = itemOf(row); if (!r || !r.it) return;
    if (e.target.closest('[data-more]')) {
      const ctx = {};
      if (r.l === 'queue') ctx.queueIndex = r.i;
      if (routeName === 'playlist') ctx.playlistId = V.playlistId;
      trackMenu(r.it, ctx); return;
    }
    if (r.mode === 'queue') { if (r.l === 'queue') playAt(r.i); else playList(r.items, r.i); } else playNow(r.it);
  }
});
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') { if ($('#sheet').classList.contains('on')) closeSheet(); else closeFp(); return; }
  const row = e.target.closest && e.target.closest('.row[data-id]');
  if (row && (e.key === 'Enter' || e.key === ' ') && e.target === row) { e.preventDefault(); row.click(); return; }
  if (e.target.tagName === 'INPUT' && e.target.type !== 'range') return;
  if (e.code === 'Space') { e.preventDefault(); toggle(); }
  if (e.key === 'ArrowRight' && e.shiftKey) next();
  if (e.key === 'ArrowLeft' && e.shiftKey) prev();
  if (e.key === '/') { e.preventDefault(); go('/search'); }
});
async function startMix(it) {
  snack('Starting mix…');
  try { const r = await api('/api/mix?id=' + encodeURIComponent(it.id)); playList(r.map(T), 0); } catch (e) { snack('Could not start mix: ' + e.message); }
}

$$('[data-nav]').forEach(b => b.addEventListener('click', () => go('/' + (b.dataset.nav === 'home' ? '' : b.dataset.nav))));
$('#scrim').onclick = closeSheet;
for (const id of ['play', 'miniPlay', 'dPlay']) $('#' + id).addEventListener('click', e => { e.stopPropagation(); toggle(); });
for (const id of ['next', 'miniNext', 'dNext']) $('#' + id).addEventListener('click', e => { e.stopPropagation(); next(); });
for (const id of ['prev', 'dPrev']) $('#' + id).addEventListener('click', prev);
for (const id of ['shuf', 'dShuf']) $('#' + id).addEventListener('click', () => { shuffle = !shuffle; store.set('mw.shuffle', shuffle); updateNow(); });
for (const id of ['rep', 'dRep']) $('#' + id).addEventListener('click', () => { repeat = (repeat + 1) % 3; store.set('mw.repeat', repeat); updateNow(); snack(['Repeat off', 'Repeat queue', 'Repeat song'][repeat]); });
for (const id of ['fpFav', 'dFav']) $('#' + id).addEventListener('click', () => toggleLike(cur()));
for (const id of ['fpDl', 'dDl']) $('#' + id).addEventListener('click', () => { const t = cur(); if (t) { if (stateOf(t.id) === 'err') delete dls[t.id]; download(t.id); } });
$('#fpMore').onclick = () => { if (cur()) trackMenu(cur(), {}); };
$('#fpPl').onclick = () => { if (cur()) playlistPicker([cur()], cur().title); };
$('#fpQueue').onclick = () => { closeFp(); go('/queue'); };
$('#dQueue').onclick = () => go('/queue');
$('#fpA').onclick = () => { const t = cur(); if (t && t.artistId) { closeFp(); go('/artist/' + t.artistId); } };
$('#miniInfo').onclick = openFp; $('#miniInfo').addEventListener('keydown', e => { if (e.key === 'Enter') openFp(); });
$('#dExpand').onclick = openFp; $('#fpClose').onclick = closeFp;

// swipe down closes the full player; swipe up on the mini player opens it
(() => {
  const fp = $('#fp'); let y0 = null, dy = 0;
  fp.addEventListener('touchstart', e => { if (e.target.closest('input')) return; y0 = e.touches[0].clientY; dy = 0; }, { passive: true });
  fp.addEventListener('touchmove', e => { if (y0 === null) return; dy = e.touches[0].clientY - y0; if (dy > 0) { fp.classList.add('drag'); fp.style.transform = `translateY(${dy}px)`; } }, { passive: true });
  fp.addEventListener('touchend', () => { if (y0 === null) return; fp.classList.remove('drag'); if (dy > 120) closeFp(); else fp.style.transform = ''; y0 = null; });
  const mini = $('#mini'); let my = null;
  mini.addEventListener('touchstart', e => { my = e.touches[0].clientY; }, { passive: true });
  mini.addEventListener('touchend', e => { if (my !== null && my - e.changedTouches[0].clientY > 40) openFp(); my = null; });
})();

// audio
audio.addEventListener('play', updateNow); audio.addEventListener('pause', updateNow);
audio.addEventListener('ended', () => next(true));
audio.addEventListener('error', () => { if (!audio.src) return; snack('Playback failed, skipping…'); setTimeout(() => next(), 1200); });
let seeking = false;
function paintBars() {
  const p = audio.duration ? audio.currentTime / audio.duration : 0;
  $('#cur').textContent = $('#dCur').textContent = time(audio.currentTime);
  $('#dur').textContent = $('#dDur').textContent = time(audio.duration);
  $('#miniProg').style.width = (p * 100) + '%';
  if (!seeking) for (const id of ['bar', 'dBar']) { const b = $('#' + id); b.value = p * 1000; b.style.setProperty('--p', (p * 100) + '%'); }
}
audio.addEventListener('timeupdate', () => {
  paintBars();
  if ('mediaSession' in navigator && audio.duration && isFinite(audio.duration)) {
    try { navigator.mediaSession.setPositionState({ duration: audio.duration, position: audio.currentTime, playbackRate: audio.playbackRate }); } catch {}
  }
});
for (const id of ['bar', 'dBar']) {
  const b = $('#' + id);
  b.addEventListener('input', () => { seeking = true; b.style.setProperty('--p', (b.value / 10) + '%'); });
  b.addEventListener('change', () => { if (audio.duration) audio.currentTime = b.value / 1000 * audio.duration; seeking = false; paintBars(); });
}
$('#vol').addEventListener('input', e => { audio.volume = e.target.value / 100; e.target.style.setProperty('--p', e.target.value + '%'); store.set('mw.vol', e.target.value); });
$('#vol').value = store.get('mw.vol', 80); audio.volume = $('#vol').value / 100; $('#vol').style.setProperty('--p', $('#vol').value + '%');

if ('mediaSession' in navigator) {
  navigator.mediaSession.setActionHandler('play', () => audio.play());
  navigator.mediaSession.setActionHandler('pause', () => audio.pause());
  navigator.mediaSession.setActionHandler('nexttrack', () => next());
  navigator.mediaSession.setActionHandler('previoustrack', prev);
  try { navigator.mediaSession.setActionHandler('seekto', d => { audio.currentTime = d.seekTime; }); } catch {}
}

// ============================================================================
// Boot
// ============================================================================
function paintStatic() {
  $$('[data-ic]').forEach(e => { e.innerHTML = ic(e.dataset.ic); e.style.display = 'inline-flex'; });
  const nav = { home: ['home', 'Home'], search: ['search', 'Search'], library: ['lib', 'Library'], queue: ['queue', 'Queue'] };
  $$('#nav button').forEach(b => b.querySelector('.pill').innerHTML = ic(nav[b.dataset.nav][0]));
  $$('#side button[data-nav]').forEach(b => b.innerHTML = ic(nav[b.dataset.nav][0]) + nav[b.dataset.nav][1]);
  $('#sideAcc').innerHTML = ic('user') + '<span id="sideUser">Account</span>';
  $('#sideAcc').onclick = accountSheet;
  $('#fpClose').innerHTML = ic('down', 30); $('#fpMore').innerHTML = ic('more');
  $('#miniNext').innerHTML = ic('next'); $('#dPrev').innerHTML = ic('prev'); $('#dNext').innerHTML = ic('next');
  $('#prev').innerHTML = ic('prev', 34); $('#next').innerHTML = ic('next', 34);
  $('#dExpand').innerHTML = ic('expand'); $('#dQueue').innerHTML = ic('queue');
  $('#fpQueue').innerHTML = ic('queue', 20) + 'Queue'; $('#fpPl').innerHTML = ic('lib', 20) + 'Playlist';
}
async function migrateLocalFavs() {
  const old = store.get('mw.favs', null);
  if (!old || store.get('mw.favsMigrated', false)) return;
  const list = Object.values(old).filter(t => t && t.id && t.title);
  if (list.length) { try { await send('POST', '/api/playlists/liked/tracks', { tracks: list.map(T) }); } catch { return; } }
  store.set('mw.favsMigrated', true);
}
(async function boot() {
  paintStatic(); updateNow();
  api('/api/account').then(a => { acc = a; paintAccount(); }).catch(() => {});
  await migrateLocalFavs();
  await Promise.all([loadLiked(), refreshPlaylists()]);
  pollDl();
  if (cur()) themeFrom(cur().thumb);
  route(); updateNow();
})();
