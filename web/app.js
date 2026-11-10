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
  up: 'M7.41 15.41L12 10.83l4.59 4.58L18 14l-6-6-6 6z',
  down: 'M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z', close: 'M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z',
  shuf: 'M10.59 9.17L5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41l-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z',
  rep: 'M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z',
  rep1: 'M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4zm-4-2V9h-1l-2 1v1h1.5v4H13z',
  user: 'M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z',
  out: 'M17 7l-1.41 1.41L18.17 11H8v2h10.17l-2.58 2.58L17 17l5-5zM4 5h8V3H4c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h8v-2H4V5z',
  vol: 'M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z',
  expand: 'M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z',
  warn: 'M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z',
  moon: 'M12.1 22c-5.5 0-10-4.5-10-10 0-4.3 2.7-8 6.6-9.4.5-.2 1 0 1.2.4.2.4.1.9-.2 1.2-.9.9-1.4 2.1-1.4 3.3 0 2.6 2.1 4.7 4.7 4.7 1.2 0 2.4-.5 3.3-1.4.3-.3.8-.4 1.2-.2.4.2.6.7.4 1.2-1.5 3.3-4.9 5.2-8.4 5.2z',
  grip: 'M9 4h2v2H9zm4 0h2v2h-2zM9 9h2v2H9zm4 0h2v2h-2zM9 14h2v2H9zm4 0h2v2h-2zM9 19h2v2H9zm4 0h2v2h-2z',
  mute: 'M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z',
  speed: 'M20.38 8.57l-1.23 1.85a8 8 0 0 1-.22 7.58H5.07A8 8 0 0 1 15.58 6.85l1.85-1.23A10 10 0 0 0 3.35 19a2 2 0 0 0 1.72 1h13.85a2 2 0 0 0 1.74-1 10 10 0 0 0-.27-10.44zm-9.79 6.84a2 2 0 0 0 2.83 0l5.66-8.49-8.49 5.66a2 2 0 0 0 0 2.83z',
  cloud: 'M19.35 10.04A7.49 7.49 0 0 0 12 4C9.11 4 6.6 5.64 5.35 8.04A5.994 5.994 0 0 0 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM17 13l-5 5-5-5h3V9h4v4h3z',
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
  stats: 'M5 9.2h3V19H5zM10.6 5h2.8v14h-2.8zm5.6 8H19v6h-2.8z',
  menu: 'M3 18h18v-2H3v2zm0-5h18v-2H3v2zm0-7v2h18V6H3z',
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
let origQueue = store.get('mw.orig', null);     // the queue in its original order, while shuffled
let liked = [];                                  // Track[] (oldest first)
let likedIds = new Set();
let playlists = [];                              // summaries, incl. "liked"
let dls = {};                                    // id -> queued|running|done|error...
let owned = new Set();                           // song ids already in the Jellyfin library
const ownedChecked = new Set();
let jfEnabled = false;
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
  if (owned.has(id)) return 'owned';
  return '';
}
function stIcon(id) {
  if (offTracks[id]) return `<span class="st off" title="Saved for offline">${ic('cloud')}</span>`;
  const s = stateOf(id);
  if (s === 'busy') return `<span class="st" title="Downloading…">${ic('clock')}</span>`;
  if (s === 'done') return `<span class="st" title="Downloaded to your library">${ic('check')}</span>`;
  if (s === 'owned') return `<span class="st own" title="Already in your Jellyfin library">${ic('check')}</span>`;
  if (s === 'err') return `<span class="st err" title="${esc(dls[id])}">${ic('warn')}</span>`;
  return '';
}

// A song row. o.num shows a track number instead of artwork.
function rowHTML(t, l, i, o = {}) {
  const c = cur(), isCur = c && c.id === t.id && (l !== 'queue' || i === idx);
  const sub = [t.artist, o.noAlbum ? '' : t.album].filter(Boolean).join(' • ');
  return `<div class="row${o.grip ? ' hasgrip' : ''}${isCur ? ' cur' : ''}${isCur && playing() ? ' playing' : ''}" data-l="${l}" data-i="${i}" data-id="${esc(t.id)}" role="button" tabindex="0">
    ${o.num ? `<span class="num">${o.num}</span>` : `<img class="art" loading="lazy" src="${img(t.thumb)}" alt="">`}
    <span class="eq"><i></i><i></i><i></i></span>
    <div class="meta"><div class="t">${esc(t.title)}</div><div class="a">${stIcon(t.id)}<span>${esc(sub)}</span></div></div>
    <span class="dur"${o.right !== undefined ? ' style="color:var(--accent);font-weight:700"' : ''}>${esc(o.right ?? t.duration)}</span>
    <button class="iconbtn" data-more aria-label="More options">${ic('more')}</button>
    ${o.grip ? `<span class="grip" data-grip title="Drag to reorder" aria-label="Drag to reorder">${ic('grip')}</span>` : ''}
  </div>`;
}
const songRows = (items, l, o) => items.map((t, i) => rowHTML(t, l, i, o)).join('');

function cardHTML(it, l, i) {
  const round = it.type === 'artist';
  const img_ = it.local === 'liked' ? `<div class="img likedcov">${ic('heart', 52)}</div>` : it.thumb ? `<img class="img" loading="lazy" src="${img(it.thumb)}" alt="">`
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
  [/^\/stats$/, 'stats', viewStats],
  [/^\/queue$/, 'queue', viewQueue],
  [/^\/artist\/([\w-]+)$/, 'artist', viewArtist],
  [/^\/album\/([\w-]+)$/, 'album', viewAlbum],
  [/^\/yt\/([\w-]+)$/, 'yt', viewYT],
  [/^\/playlist\/([\w-]+)$/, 'playlist', viewPlaylist],
  [/^\/mix\/([\w-]+)$/, 'mix', viewForYou],
  [/^\/status$/, 'status', viewStatus],
  [/^\/offline$/, 'offline', viewOffline],
];
const TAB_OF = { home: 'home', search: 'search', library: 'library', stats: 'stats', queue: 'queue', playlist: 'library', mix: 'home', offline: 'library' };

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
const greeting = () => { const h = new Date().getHours(); return h < 5 ? 'Still up?' : h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'; };
// one-tap starters: Liked songs (shuffled), your first mixes and playlists
function jumpTiles(mixes) {
  const t = [];
  if (liked.length) t.push({ k: 'liked', title: 'Liked songs', sub: `${liked.length} songs`, likedcov: true });
  mixes.slice(0, 3).forEach(m => t.push({ k: 'mix:' + m.id, title: m.title, thumb: m.thumb, mix: m }));
  playlists.filter(p => p.id !== 'liked' && p.count > 0).slice(0, 2).forEach(p => t.push({ k: 'pl:' + p.id, title: p.name, thumb: p.thumb, pl: p }));
  return t.slice(0, t.length >= 6 ? 6 : t.length - (t.length % 2));   // keep the 2-column grid even
}
function paintJump(c, mixes) {
  if (!c.alive()) return;
  const tiles = jumpTiles(mixes), el = $('#hJump'); if (!el) return;
  el.innerHTML = tiles.length ? `<div class="qgrid">${tiles.map((x, i) => `<button class="qtile" data-q="${i}">${x.likedcov ? `<div class="ph likedcov">${ic('heart', 26)}</div>` : x.thumb ? `<img src="${img(x.thumb)}" alt="">` : `<div class="ph">${ic('music')}</div>`}<span>${esc(x.title)}</span></button>`).join('')}</div>` : '';
  el.onclick = async e => {
    const b = e.target.closest('[data-q]'); if (!b) return;
    const x = tiles[+b.dataset.q];
    try {
      if (x.k === 'liked') playShuffled(liked.map(T));
      else if (x.mix) playList(x.mix.tracks.map(T), 0);
      else if (x.pl) { const p = await api('/api/playlists/' + x.pl.id); if (p.tracks.length) playList(p.tracks.map(T), 0); }
    } catch (err) { snack(err.message); }
  };
}
async function viewHome(c) {
  const last = recent[0] || liked[liked.length - 1];
  c.set(`<div class="top"><a class="search" href="#/search">${ic('search')}<span>Search songs, artists, albums…</span></a>${avatarBtn()}</div>
    <div id="banner">${bannerHTML()}</div>
    <div class="greet"><h1>${greeting()}</h1></div><div id="hJump"></div>
    <div id="hMixes"></div><div id="hQuick"></div><div id="hRecent"></div><div id="hLists"></div><div id="hBrowse">${section('Browse', skeletonCards(6))}</div>`);
  const own = playlists.filter(p => p.id !== 'liked' || p.count > 0);
  if (c.alive()) {
    const ul = own.map(p => ({ type: 'playlist', id: 'local:' + p.id, title: p.name, subtitle: `${p.count} song${p.count === 1 ? '' : 's'}`, thumb: p.thumb, local: p.id }));
    $('#hLists').innerHTML = ul.length ? section('Your library', hscroll(ul, reg('hl', ul))) : '';
    if (recent.length) $('#hRecent').innerHTML = section('Recently played', hscroll(recent.slice(0, 12).map(x => ({ ...x, type: 'song', subtitle: x.artist })), reg('hr', recent.slice(0, 12), 'one')));
  }
  // mixes built from your own listening history and Liked songs
  cached(`/api/mixes?tz=${-new Date().getTimezoneOffset()}`, 10 * 60 * 1000).then(mx => {
    if (!c.alive() || !mx.length) return;
    const items = mx.map(m => ({ type: 'playlist', id: 'mix:' + m.id, mixId: m.id, title: m.title, subtitle: m.subtitle, thumb: m.thumb }));
    $('#hMixes').innerHTML = section('Mixes for you', hscroll(items, reg('hm', items)));
    paintJump(c, mx);
  }).catch(() => {});
  paintJump(c, []);
  if (last) {
    $('#hQuick').innerHTML = section('Because you listen', skeletonRows(5));
    try {
      const radio = await cached('/api/radio/' + last.id);
      if (!c.alive()) return;
      const picks = radio.filter(x => x.id !== last.id).slice(0, 8);
      $('#hQuick').innerHTML = picks.length ? section('Because you listen', `<div class="sub" style="color:var(--dim);font-size:13px;padding:0 4px 6px">Because you played ${esc(last.title)}</div>` + songRows(picks, reg('hq', picks, 'one'))) : '';
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
    if (act === 'shuffle') { if (all.length) playShuffled(all); }
    if (act === 'radio') {
      if (!a.radioId) { if (all.length) startRadio(all[0]); return; }
      snack('Starting radio…');
      try { const r = await api(`/api/mix?id=${encodeURIComponent(a.radioId)}&p=${encodeURIComponent(a.radioParams || '')}&v=${encodeURIComponent(a.radioVideoId || '')}`); playList(r.map(T), 0, true); }
      catch (e) { snack('Could not start radio: ' + e.message); }
    }
    if (act === 'share') share(ytURL('artist', a.id), a.name);
  };
  bindActions(c);
}

// ---------- album & YouTube playlists ----------
function tracksHeader(o) {
  return `<div class="head2">${backBtn()}
    ${o.thumb ? `<img class="cov" src="${img(o.thumb)}" alt="">` : `<div class="cov ph${o.liked ? ' likedcov' : ''}">${ic(o.icon || 'music', o.liked ? 72 : 24)}</div>`}
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
async function viewForYou(c) {
  c.set(`<div class="head2">${backBtn()}<div class="cov skeleton"></div></div>${skeletonRows(6)}`);
  const m = (await cached(`/api/mixes?tz=${-new Date().getTimezoneOffset()}`, 10 * 60 * 1000)).find(x => x.id === c.id);
  if (!c.alive()) return;
  if (!m) { c.set(topBar('Mix') + emptyHTML('music', 'This mix is not available right now. It is rebuilt from your listening, so it may come back later.', '<a class="btn" href="#/">Home</a>')); return; }
  const tracks = m.tracks.map(T);
  c.set(tracksHeader({ title: m.title, thumb: m.thumb, sub: `${esc(m.subtitle)} • ${tracks.length} songs`, icon: 'music', buttons: listButtons().replace(/<button class="iconbtn" data-act="share"[^>]*>.*?<\/button>/, '') }) +
    songRows(tracks, reg('t', tracks, 'queue')));
  c.actions = act => listAction(act, tracks, m.title, null);
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
  <button class="iconbtn" data-act="offall" aria-label="Save for offline" title="Save for offline">${ic('cloud')}</button>
  <button class="iconbtn" data-act="share" aria-label="Share link">${ic('share')}</button>${extra}`;
function listAction(act, tracks, title, url) {
  if (act === 'play' && tracks.length) playList(tracks, 0);
  if (act === 'shuffle' && tracks.length) playShuffled(tracks);
  if (act === 'addpl') playlistPicker(tracks, title);
  if (act === 'dlall') downloadAll(tracks);
  if (act === 'offall') saveOffline(tracks.map(T));
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
  c.set(`<div class="top"><h1>Library</h1><button class="btn tonal" data-import>${ic('dl')}Import</button><button class="btn tonal" data-newpl>${ic('add')}New playlist</button>${avatarBtn()}</div>
    ${Object.keys(offTracks).length ? `<div class="chips"><a class="chip" href="#/offline">${ic('cloud', 18)}Offline songs (${Object.keys(offTracks).length})</a></div>` : ''}
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
  const tools = tracks.length > 8 ? `<div class="tools"><label class="search">${ic('search')}<input id="tf" type="search" placeholder="Search in this playlist" autocomplete="off" aria-label="Search in this playlist"></label>
      <div class="chips" id="tsort" role="tablist">${[['added', 'Added'], ['title', 'A–Z'], ['artist', 'Artist']].map(([k, n], i) => `<button class="chip${i ? '' : ' on'}" data-sort="${k}" role="tab">${n}</button>`).join('')}</div></div>` : '';
  c.set(tracksHeader({ title: p.name, thumb: isLiked ? '' : tracks[0] && tracks[0].thumb, liked: isLiked, icon: isLiked ? 'heart' : 'music', sub: `${tracks.length} song${tracks.length === 1 ? '' : 's'}`, buttons: listButtons(more).replace(/<button class="iconbtn" data-act="addpl"[^>]*>.*?<\/button>/, '') }) + tools +
    (tracks.length ? `<div id="tl">${songRows(tracks, reg('t', tracks, 'queue'))}</div>` : emptyHTML(isLiked ? 'heart' : 'music', isLiked ? 'Songs you like will show up here' : 'No songs yet. Use the ⋮ menu on any song to add it here.')));
  V.playlistId = p.id;
  let shown = tracks, sortKey = 'added', filt = '';
  const paint = () => {
    const q = filt.trim().toLowerCase();
    shown = tracks.filter(t => !q || `${t.title} ${t.artist} ${t.album || ''}`.toLowerCase().includes(q));
    if (sortKey === 'title') shown = [...shown].sort((a, b) => a.title.localeCompare(b.title));
    if (sortKey === 'artist') shown = [...shown].sort((a, b) => (a.artist || '').localeCompare(b.artist || '') || a.title.localeCompare(b.title));
    $('#tl').innerHTML = shown.length ? songRows(shown, reg('t', shown, 'queue')) : emptyHTML('search', 'No songs match your search');
  };
  if (tools) {
    $('#tf').addEventListener('input', e => { filt = e.target.value; paint(); });
    $('#tsort').addEventListener('click', e => { const b = e.target.closest('[data-sort]'); if (!b) return; sortKey = b.dataset.sort; $$('#tsort .chip').forEach(x => x.classList.toggle('on', x === b)); paint(); });
  }
  c.actions = act => {
    const tracks_ = shown;
    if (act === 'rename') promptSheet('Rename playlist', p.name, 'Save', async name => { await send('PATCH', '/api/playlists/' + p.id, { name }); await refreshPlaylists(); route(); });
    else if (act === 'delete') confirmSheet('Delete playlist?', `“${p.name}” will be deleted. The songs themselves are not affected.`, 'Delete', async () => { await send('DELETE', '/api/playlists/' + p.id); await refreshPlaylists(); snack('Playlist deleted'); go('/library'); });
    else listAction(act, tracks_.length ? tracks_ : tracks, p.name, null);
  };
  bindActions(c);
}

// ---------- server status ----------
const fmtBytes = n => n >= 1e12 ? (n / 1e12).toFixed(1) + ' TB' : n >= 1e9 ? (n / 1e9).toFixed(1) + ' GB' : n >= 1e6 ? Math.round(n / 1e6) + ' MB' : Math.max(1, Math.round(n / 1e3)) + ' KB';
const ago = u => { if (!u) return 'never'; const s = Math.max(0, Date.now() / 1000 - u); return s < 90 ? 'just now' : s < 5400 ? `${Math.round(s / 60)} min ago` : s < 129600 ? `${Math.round(s / 3600)} h ago` : `${Math.round(s / 86400)} days ago`; };
const fmtUp = s => s < 3600 ? `${Math.floor(s / 60)} min` : s < 172800 ? `${Math.floor(s / 3600)} h` : `${Math.floor(s / 86400)} days`;
async function viewStatus(c) {
  c.set(topBar('Server status') + `<div id="stBody">${skeletonRows(4)}</div>`);
  let timer;
  async function load() {
    clearTimeout(timer);
    let s; try { s = await api('/api/status'); } catch (e) { if (c.alive()) $('#stBody').innerHTML = errorHTML(e.message); return; }
    if (!c.alive()) return;
    const y = s.ytdlp, jf = s.jellyfin, low = d => d.total && d.free / d.total < 0.1;
    const tile = (label, value, sub, bad) => `<div class="stile${bad ? ' bad' : ''}"><span>${label}</span><b>${value}</b><i>${sub || ''}</i></div>`;
    $('#stBody').innerHTML = `<div class="sgrid">
      ${tile('yt-dlp', esc(y.version || 'missing'), y.running ? 'Updating…' : y.error ? 'Update failed: ' + esc(y.error) : y.updated ? `Updated ${ago(y.updated)}` : y.checked ? `Checked ${ago(y.checked)}` : y.selfCopy ? 'Self-updating' : 'Updates daily', !y.version || (y.error && !y.running))}
      ${tile('ffmpeg', esc(s.ffmpeg || 'missing'), '', !s.ffmpeg)}
      ${tile('Jellyfin', jf.enabled ? `${fmtN(jf.songs)} songs` : 'Not set up', jf.enabled ? (jf.error ? esc(jf.error) : `Indexed ${ago(jf.updated)}`) : 'Optional', jf.enabled && !!jf.error)}
      ${tile('Music folder', fmtBytes(s.library.free), 'free of ' + fmtBytes(s.library.total), low(s.library))}
      ${tile('Server data', fmtBytes(s.data.free), 'free of ' + fmtBytes(s.data.total), low(s.data))}
      ${tile('Stream cache', fmtBytes(s.cacheBytes), `${s.cacheFiles} files`)}
      ${tile('Plays recorded', fmtN(s.plays), '')}
      ${tile('Scrobbling', (s.scrobble.listenbrainz || s.scrobble.jellyfin) ? [s.scrobble.listenbrainz ? 'ListenBrainz' : '', s.scrobble.jellyfin ? 'Jellyfin' : ''].filter(Boolean).join(' + ') : 'Off', (s.scrobble.listenbrainz || s.scrobble.jellyfin) ? (s.scrobble.error ? esc(s.scrobble.error) : `${s.scrobble.sent} sent · ${s.scrobble.jfMarked} marked in Jellyfin`) : 'Set LISTENBRAINZ_TOKEN', !!s.scrobble.error)}
      ${tile('Uptime', fmtUp(s.uptime), esc(s.go))}</div>
    <div class="actions" style="padding-top:14px"><button class="btn tonal" data-st="update" ${y.running || !y.canUpdate ? 'disabled' : ''}>${ic('dl')}Update yt-dlp now</button></div>
    <h2 class="sec"><span>Backups</span></h2>
    <div class="sub" style="color:var(--dim);font-size:13px;padding:0 4px 8px">Your playlists, listening history and sync state. One is made every day and the last 14 are kept. Logins are never included.</div>
    <div class="actions"><button class="btn tonal" data-st="backup">${ic('addq')}Back up now</button><button class="btn tonal" data-st="download">${ic('dl')}Download</button></div>
    ${s.backups.map(b => `<div class="stline"><span>${esc(b.name)}</span><i>${fmtBytes(b.size)} · ${ago(b.time)}</i></div>`).join('') || '<div class="stline"><i>No backups yet</i></div>'}`;
    if (y.running) timer = setTimeout(() => { if (c.alive()) load(); }, 3000);
  }
  $('#view').onclick = async e => {
    const b = e.target.closest('[data-st]'); if (!b) return;
    try {
      if (b.dataset.st === 'update') { await send('POST', '/api/status/update-ytdlp'); snack('Updating yt-dlp…'); setTimeout(load, 800); }
      else if (b.dataset.st === 'backup') { await send('POST', '/api/backup/now'); snack('Backup saved'); load(); }
      else if (b.dataset.st === 'download') location.href = '/api/backup/download';
    } catch (err) { snack(err.message); }
  };
  await load();
}

// ---------- stats ----------
let statsCache = null;
const PALETTE = ['#d0bcff', '#efb8c8', '#a8d5a2', '#9ecaff', '#ffd180', '#80deea', '#ffab91', '#b0bec5'];
const fmtN = n => Number(n).toLocaleString('en-US');
const fmtTime = m => m >= 120 ? `${(m / 60).toFixed(m >= 6000 ? 0 : 1).replace(/\.0$/, '')} h` : `${m} min`;
const STAT_RANGES = [['30d', '30 days'], ['90d', '3 months'], ['365d', 'Year'], ['all', 'All time']];

async function viewStats(c) {
  let range = store.get('mw.statsRange', '90d'), timer;
  c.set(`<div class="top"><h1>Stats</h1>${avatarBtn()}</div>
    <div class="chips" id="srange" role="tablist">${STAT_RANGES.map(([k, n]) => `<button class="chip${k === range ? ' on' : ''}" data-r="${k}" role="tab">${n}</button>`).join('')}</div>
    <div id="sbody">${`<div class="tiles">${'<div class="tile skeleton"><div class="l1"></div><div class="l2"></div></div>'.repeat(5)}</div>`}</div>`);
  async function load() {
    clearTimeout(timer);
    let s;
    try { s = await api(`/api/stats?range=${range}&tz=${-new Date().getTimezoneOffset()}`); }
    catch (e) { if (c.alive()) $('#sbody').innerHTML = errorHTML('Could not load your stats: ' + e.message); return; }
    if (!c.alive()) return;
    $('#sbody').innerHTML = statsHTML(s);
    wireCharts(s);
    if (s.genresEnabled && s.totals.plays > 0 && !s.genres.length && s.genresPending > 0) timer = setTimeout(() => { if (c.alive()) load(); }, 15000);
  }
  $('#view').onclick = e => {
    const r = e.target.closest('[data-r]');
    if (r) { range = r.dataset.r; store.set('mw.statsRange', range); $$('#srange .chip').forEach(b => b.classList.toggle('on', b.dataset.r === range)); load(); return; }
    const ar = e.target.closest('[data-sa]');
    if (ar && ar.dataset.sa) go('/artist/' + ar.dataset.sa);
  };
  await load();
}

function statsHTML(s) {
  const t = s.totals;
  if (!t.plays) return emptyHTML('stats', 'Nothing here yet.<br>Listen to some music and your stats will show up.', '<a class="btn" href="#/search">Find music</a>');
  const peak = s.byHour.indexOf(Math.max(...s.byHour));
  const hr12 = h => `${h % 12 || 12} ${h < 12 ? 'AM' : 'PM'}`;
  const top = s.topArtists, max = top.length ? top[0].plays : 1, f = top[0];
  const pic = (a, cls = '') => a.thumb ? `<img class="${cls}" loading="lazy" src="${img(a.thumb)}" alt="">` : `<div class="ph ${cls}">${ic('user')}</div>`;
  let h = `<div class="tiles">
    <div class="tile hi"><b>${fmtN(t.plays)}</b><span>Plays</span></div>
    <div class="tile"><b>${fmtTime(t.minutes)}</b><span>Listening time</span></div>
    <div class="tile"><b>${fmtN(t.artists)}</b><span>Artists</span></div>
    <div class="tile"><b>${fmtN(t.songs)}</b><span>Songs</span></div>
    <div class="tile"><b>${t.streak}</b><span>Day streak${t.streak === 1 ? '' : 's'}</span></div></div>`;

  h += `<div class="stats-cols"><div class="panel"><h3>Top artists</h3>
    <div class="feat" data-sa="${esc(f.id || '')}">
      <div class="pic">${pic(f)}<span class="badge">1</span></div>
      <div><div class="nm">${esc(f.name)}</div><div class="mt">${fmtN(f.plays)} plays · ${fmtTime(f.minutes)}</div></div></div>
    ${top.slice(1).map((a, i) => `<div class="rank" data-sa="${esc(a.id || '')}"><span class="rk">${i + 2}</span>${pic(a)}
      <div style="min-width:0"><div class="rn">${esc(a.name)}</div><div class="bar"><i style="width:${Math.max(4, a.plays / max * 100)}%"></i></div></div>
      <span class="rv">${fmtN(a.plays)}</span></div>`).join('')}</div>`;

  const songs = s.topSongs;
  h += `<div class="panel"><h3>Top songs</h3>${(() => { const l = reg('ts', songs, 'one'); return songs.map((t, i) => rowHTML(t, l, i, { num: i + 1, right: `${fmtN(t.plays)}×` })).join(''); })()}</div></div>`;

  h += `<div class="panel"><h3>Artists over time<small>${s.granularity === 'day' ? 'per day' : s.granularity === 'week' ? 'per week' : 'per month'}</small></h3>
    <div class="chartwrap" id="areaChart"></div>
    <div class="legend">${s.series.map((x, i) => `<span><i class="dot" style="background:${PALETTE[i]}"></i><span>${esc(x.name)}</span></span>`).join('')}</div></div>`;

  h += `<div class="stats-cols"><div class="panel"><h3>Genres</h3>${genresHTML(s)}</div>
    <div class="panel"><h3>When you listen</h3>
      <div class="big-num">Most active around <b>${hr12(peak)}</b></div>
      <div class="hours" role="img" aria-label="Plays by hour of day">${s.byHour.map((v, i) => `<i class="${i === peak ? 'pk' : ''}" style="height:${Math.max(2, v / Math.max(1, s.byHour[peak]) * 100)}%" title="${hr12(i)}: ${v} plays"></i>`).join('')}</div>
      <div class="hours-x"><span>12 AM</span><span>6 AM</span><span>12 PM</span><span>6 PM</span><span>11 PM</span></div></div></div>`;

  const months = s.months.filter(m => m.plays > 0).reverse();
  h += `<h2 class="sec"><span>Month by month</span></h2><div class="hscroll">${months.map(m => `<div class="mcard" data-sa="${esc((m.topArtist && m.topArtist.id) || '')}">
      <div class="mm">${esc(monthLabel(m.key))}</div>${m.topArtist ? pic(m.topArtist) : ''}
      <div class="mn">${m.topArtist ? esc(m.topArtist.name) : ''}</div><div class="mp">${fmtN(m.plays)} plays</div></div>`).join('')}</div>`;

  if (s.topAlbums.length) {
    const items = s.topAlbums.map(a => ({ type: 'album', id: a.id, title: a.title, subtitle: `${a.artist} · ${fmtN(a.plays)} plays`, thumb: a.thumb })).filter(a => a.id);
    if (items.length) h += section('Top albums', hscroll(items, reg('talb', items)));
  }
  return h;
}
const monthLabel = key => new Date(key + '-02T12:00:00').toLocaleString('en-US', { month: 'short', year: 'numeric' });

function genresHTML(s) {
  if (!s.genres.length) {
    const msg = !s.genresEnabled ? 'Genre lookup is turned off on this server.'
      : s.genresPending > 0 ? 'Looking up genres for your artists… this page refreshes by itself.'
      : "Couldn't find genres for these artists yet.";
    return `<div class="note">${msg}</div>`;
  }
  const R = 70, C = 2 * Math.PI * R; let off = 0;
  const segs = s.genres.map((g, i) => {
    const len = Math.max(0, g.pct / 100 * C - 2), seg = `<circle cx="100" cy="100" r="${R}" fill="none" stroke="${PALETTE[i % PALETTE.length]}" stroke-width="26" stroke-dasharray="${len} ${C - len}" stroke-dashoffset="${-off}" transform="rotate(-90 100 100)"/>`;
    off += g.pct / 100 * C; return seg;
  }).join('');
  return `<div class="donut"><svg viewBox="0 0 200 200" role="img" aria-label="Genre breakdown"><circle cx="100" cy="100" r="${R}" fill="none" stroke="var(--high)" stroke-width="26"/>${segs}
      <text class="ctr2" x="100" y="94">Top genre</text><text class="ctr" x="100" y="114">${esc(s.genres[0].name.length > 14 ? s.genres[0].name.slice(0, 13) + '…' : s.genres[0].name)}</text></svg>
    <div class="glist">${s.genres.map((g, i) => `<div><i class="dot" style="background:${PALETTE[i % PALETTE.length]}"></i><span>${esc(g.name)}</span><em>${g.pct.toFixed(g.pct < 10 ? 1 : 0)}%</em></div>`).join('')}</div></div>`;
}

// Stacked, smoothed area chart with a hover/touch tooltip.
function wireCharts(s) {
  const el = $('#areaChart'); if (!el || !s.series.length || !s.timeline.length) { if (el) el.innerHTML = '<div class="note">Not enough data yet.</div>'; return; }
  let lastW = 0;
  const draw = () => { const w = Math.max(280, Math.round(el.clientWidth)); if (Math.abs(w - lastW) < 8) return; lastW = w; drawArea(s, el, w); };
  draw();
  if (window.ResizeObserver) { const ro = new ResizeObserver(() => { if (!el.isConnected) ro.disconnect(); else draw(); }); ro.observe(el); }
}
function drawArea(s, el, W) {
  const H = 210, L = 30, R = 6, TP = 10, B = 26, n = s.timeline.length;
  const x = i => n > 1 ? L + i * (W - L - R) / (n - 1) : (W + L - R) / 2;
  const totals = s.timeline.map((_, i) => s.series.reduce((a, se) => a + se.data[i], 0));
  const maxV = Math.max(1, ...totals), niceMax = Math.ceil(maxV / 4) * 4 || 4;
  const y = v => TP + (H - TP - B) * (1 - v / niceMax);
  const smooth = pts => pts.map((p, i) => i === 0 ? `M${p[0]},${p[1]}` : `C${(pts[i-1][0] + p[0]) / 2},${pts[i-1][1]} ${(pts[i-1][0] + p[0]) / 2},${p[1]} ${p[0]},${p[1]}`).join(' ');
  const cum = s.timeline.map(() => 0); let paths = '';
  s.series.forEach((se, k) => {
    const lower = cum.slice(); se.data.forEach((v, i) => cum[i] += v);
    const up = cum.map((v, i) => [x(i), y(v)]), lo = lower.map((v, i) => [x(i), y(v)]).reverse();
    const d = n === 1 ? '' : smooth(up) + ' L' + lo[0][0] + ',' + lo[0][1] + ' ' + smooth(lo).replace(/^M[^C]*/, '');
    paths += n === 1 ? `<rect x="${x(0) - 14}" y="${y(cum[0])}" width="28" height="${y(lower[0]) - y(cum[0])}" fill="${PALETTE[k]}" rx="4"/>` :
      `<path d="${d} Z" fill="${PALETTE[k]}" fill-opacity=".82"/>`;
  });
  const grid = [0, 1, 2, 3, 4].map(g => { const v = niceMax / 4 * g; return `<line x1="${L}" x2="${W - R}" y1="${y(v)}" y2="${y(v)}" stroke="var(--outline)" stroke-opacity="${g ? .35 : .8}" stroke-dasharray="${g ? '3 4' : '0'}"/><text x="${L - 6}" y="${y(v) + 3.5}" fill="var(--dim)" font-size="11" text-anchor="end">${Math.round(v)}</text>`; }).join('');
  const ticks = Math.min(n, W > 500 ? 7 : W > 360 ? 5 : 4), xl = [...new Set(Array.from({ length: ticks }, (_, k) => Math.round(k * (n - 1) / Math.max(1, ticks - 1))))]
    .map(i => `<text x="${x(i)}" y="${H - 6}" fill="var(--dim)" font-size="11" text-anchor="${i === 0 && n > 1 ? 'start' : i === n - 1 && n > 1 ? 'end' : 'middle'}">${esc(s.timeline[i].label)}</text>`).join('');
  el.innerHTML = `<svg viewBox="0 0 ${W} ${H}" role="img" aria-label="Plays per artist over time">${grid}${paths}${xl}
      <line id="cl" x1="0" x2="0" y1="${TP}" y2="${H - B}" stroke="var(--on)" stroke-opacity=".5" stroke-dasharray="3 3" visibility="hidden"/>
      <rect id="hit" x="${L}" y="0" width="${W - L - R}" height="${H}" fill="transparent"/></svg><div class="chart-tip" id="ctip"></div>`;
  const svg = el.querySelector('svg'), tip = $('#ctip'), cl = $('#cl');
  const show = ev => {
    const b = svg.getBoundingClientRect(), px = (ev.clientX - b.left) / b.width * W;
    const i = Math.max(0, Math.min(n - 1, Math.round((px - L) / ((W - L - R) / Math.max(1, n - 1)))));
    cl.setAttribute('x1', x(i)); cl.setAttribute('x2', x(i)); cl.setAttribute('visibility', 'visible');
    tip.innerHTML = `<b>${esc(s.timeline[i].label)} · ${s.timeline[i].plays} plays</b>` + s.series.filter(se => se.data[i]).map(se => `<div><i class="dot" style="background:${PALETTE[s.series.indexOf(se)]}"></i>${esc(se.name)} <b style="margin:0 0 0 auto;padding-left:10px">${se.data[i]}</b></div>`).join('');
    tip.classList.add('on');
    const tw = tip.offsetWidth, left = x(i) / W * b.width;
    tip.style.left = Math.max(0, Math.min(b.width - tw, left - tw / 2)) + 'px';
  };
  const hide = () => { tip.classList.remove('on'); cl.setAttribute('visibility', 'hidden'); };
  svg.addEventListener('pointermove', show); svg.addEventListener('pointerdown', show);
  svg.addEventListener('pointerleave', hide);
  svg.addEventListener('pointerup', e => { if (e.pointerType === 'touch') setTimeout(hide, 1800); });
}

// ---------- queue ----------
const secs = d => { const p = String(d || '').split(':').map(Number); return p.some(isNaN) ? 0 : p.reduce((a, x) => a * 60 + x, 0); };
const fmtLen = s => s >= 3600 ? `${Math.floor(s / 3600)} h ${Math.round(s % 3600 / 60)} min` : `${Math.max(1, Math.round(s / 60))} min`;
let showPlayed = false;
function viewQueue(c) {
  showPlayed = false;
  c.set(`<div class="top"><h1>Queue</h1>
      <button class="chip" id="qSave" ${queue.length ? '' : 'disabled'}>${ic('addq')}Save</button>
      <button class="chip" id="qClear" ${queue.length ? '' : 'disabled'}>${ic('close')}Clear</button></div>
    <div id="qSum" class="qsum"></div><div id="qList"></div>`);
  renderQueueList.done = false; renderQueueList();
  $('#qClear').onclick = () => { queue = cur() ? [cur()] : []; idx = queue.length ? 0 : -1; persist(); renderQueueList(); };
  $('#qSave').onclick = () => playlistPicker(queue.map(T), 'Queue');
  $('#qList').addEventListener('pointerdown', queueDrag);
  $('#qList').addEventListener('click', e => { if (e.target.closest('[data-played]')) { showPlayed = !showPlayed; renderQueueList(); } else if (e.target.closest('[data-grip]')) e.stopPropagation(); }, true);
}
function renderQueueList() {
  const el = $('#qList'); if (!el) return;
  reg('queue', queue, 'queue');
  const rows = (a, b) => { let s = ''; for (let i = a; i < b; i++) s += rowHTML(queue[i], 'queue', i, { grip: true }); return s; };
  if (!queue.length) { el.innerHTML = emptyHTML('queue', 'The queue is empty', '<a class="btn" href="#/search">Find music</a>'); const s = $('#qSum'); if (s) s.textContent = ''; return; }
  const cp = Math.max(idx, 0);
  const left = queue.slice(cp + 1).reduce((a, t) => a + secs(t.duration), 0);
  const sum = $('#qSum'); if (sum) sum.textContent = `${queue.length} song${queue.length === 1 ? '' : 's'}${queue.length > cp + 1 ? ` · ${fmtLen(left)} up next` : ''}${shuffle ? ' · shuffled' : ''}`;
  el.innerHTML = (cp > 0 ? `<button class="qhead played" data-played>${showPlayed ? 'Hide' : 'Show'} ${cp} played song${cp === 1 ? '' : 's'}${ic(showPlayed ? 'up' : 'down', 18)}</button>${showPlayed ? rows(0, cp) : ''}` : '') +
    `<h2 class="sec"><span>Now playing</span></h2>${rows(cp, cp + 1)}` +
    (queue.length > cp + 1 ? `<h2 class="sec"><span>Up next</span></h2>${rows(cp + 1, queue.length)}` : '');
  const c = el.querySelector('.row.cur'); if (c && !renderQueueList.done) { renderQueueList.done = true; if (cp > 2) c.scrollIntoView({ block: 'start' }); }
}
function moveQueueItem(from, to) {
  if (from === to || from < 0 || to < 0 || from >= queue.length || to >= queue.length) return;
  const [it] = queue.splice(from, 1); queue.splice(to, 0, it);
  if (from === idx) idx = to; else if (from < idx && to >= idx) idx--; else if (from > idx && to <= idx) idx++;
  persist(); renderQueueList(); updateNow();
}
// drag a row by its grip (mouse or touch) to reorder the queue
function queueDrag(e) {
  const grip = e.target.closest('[data-grip]'); if (!grip) return;
  const list = $('#qList'), row = grip.closest('.row'), scroller = $('#content');
  const rows = [...list.querySelectorAll('.row[data-id]')], rects = rows.map(r => r.getBoundingClientRect());
  const me = rows.indexOf(row), from = +row.dataset.i, h = rects[me].height, y0 = e.clientY, s0 = scroller.scrollTop;
  e.preventDefault(); e.stopPropagation(); grip.setPointerCapture(e.pointerId);
  row.classList.add('dragging'); let to = from, raf = 0, lastY = y0;
  const place = () => {
    const dy = lastY - y0 + (scroller.scrollTop - s0);
    row.style.transform = `translateY(${dy}px)`;
    const mid = rects[me].top + h / 2 + dy; let best = me;
    rects.forEach((r, k) => { if (Math.abs(r.top + r.height / 2 - mid) < Math.abs(rects[best].top + rects[best].height / 2 - mid)) best = k; });
    to = +rows[best].dataset.i;
    rows.forEach(r => { if (r === row) return; const i = +r.dataset.i; r.style.transform = (from < to && i > from && i <= to) ? `translateY(${-h}px)` : (from > to && i < from && i >= to) ? `translateY(${h}px)` : ''; });
  };
  const tick = () => {
    const box = scroller.getBoundingClientRect();
    if (lastY < box.top + 70) scroller.scrollTop -= 10; else if (lastY > box.bottom - 150) scroller.scrollTop += 10;
    place(); raf = requestAnimationFrame(tick);
  };
  raf = requestAnimationFrame(tick);
  const move = ev => { lastY = ev.clientY; };
  const end = () => {
    cancelAnimationFrame(raf); grip.removeEventListener('pointermove', move); grip.removeEventListener('pointerup', end); grip.removeEventListener('pointercancel', end);
    rows.forEach(r => { r.style.transform = ''; }); row.classList.remove('dragging');
    moveQueueItem(from, to);
  };
  grip.addEventListener('pointermove', move); grip.addEventListener('pointerup', end); grip.addEventListener('pointercancel', end);
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
    else { await send('POST', '/api/playlists/liked/tracks', { tracks: [T(t)] }); snack(stateOf(t.id) === 'owned' ? 'Liked — already in your Jellyfin library' : 'Liked — saving to your library'); download(t.id, true); }
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
// ---- import a playlist: a YouTube / YouTube Music link, or a pasted list / Spotify CSV
function importSheet() {
  openSheet(`<h3>Import a playlist</h3><div class="sub">Paste a YouTube or YouTube Music playlist link, or a list of songs: one “Artist - Title” per line, or a Spotify CSV (for example from Exportify).</div>
    <form id="impf"><input type="text" id="impn" placeholder="Playlist name" maxlength="100" autocomplete="off" required>
    <textarea id="impt" rows="7" placeholder="https://music.youtube.com/playlist?list=…&#10;or&#10;Daft Punk - One More Time&#10;Radiohead - Creep" spellcheck="false" required></textarea>
    <div class="msg" id="impm"></div><div class="btns"><button type="button" class="btn ghost" id="impc">Cancel</button><button class="btn" type="submit" id="impg">Import</button></div></form>`);
  $('#sheet').onclick = null; $('#impc').onclick = closeSheet; setTimeout(() => $('#impn').focus(), 250);
  $('#impf').onsubmit = async e => {
    e.preventDefault();
    const name = $('#impn').value.trim(), text = $('#impt').value.trim(), m = $('#impm'), go_ = $('#impg');
    m.className = 'msg'; m.textContent = ''; go_.disabled = true;
    try {
      const first = text.split('\n')[0].trim(), link = /^https?:\/\//i.test(first) && !text.includes('\n') ? parseYTLink(first) : null;
      if (link && (link.kind === 'yt' || link.kind === 'album')) {
        m.textContent = 'Reading the playlist…';
        const tracks = (link.kind === 'album' ? (await api('/api/album/' + link.id)).tracks : await api('/api/songs?b=' + encodeURIComponent(link.id))).map(T);
        if (!tracks.length) throw new Error('That playlist has no songs we can read');
        const made = await send('POST', '/api/playlists', { name, tracks });
        await refreshPlaylists(); closeSheet(); snack(`Imported ${tracks.length} songs into “${made.name}”`); if (routeName === 'library') route();
        return;
      }
      const { id } = await send('POST', '/api/import/list', { name, text });
      for (;;) {
        await new Promise(r => setTimeout(r, 1500));
        const st = await api('/api/import/list?id=' + id);
        if (st.running) { m.textContent = `Matching songs… ${st.done} of ${st.total}`; continue; }
        if (st.error) throw new Error(st.error);
        await refreshPlaylists();
        openSheet(`<h3>Imported “${esc(name)}”</h3><div class="sub">${st.found} of ${st.total} songs added.${st.missing && st.missing.length ? ` ${st.missing.length} could not be matched:` : ''}</div>
          ${st.missing && st.missing.length ? `<div class="miss">${st.missing.slice(0, 12).map(x => `<div>${esc(x)}</div>`).join('')}${st.missing.length > 12 ? `<div>…and ${st.missing.length - 12} more</div>` : ''}</div>` : ''}
          <div class="btns"><button class="btn" id="impo">Open playlist</button></div>`);
        $('#sheet').onclick = null; $('#impo').onclick = () => { closeSheet(); go('/playlist/' + st.playlistId); };
        if (routeName === 'library') route();
        return;
      }
    } catch (err) { m.textContent = err.message; go_.disabled = false; }
  };
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

// ---- offline songs: saved in the browser (Cache Storage) and played by the service worker
let offTracks = store.get('mw.off', {});
const OFF_CACHE = 'mw-offline-v1';
async function saveOffline(tracks) {
  if (!('caches' in window) || !('serviceWorker' in navigator)) { snack('Offline songs are not available in this browser'); return; }
  const todo = tracks.filter(t => !offTracks[t.id]);
  if (!todo.length) { snack('Already saved for offline'); return; }
  snack(`Saving ${todo.length} song${todo.length === 1 ? '' : 's'} for offline…`);
  const cache = await caches.open(OFF_CACHE); let ok = 0, last = '';
  for (const t of todo) {
    try {
      const r = await fetch(streamURL(t));
      if (!r.ok) throw new Error('HTTP ' + r.status);
      const blob = await r.blob();
      if (blob.size < 20000) throw new Error('empty audio');
      await cache.put('/offline/' + t.id, new Response(blob, { headers: { 'Content-Type': r.headers.get('Content-Type') || 'audio/mp4' } }));
      offTracks[t.id] = T(t); store.set('mw.off', offTracks); ok++;
    } catch (e) { last = e.message; }
  }
  snack(ok === todo.length ? `${ok} song${ok === 1 ? '' : 's'} saved for offline` : ok ? `${ok} of ${todo.length} saved (${last})` : `Could not save: ${last}`);
  updateNow();
  if (routeName === 'offline' || routeName === 'library') route();
}
async function removeOffline(ids) {
  try { const cache = await caches.open(OFF_CACHE); for (const id of ids) { await cache.delete('/offline/' + id); delete offTracks[id]; } } catch {}
  store.set('mw.off', offTracks); updateNow();
  if (routeName === 'offline') route();
}
async function offlineUsage() {
  try { const e = await navigator.storage.estimate(); return e.usage ? fmtBytes(e.usage) : ''; } catch { return ''; }
}
async function viewOffline(c) {
  const tracks = Object.values(offTracks), use = await offlineUsage();
  if (!c.alive()) return;
  c.set(tracksHeader({ title: 'Offline songs', thumb: tracks[0] && tracks[0].thumb, icon: 'cloud', sub: `${tracks.length} song${tracks.length === 1 ? '' : 's'}${use ? ' • ' + use : ''}`,
      buttons: `<button class="btn" data-act="play">${ic('play')}Play</button><button class="btn tonal" data-act="shuffle">${ic('shuf')}Shuffle</button>${tracks.length ? `<button class="iconbtn" data-act="clearoff" aria-label="Remove all">${ic('del')}</button>` : ''}` }) +
    (tracks.length ? songRows(tracks, reg('t', tracks, 'queue')) : emptyHTML('cloud', 'Songs you save for offline show up here.<br>Use the ⋮ menu on a song, or the cloud button on a playlist or album.')));
  c.actions = act => {
    if (act === 'clearoff') confirmSheet('Remove all offline songs?', 'They stay in your library, only the copies saved on this device are removed.', 'Remove', () => removeOffline(Object.keys(offTracks)));
    else listAction(act, tracks, 'Offline songs', null);
  };
  bindActions(c);
}
if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
window.addEventListener('offline', () => snack('You are offline. Songs saved for offline still play.'));

// ---- settings (playback)
const switchRow = (id, title, sub, on) => `<button class="swrow" data-sw="${id}" role="switch" aria-checked="${on}"><span><b>${esc(title)}</b><i>${esc(sub)}</i></span><span class="sw${on ? ' on' : ''}"></span></button>`;
function settingsSheet() {
  const paint = () => {
    openSheet(`<h3>Settings</h3>
      <h4 class="sethead">Playback</h4>
      ${switchRow('fade', 'Smooth transitions', 'Fade out the end of a song and fade the next one in', fadeOn)}
      ${switchRow('level', 'Volume leveling', 'Evens out loud and quiet songs. Experimental: on iPhone it may stop in the background', levelOn)}
      <button class="swrow" data-sw="offline"><span><b>Offline songs</b><i>${Object.keys(offTracks).length} saved on this device</i></span>${ic('cloud')}</button>
      <button class="swrow" data-sw="speed"><span><b>Playback speed</b><i>${rate === 1 ? 'Normal' : rate + '×'}</i></span>${ic('speed')}</button>
      ${appearanceHTML ? appearanceHTML() : ''}`);
    $('#sheet').onclick = e => {
      const b = e.target.closest('[data-sw]'); if (!b) return;
      const k = b.dataset.sw;
      if (k === 'fade') { fadeOn = !fadeOn; store.set('mw.fade', fadeOn); paint(); }
      else if (k === 'level') { setLeveling(!levelOn); paint(); }
      else if (k === 'speed') speedSheet();
      else if (k === 'offline') { closeSheet(); go('/offline'); }
      else if (appearanceClick) appearanceClick(k, paint);
    };
  };
  paint();
}
const appearanceHTML = null, appearanceClick = null;

// ---- smooth transitions: fade the end of a song out and the start of the next in
let fadeOn = store.get('mw.fade', false), sleeping = false;
const FADE_OUT = 3, FADE_IN = 1.8;
const baseVol = () => +$('#vol').value / 100;
function fadeGain() {
  if (!fadeOn || sleeping || !audio.duration || !isFinite(audio.duration)) return 1;
  const rem = audio.duration - audio.currentTime, hasNext = idx < queue.length - 1 || repeat > 0;
  let g = Math.min(1, audio.currentTime / FADE_IN);
  if (hasNext && repeat !== 2) g = Math.min(g, Math.max(0, rem / FADE_OUT));
  return Math.max(0.0001, g);
}
setInterval(() => { if (fadeOn && !sleeping && !audio.paused) audio.volume = Math.max(0, Math.min(1, baseVol() * fadeGain())); }, 100);

// ---- volume leveling: a gentle compressor so loud and quiet songs sit closer together
let levelOn = false, actx = null, lvSrc = null, lvComp = null, lvGain = null;
function setLeveling(on) {
  try {
    if (on && !actx) {
      actx = new (window.AudioContext || window.webkitAudioContext)();
      lvSrc = actx.createMediaElementSource(audio);
      lvComp = actx.createDynamicsCompressor();
      lvComp.threshold.value = -26; lvComp.knee.value = 30; lvComp.ratio.value = 4; lvComp.attack.value = 0.005; lvComp.release.value = 0.25;
      lvGain = actx.createGain(); lvGain.gain.value = 1.3;
    }
    if (actx) {
      lvSrc.disconnect(); lvComp.disconnect(); lvGain.disconnect();
      if (on) { lvSrc.connect(lvComp); lvComp.connect(lvGain); lvGain.connect(actx.destination); actx.resume(); }
      else lvSrc.connect(actx.destination);
    }
    levelOn = on; store.set('mw.level', on);
  } catch (e) { levelOn = false; store.set('mw.level', false); snack('Volume leveling is not available in this browser'); }
}
// the browser only allows audio processing after a tap: apply the saved choice on the first play
audio.addEventListener('play', () => { if (store.get('mw.level', false) && !levelOn) setLeveling(true); if (actx && actx.state === 'suspended') actx.resume(); }, { once: false });

// ---- playback speed
let rate = +store.get('mw.rate', 1) || 1;
function applyRate() { audio.defaultPlaybackRate = rate; audio.playbackRate = rate; }
function speedSheet() {
  openSheet(`<h3>Playback speed</h3><div class="sub">Pitch stays the same</div>${[0.75, 1, 1.25, 1.5, 2].map(r => `<button class="act${r === rate ? ' accent' : ''}" data-r="${r}">${ic(r === rate ? 'check' : 'speed')}${r === 1 ? 'Normal' : r + '×'}</button>`).join('')}`);
  $('#sheet').onclick = e => { const b = e.target.closest('[data-r]'); if (!b) return; rate = +b.dataset.r; store.set('mw.rate', rate); applyRate(); closeSheet(); snack(rate === 1 ? 'Normal speed' : `Speed ${rate}×`); };
}
applyRate();

// ---- sleep timer: pause after N minutes (fading out) or at the end of the song
let sleep = null;
function sleepLabel() {
  if (!sleep) return 'Sleep timer';
  if (sleep.eos) return 'Sleep timer: end of song';
  const m = Math.max(1, Math.round((sleep.at - Date.now()) / 60000));
  return `Sleep timer: ${m} min left`;
}
function updateSleepUI() { const b = $('#dSleep'); if (!b) return; b.classList.toggle('on', !!sleep); b.style.color = sleep ? 'var(--accent)' : ''; b.title = sleepLabel(); }
function clearSleep() { if (sleep && sleep.timer) clearTimeout(sleep.timer); sleep = null; updateSleepUI(); }
function sleepFire() {
  sleep = null; sleeping = true; updateSleepUI();
  const v = audio.volume; let n = 0;
  const iv = setInterval(() => {
    n++; audio.volume = Math.max(0, v * (1 - n / 20));
    if (n >= 20) { clearInterval(iv); audio.pause(); sleeping = false; audio.volume = v; snack('Sleep timer: paused. Good night.'); }
  }, 200);
}
function setSleep(mins) {
  clearSleep();
  if (mins === 'eos') sleep = { eos: true };
  else if (mins) sleep = { at: Date.now() + mins * 60000, timer: setTimeout(sleepFire, mins * 60000) };
  updateSleepUI();
  snack(sleep ? (sleep.eos ? 'Will pause at the end of this song' : `Will pause in ${mins} minutes`) : 'Sleep timer off');
}
function sleepSheet() {
  const o = (v, label) => `<button class="act" data-s="${v}">${ic('clock')}${label}</button>`;
  openSheet(`<h3>Sleep timer</h3><div class="sub">${esc(sleepLabel())}</div>${[15, 30, 45, 60].map(m => o(m, `${m} minutes`)).join('')}${o('eos', 'End of this song')}${sleep ? `<button class="act accent" data-s="0">${ic('close')}Turn off</button>` : ''}`);
  $('#sheet').onclick = e => { const b = e.target.closest('[data-s]'); if (!b) return; closeSheet(); const v = b.dataset.s; setSleep(v === 'eos' ? 'eos' : +v); };
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
function findTrack(id) {
  const c = cur(); if (c && c.id === id) return c;
  for (const L of Object.values(V.lists)) { const t = L.items.find(x => x.id === id); if (t) return t; }
  return queue.find(t => t.id === id) || recent.find(t => t.id === id) || liked.find(t => t.id === id) || null;
}
async function download(id, quiet) {
  const s = stateOf(id);
  if (s === 'owned') { if (!quiet) snack('Already in your Jellyfin library'); return; }
  if (s === 'busy' || s === 'done') return;
  dls[id] = 'queued'; updateNow();
  try {
    const t = findTrack(id) || {};
    const d = await send('POST', '/api/download/' + encodeURIComponent(id), { title: t.title || '', artist: t.artist || '' });
    if (d.status === 'owned') { delete dls[id]; owned.add(id); ownedChecked.add(id); updateNow(); if (!quiet) snack('Already in your Jellyfin library, skipped'); return; }
    if (d.status) dls[id] = d.status;
    updateNow(); if (!quiet) snack('Downloading…');
    if (!dlTimer) dlTimer = setTimeout(pollDl, 1500);
  } catch (e) { delete dls[id]; updateNow(); snack(e.message); }
}
function downloadAll(tracks) {
  const todo = tracks.filter(t => !['done', 'busy', 'owned'].includes(stateOf(t.id)));
  todo.forEach(t => download(t.id, true));
  const skipped = tracks.filter(t => stateOf(t.id) === 'owned').length;
  snack(todo.length ? `Downloading ${todo.length} song${todo.length === 1 ? '' : 's'}…${skipped ? ` (${skipped} already in Jellyfin)` : ''}` : 'Everything is already in your library');
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
    ${act('off', 'cloud', offTracks[t.id] ? 'Remove from offline' : 'Save for offline')}
    ${act('dl', st === 'done' || st === 'owned' ? 'check' : 'dl', st === 'done' ? 'Downloaded to your library' : st === 'owned' ? 'Already in your Jellyfin library' : st === 'busy' ? 'Downloading…' : 'Download to library')}
    ${ctx.player ? act('speed', 'speed', rate === 1 ? 'Playback speed' : `Playback speed: ${rate}×`) : ''}
    ${ctx.player ? act('sleep', 'moon', sleepLabel()) : ''}
    ${t.artistId ? act('artist', 'user', 'Go to artist') : ''}
    ${t.albumId ? act('album', 'album', 'Go to album') : ''}
    ${act('share', 'share', 'Share link')}
    ${ctx.queueIndex !== undefined ? act('rmq', 'close', 'Remove from queue') : ''}
    ${ctx.playlistId && ctx.playlistId !== 'liked' ? act('rmp', 'del', 'Remove from this playlist') : ''}`);
  $('#sheet').onclick = async e => {
    const b = e.target.closest('[data-a]'); if (!b) return;
    const a = b.dataset.a; closeSheet();
    if (a === 'sleep') sleepSheet();
    if (a === 'speed') speedSheet();
    if (a === 'radio') startRadio(t);
    if (a === 'next') { queue.splice(idx + 1, 0, T(t)); persist(); renderQueueList(); snack('Playing next'); }
    if (a === 'queue') { queue.push(T(t)); persist(); renderQueueList(); snack('Added to queue'); }
    if (a === 'like') toggleLike(t);
    if (a === 'pl') playlistPicker([t], t.title);
    if (a === 'off') { if (offTracks[t.id]) removeOffline([t.id]); else saveOffline([T(t)]); }
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
      <input type="text" name="user" placeholder="Username" value="${esc(acc.user)}" maxlength="64" autocapitalize="none" spellcheck="false" aria-label="Username" required>
      <input type="password" name="password" placeholder="New password (min. 10 characters; empty keeps it)" autocomplete="new-password" minlength="10" aria-label="New password">
      <div class="msg" id="accMsg"></div>
      <div class="btns"><button type="button" class="btn ghost" id="logout">${ic('out')}Sign out</button><button class="btn" type="submit">Save</button></div>
    </form>
    <div class="syncline" id="syncLine"></div>
    <div class="acclinks"><button data-act="settings">${ic('speed')}Settings</button><button data-go="/status">${ic('stats')}Server status</button></div>`);
  if (jfEnabled) fillSync();
  $('#sheet').onclick = e => { const g = e.target.closest('[data-go]'); if (g) { closeSheet(); go(g.dataset.go); } else if (e.target.closest('[data-act="settings"]')) settingsSheet(); };
  $('#logout').onclick = async () => { await fetch('/logout', { method: 'POST' }); location.href = '/login'; };
  $('#accForm').onsubmit = async e => {
    e.preventDefault(); const f = e.target, m = $('#accMsg'); m.className = 'msg'; m.textContent = '';
    try {
      acc = await send('POST', '/api/account', { current: f.current.value, user: f.user.value, password: f.password.value });
      m.className = 'msg ok'; m.textContent = 'Credentials updated.'; paintAccount(); setTimeout(closeSheet, 900);
    } catch (err) { m.textContent = err.message; }
  };
}
// Discreet Jellyfin sync status (the sync itself runs by itself about once a week).
async function fillSync() {
  const el = $('#syncLine'); if (!el) return;
  try {
    const [lib, imp] = await Promise.all([api('/api/library/status'), api('/api/library/import')]);
    if (!$('#syncLine')) return;
    const d = ts => new Date(ts * 1000).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    const state = imp.running ? `syncing ${imp.done}/${imp.total}…`
      : imp.lastRun ? `last sync ${d(imp.lastRun)}${imp.added ? `, +${imp.added} liked` : ''} · next ~${d(imp.next)}` : 'first sync is scheduled';
    el.innerHTML = `Jellyfin · ${lib.songs} songs · ${state} ${imp.running ? '' : '<button type="button" id="syncNow">Sync now</button>'}`;
    const b = $('#syncNow');
    if (b) b.onclick = async () => {
      try { await send('POST', '/api/library/import'); snack('Syncing your Jellyfin songs in the background'); fillSync(); const t = setInterval(async () => { if (!$('#syncLine')) return clearInterval(t); const s = await api('/api/library/import'); if (!s.running) { clearInterval(t); await Promise.all([loadLiked(), refreshPlaylists()]); fillSync(); snack(s.added ? `Liked ${s.added} songs from Jellyfin` : 'Jellyfin sync finished'); } else fillSync(); }, 3000); }
      catch (e) { snack(e.message); }
    };
  } catch { el.textContent = ''; }
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

let counted = false;     // has the current listen been reported to the stats?
function playAt(i) {
  if (i < 0 || i >= queue.length) return;
  idx = i; counted = false;
  if (mode === 'video') { stopVideo(); setTimeout(() => { if (mode === 'video') enterVideo(); }, 0); }
  const t = queue[i];
  audio.src = streamURL(t);
  audio.play().catch(e => console.warn(e));
  setTimeout(prefetchNext, 1500);
  document.title = `${t.title} · metroweb`;
  recent = [T(t), ...recent.filter(x => x.id !== t.id)].slice(0, 40); store.set('mw.history', recent);
  if ('mediaSession' in navigator) {
    navigator.mediaSession.metadata = new MediaMetadata({ title: t.title, artist: t.artist, album: t.album,
      artwork: t.thumb ? [{ src: new URL(img(t.thumb), location.href).href, sizes: '512x512' }] : [] });
  }
  themeFrom(t.thumb);
  persist(); renderQueueList(); updateNow();
  if (mode === 'lyrics') loadLyrics(t);
  if (idx >= queue.length - 2) topUpRadio();
}
// Title and artist let the server play the song straight from your Jellyfin library when it is there.
const streamURL = (t, extra = '') => `/api/stream/${t.id}?fmt=${fmt}&t=${encodeURIComponent(t.title || '')}&a=${encodeURIComponent(t.artist || '')}${extra}`;
// warm up the next song so it starts without a gap
function prefetchNext() {
  const n = queue[idx + 1]; if (!n) return;
  fetch(`/api/prefetch/${n.id}?fmt=${fmt}&t=${encodeURIComponent(n.title || '')}&a=${encodeURIComponent(n.artist || '')}`).catch(() => {});
}
function playList(items, i, keepOrder) {
  if (shuffle && !keepOrder && items.length > 1) return playShuffled(items, items[i]);
  origQueue = null; store.set('mw.orig', null);
  queue = items.map(T); idx = -1; playAt(i);
}

// ---- smart shuffle (see shuffle.js): the queue itself is reordered, so next, previous, the queue list and prefetch all just work
async function shuffleOpts() {
  let stats = {};
  try { stats = await cached('/api/plays/summary', 2 * 60 * 1000); } catch {}
  return { stats, liked: likedIds, now: Date.now() / 1000 };
}
function keepOrig(items) { origQueue = items.map(T); store.set('mw.orig', origQueue.slice(0, 300)); }
// start playing `items` shuffled; the first song is `first`, or the best opener when omitted
async function playShuffled(items, first) {
  const o = await shuffleOpts();
  let order;
  if (first) order = Shuffle.smartOrder(items, { ...o, first });
  else { order = Shuffle.smartOrder(items, o); }
  shuffle = true; store.set('mw.shuffle', true);
  keepOrig(items); queue = order.map(T); idx = -1; playAt(0);
  const g = Shuffle.gems(items, o.stats, o.liked, o.now);
  if (g >= 3) snack(`Shuffle mixed in ${g} songs you haven't heard in a while`);
}
async function setShuffle(on) {
  shuffle = on; store.set('mw.shuffle', on);
  if (on && queue.length > 1) {
    keepOrig(queue);
    const o = await shuffleOpts(), c = cur();
    const order = Shuffle.smartOrder(queue, { ...o, first: c });
    queue = order.map(T); idx = c ? 0 : -1;
    const g = Shuffle.gems(queue, o.stats, o.liked, o.now);
    if (g >= 3) snack(`Shuffle on: ${g} songs you haven't heard in a while are in the mix`);
  } else if (!on && origQueue) {
    const have = new Set(queue.map(t => t.id)), c = cur();
    const back = origQueue.filter(t => have.has(t.id)), seen = new Set(back.map(t => t.id));
    queue = [...back, ...queue.filter(t => !seen.has(t.id))];
    idx = c ? Math.max(0, queue.findIndex(t => t.id === c.id)) : -1;
    origQueue = null; store.set('mw.orig', null);
  }
  persist(); renderQueueList(); updateNow();
}
// everything played once: deal a fresh round, never opening with the song or artist just heard
async function nextRound() {
  const o = await shuffleOpts(), c = cur();
  const rest = queue.filter(t => !c || t.id !== c.id);
  const order = Shuffle.smartOrder(rest, { ...o, avoid: c ? [Shuffle.artistOf(c)] : [] });
  queue = c ? [...order, c] : order; idx = -1; keepOrig(queue); playAt(0);
}
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
    playList([T(t), ...data.filter(x => x.id !== t.id)], 0, true);
  } catch (e) { snack('Could not start radio: ' + e.message); }
}
function next(auto) {
  if (!queue.length) return;
  if (auto && repeat === 2) { audio.currentTime = 0; audio.play(); return; }
  if (idx < queue.length - 1) playAt(idx + 1);
  else if (repeat === 1) { if (shuffle && queue.length > 2) nextRound(); else playAt(0); }
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

// A listen counts after 30 seconds (or half of a very short track).
function maybeCountPlay() {
  const t = cur();
  if (counted || !t || audio.paused) return;
  const d = audio.duration;
  if (audio.currentTime >= 30 || (isFinite(d) && d < 60 && audio.currentTime >= d / 2)) {
    counted = true;
    send('POST', '/api/plays', { track: T(t) }).then(() => { statsCache = null; }).catch(() => { counted = false; });
  }
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
  const nx = queue[idx + 1], un = $('#fpNext');
  un.hidden = !nx; if (nx) un.innerHTML = `<span>Up next</span><b>${esc(nx.title)}</b><i>${esc(nx.artist || '')}</i>${ic('queue', 18)}`;
  const pi = pl ? 'pause' : 'play';
  for (const id of ['#miniPlay', '#play', '#dPlay']) $(id).innerHTML = ic(pi);
  $('#fp').classList.toggle('playing', pl); $('#fp').classList.toggle('paused', !pl);
  const fav = t && likedIds.has(t.id);
  for (const sel of ['#fpFav', '#dFav']) { $(sel).innerHTML = ic(fav ? 'heart' : 'heart0'); $(sel).style.color = fav ? 'var(--accent)' : ''; $(sel).setAttribute('aria-label', fav ? 'Unlike' : 'Like'); }
  const st = t ? stateOf(t.id) : '';
  const have = st === 'done' || st === 'owned';
  const dlIcon = have ? 'check' : st === 'busy' ? 'clock' : st === 'err' ? 'warn' : 'dl';
  const lbl = st === 'owned' ? 'In Jellyfin' : st === 'done' ? 'In library' : st === 'busy' ? 'Downloading…' : st === 'err' ? 'Retry' : 'Download';
  $('#fpDl').innerHTML = ic(dlIcon, 20) + lbl; $('#fpDl').className = have ? 'done' : '';
  $('#dDl').innerHTML = ic(dlIcon); $('#dDl').style.color = have ? 'var(--ok)' : st === 'err' ? 'var(--err)' : '';
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

function openFp() { if (!cur()) return; $('#fp .seg [data-mode="video"]').classList.toggle('none', clipCache.get(cur().id) === ''); $('#fp').classList.add('open'); $('#fp').setAttribute('aria-hidden', 'false'); }
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
  if (e.target.closest('[data-import]')) { importSheet(); return; }
  if (e.target.closest('[data-newpl]')) { promptSheet('New playlist', '', 'Create', async name => { await send('POST', '/api/playlists', { name }); await refreshPlaylists(); snack('Playlist created'); route(); }); return; }
  const ar = e.target.closest('[data-artist]'); if (ar) { go('/artist/' + ar.dataset.artist); return; }

  const card = e.target.closest('[data-card]');
  if (card) {
    const r = itemOf(card); if (!r || !r.it) return;
    const it = r.it;
    if (it.local) { go('/playlist/' + it.local); return; }
    if (it.mixId) { go('/mix/' + it.mixId); return; }
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
  const onRange = e.target.tagName === 'INPUT';
  if (e.code === 'Space') { e.preventDefault(); toggle(); }
  if (e.key === 'ArrowRight' && e.shiftKey) next();
  else if (e.key === 'ArrowLeft' && e.shiftKey) prev();
  else if (!onRange && !e.metaKey && !e.ctrlKey && !e.altKey) {
    if (e.key === 'ArrowRight') { e.preventDefault(); if (audio.duration) audio.currentTime = Math.min(audio.duration, audio.currentTime + 5); }
    else if (e.key === 'ArrowLeft') { e.preventDefault(); audio.currentTime = Math.max(0, audio.currentTime - 5); }
    else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      e.preventDefault(); const v = $('#vol'); v.value = Math.max(0, Math.min(100, +v.value + (e.key === 'ArrowUp' ? 5 : -5))); v.dispatchEvent(new Event('input'));
    }
    else if (e.key === 'm') { const vi = $('.volic'); if (vi) vi.click(); else audio.muted = !audio.muted; snack(audio.muted ? 'Muted' : 'Unmuted'); }
    else if (e.key === 'l' && cur()) toggleLike(cur());
    else if (e.key === 's') { setShuffle(!shuffle); snack(shuffle ? 'Shuffle off' : 'Shuffle on'); }
    else if (e.key === 'r') $('#dRep').click();
    else if (e.key === '?') shortcutsSheet();
  }
  if (e.key === '/') { e.preventDefault(); go('/search'); }
});
function shortcutsSheet() {
  const k = [['Space', 'Play / pause'], ['← →', 'Seek 5 seconds'], ['Shift + ← →', 'Previous / next song'], ['↑ ↓', 'Volume'], ['M', 'Mute'], ['L', 'Like the current song'],
    ['S', 'Shuffle'], ['R', 'Repeat'], ['/', 'Search'], ['Esc', 'Close'], ['?', 'This list']];
  openSheet(`<h3>Keyboard shortcuts</h3><div class="keys">${k.map(([a, b]) => `<div><kbd>${a}</kbd><span>${b}</span></div>`).join('')}</div>`);
  $('#sheet').onclick = null;
}
async function startMix(it) {
  snack('Starting mix…');
  try { const r = await api('/api/mix?id=' + encodeURIComponent(it.id)); playList(r.map(T), 0, true); } catch (e) { snack('Could not start mix: ' + e.message); }
}

$$('[data-nav]').forEach(b => b.addEventListener('click', () => go('/' + (b.dataset.nav === 'home' ? '' : b.dataset.nav))));
$('#scrim').onclick = closeSheet;
for (const id of ['play', 'miniPlay', 'dPlay']) $('#' + id).addEventListener('click', e => { e.stopPropagation(); toggle(); });
for (const id of ['next', 'miniNext', 'dNext']) $('#' + id).addEventListener('click', e => { e.stopPropagation(); next(); });
for (const id of ['prev', 'dPrev']) $('#' + id).addEventListener('click', prev);
for (const id of ['shuf', 'dShuf']) $('#' + id).addEventListener('click', () => setShuffle(!shuffle));
for (const id of ['rep', 'dRep']) $('#' + id).addEventListener('click', () => { repeat = (repeat + 1) % 3; store.set('mw.repeat', repeat); updateNow(); snack(['Repeat off', 'Repeat queue', 'Repeat song'][repeat]); });
for (const id of ['fpFav', 'dFav']) $('#' + id).addEventListener('click', () => toggleLike(cur()));
for (const id of ['fpDl', 'dDl']) $('#' + id).addEventListener('click', () => { const t = cur(); if (t) { if (stateOf(t.id) === 'err') delete dls[t.id]; download(t.id); } });
$('#fpMore').onclick = () => { if (cur()) trackMenu(cur(), { player: true }); };
$('#dSleep').onclick = sleepSheet;
$('#fpPl').onclick = () => { if (cur()) playlistPicker([cur()], cur().title); };
$('#fpQueue').onclick = () => { closeFp(); go('/queue'); };
$('#fpNext').onclick = () => { closeFp(); go('/queue'); };
$('#dQueue').onclick = () => go('/queue');
$('#fpA').onclick = () => { const t = cur(); if (t && t.artistId) { closeFp(); go('/artist/' + t.artistId); } };
$('#miniInfo').onclick = openFp; $('#miniInfo').addEventListener('keydown', e => { if (e.key === 'Enter') openFp(); });
$('#dExpand').onclick = openFp; $('#fpClose').onclick = closeFp;

// swipe down closes the full player; swipe up on the mini player opens it
(() => {   // swipe the cover left/right: next / previous song
  const img_ = $('#fpImg'); let x0 = null, dx = 0, pid = null;
  img_.addEventListener('pointerdown', e => { if (e.pointerType === 'mouse') return; x0 = e.clientX; dx = 0; pid = e.pointerId; });
  img_.addEventListener('pointermove', e => { if (x0 === null || e.pointerId !== pid) return; dx = e.clientX - x0; if (Math.abs(dx) > 8) { img_.style.transition = 'none'; img_.style.transform = `translateX(${dx}px) rotate(${dx / 40}deg)`; img_.style.opacity = String(1 - Math.min(.5, Math.abs(dx) / 500)); } });
  const done = () => {
    if (x0 === null) return; const d = dx; x0 = null; img_.style.transition = '';
    if (Math.abs(d) > 80) { img_.style.transform = `translateX(${d > 0 ? 120 : -120}%)`; img_.style.opacity = '0'; setTimeout(() => { d > 0 ? prev() : next(); img_.style.transition = 'none'; img_.style.transform = `translateX(${d > 0 ? -40 : 40}%)`; requestAnimationFrame(() => requestAnimationFrame(() => { img_.style.transition = ''; img_.style.transform = ''; img_.style.opacity = ''; })); }, 180); }
    else { img_.style.transform = ''; img_.style.opacity = ''; }
  };
  img_.addEventListener('pointerup', done); img_.addEventListener('pointercancel', done);
})();
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
audio.addEventListener('ended', () => { counted = false; if (sleep && sleep.eos) { clearSleep(); snack('Sleep timer: paused. Good night.'); return; } next(true); });
audio.addEventListener('error', () => {
  if (!audio.src) return;
  // a file from Jellyfin the browser can't play: retry once through YouTube
  if (audio.src.includes('/api/stream/') && !audio.src.includes('nojf=1')) { audio.src += '&nojf=1'; audio.play().catch(() => {}); return; } snack('Playback failed, skipping…'); setTimeout(() => next(), 1200); });
let seeking = false;
function paintBars() {
  const p = audio.duration ? audio.currentTime / audio.duration : 0;
  $('#cur').textContent = $('#dCur').textContent = time(audio.currentTime);
  $('#dur').textContent = $('#dDur').textContent = time(audio.duration);
  $('#miniProg').style.width = (p * 100) + '%';
  if (!seeking) for (const id of ['bar', 'dBar']) { const b = $('#' + id); b.value = p * 1000; b.style.setProperty('--p', (p * 100) + '%'); }
}
audio.addEventListener('timeupdate', () => {
  paintBars(); syncLyrics(); syncVideo();
  maybeCountPlay();
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

// ---------- full-screen player: Song / Lyrics / Video ----------
let mode = 'song';                    // 'song' | 'lyrics' | 'video'
let clip = null;                      // { videoId, trackId } while the music video is playing
const lyricsCache = new Map(), clipCache = new Map();
let lyr = { id: null, lines: [], active: -1, synced: false };
let userScrollUntil = 0;

function setMode(m, quiet) {
  if (m === 'video' && mode !== 'video') { enterVideo(); return; }
  if (mode === 'video' && m !== 'video') stopVideo();
  mode = m;
  const fp = $('#fp'); fp.dataset.mode = m;
  $$('#fp .seg button').forEach(b => { b.classList.toggle('on', b.dataset.mode === m); b.setAttribute('aria-selected', b.dataset.mode === m); });
  $('#lyr').hidden = m !== 'lyrics'; $('#vidbox').hidden = m !== 'video';
  if (m === 'lyrics' && cur()) loadLyrics(cur());
}

async function loadLyrics(t) {
  const box = $('#lyr'); lyr = { id: t.id, lines: [], active: -1, synced: false };
  box.innerHTML = '<div class="msg"><div style="width:30px;height:30px;border-radius:50%;border:4px solid var(--outline);border-top-color:var(--accent);animation:spin 1s linear infinite"></div></div>';
  let r = lyricsCache.get(t.id);
  if (!r) {
    try {
      const q = new URLSearchParams({ title: t.title, artist: t.artist || '', album: t.album || '', duration: t.duration || '' });
      r = await api('/api/lyrics?' + q); if (r.found) lyricsCache.set(t.id, r);
    } catch (e) {
      if (cur() && cur().id === t.id) { box.innerHTML = '<div class="msg">Could not load lyrics.<button class="btn tonal" id="lyRetry">Try again</button></div>'; $('#lyRetry').onclick = () => loadLyrics(t); }
      return;
    }
  }
  if (!cur() || cur().id !== t.id || mode !== 'lyrics') return;
  if (!r.enabled) { box.innerHTML = '<div class="msg">Lyrics are turned off on this server.</div>'; return; }
  if (r.synced && r.synced.length) {
    lyr.lines = r.synced; lyr.synced = true;
    box.innerHTML = r.synced.map((l, i) => `<p class="ln" data-i="${i}">${esc(l.text) || '♪'}</p>`).join('');
    syncLyrics(true);
  } else if (r.plain) {
    box.innerHTML = `<span class="tag">Not synced</span><div class="plain">${esc(r.plain)}</div>`;
    box.scrollTop = 0;
  } else if (r.instrumental) {
    box.innerHTML = '<div class="msg">♪ Instrumental ♪</div>';
  } else box.innerHTML = '<div class="msg">No lyrics found for this song.</div>';
}
// highlight the current line and keep it centred
function syncLyrics(force) {
  if (mode !== 'lyrics' || !lyr.synced || !lyr.lines.length) return;
  const t = audio.currentTime + 0.25, L = lyr.lines;
  let lo = 0, hi = L.length - 1, a = -1;
  while (lo <= hi) { const m = (lo + hi) >> 1; if (L[m].t <= t) { a = m; lo = m + 1; } else hi = m - 1; }
  if (a === lyr.active && !force) return;
  lyr.active = a;
  const box = $('#lyr'), els = box.children;
  for (let i = 0; i < els.length; i++) { els[i].classList.toggle('cur', i === a); els[i].classList.toggle('past', i < a); }
  if (a >= 0 && Date.now() > userScrollUntil) {
    const el = els[a]; box.scrollTo({ top: el.offsetTop - box.clientHeight / 2 + el.clientHeight / 2, behavior: force ? 'auto' : 'smooth' });
  }
}
$('#lyr').addEventListener('click', e => {
  const ln = e.target.closest('.ln'); if (!ln || !lyr.lines[+ln.dataset.i]) return;
  audio.currentTime = lyr.lines[+ln.dataset.i].t; userScrollUntil = 0; if (audio.paused) audio.play();
});
['wheel', 'touchmove'].forEach(ev => $('#lyr').addEventListener(ev, () => { userScrollUntil = Date.now() + 3500; }, { passive: true }));

// ---- music video ----
// The song's own audio keeps playing, untouched. The clip loads quietly in the
// background into a muted <video>; once it is ready it is placed at the exact
// point of the song (the server measures the clip's offset) and fades in.
let clipToken = 0;
async function clipFor(t) {
  if (clipCache.has(t.id)) return clipCache.get(t.id);
  const q = new URLSearchParams({ title: t.title, artist: t.artist || '', duration: t.duration || '' });
  const r = await api('/api/clip/info?' + q); clipCache.set(t.id, r.videoId || ''); return r.videoId || '';
}
function showVideoUI(on) {
  mode = on ? 'video' : mode;
  $('#fp').dataset.mode = on ? 'video' : mode;
  $('#lyr').hidden = true; $('#vidbox').hidden = !on;
  $$('#fp .seg button').forEach(b => { b.classList.toggle('on', b.dataset.mode === mode); b.setAttribute('aria-selected', b.dataset.mode === mode); });
}
function giveUpVideo(msg) {
  stopVideo(); mode = 'song'; setMode('song', true); if (msg) snack(msg);
}
async function enterVideo() {
  const t = cur(); if (!t) return;
  const token = ++clipToken;
  showVideoUI(true);                                   // switch the view at once; audio is not touched
  $('#vidPoster').src = img(t.thumb); $('#vidbox').classList.remove('live'); $('#vspin').hidden = false;
  let vid = '';
  try { vid = await clipFor(t); } catch (e) { if (token === clipToken) giveUpVideo('Could not look for the music video'); return; }
  if (token !== clipToken) return;
  if (!vid) { $('#fp .seg [data-mode="video"]').classList.add('none'); giveUpVideo('No official music video found for this song'); return; }
  const v = $('#fpVid');
  clip = { videoId: vid, songId: t.id, offset: 0, ready: false };
  const loaded = new Promise((res, rej) => { v.addEventListener('loadeddata', res, { once: true }); v.addEventListener('error', () => rej(new Error('video')), { once: true }); });
  v.src = `/api/clip/${vid}`; v.load();
  const offset = api(`/api/clip/offset?song=${t.id}&clip=${vid}`).catch(() => ({ offset: 0, confidence: 0 }));
  try { await loaded; } catch { if (token === clipToken) giveUpVideo('The video could not be played'); return; }
  const off = await offset;
  if (token !== clipToken || !clip) return;
  clip.offset = off.offset || 0; clip.ready = true;
  placeVideo(true);
  $('#vspin').hidden = true; $('#vidbox').classList.add('live');   // fade in over the cover
}
// where the video should be for the song's current position
function placeVideo(force) {
  if (!clip || !clip.ready) return;
  const v = $('#fpVid'), target = audio.currentTime + clip.offset;
  if (target < 0 || (isFinite(v.duration) && target > v.duration)) { v.pause(); return; }   // the clip has not started / has ended
  const diff = target - v.currentTime;
  if (force || Math.abs(diff) > 0.5) v.currentTime = target;
  else v.playbackRate = Math.abs(diff) > 0.06 ? (diff > 0 ? 1.04 : 0.96) * audio.playbackRate : audio.playbackRate;      // nudge gently instead of seeking
  if (!audio.paused && v.paused) v.play().catch(() => {});
}
function stopVideo() {
  clipToken++; clip = null;
  const box = $('#vidbox'), v = $('#fpVid');
  box.classList.remove('live'); $('#vspin').hidden = true;
  setTimeout(() => { if (!clip) { v.pause(); v.removeAttribute('src'); v.load(); } }, 350);   // after the fade-out
}
function syncVideo() { if (clip && clip.ready) placeVideo(false); }
audio.addEventListener('pause', () => { if (clip) $('#fpVid').pause(); });
audio.addEventListener('play', () => { if (clip && clip.ready) placeVideo(true); });
audio.addEventListener('seeked', () => { if (clip && clip.ready) placeVideo(true); });
$$('#fp .seg button').forEach(b => b.addEventListener('click', () => { if (b.dataset.mode !== mode) setMode(b.dataset.mode); }));

// ---------- Jellyfin awareness: mark songs you already own ----------
let scanTimer = null;
function scheduleOwnedScan() { if (jfEnabled) { clearTimeout(scanTimer); scanTimer = setTimeout(scanOwned, 350); } }
async function scanOwned() {
  if (!jfEnabled) return;
  const want = new Map();
  const add = t => { if (t && t.id && t.title && !ownedChecked.has(t.id) && /^[\w-]{11}$/.test(t.id) && (t.type === undefined || t.type === 'song')) want.set(t.id, t); };
  Object.values(V.lists).forEach(L => L.items.forEach(add)); add(cur());
  const batch = [...want.values()].slice(0, 400);
  if (!batch.length) return;
  batch.forEach(t => ownedChecked.add(t.id));
  try {
    const r = await send('POST', '/api/library/check', { tracks: batch.map(T) });
    r.owned.forEach(id => owned.add(id));
    // songs that are in your Jellyfin library count as liked: the server liked the new ones
    if (r.liked && r.liked.length) {
      r.liked.forEach(id => { likedIds.add(id); const t = batch.find(x => x.id === id); if (t && !liked.some(x => x.id === id)) liked.push(T(t)); });
      refreshPlaylists();
    }
    if (r.owned.length) updateNow();
  } catch { batch.forEach(t => ownedChecked.delete(t.id)); }
  if (want.size > batch.length) scheduleOwnedScan();
}
async function loadJellyfin() {
  try { jfEnabled = !!(await api('/api/library/status')).enabled; } catch { jfEnabled = false; }
  if (jfEnabled) { new MutationObserver(scheduleOwnedScan).observe($('#view'), { childList: true, subtree: true }); scheduleOwnedScan(); }
}

// Desktop sidebar: collapsed to icons by default, expanded on demand (remembered).
function setSide(open, quiet) {
  document.body.classList.toggle('side-open', open);
  const b = $('#sideToggle'); if (b) { b.setAttribute('aria-expanded', open); b.setAttribute('aria-label', open ? 'Collapse sidebar' : 'Expand sidebar'); }
  if (!quiet) store.set('mw.sideOpen', open);
}

// ============================================================================
// Boot
// ============================================================================
function paintStatic() {
  $$('[data-ic]').forEach(e => { e.innerHTML = ic(e.dataset.ic); e.style.display = 'inline-flex'; });
  const vi = $('.volic'); if (vi) { vi.setAttribute('role', 'button'); vi.title = 'Mute'; vi.style.cursor = 'pointer'; vi.onclick = () => { audio.muted = !audio.muted; vi.innerHTML = ic(audio.muted ? 'mute' : 'vol'); vi.style.color = audio.muted ? 'var(--accent)' : ''; }; }
  const nav = { home: ['home', 'Home'], search: ['search', 'Search'], library: ['lib', 'Library'], stats: ['stats', 'Stats'], queue: ['queue', 'Queue'] };
  $$('#nav button').forEach(b => b.querySelector('.pill').innerHTML = ic(nav[b.dataset.nav][0]));
  $$('#side button[data-nav]').forEach(b => { b.innerHTML = ic(nav[b.dataset.nav][0]) + `<span class="lbl">${nav[b.dataset.nav][1]}</span>`; b.title = nav[b.dataset.nav][1]; });
  $('#sideAcc').innerHTML = ic('user') + '<span class="lbl" id="sideUser">Account</span>'; $('#sideAcc').title = 'Account';
  $('#sideToggle').innerHTML = ic('menu');
  $('#sideToggle').onclick = () => setSide(!document.body.classList.contains('side-open'));
  setSide(store.get('mw.sideOpen', false), true);
  $('#sideAcc').onclick = accountSheet;
  $('#fpClose').innerHTML = ic('down', 30); $('#fpMore').innerHTML = ic('more');
  $('#miniNext').innerHTML = ic('next'); $('#dPrev').innerHTML = ic('prev'); $('#dNext').innerHTML = ic('next');
  $('#prev').innerHTML = ic('prev', 34); $('#next').innerHTML = ic('next', 34);
  $('#dExpand').innerHTML = ic('expand'); $('#dSleep').innerHTML = ic('moon'); $('#dQueue').innerHTML = ic('queue');
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
  await Promise.all([loadLiked(), refreshPlaylists(), loadJellyfin()]);
  pollDl();
  if (cur()) themeFrom(cur().thumb);
  if (!navigator.onLine && Object.keys(offTracks).length && !location.hash.startsWith('#/offline')) location.hash = '#/offline';
  route(); updateNow();
})();
