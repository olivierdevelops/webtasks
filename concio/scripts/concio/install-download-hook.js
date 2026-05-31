// Install an interceptor on URL.createObjectURL so that every Blob that
// Concio decrypts becomes available to the Java side via executeScript.
//
// We re-install on every call (idempotent via `__webtasks_hookInstalled`).
// Captured entries are appended to `window.__webtasks_captures`. The Java
// side polls and drains them; each entry has { id, url, mime, size, name,
// bytesB64, ts }.
(function() {
    if (window.__webtasks_hookInstalled) return { installed: false, alreadyInstalled: true };
    window.__webtasks_hookInstalled = true;
    window.__webtasks_captures = [];
    window.__webtasks_nextId = 1;

    const origCreate = URL.createObjectURL.bind(URL);
    URL.createObjectURL = function(blob) {
        const url = origCreate(blob);
        const id = window.__webtasks_nextId++;
        const entry = {
            id,
            url,
            mime: blob && blob.type ? blob.type : null,
            size: blob && blob.size != null ? blob.size : null,
            name: null,
            bytesB64: null,
            ts: Date.now()
        };
        window.__webtasks_captures.push(entry);
        // Materialize the bytes asynchronously. Java polls until bytesB64 != null.
        if (blob && typeof blob.arrayBuffer === 'function') {
            blob.arrayBuffer().then(buf => {
                const u8 = new Uint8Array(buf);
                let bin = '';
                const chunk = 0x8000;
                for (let i = 0; i < u8.length; i += chunk) {
                    bin += String.fromCharCode.apply(null, u8.subarray(i, i + chunk));
                }
                entry.bytesB64 = btoa(bin);
            }).catch(err => { entry.error = String(err); });
        } else {
            entry.error = 'blob has no arrayBuffer()';
        }
        return url;
    };

    // Also intercept anchor.click() so we can record the suggested filename
    // (the `download` attribute) that Concio attaches to the temporary <a>.
    const origAnchorClick = HTMLAnchorElement.prototype.click;
    HTMLAnchorElement.prototype.click = function() {
        if (this.href && this.href.startsWith('blob:') && window.__webtasks_captures) {
            const entry = window.__webtasks_captures.find(c => c.url === this.href);
            if (entry) entry.name = this.getAttribute('download') || this.download || null;
        }
        return origAnchorClick.apply(this, arguments);
    };

    return { installed: true };
})();
