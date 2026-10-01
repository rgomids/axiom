/* Axiom — deep-sea background.

   One job: say how far the two side tentacles have slid in. The value goes
   into the --tentacle-reveal custom property and the stylesheet does the rest,
   including standing still under prefers-reduced-motion. */
(function () {
  'use strict';

  /* 0 at the top of the page, 1 once the reader has scrolled roughly two
     thirds of a viewport. */
  var REVEAL_OVER = 0.66;

  var root = document.documentElement;
  var frame = null;

  function write() {
    frame = null;
    var span = window.innerHeight * REVEAL_OVER;
    var reveal = span > 0 ? Math.min(window.scrollY / span, 1) : 1;
    root.style.setProperty('--tentacle-reveal', reveal.toFixed(3));
  }

  function onScroll() {
    if (frame === null) frame = window.requestAnimationFrame(write);
  }

  write();
  window.addEventListener('scroll', onScroll, { passive: true });
  window.addEventListener('resize', onScroll, { passive: true });
})();
