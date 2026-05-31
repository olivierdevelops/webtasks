// scroll-history.js — scrolls a chat's message pane to the top, loading all
// history. Run via `js` with `await: true` (it is an async routine).
//
// Replaces reliance on the flaky scroll-until-stable action: it waits a real
// beat for each lazy-load and only stops after several consecutive no-growth
// iterations. arguments[0] = stopBeforeTs (epoch ms, optional): once a message
// older than this is loaded, stop early — an incremental run need not re-scroll
// history it has already captured.
var el = document.querySelector('.chats.chat-content-scroll');
if (!el) return { ok: false, error: 'no message pane' };

var stopBeforeTs = Number(arguments[0] || 0);
var sleep = function (ms) { return new Promise(function (r) { setTimeout(r, ms); }); };

var lastHeight = -1;
var stable = 0;
var iters = 0;
while (stable < 4 && iters < 600) {
  el.scrollTop = 0;
  await sleep(750);
  var h = el.scrollHeight;
  if (h === lastHeight) {
    stable++;
  } else {
    stable = 0;
    lastHeight = h;
  }
  iters++;

  if (stopBeforeTs > 0) {
    var rows = el.querySelectorAll('.chatList');
    if (rows.length) {
      var firstId = rows[0].id || '';
      var m = firstId.match(/\b(1\d{12})\b/); // 13-digit epoch ms in the dom id
      if (m && Number(m[1]) < stopBeforeTs) break;
    }
  }
}
return { ok: true, iterations: iters, height: lastHeight };
