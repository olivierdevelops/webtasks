// mark-chat-done.js — records that a chat has been extracted up to its current
// latest message, then returns state.json for write-files. Called after each
// chat in concio/watch, so a crash mid-sweep resumes cleanly: chats already
// marked are skipped by chats-needing-refresh.js on the next run.
//
// arguments[0] = the chat object { peerName, isGroup, timestampMs }.
var chat = arguments[0] || {};
var state = window.__webtasks_state;
if (!state || typeof state !== 'object') state = { chats: {} };
if (!state.chats) state.chats = {};

if (chat.peerName) {
  state.chats[chat.peerName] = {
    lastTs: Number(chat.timestampMs || 0),
    isGroup: !!chat.isGroup,
    updatedAt: Date.now(),
  };
}
window.__webtasks_state = state;

return [{ path: 'state.json', content: JSON.stringify(state, null, 2) }];
