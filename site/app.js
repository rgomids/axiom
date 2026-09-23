/* Axiom — theme toggle.
   The stored theme is applied by an inline script in <head> so the page never
   flashes the wrong palette; this file only handles switching it afterwards. */
(function () {
  'use strict';

  var STORAGE_KEY = 'axiom-theme';
  var root = document.documentElement;
  var toggle = document.getElementById('theme-toggle');

  if (!toggle) return;

  function write(theme) {
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch (e) {
      /* private mode or blocked storage: the theme still applies for this visit */
    }
  }

  function apply(theme) {
    root.setAttribute('data-theme', theme);
    toggle.setAttribute(
      'aria-label',
      theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'
    );
  }

  /* #86 defines dark as the default, so the OS colour-scheme preference is
     deliberately not followed: only the toggle changes the theme, and only a
     stored choice survives a reload. */
  apply(root.getAttribute('data-theme') === 'light' ? 'light' : 'dark');

  toggle.addEventListener('click', function () {
    var next = root.getAttribute('data-theme') === 'light' ? 'dark' : 'light';
    apply(next);
    write(next);
  });
})();
