/* Axiom — flow diagram in "How it works" (based on axiom-diagrama/).

   Draws one wire from each input to the core and from the core to each
   output, then moves a small signal along them: inputs first, outputs after.
   The rings around the core pulse in CSS (styles.css "Flow diagram").

   Under prefers-reduced-motion the wires stay and nothing moves. Nothing is
   stored. */
(function () {
  'use strict';

  var root = document.getElementById('flow');
  if (!root) return;

  var stage = root.querySelector('.flow-stage');
  var svg = root.querySelector('.flow-wires');
  var NS = 'http://www.w3.org/2000/svg';
  var CYCLE = 4800;        /* one full pass, in ms: inputs then outputs */
  var STACKED = 760;       /* below this width the diagram is vertical */

  var reduced = window.matchMedia('(prefers-reduced-motion: reduce)');
  var paths = [];
  var dots = [];
  var time = 0;
  var last = 0;

  function makeSvg(tag, attrs) {
    var node = document.createElementNS(NS, tag);
    Object.keys(attrs).forEach(function (key) { node.setAttribute(key, attrs[key]); });
    return node;
  }

  function bounds(name) {
    return root.querySelector('[data-node="' + name + '"]').getBoundingClientRect();
  }

  /* Wires: recomputed whenever the stage changes size */
  function drawWires() {
    var box = stage.getBoundingClientRect();
    var stacked = box.width < STACKED;
    var core = bounds('core');   /* whole core, label included: vertical wires */
    var frame = bounds('frame'); /* logo tile only: horizontal wires meet its middle */
    var coreY = frame.top + frame.height / 2 - box.top;
    var coreX = core.left + core.width / 2 - box.left;

    svg.replaceChildren();
    svg.setAttribute('viewBox', '0 0 ' + box.width + ' ' + box.height);
    paths = [];
    dots = [];

    for (var i = 0; i < 6; i++) {
      var incoming = i < 3;
      var node = bounds((incoming ? 'in' : 'out') + (i % 3));
      var from;
      var to;

      if (stacked) {
        var nodeX = node.left + node.width / 2 - box.left;
        from = incoming ? [nodeX, node.bottom - box.top] : [coreX, core.bottom - box.top];
        to = incoming ? [coreX, core.top - box.top] : [nodeX, node.top - box.top];
      } else {
        var nodeY = node.top + node.height / 2 - box.top;
        from = incoming ? [node.right - box.left + 8, nodeY] : [frame.right - box.left, coreY];
        to = incoming ? [frame.left - box.left, coreY] : [node.left - box.left - 8, nodeY];
      }

      var mid = stacked ? (from[1] + to[1]) / 2 : (from[0] + to[0]) / 2;
      var d = stacked
        ? 'M' + from + ' C' + from[0] + ',' + mid + ' ' + to[0] + ',' + mid + ' ' + to
        : 'M' + from + ' C' + mid + ',' + from[1] + ' ' + mid + ',' + to[1] + ' ' + to;

      var path = makeSvg('path', { d: d, class: 'flow-wire' });
      var dot = makeSvg('circle', { r: 3, class: 'flow-dot' });
      svg.append(path, dot);
      paths.push(path);
      dots.push(dot);
    }
    paintSignals();
  }

  /* Signals: each wire carries one dot during its slice of the cycle */
  function paintSignals() {
    var phase = (time % CYCLE) / CYCLE;

    paths.forEach(function (path, i) {
      var offset = (i % 3) * 0.045;
      var start = i < 3 ? offset : 0.5 + offset;
      var end = i < 3 ? 0.38 + offset : 0.9 + offset;
      var progress = Math.max(0, Math.min(1, (phase - start) / (end - start)));
      var point = path.getPointAtLength(path.getTotalLength() * progress);

      dots[i].setAttribute('cx', point.x);
      dots[i].setAttribute('cy', point.y);
      dots[i].style.opacity = !reduced.matches && phase >= start && phase <= end ? '1' : '0';
    });
  }

  function tick(now) {
    if (last && !reduced.matches) time += Math.min(now - last, 60);
    last = now;
    paintSignals();
    window.requestAnimationFrame(tick);
  }

  new ResizeObserver(drawWires).observe(stage);
  window.requestAnimationFrame(tick);
})();
