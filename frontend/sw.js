/**
 * Service Worker für mAIrchen: Cache-First für den App-Shell (HTML/CSS/JS/
 * Icons), Network-Only für die Backend-API (/api/*, /health), damit
 * Geschichten und Status-Infos nie aus dem Cache statt live beantwortet
 * werden.
 *
 * Versionierung: CACHE_VERSION muss bei jedem Deployment erhöht werden. Alte
 * Caches werden beim `activate`-Event automatisch gelöscht, damit iOS/Chrome
 * nicht auf einer alten Version hängen bleiben.
 */
const CACHE_VERSION = 'v1';
const CACHE_NAME = `mairchen-${CACHE_VERSION}`;

// Alle Pfade sind relativ, damit der Service Worker unabhängig vom
// Deployment-Pfad funktioniert.
const APP_SHELL = [
    './',
    './index.html',
    './about.html',
    './styles.css',
    './app.js',
    './illustration.js',
    './sw-register.js',
    './manifest.webmanifest',
    './app_icon.png',
    './app_icon.svg',
    './logo.png',
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            // Einzeln cachen statt addAll(), damit ein einzelner fehlschlagender
            // Request nicht die komplette Installation blockiert.
            return Promise.all(
                APP_SHELL.map((url) => cache.add(url).catch((err) => {
                    console.warn('[SW] Konnte nicht gecacht werden:', url, err);
                }))
            );
        }).then(() => self.skipWaiting())
    );
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((keys) => Promise.all(
            keys
                .filter((key) => key.startsWith('mairchen-') && key !== CACHE_NAME)
                .map((key) => caches.delete(key))
        )).then(() => self.clients.claim())
    );
});

self.addEventListener('fetch', (event) => {
    const { request } = event;
    if (request.method !== 'GET') return;

    const url = new URL(request.url);

    // Backend-API und Health-Check immer frisch aus dem Netzwerk holen, nie
    // aus dem Cache - Story-Generierung und Statusdaten dürfen im Cache
    // niemals einen veralteten Stand vortäuschen.
    if (url.pathname.startsWith('/api/') || url.pathname === '/health') {
        event.respondWith(fetch(request));
        return;
    }

    // App-Shell: Cache-First, nur bei einem Cache-Miss ins Netzwerk. Ein
    // Treffer wird direkt zurückgegeben, ohne im Hintergrund trotzdem noch
    // einen Netzwerk-Request zu starten - Aktualisierungen kommen stattdessen
    // über CACHE_VERSION.
    event.respondWith(
        caches.match(request).then((cached) => {
            if (cached) return cached;

            return fetch(request).then((response) => {
                if (response && response.ok) {
                    const responseClone = response.clone();
                    caches.open(CACHE_NAME).then((cache) => cache.put(request, responseClone));
                }
                return response;
            });
        })
    );
});

// Erlaubt der Seite, einen wartenden Service Worker sofort zu aktivieren
// (z.B. nach Klick auf "Neu laden" im Update-Hinweis).
self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        self.skipWaiting();
    }
});
