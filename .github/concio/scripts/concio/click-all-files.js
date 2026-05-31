// Click every file bubble in the open chat panel in DOM order, hiding
// overlays before each click. The actual download bytes are captured by
// the URL.createObjectURL hook installed by concio/install-download-hook
// (see concio/poll-captures or the `save-captures-to-dir` action).
const bubbles = Array.from(document.querySelectorAll(
    '.chats.chat-content-scroll .chat-left-file-box, '
    + '.chats.chat-content-scroll .chat-right-file-box'
));
const clicked = [];
for (const el of bubbles) {
    el.scrollIntoView({ block: 'center', inline: 'center' });
    document.querySelectorAll('.to-bottom-chat, .modalRoot.adInvite, [class*="unread-message-tip"]')
        .forEach(o => { o.style.display = 'none'; o.style.pointerEvents = 'none'; });
    // dispatched events don't trigger Concio's handler, so this is best-effort
    // for the rare case where it does; the actual click is performed by the
    // calling action via native WebElement.click()
    const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
    el.dispatchEvent(new MouseEvent('mousedown', opts));
    el.dispatchEvent(new MouseEvent('mouseup', opts));
    el.dispatchEvent(new MouseEvent('click', opts));
    clicked.push({
        side: el.className.includes('chat-right') ? 'right' : 'left',
        rect: el.getBoundingClientRect && (() => { const r = el.getBoundingClientRect(); return { x: r.x, y: r.y, w: r.width, h: r.height }; })()
    });
}
return { clicked: clicked.length, items: clicked };
