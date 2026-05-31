// Clear the in-page mapping accumulators so a fresh extract-all run doesn't
// inherit peers from a previous sweep in the same browser session.
window.__webtasks_users = {};
window.__webtasks_groups = {};
return true;
