/* SentinelCore service worker
 * - Offline-first app shell (navigate SPA offline)
 * - Runtime caching for GET /api/v1/* using stale-while-revalidate
 */

const VERSION = 'sc-sw-v1'
const APP_SHELL_CACHE = `${VERSION}-shell`
const API_CACHE = `${VERSION}-api`

// Minimal shell list. Vite will emit hashed assets; we mainly need index.html.
const APP_SHELL_URLS = [
  '/',
  '/index.html'
]

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(APP_SHELL_CACHE)
      .then((cache) => cache.addAll(APP_SHELL_URLS))
      .then(() => self.skipWaiting())
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(
        keys
          .filter((k) => k.startsWith('sc-sw-') && !k.startsWith(VERSION))
          .map((k) => caches.delete(k))
      )
    ).then(() => self.clients.claim())
  )
})

function isApiGet(req) {
  try {
    const url = new URL(req.url)
    return req.method === 'GET' && url.pathname.startsWith('/api/v1/')
  } catch {
    return false
  }
}

function isNavigate(req) {
  return req.mode === 'navigate'
}

self.addEventListener('fetch', (event) => {
  const req = event.request

  // API runtime caching
  if (isApiGet(req)) {
    event.respondWith((async () => {
      const cache = await caches.open(API_CACHE)
      const cached = await cache.match(req)

      const networkFetch = fetch(req)
        .then((res) => {
          if (res && res.ok) {
            cache.put(req, res.clone())
          }
          return res
        })
        .catch(() => null)

      if (cached) {
        event.waitUntil(networkFetch)
        return cached
      }

      const res = await networkFetch
      if (res) return res
      return new Response(JSON.stringify({ error: 'offline' }), {
        status: 503,
        headers: { 'Content-Type': 'application/json' }
      })
    })())
    return
  }

  // SPA navigations: serve cached index.html when offline
  if (isNavigate(req)) {
    event.respondWith((async () => {
      try {
        const res = await fetch(req)
        return res
      } catch {
        const cache = await caches.open(APP_SHELL_CACHE)
        const cached = await cache.match('/index.html')
        return cached || new Response('offline', { status: 503 })
      }
    })())
    return
  }

  // Default: passthrough
})
