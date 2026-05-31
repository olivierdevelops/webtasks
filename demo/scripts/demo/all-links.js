// Return every <a href> on the page, normalised to absolute URLs.
const limit = arguments[0] || 100;
const out = [];
const anchors = document.querySelectorAll('a[href]');
for (let i = 0; i < Math.min(anchors.length, limit); i++) {
    const a = anchors[i];
    out.push({
        text: (a.textContent || '').trim().slice(0, 120),
        href: a.href,
        rel:  a.getAttribute('rel') || ''
    });
}
return out;
