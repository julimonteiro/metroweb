// Smart shuffle: "wait, THIS song? yes!"
//
// A plain random order treats every song alike. This one asks of each song:
// how much do I love it, and how long since I last heard it? Songs that are
// loved but forgotten (and songs you own but never played here) are drawn
// first, so a shuffle keeps surfacing old favourites at the right moment.
// On top of that: each song appears once per round (no repeats), the same
// artist is spread apart, and a slice of pure luck keeps it unpredictable.
(function (root) {
  const DAY = 86400;

  // How much a song deserves to come up early. Higher = sooner.
  function weight(track, stat, now, likedIds) {
    const liked = likedIds ? likedIds.has(track.id) : false;
    const n = stat ? stat.n : 0;
    const love = 1 + (liked ? 1.2 : 0) + 0.7 * Math.log2(1 + n);   // loved: liked and often played
    let fresh;                                                       // forgotten-ness, 0..1
    if (!stat) fresh = liked ? 1 : 0.6;                              // never played here: a hidden gem
    else {
      const days = Math.max(0, (now - stat.last) / DAY);
      fresh = Math.min(1, days / 45);                                // fully "forgotten" after ~6 weeks
      if (days < 2) fresh *= 0.15;                                   // you just heard it
    }
    return love * (0.12 + fresh);
  }

  // Weighted draw without replacement (Efraimidis–Spirakis): sort by u^(1/w).
  function weightedOrder(items, weights, rng) {
    return items
      .map((it, i) => ({ it, k: Math.pow(Math.max(rng(), 1e-12), 1 / Math.max(weights[i], 1e-6)) }))
      .sort((a, b) => b.k - a.k)
      .map(x => x.it);
  }

  const artistOf = t => (t.artist || '').split(/,|&| feat\.? | x /i)[0].trim().toLowerCase();

  // Keep the same artist at least `gap` songs apart whenever that is possible.
  function spread(list, gap, recentArtists) {
    const artists = new Set(list.map(artistOf));
    const win = Math.min(gap, Math.max(0, artists.size - 1));
    if (win === 0) return list;
    const rest = list.slice(), out = [], last = (recentArtists || []).slice(-win);
    while (rest.length) {
      let pick = rest.findIndex(t => !last.includes(artistOf(t)));
      if (pick < 0) pick = 0;
      const t = rest.splice(pick, 1)[0];
      out.push(t); last.push(artistOf(t)); if (last.length > win) last.shift();
    }
    return out;
  }

  // The order to play `tracks` in. `first` (optional) stays on top, e.g. the
  // song that is playing now. opts: stats {id:{n,last}}, liked Set, now (s),
  // rng, wildcard (share of pure luck), avoid (artists to keep off the start).
  function smartOrder(tracks, opts) {
    opts = opts || {};
    const rng = opts.rng || Math.random, now = opts.now || Date.now() / 1000;
    const stats = opts.stats || {}, wildcard = opts.wildcard == null ? 0.15 : opts.wildcard;
    const pool = tracks.filter(t => !opts.first || t.id !== opts.first.id);
    const w = pool.map(t => rng() < wildcard ? 1 : weight(t, stats[t.id], now, opts.liked));
    let order = weightedOrder(pool, w, rng);
    order = spread(order, 3, opts.first ? [artistOf(opts.first)] : opts.avoid);
    return opts.first ? [opts.first, ...order] : order;
  }

  // How many of these songs are "forgotten favourites" (for the little hello).
  function gems(tracks, stats, liked, now) {
    return tracks.filter(t => {
      const s = stats[t.id];
      return (liked && liked.has(t.id)) && (!s || (now - s.last) / DAY > 30);
    }).length;
  }

  const api = { smartOrder, weight, spread, gems, artistOf };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.Shuffle = api;
})(typeof self !== 'undefined' ? self : this);
