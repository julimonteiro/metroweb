const test = require('node:test'), assert = require('node:assert');
const S = require('./shuffle.js');
const DAY = 86400, NOW = 1_800_000_000;
const mk = (n, artists = 20) => Array.from({ length: n }, (_, i) => ({ id: 's' + i, title: 't' + i, artist: 'A' + (i % artists) }));
function lcg(seed) { let s = seed; return () => (s = (s * 1664525 + 1013904223) % 4294967296) / 4294967296; }

test('plays every song exactly once', () => {
  const t = mk(100), o = S.smartOrder(t, { rng: lcg(1), now: NOW });
  assert.strictEqual(o.length, 100); assert.strictEqual(new Set(o.map(x => x.id)).size, 100);
});
test('keeps the current song first', () => {
  const t = mk(30), o = S.smartOrder(t, { first: t[7], rng: lcg(2), now: NOW });
  assert.strictEqual(o[0].id, 's7'); assert.strictEqual(o.length, 30);
});
test('spreads the same artist apart', () => {
  const t = mk(60, 10), o = S.smartOrder(t, { rng: lcg(3), now: NOW });
  for (let i = 1; i < 4; i++) for (let j = 0; j + i < o.length - 0; j++) if (j + i < o.length && i <= 3 && j + i < 40)
    assert.notStrictEqual(o[j].artist, o[j + i].artist, `pos ${j} vs ${j + i}`);
});
test('forgotten favourites come up early, just-heard songs late', () => {
  const t = mk(200), stats = {}, liked = new Set(t.map(x => x.id));
  t.slice(0, 20).forEach(x => stats[x.id] = { n: 12, last: NOW - 200 * DAY });   // old loved
  t.slice(20, 40).forEach(x => stats[x.id] = { n: 12, last: NOW - 3600 });        // just played
  let early = 0, late = 0;
  for (let k = 0; k < 50; k++) {
    const o = S.smartOrder(t, { stats, liked, now: NOW, rng: lcg(100 + k) }).slice(0, 40).map(x => +x.id.slice(1));
    early += o.filter(i => i < 20).length; late += o.filter(i => i >= 20 && i < 40).length;
  }
  assert.ok(early > late * 3, `forgotten ${early} vs just-played ${late}`);
});
test('is not deterministic and still gives everyone a chance', () => {
  const t = mk(50), a = S.smartOrder(t, { rng: lcg(5), now: NOW }).map(x => x.id).join(), b = S.smartOrder(t, { rng: lcg(6), now: NOW }).map(x => x.id).join();
  assert.notStrictEqual(a, b);
});
test('single artist and tiny lists do not break', () => {
  assert.strictEqual(S.smartOrder([], {}).length, 0);
  assert.strictEqual(S.smartOrder(mk(5, 1), { rng: lcg(1), now: NOW }).length, 5);
});
