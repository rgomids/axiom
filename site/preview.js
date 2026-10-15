/* Isolated documentation views; no preview parameter changes production. */
const previews = {
  navigation: '.header',
  story: '.storytelling',
  hero: '.storytelling',
  workflow: '.storytelling',
  installation: '.installation',
  'product-status': '.reality',
  closing: '.closing',
  footer: 'footer',
};
const parameters = new URLSearchParams(location.search);
const preview = parameters.has('scene') ? 'story' : parameters.get('preview');
if (Object.hasOwn(previews, preview)) {
  document.body.dataset.preview = preview;
  for (const element of document.querySelectorAll('main > section, .header, footer, .skip'))
    element.hidden = !element.matches(previews[preview]);
  document.querySelector(previews[preview]).hidden = false;
}
