// Your year in music: a full-screen story you tap through, ending in a poster
// you can save or share. Loaded before app.js; it uses app.js helpers at run time.

const STORY_SLIDE_MS = 6500;

// The recap opens in December (for that year) and stays through January (for the year that just ended).
// To peek earlier while testing: localStorage.setItem('mw.wrappedPreview', 'true').
function wrappedInfo(now = new Date()) {
  const m = now.getMonth(), y = now.getFullYear();
  if (m === 11) return { open: true, year: y };
  if (m === 0) return { open: true, year: y - 1 };
  if (store.get('mw.wrappedPreview', false)) return { open: true, year: y, preview: true };
  const opens = new Date(y, 11, 1);
  return { open: false, year: y, opens, days: Math.ceil((opens - now) / 86400000), yearDone: (now - new Date(y, 0, 1)) / (opens - new Date(y, 0, 1)) };
}
const wrappedSeen = year => store.get('mw.wrapped.seen', 0) === year;

function closeStory() {
  const el = document.getElementById('story');
  if (el) el.remove();
  document.removeEventListener('keydown', storyKeys, true);
  storyState = null;
}
let storyState = null;
function storyKeys(e) {
  if (!storyState) return;
  if (e.key === 'Escape') { e.stopPropagation(); storyState.close(); }
  else if (e.key === 'ArrowRight' || e.key === ' ') { e.preventDefault(); e.stopPropagation(); storyState.go(storyState.i + 1); }
  else if (e.key === 'ArrowLeft') { e.preventDefault(); e.stopPropagation(); storyState.go(storyState.i - 1); }
}

async function viewWrapped(c) {
  const info = wrappedInfo(), tz = -new Date().getTimezoneOffset();
  c.set(topBar('Your year in music') + `<div class="wloading">${skeletonRows(3)}</div>`);
  if (!info.open) {
    let soFar = null;
    try { soFar = (await api(`/api/stats?range=year&tz=${tz}`)).totals; } catch {}
    if (!c.alive()) return;
    const when = info.opens.toLocaleDateString('en-US', { month: 'long', day: 'numeric' });
    const pct = Math.round(Math.min(1, info.yearDone) * 100);
    c.set(topBar('Your year in music') + `<div class="wlock">
      <div class="wring" style="--p:${pct}"><span>${ic('lock', 34)}</span></div>
      <h2>Your ${info.year} in music<br>opens on ${when}</h2>
      <p>${info.days} day${info.days === 1 ? '' : 's'} to go. Your top artists, songs and sound of the year, in one story you can share.</p>
      ${soFar && soFar.plays ? `<div class="wsofar"><b>${fmtN(soFar.plays)}</b> plays so far · <b>${fmtTime(soFar.minutes)}</b> of music</div>` : ''}
      <small>Every song you listen to for 30 seconds counts.</small></div>`);
    return;
  }
  let s;
  try { s = await api(`/api/stats?range=${info.year === new Date().getFullYear() ? 'year' : 'year:' + info.year}&tz=${tz}`); }
  catch (e) { c.set(topBar('Your year in music') + errorHTML(e.message)); return; }
  if (!c.alive()) return;
  if (!s.totals.plays) {
    c.set(topBar('Your year in music') + emptyHTML('stats', `No plays in ${info.year}.<br>Listen for a while and it will show up here.`, '<a class="btn" href="#/search">Find music</a>'));
    return;
  }
  c.set(topBar('Your year in music') + `<div class="wloading">${emptyHTML('stats', 'Ready when you are.', '<button class="btn" data-replay>Play my year</button>')}</div>`);
  $('#view').onclick = e => { if (e.target.closest('[data-replay]')) openStory(storyData(s, info.year)); };
  openStory(storyData(s, info.year));
}

// ---- turn the stats into what the slides and the poster show
function storyData(s, year) {
  const t = s.totals, byHour = s.byHour, months = s.timeline || [];
  const peak = byHour.indexOf(Math.max(...byHour));
  const persona = peak >= 22 || peak < 4 ? { name: 'Night owl', line: 'Most of your music plays after dark.', night: true }
    : peak < 10 ? { name: 'Early bird', line: 'You start the day with music.' }
    : peak < 18 ? { name: 'Daytime listener', line: 'Music keeps you company through the day.' }
    : { name: 'Evening listener', line: 'You wind down with music.' };
  const best = months.reduce((a, m) => (m.plays > (a ? a.plays : 0) ? m : a), null);
  const unit = t.minutes >= 120 ? { n: Math.round(t.minutes / 60), word: 'hours' } : { n: t.minutes, word: 'minutes' };
  const fun = t.minutes >= 1440 ? `That is ${(t.minutes / 1440).toFixed(1)} days of non-stop music.` : `About ${Math.max(1, Math.round(t.minutes / 45))} albums back to back.`;
  return { s, t, year, peak, persona, best, unit, fun, months,
    top: s.topArtists[0], artists: s.topArtists.slice(0, 5), songs: s.topSongs.slice(0, 5), genres: (s.genres || []).slice(0, 5) };
}

const hourLabel = h => `${h % 12 || 12} ${h < 12 ? 'AM' : 'PM'}`;
const rise = (d, html, tag = 'div', cls = '') => `<${tag} class="rise ${cls}" style="--d:${d}s">${html}</${tag}>`;
const sImg = (u, cls = '') => u ? `<img class="${cls}" src="${img(u)}" alt="">` : `<div class="s-ph ${cls}">${ic('music', 36)}</div>`;

function clockSVG(byHour, peak) {
  const max = Math.max(...byHour, 1), cx = 130, cy = 130;
  const bars = byHour.map((v, h) => {
    const a = (h / 24) * Math.PI * 2 - Math.PI / 2, r0 = 62, len = 8 + 54 * (v / max);
    const x1 = cx + Math.cos(a) * r0, y1 = cy + Math.sin(a) * r0, x2 = cx + Math.cos(a) * (r0 + len), y2 = cy + Math.sin(a) * (r0 + len);
    return `<line x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" class="${h === peak ? 's-pk' : ''}" style="--i:${h}"/>`;
  }).join('');
  const lab = [['12a', 0], ['6a', 6], ['12p', 12], ['6p', 18]].map(([t, h]) => {
    const a = (h / 24) * Math.PI * 2 - Math.PI / 2;
    return `<text x="${(cx + Math.cos(a) * 44).toFixed(1)}" y="${(cy + Math.sin(a) * 44 + 4).toFixed(1)}" text-anchor="middle">${t}</text>`;
  }).join('');
  return `<svg viewBox="0 0 260 260" class="clock" aria-hidden="true">${bars}${lab}</svg>`;
}

function monthsChart(months, best) {
  const max = Math.max(...months.map(m => m.plays), 1);
  return `<div class="mchart">${months.map((m, i) => `<div class="mcol${best && m.key === best.key ? ' best' : ''}"><i style="--h:${Math.max(3, Math.round((m.plays / max) * 100))}%;--i:${i}"></i><span>${esc((m.label || '')[0])}</span></div>`).join('')}</div>`;
}

function buildSlides(D) {
  const { t, top } = D, sl = [];
  sl.push({ cls: 's-intro', html:
    rise(0.1, `<span class="kicker">metroweb</span>`) +
    rise(0.3, `<h1 class="mega">Your<br>${D.year}<br>in music</h1>`) +
    rise(0.9, `<p class="lead">Let's look back at what you played.</p>`) });
  sl.push({ cls: 's-time', html:
    rise(0.1, `<span class="kicker">You listened for</span>`) +
    rise(0.3, `<b class="count" data-to="${D.unit.n}">0</b>`, 'div', 's-num') +
    rise(0.5, `<h2 class="title">${D.unit.word} of music</h2>`) +
    rise(0.9, `<p class="lead">${fmtN(t.plays)} plays · ${fmtN(t.songs)} songs · ${fmtN(t.artists)} artists</p>`) +
    rise(1.2, `<p class="fine">${esc(D.fun)}</p>`) });
  if (top) sl.push({ cls: 's-artist', bg: top.thumb, html:
    rise(0.1, `<span class="kicker">Your #1 artist</span>`) +
    rise(0.35, sImg(top.thumb, 'hero-img'), 'div', 'hero-wrap') +
    rise(0.7, `<h2 class="title big">${esc(top.name)}</h2>`) +
    rise(0.95, `<p class="lead">${fmtN(top.plays)} plays · ${fmtTime(top.minutes)}</p>`) });
  if (D.artists.length > 1) sl.push({ cls: 's-artists', html:
    rise(0.1, `<span class="kicker">Your top artists</span>`) +
    `<ol class="s-rank">${D.artists.map((a, i) => `<li class="rise" style="--d:${0.3 + i * 0.14}s">${sImg(a.thumb, 'av')}<div><b>${esc(a.name)}</b><span class="s-bar"><i style="--w:${Math.max(8, Math.round(a.plays / D.artists[0].plays * 100))}%"></i></span></div><u>${fmtN(a.plays)}</u></li>`).join('')}</ol>` });
  if (D.songs.length) sl.push({ cls: 's-songs', html:
    rise(0.1, `<span class="kicker">Your top songs</span>`) +
    `<ol class="s-rank songs">${D.songs.map((x, i) => `<li class="rise${i === 0 ? ' first' : ''}" style="--d:${0.3 + i * 0.14}s"><i class="n">${i + 1}</i>${sImg(x.thumb, 'cv')}<div><b>${esc(x.title)}</b><em>${esc((x.artist || '').split(',')[0])}</em></div><u>${x.plays}×</u></li>`).join('')}</ol>` });
  if (D.genres.length) sl.push({ cls: 's-genre', html:
    rise(0.1, `<span class="kicker">Your sound</span>`) +
    rise(0.3, `<h2 class="title big">${esc(D.genres[0].name)}</h2>`) +
    `<div class="gbars">${D.genres.map((g, i) => `<div class="rise" style="--d:${0.6 + i * 0.12}s"><em>${esc(g.name)}</em><span class="s-bar"><i style="--w:${Math.max(6, Math.round(g.pct))}%"></i></span></div>`).join('')}</div>` });
  sl.push({ cls: `s-clock${D.persona.night ? ' night' : ''}`, html:
    rise(0.1, `<span class="kicker">You are a</span>`) +
    rise(0.3, `<h2 class="title big">${D.persona.name}</h2>`) +
    rise(0.5, clockSVG(D.s.byHour, D.peak), 'div', 'clockwrap') +
    rise(0.9, `<p class="lead">${esc(D.persona.line)} Your peak hour is ${hourLabel(D.peak)}.</p>`) });
  if (D.best && D.best.plays > 0) sl.push({ cls: 's-months', html:
    rise(0.1, `<span class="kicker">Month by month</span>`) +
    rise(0.3, `<h2 class="title">${esc(D.best.label)} was your biggest month</h2>`) +
    monthsChart(D.months, D.best) +
    rise(1.1, `<p class="lead">${fmtN(D.best.plays)} plays${D.best.topArtist ? ` · led by ${esc(D.best.topArtist.name)}` : ''}</p>`) });
  sl.push({ cls: 's-final', last: true, html:
    rise(0.1, `<span class="kicker">That was your ${D.year}</span>`) +
    `<div class="poster-slot" aria-live="polite"><div class="poster-loading">Making your poster…</div></div>` +
    `<div class="sbtns rise" style="--d:.5s"><button class="sbtn main" data-share>${ic('share')}Share</button><button class="sbtn" data-save>${ic('dl')}Save image</button><button class="sbtn s-ghost" data-done>Done</button></div>` });
  return sl;
}

// ---- the story player
function openStory(D) {
  closeStory();
  const slides = buildSlides(D), el = document.createElement('div');
  el.id = 'story'; el.setAttribute('role', 'dialog'); el.setAttribute('aria-modal', 'true'); el.setAttribute('aria-label', 'Your year in music');
  el.innerHTML = `<button class="snav prev" aria-label="Previous">${ic('back', 28)}</button>
    <div class="sphone"><div class="sbars">${slides.map(() => '<i><b></b></i>').join('')}</div>
    <button class="sx" aria-label="Close">${ic('close')}</button><div class="sview"></div></div>
    <button class="snav next" aria-label="Next">${ic('back', 28)}</button>`;
  store.set('mw.wrapped.seen', D.year);
  document.body.appendChild(el);
  const view = el.querySelector('.sview'), bars = [...el.querySelectorAll('.sbars i')];
  let poster = null;
  const st = storyState = { i: -1, close() { closeStory(); go('/stats'); }, go };
  function go(n) {
    if (n < 0) n = 0;
    if (n >= slides.length) return;
    st.i = n;
    const sl = slides[n];
    bars.forEach((b, k) => { b.className = k < n ? 'is-done' : ''; });
    if (!sl.last) { void bars[n].offsetWidth; bars[n].className = 'is-cur'; bars[n].firstChild.style.animationDuration = STORY_SLIDE_MS + 'ms'; }
    else bars[n].className = 'is-done';
    view.innerHTML = `<section class="slide ${sl.cls}">${sl.bg ? `<div class="bgimg" style="background-image:url('${esc(img(sl.bg))}')"></div>` : ''}<div class="blob s-a"></div><div class="blob s-b"></div><div class="s-inner">${sl.html}</div></section>`;
    view.querySelectorAll('.count').forEach(countUp);
    if (sl.last) showPoster();
  }
  async function showPoster() {
    try { poster = poster || await drawPoster(D); } catch (e) { console.error(e); }
    const slot = view.querySelector('.poster-slot');
    if (!slot) return;
    if (!poster) { slot.innerHTML = '<div class="poster-loading">Could not make the poster</div>'; return; }
    poster.className = 'poster-img'; poster.setAttribute('aria-label', 'Your year in music poster');
    slot.innerHTML = ''; slot.appendChild(poster);
  }
  const blob = () => new Promise(res => poster.toBlob(res, 'image/png'));
  async function shareOrSave(save) {
    if (!poster) return;
    const b = await blob(), file = new File([b], `my-${D.year}-in-music.png`, { type: 'image/png' });
    if (!save && navigator.canShare && navigator.canShare({ files: [file] })) {
      try { await navigator.share({ files: [file], title: 'My year in music' }); return; } catch (e) { if (e.name === 'AbortError') return; }
    }
    const a = document.createElement('a'); a.href = URL.createObjectURL(b); a.download = file.name; a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 4000); snack('Image saved');
  }
  // tap right = next, tap left = previous, hold = pause
  let downAt = 0, holdTimer = 0, x0 = 0;
  view.addEventListener('pointerdown', e => {
    if (e.target.closest('button, a')) return;
    downAt = Date.now(); x0 = e.clientX; holdTimer = setTimeout(() => el.classList.add('hold'), 220);
  });
  const release = e => {
    clearTimeout(holdTimer); el.classList.remove('hold');
    if (!downAt) return; const quick = Date.now() - downAt < 220; downAt = 0;
    if (!quick || (e.target.closest && e.target.closest('button, a'))) return;
    const r = view.getBoundingClientRect(); go(e.clientX - r.left < r.width * 0.3 ? st.i - 1 : st.i + 1);
  };
  view.addEventListener('pointerup', release);
  view.addEventListener('pointercancel', () => { clearTimeout(holdTimer); el.classList.remove('hold'); downAt = 0; });
  bars.forEach(b => b.firstChild.addEventListener('animationend', () => go(st.i + 1)));
  el.addEventListener('click', e => {
    if (e.target.closest('.sx') || e.target.closest('[data-done]') || e.target === el) st.close();
    else if (e.target.closest('.snav.prev')) go(st.i - 1);
    else if (e.target.closest('.snav.next')) go(st.i + 1);
    else if (e.target.closest('[data-share]')) shareOrSave(false);
    else if (e.target.closest('[data-save]')) shareOrSave(true);
  });
  document.addEventListener('keydown', storyKeys, true);
  go(0);
}

function countUp(node) {
  const to = +node.dataset.to || 0, t0 = performance.now(), dur = 1300;
  if (!to || matchMedia('(prefers-reduced-motion: reduce)').matches) { node.textContent = fmtN(to); return; }
  const tick = now => {
    const p = Math.min(1, (now - t0) / dur);
    node.textContent = fmtN(Math.round(to * (1 - Math.pow(1 - p, 3))));
    if (p < 1 && node.isConnected) requestAnimationFrame(tick);
  };
  requestAnimationFrame(tick);
}

// ---- the poster (1080 x 1920, made to be posted as a story)
async function loadImg(u) {
  if (!u) return null;
  try { const im = new Image(); im.src = img(u); await im.decode(); return im; } catch { return null; }
}
function rr(x, ctx, X, Y, W, H, R) { x.beginPath(); if (x.roundRect) x.roundRect(X, Y, W, H, R); else x.rect(X, Y, W, H); }
function fitText(x, text, maxW, size, weight) {
  let s = size; x.font = `${weight} ${s}px system-ui, -apple-system, "Segoe UI", sans-serif`;
  while (x.measureText(text).width > maxW && s > 28) { s -= 2; x.font = `${weight} ${s}px system-ui, -apple-system, "Segoe UI", sans-serif`; }
  return s;
}
async function drawPoster(D) {
  const W = 1080, H = 1920, cv = document.createElement('canvas'); cv.width = W; cv.height = H;
  const x = cv.getContext('2d'), F = 'system-ui, -apple-system, "Segoe UI", sans-serif';
  const [artistImg, ...coverImgs] = await Promise.all([loadImg(D.top && D.top.thumb), ...D.songs.map(s => loadImg(s.thumb))]);
  // background
  const g = x.createLinearGradient(0, 0, W * 0.6, H); g.addColorStop(0, '#1b0b3a'); g.addColorStop(0.55, '#4b1b78'); g.addColorStop(1, '#b02a6b');
  x.fillStyle = g; x.fillRect(0, 0, W, H);
  const blob = (cx, cy, r, c) => { const rg = x.createRadialGradient(cx, cy, 0, cx, cy, r); rg.addColorStop(0, c); rg.addColorStop(1, 'rgba(255,255,255,0)'); x.fillStyle = rg; x.fillRect(cx - r, cy - r, r * 2, r * 2); };
  blob(W * 0.9, 240, 560, 'rgba(255,140,200,.35)'); blob(W * 0.05, 1100, 620, 'rgba(110,80,255,.35)'); blob(W * 0.8, H, 520, 'rgba(255,190,120,.25)');
  // header
  x.fillStyle = 'rgba(255,255,255,.75)'; x.font = `700 34px ${F}`; x.letterSpacing = '8px'; x.fillText('METROWEB', 72, 120);
  x.fillText(String(D.year), W - 72 - x.measureText(String(D.year)).width, 120); x.letterSpacing = '0px';
  // hours
  x.fillStyle = '#fff'; x.font = `800 340px ${F}`; x.fillText(fmtN(D.unit.n), 62, 470);
  x.fillStyle = '#ffd9ec'; x.font = `700 62px ${F}`; x.fillText(`${D.unit.word} of music`, 72, 548);
  x.fillStyle = 'rgba(255,255,255,.75)'; x.font = `500 36px ${F}`; x.fillText(`${fmtN(D.t.plays)} plays · ${fmtN(D.t.songs)} songs · ${fmtN(D.t.artists)} artists`, 72, 612);
  // top artist
  const cx = 250, cy = 880, R = 170;
  x.fillStyle = 'rgba(255,255,255,.9)'; x.beginPath(); x.arc(cx, cy, R + 10, 0, 7); x.fill();
  x.save(); x.beginPath(); x.arc(cx, cy, R, 0, 7); x.clip();
  if (artistImg) x.drawImage(artistImg, cx - R, cy - R, R * 2, R * 2); else { x.fillStyle = '#2b2930'; x.fillRect(cx - R, cy - R, R * 2, R * 2); }
  x.restore();
  if (D.top) {
    x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `700 30px ${F}`; x.letterSpacing = '6px'; x.fillText('TOP ARTIST', 470, cy - 70); x.letterSpacing = '0px';
    x.fillStyle = '#fff'; fitText(x, D.top.name, 540, 84, 800); x.fillText(D.top.name, 470, cy + 14);
    x.fillStyle = 'rgba(255,255,255,.8)'; x.font = `500 38px ${F}`; x.fillText(`${fmtN(D.top.plays)} plays`, 470, cy + 76);
  }
  // top songs
  let y = 1180; x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `700 30px ${F}`; x.letterSpacing = '6px'; x.fillText('TOP SONGS', 72, y); x.letterSpacing = '0px';
  D.songs.forEach((s, i) => {
    const ry = y + 36 + i * 112;
    x.fillStyle = 'rgba(255,255,255,.12)'; rr(x, 0, 72, ry, W - 144, 96, 24); x.fill();
    x.fillStyle = '#ffd9ec'; x.font = `800 44px ${F}`; x.fillText(String(i + 1), 100, ry + 63);
    x.save(); rr(x, 0, 160, ry + 8, 80, 80, 16); x.clip(); if (coverImgs[i]) x.drawImage(coverImgs[i], 160, ry + 8, 80, 80); else { x.fillStyle = '#2b2930'; x.fillRect(160, ry + 8, 80, 80); } x.restore();
    x.fillStyle = '#fff'; fitText(x, s.title, 600, 38, 700); x.fillText(s.title, 262, ry + 44);
    x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `500 28px ${F}`; x.fillText((s.artist || '').split(',')[0].slice(0, 34), 262, ry + 78);
    x.fillStyle = 'rgba(255,255,255,.85)'; x.font = `700 32px ${F}`; const pl = `${s.plays}×`; x.fillText(pl, W - 100 - x.measureText(pl).width, ry + 62);
  });
  // footer
  x.fillStyle = 'rgba(255,255,255,.85)'; x.font = `600 36px ${F}`;
  const foot = [D.persona.name, D.genres[0] && D.genres[0].name].filter(Boolean).join(' · ');
  x.fillText(foot, 72, H - 72);
  x.fillStyle = 'rgba(255,255,255,.5)'; x.font = `500 30px ${F}`; const tag = 'my year in music'; x.fillText(tag, W - 72 - x.measureText(tag).width, H - 72);
  return cv;
}
