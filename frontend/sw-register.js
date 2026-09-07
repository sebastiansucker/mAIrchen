// Registriert den Service Worker für Offline-Betrieb (PWA). Eigenständige
// Datei statt Teil von app.js, damit sowohl index.html als auch about.html
// sie einbinden können - about.html lädt app.js nicht.
(function registerServiceWorker() {
    if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) {
        return;
    }
    // Automatisierte Browser (Playwright & Co.) markieren sich über
    // navigator.webdriver - dort keinen Service Worker registrieren, damit
    // die E2E-Tests nicht durch gecachte Antworten oder einen aktiven
    // Controller beeinflusst werden.
    if (navigator.webdriver) {
        return;
    }

    window.addEventListener('load', () => {
        navigator.serviceWorker.register('./sw.js').then((registration) => {
            registration.addEventListener('updatefound', () => {
                const newWorker = registration.installing;
                if (!newWorker) {
                    return;
                }
                newWorker.addEventListener('statechange', () => {
                    if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                        showUpdateAvailableNotice(newWorker);
                    }
                });
            });
        }).catch((err) => {
            console.warn('Service Worker Registrierung fehlgeschlagen:', err);
        });
    });

    // Zeigt einen Hinweis "Neue Version verfügbar, neu laden" an.
    function showUpdateAvailableNotice(waitingWorker) {
        if (document.getElementById('sw-update-notice')) {
            return;
        }

        const notice = document.createElement('div');
        notice.id = 'sw-update-notice';
        notice.className = 'sw-update-notice';
        notice.setAttribute('role', 'status');
        notice.innerHTML = '<span>Neue Version verfügbar</span>';

        const reloadBtn = document.createElement('button');
        reloadBtn.type = 'button';
        reloadBtn.className = 'sw-update-notice-btn';
        reloadBtn.textContent = 'Neu laden';
        reloadBtn.addEventListener('click', () => {
            waitingWorker.postMessage('SKIP_WAITING');
            navigator.serviceWorker.addEventListener('controllerchange', () => {
                window.location.reload();
            });
        });

        notice.appendChild(reloadBtn);
        document.body.appendChild(notice);
    }
})();
