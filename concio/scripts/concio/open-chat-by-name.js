// Click the sidebar chat-list row whose visible name equals arguments[0].
// Vue often binds the click handler to the inner `.message-panel`, so we
// dispatch a real mousedown/mouseup/click sequence rather than calling
// HTMLElement.click() which the SPA may ignore.
const name = arguments[0];
const rows = document.querySelectorAll('.chat-list-inner');
const match = Array.from(rows).find(r => {
    const n = r.querySelector('.name');
    return n && n.textContent.trim() === name;
});
if (!match) return false;
const target = match.querySelector('.message-panel') || match.querySelector('.user-name') || match;
const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
target.dispatchEvent(new MouseEvent('mousedown', opts));
target.dispatchEvent(new MouseEvent('mouseup', opts));
target.dispatchEvent(new MouseEvent('click', opts));
return true;
