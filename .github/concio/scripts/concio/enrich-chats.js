// Read the sidebar chat list and classify each row as a group or a DM by
// checking its name against the groups directory. Returns a de-duplicated
// list of { peerName, isGroup } the extract-all task iterates with for-each.

function stripCount(s) {
  return (s || '').replace(/\s*\(\d+\)\s*$/, '').trim();
}

const groupNames = new Set();
document.querySelectorAll('.leftSide-group-list-item').forEach(function (g) {
  const n = g.querySelector('.name');
  if (!n) return;
  const nm = stripCount(n.textContent.trim());
  if (nm && nm !== 'Add Group') groupNames.add(nm);
});

const out = [];
const seen = new Set();
document.querySelectorAll('.chat-list-inner').forEach(function (row) {
  const n = row.querySelector('.name');
  if (!n) return;
  const name = n.textContent.trim();
  if (!name || seen.has(name)) return;
  seen.add(name);
  out.push({ peerName: name, isGroup: groupNames.has(name) });
});

return out;
