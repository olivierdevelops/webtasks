// Return a flat map of every <meta> tag in the current document by either
// `name` or `property` attribute.
const out = {};
for (const m of document.querySelectorAll('meta')) {
    const key = m.getAttribute('name') || m.getAttribute('property');
    if (!key) continue;
    out[key] = m.getAttribute('content') || '';
}
return out;
