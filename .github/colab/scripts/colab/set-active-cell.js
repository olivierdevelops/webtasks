// set-active-cell.js — places code into the first/active code cell.
// arguments[0] = the code string.
//
// Colab cells embed a Monaco editor; setting the model value is far more
// reliable than dispatching raw key events. Falls back to a textarea.
var code = arguments[0] || '';

if (window.monaco && monaco.editor && monaco.editor.getModels().length) {
  var models = monaco.editor.getModels();
  models[0].setValue(code);
  return { ok: true, via: 'monaco' };
}

var ta = document.querySelector('.cell textarea, textarea.inputarea, .inputarea');
if (ta) {
  var proto = window.HTMLTextAreaElement.prototype;
  var setter = Object.getOwnPropertyDescriptor(proto, 'value').set;
  setter.call(ta, code);
  ta.dispatchEvent(new Event('input', { bubbles: true }));
  return { ok: true, via: 'textarea' };
}
return { ok: false, error: 'no editable Colab cell found' };
