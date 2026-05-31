// run-active-cell.js — triggers execution of the focused/first cell.
// Colab runs the current cell on Ctrl+Enter; we dispatch that to the cell's
// editor element. Returns { ok }.
var editor = document.querySelector('.monaco-editor');
var cell = editor || document.querySelector('.cell');
if (!cell) return { ok: false, error: 'no cell to run' };

var target = document.querySelector('.monaco-editor textarea.inputarea')
  || document.querySelector('.cell textarea')
  || cell;
if (target.focus) target.focus();

function key(type) {
  return new KeyboardEvent(type, {
    key: 'Enter', code: 'Enter', keyCode: 13, which: 13,
    ctrlKey: true, bubbles: true, cancelable: true
  });
}
target.dispatchEvent(key('keydown'));
target.dispatchEvent(key('keypress'));
target.dispatchEvent(key('keyup'));
return { ok: true };
