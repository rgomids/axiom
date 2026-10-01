/* Axiom — language switch (English / Portuguese).

   The page is written in English. Every translatable element carries the
   Portuguese version next to it, in `data-pt` for plain text or `data-pt-html`
   when the string wraps inline markup. The first switch stores the English
   original in the matching `data-en` attribute, so switching back needs no
   second copy of the page.

   The choice is remembered per visitor. Nothing else is stored: the landing
   page has one palette and no other preference. */
(function () {
  'use strict';

  var STORAGE_KEY = 'axiom-lang';
  var root = document.documentElement;
  var toggle = document.getElementById('lang-toggle');

  if (!toggle) return;

  var labels = { en: 'PT', pt: 'EN' };
  var hints = {
    en: 'Mudar para português',
    pt: 'Switch to English'
  };

  function swap(element, plain) {
    var back = plain ? 'data-en' : 'data-en-html';

    if (!element.hasAttribute(back)) {
      element.setAttribute(back, plain ? element.textContent : element.innerHTML);
    }
  }

  function apply(lang) {
    var i;

    var plain = document.querySelectorAll('[data-pt]');
    for (i = 0; i < plain.length; i++) {
      swap(plain[i], true);
      plain[i].textContent = plain[i].getAttribute(lang === 'pt' ? 'data-pt' : 'data-en');
    }

    var rich = document.querySelectorAll('[data-pt-html]');
    for (i = 0; i < rich.length; i++) {
      swap(rich[i], false);
      rich[i].innerHTML = rich[i].getAttribute(lang === 'pt' ? 'data-pt-html' : 'data-en-html');
    }

    root.setAttribute('lang', lang === 'pt' ? 'pt-BR' : 'en');
    toggle.textContent = labels[lang];
    toggle.setAttribute('aria-label', hints[lang]);
    toggle.setAttribute('title', hints[lang]);
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

  var current = read();
  apply(current);

  toggle.addEventListener('click', function () {
    current = current === 'pt' ? 'en' : 'pt';
    apply(current);
    write(current);
  });
})();
