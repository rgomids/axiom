/* Axiom — Matrix rain background.
   Two glyph columns spawn per second, starting anywhere across the width and
   between the top and the middle of the viewport. Every glyph in a column is
   re-randomised each time the column falls MUTATE_EVERY pixels. Colour comes
   from a CSS custom property, so the theme toggle recolours drops that are
   already falling. */
(function () {
  'use strict';

  /* Portuguese and English letters only — no other alphabets, digits or symbols. */
  var GLYPHS =
    'ABCDEFGHIJKLMNOPQRSTUVWXYZ' +
    'abcdefghijklmnopqrstuvwxyz' +
    '1234567890.,' +
    '+=_-´~][{}';

  var SPAWN_MIN = 350;    /* ms between drops — jittered around 2 per second */
  var SPAWN_MAX = 650;
  var MAX_DROPS = 48;     /* hard cap, so nothing can pile up */
  var FADE_IN = 0.1;      /* fraction of the fall spent fading in */
  var FADE_OUT = 0.2;     /* …and fading out */
  var MUTATE_EVERY = 3;   /* px the column falls between glyph re-rolls */
  var SPEED_MIN = 75;     /* px per second — each column picks its own speed */
  var SPEED_MAX = 250;

  var motion = window.matchMedia('(prefers-reduced-motion: reduce)');
  var drops = [];
  var layer = null;
  var timer = null;
  var frame = null;
  var last = 0;

  function rand(min, max) {
    return Math.random() * (max - min) + min;
  }

  function glyph() {
    return GLYPHS.charAt(Math.floor(Math.random() * GLYPHS.length));
  }

  function spawn() {
    if (document.hidden || drops.length >= MAX_DROPS) return;

    var height = window.innerHeight;
    var startY = rand(0, height * 0.5);          /* top half only */
    var distance = height - startY + 160;
    var count = Math.round(rand(6, 14));

    var el = document.createElement('span');
    el.className = 'rain-drop';
    el.style.left = rand(1, 97).toFixed(2) + 'vw';
    el.style.top = Math.round(startY) + 'px';
    el.style.fontSize = rand(11, 17).toFixed(1) + 'px';

    var cells = [];
    for (var i = 0; i < count; i++) {
      var cell = document.createElement('i');
      cell.className = 'rain-glyph';
      cell.textContent = glyph();
      el.appendChild(cell);
      cells.push(cell);
    }

    layer.appendChild(el);

    drops.push({
      el: el,
      cells: cells,
      y: 0,
      step: -1,                                  /* last MUTATE_EVERY bucket */
      distance: distance,
      speed: rand(SPEED_MIN, SPEED_MAX)          /* px per second */
    });
  }

  function mutate(drop) {
    /* Every glyph in the column is swapped for a random one, head included. */
    for (var i = 0; i < drop.cells.length; i++) {
      drop.cells[i].textContent = glyph();
    }
  }

  function opacityAt(progress) {
    if (progress < FADE_IN) return progress / FADE_IN;
    if (progress > 1 - FADE_OUT) return (1 - progress) / FADE_OUT;
    return 1;
  }

  function step(now) {
    var delta = last ? Math.min((now - last) / 1000, 0.1) : 0;
    last = now;

    for (var i = drops.length - 1; i >= 0; i--) {
      var drop = drops[i];
      drop.y += drop.speed * delta;

      if (drop.y >= drop.distance) {
        drop.el.remove();
        drops.splice(i, 1);
        continue;
      }

      var bucket = Math.floor(drop.y / MUTATE_EVERY);
      if (bucket !== drop.step) {
        drop.step = bucket;
        mutate(drop);
      }

      drop.el.style.transform = 'translate3d(0,' + drop.y.toFixed(1) + 'px,0)';
      drop.el.style.opacity = opacityAt(drop.y / drop.distance).toFixed(3);
    }

    frame = drops.length ? window.requestAnimationFrame(step) : null;
  }

  function pump() {
    spawn();
    if (frame === null && drops.length) {
      last = 0;
      frame = window.requestAnimationFrame(step);
    }
    timer = window.setTimeout(pump, rand(SPAWN_MIN, SPAWN_MAX));
  }

  function start() {
    if (timer !== null || motion.matches) return;
    if (!layer) {
      layer = document.createElement('div');
      layer.className = 'rain';
      layer.setAttribute('aria-hidden', 'true');
      document.body.appendChild(layer);
    }
    pump();
  }

  function stop() {
    if (timer !== null) {
      window.clearTimeout(timer);
      timer = null;
    }
    if (frame !== null) {
      window.cancelAnimationFrame(frame);
      frame = null;
    }
    drops = [];
    if (layer) layer.textContent = '';
  }

  if (!document.body) return;

  document.addEventListener('visibilitychange', function () {
    if (document.hidden) stop();
    else start();
  });

  var onMotionChange = function () { motion.matches ? stop() : start(); };
  if (motion.addEventListener) motion.addEventListener('change', onMotionChange);
  else if (motion.addListener) motion.addListener(onMotionChange);

  start();
})();
