// Service worker for the static (offline) build of the chordpro web UI.
//
// Caches the whole app on install and serves it cache-first, so it starts
// and renders songs with no network. A new version installs in the
// background and only takes over once every chordpro window has been closed
// (i.e. on the next launch), never in the middle of a set.
//
// scripts/build-site.sh replaces __VERSION__ with a hash of the build, so any
// change to the app produces a new service worker and a fresh cache.
const CACHE = 'chordpro-__VERSION__';
const ASSETS = [
  './',
  'static/app.css',
  'static/app.js',
  'static/wasm_exec.js',
  'static/chordpro.wasm',
  'static/manifest.webmanifest',
  'static/icon-192.png',
  'static/icon-512.png',
  'static/apple-touch-icon.png',
];

self.addEventListener('install', (e) => {
  // cache: 'reload' bypasses the HTTP cache so a new version never caches
  // stale copies of its own files.
  e.waitUntil(caches.open(CACHE).then((c) =>
    c.addAll(ASSETS.map((url) => new Request(url, { cache: 'reload' })))));
});

self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then((keys) => Promise.all(
    keys.filter((k) => k.startsWith('chordpro-') && k !== CACHE).map((k) => caches.delete(k)))));
});

self.addEventListener('fetch', (e) => {
  const req = e.request;
  if (req.method !== 'GET' || new URL(req.url).origin !== location.origin) return;
  e.respondWith((async () => {
    const cache = await caches.open(CACHE);
    // Any page navigation within the app gets the cached app shell.
    const hit = await cache.match(req.mode === 'navigate' ? './' : req, { ignoreSearch: true });
    return hit || fetch(req);
  })());
});
