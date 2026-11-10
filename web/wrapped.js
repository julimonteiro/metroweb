// Your year in music: a full-screen story with your own music playing under it,
// visuals that move with the beat, and posters you can share at the end.
// Loaded before app.js; it uses app.js helpers at run time. Posters live in wrapped-posters.js.

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

let storyState = null;
function closeStory() {
  if (storyState) storyState.dispose();
  const el = document.getElementById('story');
  if (el) el.remove();
  document.removeEventListener('keydown', storyKeys, true);
  storyState = null;
}
function storyKeys(e) {
  if (!storyState) return;
  if (e.key === 'Escape') { e.stopPropagation(); storyState.close(); }
  else if (e.key === 'ArrowRight' || e.key === ' ') { e.preventDefault(); e.stopPropagation(); storyState.go(storyState.i + 1); }
  else if (e.key === 'ArrowLeft') { e.preventDefault(); e.stopPropagation(); storyState.go(storyState.i - 1); }
  else if (e.key === 'm' || e.key === 'M') { e.stopPropagation(); storyState.mute(); }
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
  let rv;
  try { rv = await api(`/api/review?year=${info.year}&tz=${tz}`); }
  catch (e) { c.set(topBar('Your year in music') + errorHTML(e.message)); return; }
  if (!c.alive()) return;
  if (!rv.stats.totals.plays) {
    c.set(topBar('Your year in music') + emptyHTML('stats', `No plays in ${info.year}.<br>Listen for a while and it will show up here.`, '<a class="btn" href="#/search">Find music</a>'));
    return;
  }
  c.set(topBar('Your year in music') + `<div class="wloading">${emptyHTML('stats', 'Ready when you are.', '<button class="btn" data-replay>Play my year</button>')}</div>`);
  $('#view').onclick = e => { if (e.target.closest('[data-replay]')) openStory(rv, info.year, tz); };
  openStory(rv, info.year, tz);
}

// ---- what the slides and posters show, from the review
async function storyData(rv, year) {
  const s = rv.stats, t = s.totals, byHour = s.byHour, peak = byHour.indexOf(Math.max(...byHour));
  const unit = t.minutes >= 120 ? { n: Math.round(t.minutes / 60), word: 'hours' } : { n: t.minutes, word: 'minutes' };
  const D = { rv, s, t, year, peak, unit, persona: rv.persona, top: s.topArtists[0], artists: s.topArtists.slice(0, 5), songs: s.topSongs.slice(0, 5),
    genres: (s.genres || []).slice(0, 5), hue: 280 };
  const cover = (rv.topSongs[0] && rv.topSongs[0].thumb) || (D.top && D.top.thumb);
  const hsl = await coverHSL(cover);
  if (hsl) D.hue = Math.round(hsl[0]);
  D.songTrack = rv.topSongs[0] || null;
  D.artistTrack = rv.topArtistSong || D.songTrack;
  return D;
}

const hourLabel = h => `${h % 12 || 12} ${h < 12 ? 'AM' : 'PM'}`;
const dayLabel = iso => new Date(iso + 'T12:00:00').toLocaleDateString('en-US', { month: 'long', day: 'numeric' });
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

// one square per day. Two layouts, the CSS picks by width: weeks as columns (wide) or one row per month (narrow)
function heatmapHTML(days, year) {
  const offset = (new Date(year, 0, 1).getDay() + 6) % 7, max = Math.max(...days, 1), cols = Math.ceil((offset + days.length) / 7);
  const lvl = n => n === 0 ? 0 : 1 + Math.min(3, Math.floor(3 * Math.sqrt(n / max)));
  const cells = days.map((n, i) => `<i class="hl${lvl(n)}" style="--c:${Math.floor((offset + i) / 7)}"></i>`).join('');
  const names = Array.from({ length: 12 }, (_, m) => new Date(year, m, 1).toLocaleDateString('en-US', { month: 'short' }));
  const labels = names.map((nm, m) => {
    const k = offset + Math.floor((new Date(year, m, 1) - new Date(year, 0, 1)) / 86400000);
    return `<span style="left:${(Math.floor(k / 7) / cols * 100).toFixed(2)}%">${nm}</span>`;
  }).join('');
  let rows = '', di = 0;
  for (let m = 0; m < 12; m++) {
    const dim = new Date(year, m + 1, 0).getDate();
    rows += `<em>${names[m]}</em>`;
    for (let d = 1; d <= 31; d++) rows += d <= dim ? `<i class="hl${lvl(days[di++] || 0)}" style="--c:${d};--r:${m}"></i>` : '<b></b>';
  }
  return `<div class="hm-wrap hm-by-week"><div class="hm-months">${labels}</div><div class="hm">${'<b></b>'.repeat(offset)}${cells}</div></div>` +
    `<div class="hm-wrap hm-by-month"><div class="hmm">${rows}</div></div>`;
}

function buildSlides(D) {
  const { rv, t, top } = D, st = D.s, sl = [], m = rv.metrics;
  const snd = (track, frac, vol) => track ? { track, frac, vol } : null, songTrack = D.songTrack, artistTrack = D.artistTrack;
  const base = snd(songTrack, 0.38, 0.4);

  sl.push({ cls: 's-open', sound: base, html:
    rise(0.1, `<span class="kicker">metroweb presents</span>`) +
    rise(0.35, `<h1 class="mega year">${String(D.year).split('').map((d, i) => `<i style="--k:${i}">${d}</i>`).join('')}</h1>`) +
    rise(1.1, `<p class="lead">Your year in music.</p>`) +
    rise(1.6, `<p class="fine">Turn the sound up.</p>`) });

  const movies = Math.round(t.minutes / 125), albums = Math.max(1, Math.round(t.minutes / 45));
  sl.push({ cls: 's-time', sound: base, html:
    rise(0.1, `<span class="kicker">This year you listened for</span>`) +
    rise(0.3, `<b class="count" data-to="${D.unit.n}">0</b>`, 'div', 's-num s-vis') +
    rise(0.5, `<h2 class="title">${D.unit.word} of music</h2>`) +
    rise(0.9, `<p class="lead">${fmtN(t.plays)} plays · ${fmtN(t.songs)} songs · ${fmtN(t.artists)} artists</p>`) +
    rise(1.2, `<p class="fine">${movies >= 2 ? `That is ${fmtN(movies)} movies back to back. Or ` : 'That is '}${fmtN(albums)} albums.</p>`) });

  const active = rv.days.filter(n => n > 0).length;
  sl.push({ cls: 's-heat', sound: base, html:
    rise(0.1, `<span class="kicker">Your year in squares</span>`) +
    rise(0.3, `<h2 class="title">${active} day${active === 1 ? '' : 's'} with music</h2>`) +
    rise(0.5, heatmapHTML(rv.days, D.year), 'div', 's-wide') +
    (rv.busiest ? rise(1.4, `<p class="lead">Biggest day: <b>${dayLabel(rv.busiest.date)}</b>, ${rv.busiest.plays} songs${rv.busiest.topArtist ? `, mostly ${esc(rv.busiest.topArtist)}` : ''}.${rv.streak.longest >= 3 ? ` Longest streak: <b>${rv.streak.longest} days</b> in a row.` : ''}</p>`) : '') });

  if (rv.obsession) {
    const o = rv.obsession;
    sl.push({ cls: 's-obsession', hueFrom: o.thumb, sound: base, html:
      rise(0.1, `<span class="kicker">Some songs hit different</span>`) +
      rise(0.3, `<p class="lead">On ${dayLabel(o.date)}, you played</p>`) +
      rise(0.55, sImg(o.thumb, 'cover-img'), 'div', 's-vis s-cover') +
      rise(0.7, `<h2 class="title">${esc(o.title)}</h2>`) +
      rise(0.9, `<div class="s-num s-big"><b class="count" data-to="${o.count}">0</b><span>times in one day</span></div>`) +
      rise(1.5, `<p class="fine">${o.count >= 8 ? 'That is commitment.' : 'No notes.'}</p>`) });
  }

  const dsc = rv.discovery;
  if (dsc && dsc.newArtists >= 3 && dsc.top.length) {
    sl.push({ cls: 's-discovery', hueFrom: dsc.top[0].thumb, sound: base, html:
      rise(0.1, `<span class="kicker">New in your life</span>`) +
      rise(0.3, `<div class="s-num s-big"><b class="count" data-to="${dsc.newArtists}">0</b><span>new artists</span></div>`) +
      `<div class="s-vis disco">${dsc.top.map((a, i) => `<div class="rise" style="--d:${0.6 + i * 0.15}s">${sImg(a.thumb, 'av')}<em>${esc(a.name)}</em></div>`).join('')}</div>` +
      rise(1.5, `<p class="lead"><b>${esc(dsc.top[0].name)}</b> was the one. You found them on ${dayLabel(dsc.topDate)} and played them ${fmtN(dsc.top[0].plays)} times.</p>`) });
  }

  if (top) sl.push({ cls: 's-artist s-reveal', bg: top.thumb, hueFrom: top.thumb, sound: snd(artistTrack, 0.4, 0.85), reveal: true, kickerAfter: 'Your #1 artist', html:
    rise(0.1, `<span class="kicker">Your #1 artist is…</span>`) +
    `<div class="s-vis hero-wrap s-cover"><div class="pop">${sImg(top.thumb, 'hero-img')}</div></div>` +
    `<h2 class="title big reveal-text">${esc(top.name)}</h2>` +
    `<p class="lead reveal-text">${fmtN(top.plays)} plays · ${fmtTime(top.minutes)}</p>` +
    `<p class="fine reveal-text">${Math.round(m.topArtistShare * 100)}% of everything you listened to.</p>` });

  if (D.artists.length > 1) sl.push({ cls: 's-artists', sound: snd(artistTrack, 0.4, 0.55), html:
    rise(0.1, `<span class="kicker">Your top artists</span>`) +
    `<ol class="s-rank s-vis">${D.artists.map((a, i) => `<li class="rise" style="--d:${0.3 + i * 0.14}s">${sImg(a.thumb, 'av')}<div><b>${esc(a.name)}</b><span class="s-bar"><i style="--w:${Math.max(8, Math.round(a.plays / D.artists[0].plays * 100))}%"></i></span></div><u>${fmtN(a.plays)}</u></li>`).join('')}</ol>` });

  const s1 = D.songs[0];
  if (s1) sl.push({ cls: 's-song s-reveal', bg: s1.thumb, hueFrom: s1.thumb, sound: snd(songTrack, 0.38, 0.95), reveal: true, kickerAfter: 'Your #1 song', html:
    rise(0.1, `<span class="kicker">Your #1 song is…</span>`) +
    `<div class="s-vis s-cover vinyl"><div class="pop">${sImg(s1.thumb, 'cover-img')}</div></div>` +
    `<h2 class="title big reveal-text">${esc(s1.title)}</h2>` +
    `<p class="lead reveal-text">${esc((s1.artist || '').split(',')[0])}</p>` +
    `<div class="s-num s-big reveal-text"><b class="count" data-to="${s1.plays}">0</b><span>plays</span></div>` });

  if (D.songs.length > 1) sl.push({ cls: 's-songs', sound: snd(songTrack, 0.38, 0.6), html:
    rise(0.1, `<span class="kicker">Your top songs</span>`) +
    `<ol class="s-rank songs s-vis">${D.songs.map((x, i) => `<li class="rise${i === 0 ? ' first' : ''}" style="--d:${0.3 + i * 0.14}s"><i class="n">${i + 1}</i>${sImg(x.thumb, 'cv')}<div><b>${esc(x.title)}</b><em>${esc((x.artist || '').split(',')[0])}</em></div><u>${x.plays}×</u></li>`).join('')}</ol>` });

  if (D.genres.length) sl.push({ cls: 's-genre', sound: snd(songTrack, 0.38, 0.5), html:
    rise(0.1, `<span class="kicker">Your sound</span>`) +
    rise(0.3, `<h2 class="title big">${esc(D.genres[0].name)}</h2>`) +
    `<div class="gbars s-vis">${D.genres.map((g, i) => `<div class="rise" style="--d:${0.6 + i * 0.12}s"><em>${esc(g.name)}</em><span class="s-bar"><i style="--w:${Math.max(6, Math.round(g.pct))}%"></i></span></div>`).join('')}</div>` });

  const night = D.persona.key === 'night_owl' || D.peak >= 22 || D.peak < 4;
  sl.push({ cls: `s-clock${night ? ' night' : ''}`, sound: snd(songTrack, 0.38, 0.5), html:
    rise(0.1, `<span class="kicker">You listen the most at</span>`) +
    rise(0.3, `<h2 class="title big">${hourLabel(D.peak)}</h2>`) +
    rise(0.5, clockSVG(st.byHour, D.peak), 'div', 'clockwrap s-vis') +
    rise(0.9, `<p class="lead">${night ? 'Most of your music plays after dark.' : D.peak < 10 ? 'You start the day with music.' : 'Music keeps you company through the day.'}</p>`) +
    (rv.nightSong ? rise(1.3, `<p class="fine">Your after-midnight song: <b>${esc(rv.nightSong.title)}</b>, ${rv.nightSong.count} times.</p>`) : '') });

  if (rv.months.filter(x => x.song).length >= 3) sl.push({ cls: 's-months', sound: snd(songTrack, 0.38, 0.55), html:
    rise(0.1, `<span class="kicker">Your year, month by month</span>`) +
    `<div class="mgrid s-wide">${rv.months.map((x, i) => `<div class="mcell${x.song ? '' : ' is-empty'} rise" style="--d:${0.35 + i * 0.09}s">${x.song ? sImg(x.song.thumb, 'cv') : '<div class="s-ph cv"></div>'}<span class="m-name">${x.label}</span>${x.song ? `<span class="m-title">${esc(x.song.title)}</span>` : ''}</div>`).join('')}</div>` +
    rise(1.6, `<p class="fine">The song you played most, every month.</p>`) });

  const traits = (D.persona.traits || []).slice(0, 3);
  sl.push({ cls: 's-persona', sound: base, html:
    rise(0.1, `<span class="kicker">As a listener, you are</span>`) +
    rise(0.35, `<h1 class="mega persona">${esc(D.persona.name)}</h1>`) +
    rise(0.8, `<p class="lead">${esc(D.persona.line)}${D.persona.second ? ` With a side of ${esc(D.persona.second)}.` : ''}</p>`) +
    `<div class="pbars s-vis">${[['Loyalty', m.topArtistShare], ['Variety', m.variety], ['Night plays', m.nightShare], ['Weekend plays', m.weekendShare]].map(([l, v], i) => `<div class="rise" style="--d:${0.9 + i * 0.15}s"><em>${l}</em><span class="s-bar"><i style="--w:${Math.max(4, Math.round(v * 100))}%"></i></span><u>${Math.round(v * 100)}%</u></div>`).join('')}</div>` +
    (traits.length ? rise(1.6, traits.map(x => `<span class="s-chip">${esc(x)}</span>`).join(''), 'div', 'traits') : '') });

  sl.push({ cls: 's-final', last: true, sound: snd(songTrack, 0.38, 0.35), html:
    rise(0.1, `<span class="kicker">That was your ${D.year}</span>`) +
    `<div class="ptabs rise" style="--d:.25s" role="tablist">${POSTERS.map((p, i) => `<button class="${i ? '' : 'is-on'}" data-pk="${i}" role="tab">${p.label}</button>`).join('')}</div>` +
    `<div class="poster-slot" aria-live="polite"><div class="poster-loading">Making your posters…</div></div>` +
    `<div class="sbtns rise" style="--d:.5s"><button class="sbtn main" data-share>${ic('share')}Share</button><button class="sbtn" data-save>${ic('dl')}Save image</button>` +
    `<button class="sbtn" data-pl hidden>${ic('lib')}Top Songs playlist</button><button class="sbtn s-ghost" data-replay2>Replay</button><button class="sbtn s-ghost" data-done>Done</button></div>` });
  return sl;
}

// ---- audio: your own music under the story, and the beat that drives the visuals
function createStorySound(root) {
  const el = new Audio(); el.preload = 'auto';
  let on = store.get('mw.storySound', true), cur = null, target = 0, token = 0, ctx = null, an = null, data = null, raf = 0, lvl = 0, mid = 0;
  const ramp = setInterval(() => {
    const d = target - el.volume; if (Math.abs(d) < 0.02) { el.volume = Math.max(0, Math.min(1, target)); return; }
    el.volume = Math.max(0, Math.min(1, el.volume + Math.sign(d) * 0.04));
  }, 40);
  function wire() {
    if (ctx || matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    try {
      ctx = new (window.AudioContext || window.webkitAudioContext)();
      const src = ctx.createMediaElementSource(el); an = ctx.createAnalyser(); an.fftSize = 512; an.smoothingTimeConstant = 0.78;
      src.connect(an); an.connect(ctx.destination); data = new Uint8Array(an.frequencyBinCount);
      const tick = () => {
        an.getByteFrequencyData(data);
        let b = 0, mm = 0; for (let i = 1; i <= 8; i++) b += data[i]; for (let i = 12; i <= 48; i++) mm += data[i];
        const beat = el.paused ? 0 : Math.min(1, Math.max(0, (b / 8 / 255 - 0.4) * 2.4));
        lvl += (beat - lvl) * (beat > lvl ? 0.55 : 0.12); mid += ((el.paused ? 0 : mm / 37 / 255) - mid) * 0.2;
        root.style.setProperty('--beat', lvl.toFixed(3)); root.style.setProperty('--mid', mid.toFixed(3));
        raf = requestAnimationFrame(tick);
      };
      tick();
    } catch { ctx = null; }
  }
  return {
    get on() { return on; },
    toggle() { on = !on; store.set('mw.storySound', on); if (!on) { target = 0; setTimeout(() => el.pause(), 300); } else if (cur) { el.play().catch(() => {}); target = cur.vol; } return on; },
    async play(spec) {
      if (!spec || !spec.track) return;
      const my = ++token;
      if (cur && cur.track.id === spec.track.id) { cur.vol = spec.vol; if (on) target = spec.vol; return; }
      const first = !cur; cur = spec; target = 0;
      if (!first) await new Promise(r => setTimeout(r, 350));
      if (my !== token) return;
      el.src = streamURL(spec.track); el.volume = 0;
      el.onloadedmetadata = () => { if (my === token && isFinite(el.duration)) el.currentTime = Math.max(0, el.duration * spec.frac); };
      if (!on) return;
      try { await el.play(); wire(); if (ctx && ctx.state === 'suspended') ctx.resume(); if (my === token) target = spec.vol; } catch { /* needs a tap: the sound button retries */ }
    },
    stop() { clearInterval(ramp); cancelAnimationFrame(raf); token++; try { el.pause(); el.removeAttribute('src'); el.load(); } catch {} if (ctx) ctx.close().catch(() => {}); },
  };
}

// ---- confetti for the big reveals
function confetti(canvas, hue) {
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
  const w = canvas.width = canvas.clientWidth, h = canvas.height = canvas.clientHeight, x = canvas.getContext('2d');
  const ps = Array.from({ length: 130 }, () => {
    const a = Math.random() * Math.PI * 2, v = 4 + Math.random() * 10;
    return { x: w / 2, y: h * 0.4, vx: Math.cos(a) * v, vy: Math.sin(a) * v - 5, r: 3 + Math.random() * 5, c: `hsl(${(hue + Math.random() * 120 - 20 + 360) % 360} 95% ${60 + Math.random() * 25}%)`, rot: Math.random() * 6, vr: Math.random() * 0.3 - 0.15 };
  });
  const t0 = performance.now();
  (function frame(now) {
    const age = (now - t0) / 1000; x.clearRect(0, 0, w, h);
    ps.forEach(p => { p.vy += 0.28; p.vx *= 0.99; p.x += p.vx; p.y += p.vy; p.rot += p.vr; x.save(); x.globalAlpha = Math.max(0, 1 - age / 2.2); x.translate(p.x, p.y); x.rotate(p.rot); x.fillStyle = p.c; x.fillRect(-p.r, -p.r / 2, p.r * 2, p.r); x.restore(); });
    if (age < 2.3 && canvas.isConnected) requestAnimationFrame(frame); else x.clearRect(0, 0, w, h);
  })(t0);
}

// ---- the story player
async function openStory(rv, year, tz) {
  closeStory();
  const D = await storyData(rv, year);
  const el = document.createElement('div');
  el.id = 'story'; el.setAttribute('role', 'dialog'); el.setAttribute('aria-modal', 'true'); el.setAttribute('aria-label', 'Your year in music');
  el.style.setProperty('--h', D.hue);
  const sound = createStorySound(el), wasPlaying = playing();
  if (wasPlaying) audio.pause();
  let muteBtn = null;
  const st = storyState = { i: -1, timers: [], sound, D,
    close() { closeStory(); go('/stats'); }, go: show,
    mute() { const on = sound.toggle(); muteBtn.innerHTML = ic(on ? 'vol' : 'mute', 22); muteBtn.setAttribute('aria-pressed', String(!on)); },
    dispose() { st.timers.forEach(clearTimeout); sound.stop(); if (wasPlaying) audio.play().catch(() => {}); } };
  const slides = buildSlides(D);
  el.innerHTML = `<button class="snav prev" aria-label="Previous">${ic('back', 28)}</button>
    <div class="sphone"><div class="sbars">${slides.map(() => '<i><b></b></i>').join('')}</div>
    <button class="sx" aria-label="Close">${ic('close')}</button><button class="smute" aria-label="Sound" aria-pressed="${!sound.on}">${ic(sound.on ? 'vol' : 'mute', 22)}</button>
    <div class="sview"></div><canvas class="sconf" aria-hidden="true"></canvas></div>
    <button class="snav next" aria-label="Next">${ic('back', 28)}</button>`;
  store.set('mw.wrapped.seen', year);
  document.body.appendChild(el);
  const view = el.querySelector('.sview'), bars = [...el.querySelectorAll('.sbars i')], conf = el.querySelector('.sconf');
  muteBtn = el.querySelector('.smute');

  // warm up the songs the story plays, and make the "Top Songs" playlist
  [D.songTrack, D.artistTrack].filter(Boolean).forEach(t => fetch(`/api/prefetch/${t.id}?fmt=${fmt}&t=${encodeURIComponent(t.title || '')}&a=${encodeURIComponent(t.artist || '')}`).catch(() => {}));
  send('POST', `/api/review/playlist?year=${year}&tz=${tz}`).then(p => { D.playlist = p; refreshPlaylists(); const b = view.querySelector('[data-pl]'); if (b) b.hidden = false; }).catch(() => {});

  const posters = [];
  function show(n) {
    if (n < 0) n = 0;
    if (n >= slides.length) return;
    st.timers.forEach(clearTimeout); st.timers = [];
    st.i = n;
    const sl = slides[n];
    bars.forEach((b, k) => { b.className = k < n ? 'is-done' : ''; });
    if (!sl.last) { void bars[n].offsetWidth; bars[n].className = 'is-cur'; bars[n].firstChild.style.animationDuration = (sl.reveal ? STORY_SLIDE_MS + 1500 : STORY_SLIDE_MS) + 'ms'; }
    else bars[n].className = 'is-done';
    view.innerHTML = `<section class="slide ${sl.cls}">${sl.bg ? `<div class="bgimg" style="background-image:url('${esc(img(sl.bg))}')"></div>` : ''}<div class="blob s-a"></div><div class="blob s-b"></div><div class="s-inner">${sl.html}</div></section>`;
    const slide = view.firstElementChild;
    view.querySelectorAll('.count').forEach(countUp);
    if (sl.hueFrom) coverHSL(sl.hueFrom).then(c => { if (c && st.i === n) slide.style.setProperty('--h', Math.round(c[0])); });
    if (sl.sound) sound.play(sl.sound);
    if (sl.reveal) st.timers.push(setTimeout(() => { slide.classList.add('is-revealed'); const kk = slide.querySelector('.kicker'); if (kk && sl.kickerAfter) kk.textContent = sl.kickerAfter; confetti(conf, +(slide.style.getPropertyValue('--h') || D.hue)); }, 1900));
    if (sl.last) showPoster(0);
  }
  async function showPoster(k) {
    const slot = view.querySelector('.poster-slot'); if (!slot) return;
    view.querySelectorAll('.ptabs button').forEach(b => b.classList.toggle('is-on', +b.dataset.pk === k));
    st.poster = k;
    try { posters[k] = posters[k] || await POSTERS[k].draw(D); } catch (e) { console.error(e); }
    if (st.poster !== k || !slot.isConnected) return;
    const cv = posters[k];
    if (!cv) { slot.innerHTML = '<div class="poster-loading">Could not make the poster</div>'; return; }
    cv.className = 'poster-img'; cv.style.aspectRatio = `${cv.width} / ${cv.height}`;
    slot.innerHTML = ''; slot.appendChild(cv);
    const b = view.querySelector('[data-pl]'); if (b && D.playlist) b.hidden = false;
  }
  async function shareOrSave(save) {
    const k = st.poster || 0, cv = posters[k]; if (!cv) return;
    const b = await new Promise(res => cv.toBlob(res, 'image/png')), file = new File([b], `my-${year}-${POSTERS[k].key}.png`, { type: 'image/png' });
    if (!save && navigator.canShare && navigator.canShare({ files: [file] })) {
      try { await navigator.share({ files: [file], title: `My ${year} in music` }); return; } catch (e) { if (e.name === 'AbortError') return; }
    }
    const a = document.createElement('a'); a.href = URL.createObjectURL(b); a.download = file.name; a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 4000); snack('Image saved');
  }

  // tap right = next, tap left = previous, hold = pause
  let downAt = 0, holdTimer = 0;
  view.addEventListener('pointerdown', e => {
    if (e.target.closest('button, a')) return;
    downAt = Date.now(); holdTimer = setTimeout(() => el.classList.add('hold'), 220);
  });
  view.addEventListener('pointerup', e => {
    clearTimeout(holdTimer); el.classList.remove('hold');
    if (!downAt) return; const quick = Date.now() - downAt < 220; downAt = 0;
    if (!quick || e.target.closest('button, a')) return;
    const r = view.getBoundingClientRect(); show(e.clientX - r.left < r.width * 0.3 ? st.i - 1 : st.i + 1);
  });
  view.addEventListener('pointercancel', () => { clearTimeout(holdTimer); el.classList.remove('hold'); downAt = 0; });
  bars.forEach(b => b.firstChild.addEventListener('animationend', () => show(st.i + 1)));
  el.addEventListener('click', e => {
    if (e.target.closest('.sx') || e.target.closest('[data-done]') || e.target === el) st.close();
    else if (e.target.closest('.smute')) st.mute();
    else if (e.target.closest('.snav.prev')) show(st.i - 1);
    else if (e.target.closest('.snav.next')) show(st.i + 1);
    else if (e.target.closest('[data-share]')) shareOrSave(false);
    else if (e.target.closest('[data-save]')) shareOrSave(true);
    else if (e.target.closest('[data-replay2]')) show(0);
    else if (e.target.closest('[data-pl]') && D.playlist) { const id = D.playlist.id; closeStory(); go('/playlist/' + id); }
    else { const t = e.target.closest('[data-pk]'); if (t) showPoster(+t.dataset.pk); }
  });
  document.addEventListener('keydown', storyKeys, true);
  show(0);
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
