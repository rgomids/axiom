/* Axiom — skills window in "Getting started".

   Each skill in the side list is a tab; clicking it (or using the arrow keys)
   shows that skill's panel. Each panel has one example command and a Copy
   button. The button label follows the page language set by lang.js.
   Nothing is stored. */
(function () {
  'use strict';

  var tabs = Array.prototype.slice.call(document.querySelectorAll('.skill-tab'));
  if (!tabs.length) return;

  var copied = { en: 'Copied', pt: 'Copiado' };

  /* --- Tabs: show one panel at a time --- */
  function selectTab(tab, focus) {
    tabs.forEach(function (other) {
      var active = other === tab;
      other.setAttribute('aria-selected', String(active));
      other.tabIndex = active ? 0 : -1;
      document.getElementById(other.getAttribute('aria-controls')).hidden = !active;
    });
    if (focus) tab.focus();
  }

  tabs.forEach(function (tab, index) {
    tab.addEventListener('click', function () { selectTab(tab, false); });
    tab.addEventListener('keydown', function (event) {
      var step = { ArrowDown: 1, ArrowRight: 1, ArrowUp: -1, ArrowLeft: -1 }[event.key];
      if (event.key === 'Home') step = -index;
      if (event.key === 'End') step = tabs.length - 1 - index;
      if (step === undefined) return;
      event.preventDefault();
      selectTab(tabs[(index + step + tabs.length) % tabs.length], true);
    });
  });

  /* --- Copy: the example text lives in data-copy --- */
  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }
    /* http:// previews (e.g. a local server) have no Clipboard API */
    return new Promise(function (resolve, reject) {
      var area = document.createElement('textarea');
      area.value = text;
      area.setAttribute('readonly', '');
      area.style.position = 'fixed';
      area.style.opacity = '0';
      document.body.appendChild(area);
      area.select();
      var ok = document.execCommand('copy');
      document.body.removeChild(area);
      if (ok) resolve(); else reject(new Error('copy failed'));
    });
  }

  Array.prototype.forEach.call(document.querySelectorAll('.copy-btn'), function (button) {
    var label = button.querySelector('span');

    button.addEventListener('click', function () {
      var original = label.textContent;
      var lang = document.documentElement.lang === 'pt-BR' ? 'pt' : 'en';

      copyText(button.getAttribute('data-copy')).then(function () {
        label.textContent = copied[lang];
        button.classList.add('is-copied');
        window.setTimeout(function () {
          label.textContent = original;
          button.classList.remove('is-copied');
        }, 1600);
      }).catch(function () { /* nothing to undo: the text stays selectable */ });
    });
  });
})();
