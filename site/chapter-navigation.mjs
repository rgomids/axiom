const dock = document.querySelector('.story-navigation-dock');
if (dock) {
  const toggle = dock.querySelector('.story-dock-toggle');
  const content = dock.querySelector('.story-dock-content');
  const navigation = dock.querySelector('.story-navigation');
  const story = dock.closest('.storytelling');
  let hovered = false;
  let pinned = false;
  let suppressFocus = false;
  let leaveTimer;
  const motion = () => document.body.classList.contains('story-motion');
  function render() {
    // Touch focus must not move the trigger before its click can pin the drawer.
    const keyboardFocus =
      dock.contains(document.activeElement) &&
      (document.activeElement !== toggle || toggle.matches(':focus-visible'));
    const expanded = !motion() || pinned || hovered || (!suppressFocus && keyboardFocus);
    dock.toggleAttribute('data-open', expanded);
    toggle.hidden = !motion();
    toggle.setAttribute('aria-expanded', String(expanded));
    content.inert = !expanded;
    content.setAttribute('aria-hidden', String(!expanded));
  }
  function close(returnFocus = false) {
    clearTimeout(leaveTimer);
    pinned = hovered = false;
    suppressFocus = true;
    if (returnFocus) toggle.focus({ preventScroll: true });
    render();
  }
  dock.addEventListener('pointerenter', (event) => {
    if (event.pointerType !== 'mouse') return;
    clearTimeout(leaveTimer);
    hovered = true;
    suppressFocus = false;
    render();
  });
  dock.addEventListener('pointerleave', (event) => {
    if (event.pointerType !== 'mouse') return;
    leaveTimer = setTimeout(() => {
      hovered = false;
      render();
    }, 160);
  });
  dock.addEventListener('focusin', () => {
    if (!suppressFocus) render();
  });
  dock.addEventListener('focusout', () =>
    queueMicrotask(() => {
      if (!dock.contains(document.activeElement)) suppressFocus = false;
      render();
    }),
  );
  toggle.addEventListener('click', () => {
    if (pinned) close();
    else {
      pinned = true;
      suppressFocus = false;
      render();
    }
  });
  dock.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      close(true);
    }
    // Reopen after Escape when the user continues navigating with the keyboard.
    if (event.key === 'Tab' && !event.shiftKey && document.activeElement === toggle) {
      event.preventDefault();
      suppressFocus = false;
      pinned = true;
      render();
      dock.querySelector('.story-navigation a').focus({ preventScroll: true });
    }
  });
  dock.querySelectorAll('.story-navigation a').forEach((link) => {
    link.addEventListener('click', () => close(true));
  });
  document.addEventListener('pointerdown', (event) => {
    if (!dock.contains(event.target)) close();
  });
  new MutationObserver(render).observe(document.body, {
    attributes: true,
    attributeFilter: ['class'],
  });
  // Measure the full-width strip, not its animated/clipped wrapper.
  const reserveSpace = () => {
    story.style.setProperty(
      '--story-dock-reserve',
      `${Math.ceil(navigation.getBoundingClientRect().height)}px`,
    );
  };
  new ResizeObserver(reserveSpace).observe(navigation);
  reserveSpace();
  dock.dataset.enhanced = '';
  render();
  requestAnimationFrame(() =>
    requestAnimationFrame(() => {
      dock.dataset.ready = '';
    }),
  );
}
