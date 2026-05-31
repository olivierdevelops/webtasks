// Returns a compact summary of the open chat panel for diagnostics:
//   { containerId, chatListCount, firstThreeOuter: [...], headerText }
const container = document.querySelector('.chats.chat-content-scroll');
if (!container) return { error: 'no .chats.chat-content-scroll' };
const rows = container.querySelectorAll('.chatList');
const headerEl = document.querySelector('.chat-message-link-title')
              || document.querySelector('[class*="header"] .name')
              || document.querySelector('[class*="header"]');
const ownerId = (function() {
    const av = document.querySelector('[data-org-src*="/file/"]');
    if (!av) return null;
    const m = av.getAttribute('data-org-src').match(/\/file\/([A-Z]\d+)_/);
    return m ? m[1] : null;
})();
return {
    containerId: container.id,
    chatListCount: rows.length,
    firstThreeOuter: Array.from(rows).slice(0, 3).map(r => r.outerHTML.slice(0, 600)),
    headerText: headerEl ? headerEl.textContent.trim().slice(0, 200) : null,
    bodyClasses: (document.querySelector('.style-text, .style-quote, .style-img, .style-file, .style-system, .style-sticker, .style-location') || {}).className,
    ownerAvatarPrefix: ownerId
};
