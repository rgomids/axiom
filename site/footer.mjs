const language = document.querySelector('#footer-language');
if (language && typeof window.setLanguage === 'function') {
  const sync = () => {
    language.value = document.documentElement.lang.startsWith('pt') ? 'pt' : 'en';
  };
  document.querySelector('[data-footer-language]').hidden = false;
  language.addEventListener('change', () => window.setLanguage(language.value));
  addEventListener('axiom:settings', sync);
  sync();
}
