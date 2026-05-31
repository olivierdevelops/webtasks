// seed-state.js — initialises the in-page watch state from state.json.
// arguments[0] = the raw contents of <out>/state.json ("" when it does not
// exist yet). The state shape is { chats: { <peerName>: { lastTs, isGroup } } }.
// chats-needing-refresh.js and mark-chat-done.js read/update this global.
var raw = arguments[0] || '';
var state = {};
if (raw) {
  try { state = JSON.parse(raw); } catch (e) { state = {}; }
}
if (!state || typeof state !== 'object') state = {};
if (!state.chats) state.chats = {};
window.__webtasks_state = state;
return { ok: true, knownChats: Object.keys(state.chats).length };
