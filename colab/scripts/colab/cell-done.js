// cell-done.js — used as the `untilFn` of colab/run-code's streaming loop.
// Returns true when the cell is no longer executing.
//
// Heuristic: while a cell runs, Colab shows a spinner / an enabled "Interrupt
// execution" control and an animated execution prompt. The cell is "done"
// when none of those are present. Colab's DOM is undocumented — adjust these
// selectors if the loop never terminates or terminates too early.
var running = document.querySelector(
  '.cell .spinner:not([hidden]), ' +
  '.cell .running, ' +
  'paper-spinner[active], ' +
  '[aria-label="Interrupt execution"]:not([disabled])'
);
return !running;
