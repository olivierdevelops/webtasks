// Parse the currently-open Concio chat into per-message JSON files.
//
//   arguments[0] = owner login id  (e.g. "200 008 6861"; spaces stripped -> owner id)
//   arguments[1] = peer display name
//   arguments[2] = isGroup (boolean)
//
// Returns { peerId, peerName, isGroup, total, kept, files } where `files` is a
// list of { path, content } the generic `write-files` action persists. Also
// accumulates the peer + owner into window.__webtasks_users / __webtasks_groups
// so a later `collect-mappings` step can emit the mapping files.
//
// This is the JS port of the former Python client's per-chat parsing — kept
// entirely in the bundle so the engine stays generic.

const ownerArg = String(arguments[0] || '');
const peerName = String(arguments[1] || '');
const isGroup  = arguments[2] === true || arguments[2] === 'true';
const owner = ownerArg.replace(/\s+/g, '') || 'owner';

const EPOCH_MIN = 1000000000000;   // ~2001
const EPOCH_MAX = 2000000000000;   // ~2033
const WEEKDAYS = new Set(['mon','tue','wed','thu','fri','sat','sun',
  'monday','tuesday','wednesday','thursday','friday','saturday','sunday']);
const CT_MAP = {
  text:'text', img:'img', image:'img', cryptoimg:'cryptoImg',
  video:'video', cryptovideo:'cryptoVideo', audio:'audio', pttaudio:'pttAudio',
  cryptoaudio:'cryptoAudio', file:'file', cryptofile:'cryptoFile',
  quote:'quote', cryptoquote:'cryptoQuote', location:'location',
  cryptolocation:'cryptoLocation', sticker:'sticker', systemcall:'systemCall',
  sysmsg:'sysMsg', system:'system', shareinfo:'shareInfo', cryptotext:'cryptoText'
};

function pad(n) { return String(n).padStart(2, '0'); }

function slug(s) {
  return (s || '').normalize('NFKD').replace(/[^A-Za-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '').toLowerCase() || 'unknown';
}

// Deterministic 6-hex-char hash — stands in for the Python md5()[:6] fallback
// used only when a dom id carries no usable sequence segment.
function shortHash(s) {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = (h * 16777619) >>> 0; }
  return ('000000' + h.toString(16)).slice(-6);
}

function txt(root, sel) {
  const el = root.querySelector(sel);
  return el ? el.textContent.trim() : '';
}
function attr(root, sel, name) {
  const el = root.querySelector(sel);
  return el ? (el.getAttribute(name) || '') : '';
}

// `_msgId_<seg>_<seg>_<seg>` -> [tsMsOrNull, seq]. Scans every segment for a
// valid 13-digit epoch ms (Concio's dom-id layout is inconsistent).
function parseDomId(domId) {
  if (!domId || domId.indexOf('_msgId_') !== 0) return [null, null];
  const parts = domId.split('_');
  if (parts.length < 3) return [null, null];
  const last = parts[parts.length - 1];
  const seq = /^[A-Za-z0-9-]+$/.test(last) ? last : '0';
  for (let i = 1; i < parts.length; i++) {
    if (!/^\d+$/.test(parts[i])) continue;
    const v = Number(parts[i]);
    if (v >= EPOCH_MIN && v < EPOCH_MAX) return [v, seq];
  }
  return [null, seq];
}

// UTC-midnight Date for "now".
function utcToday() {
  const n = new Date();
  return new Date(Date.UTC(n.getUTCFullYear(), n.getUTCMonth(), n.getUTCDate()));
}
function addDays(d, n) { return new Date(d.getTime() + n * 86400000); }

// Resolve a Concio date-separator label ('Today' / 'Yesterday' / 'Mon, 05/18'
// / '04/16') to a UTC-midnight Date, or null.
function parseDateLabel(label, today) {
  if (!label) return null;
  const s = label.trim();
  if (s.toLowerCase() === 'today') return today;
  if (s.toLowerCase() === 'yesterday') return addDays(today, -1);
  const m = s.match(/^(?:[A-Za-z]+,\s*)?(\d{1,2})\/(\d{1,2})(?:\/(\d{2,4}))?$/);
  if (!m) return null;
  const month = +m[1], day = +m[2];
  let year = m[3] ? +m[3] : today.getUTCFullYear();
  if (year < 100) year += 2000;
  let d = new Date(Date.UTC(year, month - 1, day));
  if (d.getUTCMonth() !== month - 1) return null;   // invalid date rolled over
  if (!m[3] && d > today) d = new Date(Date.UTC(year - 1, month - 1, day));
  return d;
}

function parseVisibleTsTime(text, d) {
  let m = text.match(/^(AM|PM|am|pm)\s*(\d{1,2}):(\d{2})(?::(\d{2}))?\s*$/);
  let hour, minute, sec, ampm;
  if (m) {
    ampm = m[1].toUpperCase(); hour = +m[2]; minute = +m[3]; sec = +(m[4] || 0);
  } else {
    m = text.match(/^(\d{1,2}):(\d{2})(?::(\d{2}))?\s*(AM|PM|am|pm)?\s*$/);
    if (!m) return null;
    hour = +m[1]; minute = +m[2]; sec = +(m[3] || 0); ampm = (m[4] || '').toUpperCase();
  }
  if (ampm === 'PM' && hour < 12) hour += 12;
  if (ampm === 'AM' && hour === 12) hour = 0;
  return Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate(), hour, minute, sec);
}

// Parse a visible per-message timestamp ('AM 10:19', 'Fri PM 03:18',
// '04/16 PM 02:49') combined with the running date context. Returns epoch ms.
function parseVisibleTs(text, current, today) {
  if (!text) return null;
  let s = text.trim();
  const md = s.match(/^(\d{1,2})\/(\d{1,2})\s+(.*)$/);
  if (md) {
    const month = +md[1], day = +md[2];
    let d = new Date(Date.UTC(current.getUTCFullYear(), month - 1, day));
    if (d.getUTCMonth() !== month - 1) d = current;
    else if (d > today) d = new Date(Date.UTC(current.getUTCFullYear() - 1, month - 1, day));
    return parseVisibleTsTime(md[3], d);
  }
  const wd = s.match(/^([A-Za-z]+)\s+(.*)$/);
  if (wd) {
    const head = wd[1].toLowerCase();
    if (WEEKDAYS.has(head)) s = wd[2];
    else if (head === 'today') { s = wd[2]; current = today; }
    else if (head === 'yesterday') { s = wd[2]; current = addDays(today, -1); }
  }
  return parseVisibleTsTime(s, current);
}

function normalizeContentType(media, inType, outType, content, fileName) {
  const ct = (outType || inType || media || '').trim().toLowerCase();
  if (fileName) {
    return (ct.indexOf('crypto') >= 0 || (content || '').toLowerCase().indexOf('encrypt') >= 0)
      ? 'cryptoFile' : 'file';
  }
  return CT_MAP[ct] || ct || 'text';
}

// ---- main ----------------------------------------------------------------

const container = document.querySelector('.chats.chat-content-scroll');
const containerId = container ? container.id : '';
let peerId = containerId.replace(/^chatContext_/, '');
if (!peerId) peerId = (isGroup ? 'group_' : 'user_') + slug(peerName);

// Accumulate mappings in window globals (persist across for-each iterations).
window.__webtasks_users = window.__webtasks_users || {};
window.__webtasks_groups = window.__webtasks_groups || {};
window.__webtasks_users[owner] = { name: 'Me' };
if (isGroup) window.__webtasks_groups[peerId] = { name: peerName };
else window.__webtasks_users[peerId] = { name: peerName };

const files = [];
let total = 0, kept = 0;
if (container) {
  const rows = container.querySelectorAll('.chatList');
  const today = utcToday();
  let current = today;
  for (const row of rows) {
    total++;
    const media = (row.getAttribute('data-media-type') || '').trim();
    const midFirst  = txt(row, '.chat-middle-datetime > div:first-child');
    const midSecond = txt(row, '.chat-middle-datetime > div:nth-child(2)');

    // Date-separator row.
    if ((midFirst || midSecond) && !media) {
      const d = parseDateLabel(midFirst || midSecond, today);
      if (d) current = d;
      continue;
    }

    // System-message row.
    let sysMsg = null, visibleForTs = '';
    if (media === 'sysMsg' && (midFirst || midSecond)) {
      sysMsg = midSecond || midFirst;
      visibleForTs = midSecond ? midFirst : '';
    } else {
      visibleForTs = txt(row, '.chat-right-box-timestamp') || txt(row, '.chat-left-box-timestamp');
    }

    const parsed = parseDomId(row.id);
    let tsMs = parsed[0];
    const seq = parsed[1];
    if (tsMs === null) {
      tsMs = parseVisibleTs(visibleForTs, current, today);
      if (tsMs === null) continue;   // unusable row
    }
    const hi = String(tsMs);
    const lo = seq || shortHash(row.id || hi);

    const contentRight = txt(row, '.chat-right-box-message');
    const contentLeft  = txt(row, '.chat-left-box-message');
    const replyRight = txt(row, '.chat-right .reply-msg .max3line')
                    || txt(row, '.chat-right .reply-msg .historyText');
    const replyLeft  = txt(row, '.chat-left .reply-msg .max3line')
                    || txt(row, '.chat-left .reply-msg .historyText');
    const fileNameRight = txt(row, '.chat-right-file-box-text-file-name');
    const fileNameLeft  = txt(row, '.chat-left-file-box-text-file-name');
    const fileName = fileNameRight || fileNameLeft;

    let direction, sender, recipient, content;
    if (sysMsg) {
      direction = 'mt'; sender = 'system'; recipient = owner; content = sysMsg;
    } else if (contentRight || replyRight) {
      direction = 'mo'; sender = owner; recipient = peerId; content = contentRight || replyRight;
    } else if (contentLeft || replyLeft) {
      direction = 'mt'; sender = peerId; recipient = owner; content = contentLeft || replyLeft;
    } else if (fileName) {
      if (fileNameRight) { direction = 'mo'; sender = owner; recipient = peerId; }
      else { direction = 'mt'; sender = peerId; recipient = owner; }
      content = fileName;
    } else {
      continue;   // sticker w/o text, etc.
    }

    const inType  = attr(row, '.chat-left [data-contenttype]', 'data-contenttype');
    const outType = attr(row, '.chat-right [data-contenttype]', 'data-contenttype');
    let contentType = normalizeContentType(media, inType, outType, content, fileName);
    if (sysMsg) contentType = 'sysMsg';

    const msg = {
      msgId: hi + '_' + lo,
      contentType: contentType,
      content: content,
      from: sender,
      to: recipient,
      timestamp: tsMs,
      isBroadcast: false
    };
    if (isGroup) msg.groupId = peerId;

    const dt = new Date(tsMs);
    const yyyy = dt.getUTCFullYear();
    const dir = owner + '/chats/' + yyyy + '/' + pad(dt.getUTCMonth() + 1) + '/' + pad(dt.getUTCDate());
    const fname = 'chat_' + owner + '_' + peerId + '_' + tsMs + '_' + direction + '_' + hi + '_' + lo + '.json';
    files.push({ path: dir + '/' + fname, content: JSON.stringify(msg, null, 2) });
    kept++;
  }
}

return { peerId: peerId, peerName: peerName, isGroup: isGroup, total: total, kept: kept, files: files };
