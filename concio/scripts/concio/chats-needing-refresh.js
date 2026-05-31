// chats-needing-refresh.js — the "only refresh what changed" filter.
//
// Reads the sidebar (each .chat-list-inner row carries a hidden epoch-ms span)
// and compares every chat's latest-message timestamp to window.__webtasks_state
// (seeded from state.json). Returns only the chats that are new or whose
// latest message is newer than what we already extracted — so a re-run never
// re-scans unchanged chats and never resets data. No chat is opened to decide.
//
// Each returned item: { peerName, isGroup, timestampMs, lastTs }
//   timestampMs — the chat's current latest-message time (the new high-water).
//   lastTs      — what we previously had (0 if new); scroll-history stops here.

function stripCount(s) {
  return (s || '').replace(/\s*\(\d+\)\s*$/, '').trim();
}

var groupNames = new Set();
document.querySelectorAll('.leftSide-group-list-item').forEach(function (g) {
  var n = g.querySelector('.name');
  if (!n) return;
  var nm = stripCount(n.textContent.trim());
  if (nm && nm !== 'Add Group') groupNames.add(nm);
});

var state = (window.__webtasks_state && window.__webtasks_state.chats) || {};
var out = [];
var seen = new Set();

document.querySelectorAll('.chat-list-inner').forEach(function (row) {
  var n = row.querySelector('.name');
  if (!n) return;
  var name = n.textContent.trim();
  if (!name || seen.has(name)) return;
  seen.add(name);

  // Hidden epoch-ms span — the true recency value, available without opening.
  var tsSpan = row.querySelector("span[style*='display: none']:not([class])");
  var ts = tsSpan ? Number((tsSpan.textContent || '').trim()) : 0;
  if (!ts || isNaN(ts)) ts = 0;

  var prev = state[name];
  var lastTs = prev && prev.lastTs ? Number(prev.lastTs) : 0;

  // New chat, no recorded ts, or a newer message than we have → needs refresh.
  if (!prev || ts === 0 || ts > lastTs) {
    out.push({
      peerName: name,
      isGroup: groupNames.has(name),
      timestampMs: ts,
      lastTs: lastTs,
    });
  }
});

return out;
