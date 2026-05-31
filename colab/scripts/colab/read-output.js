// read-output.js — returns the current text of the first cell's output area.
// Used by colab/run-code's streaming `loop` and for the final result.
var out = document.querySelector(
  '.cell .output, .output-content, .outputview, colab-static-output-renderer'
);
return out ? (out.innerText || out.textContent || '').trim() : '';
