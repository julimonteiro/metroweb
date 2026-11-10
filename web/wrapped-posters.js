// The three posters of the year in review, drawn on a canvas so they can be
// saved or shared as images: the story (9:16), your top 9 covers (4:5) and
// your year in squares (4:5). Loaded before wrapped.js and app.js.

const POSTERS = [
  { key: 'story', label: 'Story', draw: drawStoryPoster },
  { key: 'top9', label: 'Top 9', draw: drawTop9Poster },
  { key: 'year', label: 'Year', draw: drawYearPoster },
];
const PFONT = 'system-ui, -apple-system, "Segoe UI", sans-serif';

async function loadImg(u) {
  if (!u) return null;
  try { const im = new Image(); im.src = img(u); await im.decode(); return im; } catch { return null; }
}
function newCanvas(w, h) { const cv = document.createElement('canvas'); cv.width = w; cv.height = h; return [cv, cv.getContext('2d')]; }
function pathRound(x, X, Y, W, H, R) { x.beginPath(); if (x.roundRect) x.roundRect(X, Y, W, H, R); else x.rect(X, Y, W, H); }
function fitFont(x, text, maxW, size, weight) {
  let s = size; x.font = `${weight} ${s}px ${PFONT}`;
  while (x.measureText(text).width > maxW && s > 24) { s -= 2; x.font = `${weight} ${s}px ${PFONT}`; }
  return s;
}
function cutText(x, text, maxW) {
  if (x.measureText(text).width <= maxW) return text;
  let t = text; while (t.length > 1 && x.measureText(t + '…').width > maxW) t = t.slice(0, -1);
  return t + '…';
}
function drawCover(x, im, X, Y, S, R) {
  x.save(); pathRound(x, X, Y, S, S, R); x.clip();
  if (im) x.drawImage(im, X, Y, S, S); else { x.fillStyle = 'rgba(255,255,255,.14)'; x.fillRect(X, Y, S, S); }
  x.restore();
}
// shared look: a deep gradient in the hue of the year's #1 cover, with soft glows
function paintBackdrop(x, W, H, hue) {
  const g = x.createLinearGradient(0, 0, W * 0.6, H);
  g.addColorStop(0, `hsl(${hue} 60% 9%)`); g.addColorStop(0.55, `hsl(${hue} 55% 22%)`); g.addColorStop(1, `hsl(${(hue + 50) % 360} 65% 36%)`);
  x.fillStyle = g; x.fillRect(0, 0, W, H);
  const glow = (cx, cy, r, c) => { const rg = x.createRadialGradient(cx, cy, 0, cx, cy, r); rg.addColorStop(0, c); rg.addColorStop(1, 'rgba(0,0,0,0)'); x.fillStyle = rg; x.fillRect(cx - r, cy - r, r * 2, r * 2); };
  glow(W * 0.9, H * 0.12, W * 0.6, `hsl(${(hue + 40) % 360} 90% 65% / .35)`);
  glow(W * 0.05, H * 0.6, W * 0.65, `hsl(${(hue + 300) % 360} 85% 60% / .28)`);
  glow(W * 0.8, H, W * 0.5, `hsl(${(hue + 20) % 360} 95% 70% / .22)`);
}
function spaced(x, text, X, Y, spacing) { x.letterSpacing = spacing + 'px'; x.fillText(text, X, Y); x.letterSpacing = '0px'; }
function rightText(x, text, R, Y) { x.fillText(text, R - x.measureText(text).width, Y); }

// ---- 1080 x 1920: hours, top artist, top songs
async function drawStoryPoster(D) {
  const W = 1080, H = 1920, [cv, x] = newCanvas(W, H);
  const [artistImg, ...covers] = await Promise.all([loadImg(D.top && D.top.thumb), ...D.songs.map(s => loadImg(s.thumb))]);
  paintBackdrop(x, W, H, D.hue);
  x.fillStyle = 'rgba(255,255,255,.75)'; x.font = `700 34px ${PFONT}`; spaced(x, 'METROWEB', 72, 120, 8);
  x.letterSpacing = '8px'; rightText(x, String(D.year), W - 72, 120); x.letterSpacing = '0px';
  x.fillStyle = '#fff'; x.font = `800 340px ${PFONT}`; x.fillText(fmtN(D.unit.n), 62, 470);
  x.fillStyle = `hsl(${D.hue} 100% 88%)`; x.font = `700 62px ${PFONT}`; x.fillText(`${D.unit.word} of music`, 72, 548);
  x.fillStyle = 'rgba(255,255,255,.75)'; x.font = `500 36px ${PFONT}`; x.fillText(`${fmtN(D.t.plays)} plays · ${fmtN(D.t.songs)} songs · ${fmtN(D.t.artists)} artists`, 72, 612);
  const cx = 250, cy = 880, R = 170;
  x.fillStyle = 'rgba(255,255,255,.92)'; x.beginPath(); x.arc(cx, cy, R + 10, 0, 7); x.fill();
  x.save(); x.beginPath(); x.arc(cx, cy, R, 0, 7); x.clip();
  if (artistImg) x.drawImage(artistImg, cx - R, cy - R, R * 2, R * 2); else { x.fillStyle = '#2b2930'; x.fillRect(cx - R, cy - R, R * 2, R * 2); }
  x.restore();
  if (D.top) {
    x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `700 30px ${PFONT}`; spaced(x, 'TOP ARTIST', 470, cy - 70, 6);
    x.fillStyle = '#fff'; fitFont(x, D.top.name, 540, 84, 800); x.fillText(D.top.name, 470, cy + 14);
    x.fillStyle = 'rgba(255,255,255,.8)'; x.font = `500 38px ${PFONT}`; x.fillText(`${fmtN(D.top.plays)} plays`, 470, cy + 76);
  }
  const y = 1180; x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `700 30px ${PFONT}`; spaced(x, 'TOP SONGS', 72, y, 6);
  D.songs.forEach((s, i) => {
    const ry = y + 36 + i * 112;
    x.fillStyle = 'rgba(255,255,255,.12)'; pathRound(x, 72, ry, W - 144, 96, 24); x.fill();
    x.fillStyle = `hsl(${D.hue} 100% 88%)`; x.font = `800 44px ${PFONT}`; x.fillText(String(i + 1), 100, ry + 63);
    drawCover(x, covers[i], 160, ry + 8, 80, 16);
    x.fillStyle = '#fff'; fitFont(x, s.title, 600, 38, 700); x.fillText(s.title, 262, ry + 44);
    x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `500 28px ${PFONT}`; x.fillText(cutText(x, (s.artist || '').split(',')[0], 560), 262, ry + 78);
    x.fillStyle = 'rgba(255,255,255,.85)'; x.font = `700 32px ${PFONT}`; rightText(x, `${s.plays}×`, W - 100, ry + 62);
  });
  x.fillStyle = 'rgba(255,255,255,.85)'; x.font = `600 36px ${PFONT}`;
  x.fillText([D.persona.name, D.genres[0] && D.genres[0].name].filter(Boolean).join(' · '), 72, H - 72);
  x.fillStyle = 'rgba(255,255,255,.5)'; x.font = `500 30px ${PFONT}`; rightText(x, 'my year in music', W - 72, H - 72);
  return cv;
}

// ---- 1080 x 1350: the nine most played songs as a grid of covers
async function drawTop9Poster(D) {
  const W = 1080, H = 1350, [cv, x] = newCanvas(W, H);
  const list = (D.rv.topSongs || []).slice(0, 9), covers = await Promise.all(list.map(s => loadImg(s.thumb)));
  paintBackdrop(x, W, H, D.hue);
  x.fillStyle = '#fff'; x.font = `800 64px ${PFONT}`; x.fillText(`My top songs of ${D.year}`, 60, 104);
  x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `500 30px ${PFONT}`; x.fillText(`${fmtN(D.t.plays)} plays · ${D.unit.n.toLocaleString('en-US')} ${D.unit.word} of music`, 60, 148);
  const gap = 16, size = Math.floor((W - 120 - gap * 2) / 3), top = 196;
  list.forEach((s, i) => {
    const X = 60 + (i % 3) * (size + gap), Y = top + Math.floor(i / 3) * (size + gap);
    drawCover(x, covers[i], X, Y, size, 26);
    const g = x.createLinearGradient(0, Y + size * 0.55, 0, Y + size); g.addColorStop(0, 'rgba(0,0,0,0)'); g.addColorStop(1, 'rgba(0,0,0,.78)');
    x.save(); pathRound(x, X, Y, size, size, 26); x.clip(); x.fillStyle = g; x.fillRect(X, Y, size, size); x.restore();
    x.fillStyle = '#fff'; x.font = `800 54px ${PFONT}`; x.fillText(String(i + 1), X + 22, Y + 64);
    x.font = `700 26px ${PFONT}`; x.fillText(cutText(x, s.title, size - 40), X + 20, Y + size - 44);
    x.fillStyle = 'rgba(255,255,255,.75)'; x.font = `500 22px ${PFONT}`; x.fillText(cutText(x, (s.artist || '').split(',')[0], size - 40), X + 20, Y + size - 16);
  });
  x.fillStyle = 'rgba(255,255,255,.55)'; x.font = `600 26px ${PFONT}`; spaced(x, 'METROWEB', 60, H - 36, 6); rightText(x, D.persona.name, W - 60, H - 36);
  return cv;
}

// ---- 1080 x 1350: one square per day of the year
async function drawYearPoster(D) {
  const W = 1080, H = 1350, [cv, x] = newCanvas(W, H);
  paintBackdrop(x, W, H, D.hue);
  x.fillStyle = '#fff'; x.font = `800 72px ${PFONT}`; x.fillText(`My ${D.year}`, 60, 120);
  x.fillStyle = 'rgba(255,255,255,.75)'; x.font = `500 34px ${PFONT}`; x.fillText('in squares, one for every day', 60, 170);
  const days = D.rv.days, max = Math.max(...days, 1), offset = (new Date(D.year, 0, 1).getDay() + 6) % 7;
  const cols = Math.ceil((offset + days.length) / 7), gap = 5, cell = Math.floor((W - 120 - gap * (cols - 1)) / cols), gx = 60, gy = 260;
  const levels = [`rgba(255,255,255,.10)`, `hsl(${D.hue} 85% 40%)`, `hsl(${D.hue} 90% 54%)`, `hsl(${D.hue} 95% 68%)`, `hsl(${D.hue} 100% 86%)`];
  days.forEach((n, i) => {
    const k = offset + i, c = Math.floor(k / 7), r = k % 7;
    const lv = n === 0 ? 0 : 1 + Math.min(3, Math.floor(3 * Math.sqrt(n / max)));
    x.fillStyle = levels[lv]; pathRound(x, gx + c * (cell + gap), gy + r * (cell + gap), cell, cell, 4); x.fill();
  });
  const gridBottom = gy + 7 * (cell + gap);
  x.fillStyle = 'rgba(255,255,255,.7)'; x.font = `600 24px ${PFONT}`;
  for (let m = 0; m < 12; m++) {
    const k = offset + Math.floor((new Date(D.year, m, 1) - new Date(D.year, 0, 1)) / 86400000);
    x.fillText(new Date(D.year, m, 1).toLocaleDateString('en-US', { month: 'short' }), gx + Math.floor(k / 7) * (cell + gap), gy - 18);
  }
  // numbers
  const rv = D.rv, stat = (label, value, X, Y) => {
    x.fillStyle = 'rgba(255,255,255,.65)'; x.font = `600 24px ${PFONT}`; spaced(x, label.toUpperCase(), X, Y, 3);
    x.fillStyle = '#fff'; x.font = `800 74px ${PFONT}`; x.fillText(value, X, Y + 78);
  };
  const active = rv.days.filter(n => n > 0).length;
  stat('Days with music', String(active), 60, gridBottom + 90);
  stat('Longest streak', `${rv.streak.longest}`, 400, gridBottom + 90);
  stat('Plays', fmtN(D.t.plays), 740, gridBottom + 90);
  if (rv.busiest) {
    x.fillStyle = 'rgba(255,255,255,.12)'; pathRound(x, 60, gridBottom + 230, W - 120, 110, 28); x.fill();
    x.fillStyle = '#fff'; x.font = `600 34px ${PFONT}`;
    const when = new Date(rv.busiest.date + 'T12:00:00').toLocaleDateString('en-US', { month: 'long', day: 'numeric' });
    x.fillText(cutText(x, `Biggest day: ${when} · ${rv.busiest.plays} songs`, W - 180), 90, gridBottom + 296);
  }
  x.fillStyle = 'rgba(255,255,255,.55)'; x.font = `600 26px ${PFONT}`; spaced(x, 'METROWEB', 60, H - 44, 6); rightText(x, D.persona.name, W - 60, H - 44);
  return cv;
}
