/* Storybook-only section views; production without ?preview is unaffected. */
const previews = {
  navigation: '.header',
  hero: '.opening',
  manifesto: '.manifesto',
  workflow: '.organic-workflow',
  'workflow-transcript': '.workflow-transcript',
  'human-authority': '.human',
  installation: '.installation',
  'product-status': '.reality',
  closing: '.closing',
  footer: 'footer',
};
const preview = new URLSearchParams(location.search).get('preview');
if (Object.hasOwn(previews, preview)) {
  document.body.dataset.preview = preview;
  for (const element of document.querySelectorAll('main > section, .header, footer, .skip')) {
    element.hidden = !element.matches(previews[preview]);
  }
  document.querySelector(previews[preview]).hidden = false;
}
