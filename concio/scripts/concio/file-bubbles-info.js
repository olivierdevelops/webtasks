// Inspect the currently-open chat panel and return the per-file metadata we
// need to label downloads after the fact. The browser will click each bubble
// (in DOM order) for us; this script only describes them.
//
// Returns: [{ side, fileName, msgDomId, timestampText }, ...]
const out = [];
const rows = document.querySelectorAll('.chats.chat-content-scroll .chatList');
for (const row of rows) {
    const side = row.querySelector('.chat-right-file-box, .chat-right-quote-file-box')
        ? 'right'
        : row.querySelector('.chat-left-file-box, .chat-left-quote-file-box')
            ? 'left'
            : null;
    if (!side) continue;
    const nameEl = row.querySelector(
        '.chat-' + side + '-file-box-text-file-name, '
        + '.chat-' + side + '-quote-file-box .chat-' + side + '-file-box-text-file-name'
    );
    const tsEl = row.querySelector('.chat-' + side + '-box-timestamp');
    out.push({
        side,
        fileName: nameEl ? nameEl.textContent.trim() : null,
        msgDomId: row.id,
        timestampText: tsEl ? tsEl.textContent.trim() : null
    });
}
return out;
