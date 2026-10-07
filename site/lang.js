/* Axiom — language menu (English / Portuguese).

   The page is written in English. Every translatable element carries the
   Portuguese text next to it in `data-pt`. Inline markup stays in the page;
   translation values are always assigned as text, never parsed as HTML. The
   first switch stores the English original in `data-en`, so switching back needs no
   second copy of the page.

   The globe button (#lang-toggle) opens a menu of flags; picking one applies
   that language. The menu closes on Escape, on a click outside, or after a
   choice. Arrow keys move up and down the list of flags.

   The choice is remembered per visitor. Nothing else is stored: the landing
   page has one palette and no other preference. */
(function () {
  'use strict';

  var STORAGE_KEY = 'axiom-lang';
  var root = document.documentElement;
  var toggle = document.getElementById('lang-toggle');
  var list = document.getElementById('lang-list');

  if (!toggle || !list) return;

  var options = Array.prototype.slice.call(list.querySelectorAll('.lang-option'));
  var hints = { en: 'Language', pt: 'Idioma' };

  function swap(element) {
    var back = 'data-en';

    if (!element.hasAttribute(back)) {
      element.setAttribute(back, element.textContent);
    }
  }

  function apply(lang) {
    var i;

    var plain = document.querySelectorAll('[data-pt]');
    for (i = 0; i < plain.length; i++) {
      swap(plain[i]);
      plain[i].textContent = plain[i].getAttribute(lang === 'pt' ? 'data-pt' : 'data-en');
    }

    root.setAttribute('lang', lang === 'pt' ? 'pt-BR' : 'en');
    toggle.setAttribute('aria-label', hints[lang]);
    toggle.setAttribute('title', hints[lang]);
    list.setAttribute('aria-label', hints[lang]);
    options.forEach(function (option) {
      option.setAttribute('aria-checked', String(option.getAttribute('data-lang') === lang));
    });
  }

  function read() {
    try {
      return localStorage.getItem(STORAGE_KEY) === 'pt' ? 'pt' : 'en';
    } catch (e) {
      /* private mode or blocked storage: English for this visit */
      return 'en';
    }
  }

  function write(lang) {
    try {
      localStorage.setItem(STORAGE_KEY, lang);
    } catch (e) {
      /* the switch still works for this visit */
    }
  }

  /* --- Menu: open, close, keyboard --- */
  function openMenu(focusIndex) {
    list.hidden = false;
    toggle.setAttribute('aria-expanded', 'true');
    var target = options[focusIndex] || options.filter(function (o) {
      return o.getAttribute('aria-checked') === 'true';
    })[0] || options[0];
    target.focus();
  }

  function closeMenu(returnFocus) {
    if (list.hidden) return;
    list.hidden = true;
    toggle.setAttribute('aria-expanded', 'false');
    if (returnFocus) toggle.focus();
  }

  toggle.addEventListener('click', function () {
    if (list.hidden) openMenu(); else closeMenu(false);
  });

  toggle.addEventListener('keydown', function (event) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      openMenu(event.key === 'ArrowDown' ? 0 : options.length - 1);
    }
  });

  options.forEach(function (option, index) {
    option.addEventListener('click', function () {
      current = option.getAttribute('data-lang');
      apply(current);
      write(current);
      closeMenu(true);
    });

    option.addEventListener('keydown', function (event) {
      var step = { ArrowDown: 1, ArrowRight: 1, ArrowUp: -1, ArrowLeft: -1 }[event.key];
      if (event.key === 'Escape') { event.preventDefault(); closeMenu(true); return; }
      if (event.key === 'Tab') { closeMenu(false); return; }
      if (event.key === 'Home') step = -index;
      if (event.key === 'End') step = options.length - 1 - index;
      if (step === undefined) return;
      event.preventDefault();
      options[(index + step + options.length) % options.length].focus();
    });
  });

  document.addEventListener('click', function (event) {
    if (!list.hidden && !list.contains(event.target) && !toggle.contains(event.target)) {
      closeMenu(false);
    }
  });

  var current = read();
  apply(current);
})();
