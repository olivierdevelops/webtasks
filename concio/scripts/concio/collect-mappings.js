// Emit the accumulated users / groups mappings as write-files entries.
// build-message-files.js populates window.__webtasks_users / __webtasks_groups
// across every for-each iteration; this runs once after the loop.

const users  = window.__webtasks_users  || {};
const groups = window.__webtasks_groups || {};

return [
  { path: 'users_mapping.json',  content: JSON.stringify(users, null, 2) },
  { path: 'groups_mapping.json', content: JSON.stringify(groups, null, 2) }
];
