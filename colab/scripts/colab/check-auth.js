// check-auth.js — reports whether a Colab notebook is loaded (i.e. we are
// authenticated and inside a notebook) rather than on a Google login page.
// Returns true / false.
//
// Colab's DOM is undocumented and changes; these selectors are best-effort.
var hasNotebook = !!document.querySelector(
  '.notebook-content, .cell, colab-shaded-scroller, div.codecell-input-output'
);
var onLogin = /accounts\.google\.com|\/signin/.test(location.href);
return hasNotebook && !onLogin;
