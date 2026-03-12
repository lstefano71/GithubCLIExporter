package main

const cssContent = `
:root {
  --bg: #ffffff; --fg: #1a1a2e; --bg2: #f5f5f7; --border: #d1d5db;
  --accent: #2563eb; --accent2: #7c3aed; --success: #16a34a; --error: #dc2626;
  --code-bg: #f1f5f9; --code-fg: #334155; --user-bg: #eff6ff; --assistant-bg: #f9fafb;
  --summary-bg: #f8fafc; --shadow: rgba(0,0,0,0.05);
}
[data-theme="dark"] {
  --bg: #0f172a; --fg: #e2e8f0; --bg2: #1e293b; --border: #334155;
  --accent: #60a5fa; --accent2: #a78bfa; --success: #4ade80; --error: #f87171;
  --code-bg: #1e293b; --code-fg: #e2e8f0; --user-bg: #1e293b; --assistant-bg: #0f172a;
  --summary-bg: #1e293b; --shadow: rgba(0,0,0,0.3);
}
*, *::before, *::after { box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg); color: var(--fg); margin: 0; padding: 0;
  line-height: 1.6; font-size: 15px;
}
.container { margin: 0 auto; padding: 2rem 4rem 2rem 4rem; }
.has-minimap .container { padding-right: 2.5rem; }
h1 { font-size: 1.8rem; border-bottom: 2px solid var(--accent); padding-bottom: 0.5rem; margin-top: 0; }
h2 { font-size: 1.4rem; color: var(--accent); margin-top: 2.5rem; border-bottom: 1px solid var(--border); padding-bottom: 0.3rem; }
h3 { font-size: 1.1rem; margin-top: 1.5rem; }
a { color: var(--accent); }
.meta-list { list-style: none; padding: 0; }
.meta-list li { padding: 0.2rem 0; }
.meta-list strong { min-width: 140px; display: inline-block; }
pre, code { font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace; font-size: 0.9em; }
code { background: var(--code-bg); color: var(--code-fg); padding: 0.15em 0.35em; border-radius: 4px; }
pre { background: var(--code-bg); color: var(--code-fg); padding: 1rem; border-radius: 8px; overflow-x: auto; border: 1px solid var(--border); }
pre code { background: none; padding: 0; }
table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
th, td { border: 1px solid var(--border); padding: 0.5rem 0.75rem; text-align: left; }
th { background: var(--bg2); font-weight: 600; }
.turn { margin: 1.5rem 0; padding: 1rem 1.25rem; border-radius: 10px; border: 1px solid var(--border); }
.turn-user { background: var(--user-bg); border-left: 4px solid var(--accent); }
.turn-assistant { background: var(--assistant-bg); border-left: 4px solid var(--accent2); }
.turn-header { font-weight: 600; font-size: 0.95rem; margin-bottom: 0.5rem; opacity: 0.8; }
.turn-content { word-wrap: break-word; }
.turn-content h1, .turn-content h2, .turn-content h3,
.turn-content h4, .turn-content h5, .turn-content h6 {
  color: var(--fg); border: none; margin-top: 1rem; margin-bottom: 0.5rem;
}
.turn-content h1 { font-size: 1.3rem; }
.turn-content h2 { font-size: 1.15rem; }
.turn-content h3 { font-size: 1.05rem; }
.turn-content ul, .turn-content ol { padding-left: 1.5rem; margin: 0.5rem 0; }
.turn-content blockquote {
  border-left: 3px solid var(--accent); margin: 0.5rem 0; padding: 0.25rem 1rem;
  background: var(--bg2); border-radius: 0 6px 6px 0;
}
.turn-content p { margin: 0.4rem 0; }
details { margin: 0.75rem 0; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
details > summary {
  padding: 0.6rem 1rem; background: var(--summary-bg); cursor: pointer;
  font-weight: 500; user-select: none; list-style: none;
}
details > summary::-webkit-details-marker { display: none; }
details > summary::before { content: '▶ '; font-size: 0.75em; transition: transform 0.2s; display: inline-block; }
details[open] > summary::before { transform: rotate(90deg); }
details > .detail-content { padding: 0.75rem 1rem; }
.tool-label { color: var(--accent2); }
.thinking-label { color: #b45309; }
.subagent-label { color: var(--accent); }
.success { color: var(--success); }
.error { color: var(--error); }
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin: 1rem 0; }
.stat-card {
  background: var(--bg2); border: 1px solid var(--border); border-radius: 8px;
  padding: 1rem; text-align: center;
}
.stat-card .stat-value { font-size: 1.5rem; font-weight: 700; color: var(--accent); }
.stat-card .stat-label { font-size: 0.85rem; opacity: 0.7; }
.theme-toggle {
  position: fixed; top: 1rem; right: 1rem; z-index: 100;
  background: var(--bg2); border: 1px solid var(--border); border-radius: 8px;
  padding: 0.5rem 0.75rem; cursor: pointer; font-size: 1.2rem;
}
.toc { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 1rem 1.5rem; margin: 1rem 0; }
.toc ul { list-style: none; padding-left: 1rem; }
.toc > ul { padding-left: 0; }
.toc a { text-decoration: none; }
.toc a:hover { text-decoration: underline; }
html.has-minimap { scrollbar-width: none; }
html.has-minimap::-webkit-scrollbar { display: none; }
.scroll-track {
  position: fixed; right: 0; top: 0; bottom: 0; width: 24px;
  background: var(--bg2); border-left: 1px solid var(--border);
  z-index: 90; cursor: pointer; overflow: hidden;
}
.scroll-track .viewport-indicator {
  position: absolute; left: 0; right: 0;
  background: var(--accent); opacity: 0.18;
  border-radius: 2px; pointer-events: auto; cursor: grab;
  transition: opacity 0.12s;
}
.scroll-track .viewport-indicator:hover,
.scroll-track .viewport-indicator.dragging { opacity: 0.32; cursor: grabbing; }
.scroll-track .tick {
  position: absolute; left: 4px; right: 4px; height: 3px;
  border-radius: 1.5px; pointer-events: none;
}
.scroll-track .tick-user { background: var(--accent); }
.scroll-track .tick-assistant { background: var(--accent2); }
.scroll-track .tick.active { left: 2px; right: 2px; height: 4px; opacity: 1; box-shadow: 0 0 4px var(--accent); }
.track-detail {
  display: none; position: fixed; right: 28px; z-index: 200;
  background: var(--bg); border: 1px solid var(--border); border-radius: 8px;
  padding: 0.4rem 0; min-width: 260px; max-width: 380px;
  box-shadow: 0 4px 16px var(--shadow); font-size: 0.82rem;
  pointer-events: none;
}
.track-detail.visible { display: block; }
.track-detail-entry {
  display: flex; align-items: center; gap: 0.5rem;
  padding: 0.25rem 0.75rem; white-space: nowrap; overflow: hidden;
}
.track-detail-entry.current { background: var(--bg2); font-weight: 600; }
.track-detail-dot {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
}
.track-detail-dot-user { background: var(--accent); }
.track-detail-dot-assistant { background: var(--accent2); }
.track-detail-ts { opacity: 0.6; flex-shrink: 0; }
.track-detail-text { overflow: hidden; text-overflow: ellipsis; }
.theme-toggle { right: 28px; }
@media (max-width: 768px) {
  .scroll-track, .track-detail { display: none !important; }
  html.has-minimap { scrollbar-width: auto; }
  html.has-minimap::-webkit-scrollbar { display: initial; }
  .has-minimap .container { padding-right: 1rem; }
}
@media (max-width: 640px) {
  .container { padding: 1rem; }
  h1 { font-size: 1.4rem; }
  pre { font-size: 0.8em; }
}
@media (min-width: 641px) and (max-width: 1024px) {
  .container { padding: 2rem 2rem; }
}
`

const jsContent = `
(function() {
  var btn = document.getElementById('theme-toggle');
  var html = document.documentElement;
  var saved = localStorage.getItem('theme');
  if (saved) html.setAttribute('data-theme', saved);
  btn.addEventListener('click', function() {
    var current = html.getAttribute('data-theme');
    var next = current === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
    btn.textContent = next === 'dark' ? '☀️' : '🌙';
  });
  var track = document.getElementById('scroll-track');
  var detail = document.getElementById('track-detail');
  if (!track || !detail) return;
  var turns = document.querySelectorAll('.turn[data-ts]');
  if (!turns.length) return;
  html.classList.add('has-minimap');
  var turnData = [];
  turns.forEach(function(el) {
    turnData.push({
      el: el,
      role: el.dataset.role || 'assistant',
      ts: el.dataset.ts || '',
      preview: el.dataset.preview || ''
    });
  });
  var ticks = [];
  function layoutTicks() {
    var docH = document.documentElement.scrollHeight;
    var trackH = track.clientHeight;
    turnData.forEach(function(td, i) {
      var tick = ticks[i];
      if (!tick) {
        tick = document.createElement('div');
        tick.className = 'tick tick-' + td.role;
        track.appendChild(tick);
        ticks[i] = tick;
      }
      var elTop = td.el.getBoundingClientRect().top + window.scrollY;
      var pct = elTop / docH;
      tick.style.top = (pct * trackH) + 'px';
    });
  }
  var vp = document.createElement('div');
  vp.className = 'viewport-indicator';
  track.appendChild(vp);
  function updateViewport() {
    var docH = document.documentElement.scrollHeight;
    var winH = window.innerHeight;
    var scrollY = window.scrollY;
    var trackH = track.clientHeight;
    var ratio = winH / docH;
    var vpH = Math.max(ratio * trackH, 20);
    var vpTop = (scrollY / docH) * trackH;
    vp.style.height = vpH + 'px';
    vp.style.top = vpTop + 'px';
  }
  var activeTick = null;
  function updateActiveTick() {
    var scrollY = window.scrollY;
    var winH = window.innerHeight;
    var best = -1;
    var bestDist = Infinity;
    turnData.forEach(function(td, i) {
      var elTop = td.el.getBoundingClientRect().top + window.scrollY;
      if (elTop >= scrollY && elTop <= scrollY + winH) {
        var dist = Math.abs(elTop - scrollY - winH * 0.15);
        if (dist < bestDist) { bestDist = dist; best = i; }
      }
    });
    if (activeTick !== null && ticks[activeTick]) ticks[activeTick].classList.remove('active');
    if (best >= 0 && ticks[best]) { ticks[best].classList.add('active'); activeTick = best; }
  }
  function onScroll() {
    updateViewport();
    updateActiveTick();
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  window.addEventListener('resize', function() { layoutTicks(); onScroll(); });
  track.addEventListener('click', function(e) {
    if (e.target === vp) return;
    var trackH = track.clientHeight;
    var docH = document.documentElement.scrollHeight;
    var winH = window.innerHeight;
    var rect = track.getBoundingClientRect();
    var y = e.clientY - rect.top;
    var pct = y / trackH;
    var scrollTo = pct * docH - winH / 2;
    window.scrollTo({ top: Math.max(0, scrollTo), behavior: 'smooth' });
  });
  var dragging = false;
  var dragStartY = 0, dragStartScroll = 0;
  vp.addEventListener('mousedown', function(e) {
    dragging = true;
    dragStartY = e.clientY;
    dragStartScroll = window.scrollY;
    vp.classList.add('dragging');
    e.preventDefault();
  });
  document.addEventListener('mousemove', function(e) {
    if (!dragging) return;
    var trackH = track.clientHeight;
    var docH = document.documentElement.scrollHeight;
    var dy = e.clientY - dragStartY;
    var scrollDelta = (dy / trackH) * docH;
    window.scrollTo({ top: dragStartScroll + scrollDelta });
  });
  document.addEventListener('mouseup', function() {
    if (dragging) { dragging = false; vp.classList.remove('dragging'); }
  });
  var DETAIL_COUNT = 7;
  var HALF = Math.floor(DETAIL_COUNT / 2);
  track.addEventListener('mousemove', function(e) {
    if (dragging) { detail.classList.remove('visible'); return; }
    var rect = track.getBoundingClientRect();
    var y = e.clientY - rect.top;
    var trackH = track.clientHeight;
    var docH = document.documentElement.scrollHeight;
    var pct = y / trackH;
    var docPos = pct * docH;
    var nearest = 0;
    var nearestDist = Infinity;
    turnData.forEach(function(td, i) {
      var elTop = td.el.getBoundingClientRect().top + window.scrollY;
      var dist = Math.abs(elTop - docPos);
      if (dist < nearestDist) { nearestDist = dist; nearest = i; }
    });
    var start = Math.max(0, nearest - HALF);
    var end = Math.min(turnData.length, start + DETAIL_COUNT);
    if (end - start < DETAIL_COUNT) start = Math.max(0, end - DETAIL_COUNT);
    detail.innerHTML = '';
    for (var i = start; i < end; i++) {
      var td = turnData[i];
      var row = document.createElement('div');
      row.className = 'track-detail-entry' + (i === nearest ? ' current' : '');
      var dot = document.createElement('span');
      dot.className = 'track-detail-dot track-detail-dot-' + td.role;
      row.appendChild(dot);
      var ts = document.createElement('span');
      ts.className = 'track-detail-ts';
      ts.textContent = td.ts;
      row.appendChild(ts);
      var txt = document.createElement('span');
      txt.className = 'track-detail-text';
      var icon = td.role === 'user' ? '👤 ' : '🤖 ';
      txt.textContent = icon + td.preview;
      row.appendChild(txt);
      detail.appendChild(row);
    }
    var panelH = DETAIL_COUNT * 26;
    var panelTop = Math.max(4, Math.min(e.clientY - panelH / 2, window.innerHeight - panelH - 4));
    detail.style.top = panelTop + 'px';
    detail.classList.add('visible');
  });
  track.addEventListener('mouseleave', function() {
    detail.classList.remove('visible');
  });
  layoutTicks();
  onScroll();
})();
`
