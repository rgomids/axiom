/* Axiom — theme toggle.
   The stored theme is applied by an inline script in <head> so the page never
   flashes the wrong palette; this file only handles switching it afterwards. */
(function () {
  'use strict';

  var STORAGE_KEY = 'axiom-theme';
  var root = document.documentElement;
  var toggle = document.getElementById('theme-toggle');

  if (!toggle) return;

  function read() {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch (e) {
      return null;
    }
  }

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

  apply(root.getAttribute('data-theme') === 'light' ? 'light' : 'dark');

  toggle.addEventListener('click', function () {
    var next = root.getAttribute('data-theme') === 'light' ? 'dark' : 'light';
    apply(next);
    write(next);
  });

  // Follow the OS while the visitor has not made an explicit choice.
  var query = window.matchMedia('(prefers-color-scheme: light)');
  var onChange = function (event) {
    if (!read()) apply(event.matches ? 'light' : 'dark');
  };

  if (query.addEventListener) query.addEventListener('change', onChange);
  else if (query.addListener) query.addListener(onChange);
})();
