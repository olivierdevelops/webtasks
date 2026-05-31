// Drain `window.__webtasks_captures`, returning entries that have finished
// materialising their bytes (bytesB64 != null) or have errored. Pending
// entries stay in the buffer for a future poll.
const out = [];
const remaining = [];
for (const entry of (window.__webtasks_captures || [])) {
    if (entry.bytesB64 != null || entry.error) out.push(entry);
    else remaining.push(entry);
}
window.__webtasks_captures = remaining;
return out;
