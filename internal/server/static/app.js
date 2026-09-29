(() => {
  const source = document.getElementById('source');
  const preview = document.getElementById('preview');
  const fileInput = document.getElementById('file');
  const filenameLabel = document.getElementById('filename');
  const tDown = document.getElementById('transpose-down');
  const tUp = document.getElementById('transpose-up');
  const tVal = document.getElementById('transpose-val');
  const formatSel = document.getElementById('format');
  const downloadBtn = document.getElementById('download');
  const saveBtn = document.getElementById('save');
  const saveStatus = document.getElementById('save-status');

  let transpose = 0;
  let baseName = 'song';
  // FileSystemFileHandle from showOpenFilePicker / showSaveFilePicker, when
  // supported. Lets Save overwrite the original file without a dialog.
  let fileHandle = null;
  // Display name of the current file and the source text as of the last
  // open/save, used to show an unsaved-changes marker next to the name.
  let fileName = 'untitled';
  let savedText = '';
  const chordproTypes = [{
    description: 'ChordPro',
    accept: { 'text/plain': ['.cho', '.chopro', '.pro', '.crd', '.chord'] },
  }];

  function updateDirty() {
    const dirty = source.value !== savedText;
    filenameLabel.textContent = fileName + (dirty ? ' •' : '');
    filenameLabel.title = dirty ? 'Unsaved changes' : '';
  }

  function loadFile(name, text) {
    fileName = name;
    baseName = name.replace(/\.[^/.]+$/, '') || 'song';
    source.value = text;
    savedText = text;
    updateDirty();
    schedulePreview();
  }

  let statusTimer;
  function markSaved(name) {
    fileName = name;
    savedText = source.value;
    updateDirty();
    saveStatus.textContent = 'Saved';
    saveStatus.classList.add('visible');
    clearTimeout(statusTimer);
    statusTimer = setTimeout(() => saveStatus.classList.remove('visible'), 1500);
  }

  const setTranspose = (n) => {
    transpose = n;
    tVal.textContent = (n > 0 ? '+' : '') + n;
    schedulePreview();
  };

  tDown.addEventListener('click', () => setTranspose(transpose - 1));
  tUp.addEventListener('click', () => setTranspose(transpose + 1));

  fileInput.addEventListener('change', async (e) => {
    const f = e.target.files[0];
    if (!f) return;
    // Plain <input type="file"> gives us bytes but no writable handle, so
    // a later Save will need to round-trip through a save-as dialog.
    fileHandle = null;
    loadFile(f.name, await f.text());
  });

  // Prefer showOpenFilePicker when available — it returns a handle we can
  // write back to without prompting the user for a destination.
  if (window.showOpenFilePicker) {
    const label = fileInput.closest('label');
    label.addEventListener('click', async (e) => {
      e.preventDefault();
      try {
        const [handle] = await window.showOpenFilePicker({ types: chordproTypes });
        const f = await handle.getFile();
        fileHandle = handle;
        loadFile(f.name, await f.text());
      } catch (err) {
        if (err && err.name !== 'AbortError') alert('Open failed: ' + err);
      }
    });
  }

  let previewTimer;
  source.addEventListener('input', () => {
    updateDirty();
    schedulePreview();
  });
  function schedulePreview() {
    clearTimeout(previewTimer);
    previewTimer = setTimeout(renderPreview, 250);
  }

  async function renderPreview() {
    try {
      const resp = await fetch(`/api/render?format=html&transpose=${transpose}`, {
        method: 'POST',
        body: source.value,
        headers: { 'Content-Type': 'text/plain' },
      });
      const text = await resp.text();
      if (!resp.ok) {
        preview.srcdoc = errorDoc(text);
        return;
      }
      preview.srcdoc = text;
    } catch (e) {
      preview.srcdoc = errorDoc(String(e));
    }
  }

  function errorDoc(msg) {
    const safe = msg.replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]));
    return `<!doctype html><html><body style="font-family:system-ui;padding:1rem;color:#900;"><pre style="white-space:pre-wrap;">${safe}</pre></body></html>`;
  }

  async function save() {
    try {
      if (fileHandle) {
        const w = await fileHandle.createWritable();
        await w.write(source.value);
        await w.close();
        markSaved(fileHandle.name || fileName);
        return;
      }
      if (window.showSaveFilePicker) {
        const handle = await window.showSaveFilePicker({
          suggestedName: baseName + '.cho',
          types: chordproTypes,
        });
        const w = await handle.createWritable();
        await w.write(source.value);
        await w.close();
        fileHandle = handle;
        const fname = handle.name || (baseName + '.cho');
        baseName = fname.replace(/\.[^/.]+$/, '') || baseName;
        markSaved(fname);
        return;
      }
      // Browser without File System Access API — fall back to download.
      const blob = new Blob([source.value], { type: 'text/plain;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = baseName + '.cho';
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      markSaved(baseName + '.cho');
    } catch (e) {
      if (e && e.name === 'AbortError') return;
      alert('Save failed: ' + e);
    }
  }

  saveBtn.addEventListener('click', save);
  document.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 's') {
      e.preventDefault();
      save();
    }
  });

  downloadBtn.addEventListener('click', async () => {
    const fmt = formatSel.value;
    try {
      const resp = await fetch(
        `/api/render?format=${fmt}&transpose=${transpose}&download=1&name=${encodeURIComponent(baseName)}`,
        { method: 'POST', body: source.value, headers: { 'Content-Type': 'text/plain' } },
      );
      if (!resp.ok) {
        const text = await resp.text();
        alert('Render failed: ' + text);
        return;
      }
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const ext = fmt === 'text' ? 'txt' : fmt;
      a.download = `${baseName}.${ext}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (e) {
      alert('Download failed: ' + e);
    }
  });

  // --- Perform mode -------------------------------------------------------
  // Full-screen view of the rendered song that auto-scrolls in time with the
  // song's tempo. Scroll speed assumes each lyric line is one bar, so the
  // whole song takes lines × beats-per-bar beats to scroll through.

  const performBtn = document.getElementById('perform');
  const performView = document.getElementById('perform-view');
  const performFrame = document.getElementById('perform-frame');
  const performPlay = document.getElementById('perform-play');
  const performBpm = document.getElementById('perform-bpm');
  const performLayout = document.getElementById('perform-layout');
  const performSmaller = document.getElementById('perform-smaller');
  const performBigger = document.getElementById('perform-bigger');
  const performExit = document.getElementById('perform-exit');
  const performTheme = document.getElementById('perform-theme');

  const DEFAULT_BPM = 100;
  const DEFAULT_BEATS_PER_BAR = 4;

  // Injected into the rendered song document. "columns" flows the song into
  // side-by-side columns the height of the screen (scrolling sideways only if
  // they don't all fit); "scroll" is a single tall column.
  const performCSS = `
html { --scale: 1; }
body { max-width: none; margin: 0; padding: 1.5rem 2rem; font-size: calc(1.25rem * var(--scale)); }
header, section, p.line, p.comment { break-inside: avoid; }
html.columns { overflow-y: hidden; }
html.columns, html.columns body { height: 100%; }
html.columns body { box-sizing: border-box; column-width: var(--col-width, 22em); column-gap: 3rem; column-fill: auto; column-rule: 1px solid #eee; }
html.dark { color-scheme: dark; }
html.dark body { background: #111; color: #ddd; }
html.dark.columns body { column-rule-color: #333; }
html.dark header .meta, html.dark .section-label, html.dark p.comment { color: #aaa; }
html.dark section.chorus { border-left-color: #555; }
html.dark section.tab pre { background: #222; }
html.dark p.comment.box { border-color: #666; }
html.dark p.line .chord { color: #ff7b72; }
html.dark p.line .annotation { color: #79c0ff; }
`;

  const store = {
    get(k, d) { try { return localStorage.getItem(k) ?? d; } catch { return d; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch { /* ignore */ } },
  };

  let performScale = parseFloat(store.get('chordpro.perform.scale', '1')) || 1;
  performLayout.value = store.get('chordpro.perform.layout', 'columns');
  // Follow the system theme until the user picks one explicitly.
  let performDark = store.get('chordpro.perform.theme',
    window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') === 'dark';
  let playing = false;
  let scrollPos = 0;
  let lastTs = 0;
  let rafId = 0;

  // metaValue returns the value of a whole-line {name: value} directive.
  function metaValue(name) {
    const re = new RegExp(`^\\s*\\{\\s*${name}\\s*:\\s*([^{}]*?)\\s*\\}\\s*$`, 'mi');
    const m = source.value.match(re);
    return m ? m[1] : '';
  }

  function sourceTempo() {
    const n = parseFloat(metaValue('tempo'));
    return n >= 20 && n <= 400 ? n : DEFAULT_BPM;
  }

  function beatsPerBar() {
    const n = parseInt(metaValue('time'), 10); // "3/4" -> 3
    return n > 0 ? n : DEFAULT_BEATS_PER_BAR;
  }

  const frameDoc = () => performFrame.contentDocument;
  const scroller = () => frameDoc().scrollingElement;
  const horizontal = () => performLayout.value === 'columns';

  function songSeconds() {
    const doc = frameDoc();
    let lines = doc.querySelectorAll('p.line').length;
    doc.querySelectorAll('section.tab pre').forEach((pre) => {
      lines += pre.textContent.trim().split('\n').length;
    });
    const bpm = parseFloat(performBpm.value) || DEFAULT_BPM;
    return (Math.max(lines, 1) * beatsPerBar() * 60) / bpm;
  }

  // widestLine returns the natural (unwrapped) width in px of the widest
  // song line. Chord/lyric pairs and lyric runs never wrap, so a column
  // narrower than this lets lines spill into the neighbouring column.
  function widestLine() {
    const doc = frameDoc();
    let max = 0;
    doc.querySelectorAll('p.line').forEach((p) => {
      let w = 0;
      for (const child of p.children) w += child.getBoundingClientRect().width;
      max = Math.max(max, w);
    });
    doc.querySelectorAll('section.tab pre').forEach((pre) => {
      max = Math.max(max, pre.scrollWidth);
    });
    return Math.ceil(max);
  }

  // fitColumns widens columns to fit the longest line (at the current
  // --scale), but never past the screen width minus the body padding.
  function fitColumns() {
    const root = frameDoc().documentElement;
    const body = frameDoc().body;
    const cs = frameDoc().defaultView.getComputedStyle(body);
    const room = root.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
    const minCol = 22 * parseFloat(cs.fontSize);
    root.style.setProperty('--col-width', `${Math.min(room, Math.max(minCol, widestLine()))}px`);
  }

  function applyPerformStyle() {
    const root = frameDoc().documentElement;
    root.classList.toggle('columns', horizontal());
    applyTheme();
    root.style.setProperty('--scale', performScale);
    fitColumns();
    const el = scroller();
    el.scrollTop = 0;
    el.scrollLeft = 0;
    scrollPos = 0;
  }

  // applyTheme can run without resetting the scroll position, so toggling
  // mid-song doesn't lose your place.
  function applyTheme() {
    frameDoc().documentElement.classList.toggle('dark', performDark);
    performView.classList.toggle('dark', performDark);
    performTheme.textContent = performDark ? 'Light' : 'Dark';
  }

  function toggleTheme() {
    performDark = !performDark;
    store.set('chordpro.perform.theme', performDark ? 'dark' : 'light');
    applyTheme();
  }

  function setPlaying(on) {
    playing = on;
    performPlay.textContent = on ? 'Pause' : 'Play';
    performView.classList.toggle('paused', !on);
    cancelAnimationFrame(rafId);
    lastTs = 0;
    if (on) rafId = requestAnimationFrame(tick);
  }

  // tick advances the scroll position so that reaching the end of the
  // scrollable range coincides with the end of the song.
  function tick(ts) {
    if (!playing) return;
    const el = scroller();
    const max = horizontal() ? el.scrollWidth - el.clientWidth : el.scrollHeight - el.clientHeight;
    const cur = horizontal() ? el.scrollLeft : el.scrollTop;
    // The user scrolled by hand: continue from where they left it.
    if (Math.abs(cur - scrollPos) > 2) scrollPos = cur;
    const dt = lastTs ? (ts - lastTs) / 1000 : 0;
    lastTs = ts;
    scrollPos = Math.min(max, scrollPos + (max / songSeconds()) * dt);
    if (horizontal()) el.scrollLeft = scrollPos;
    else el.scrollTop = scrollPos;
    if (scrollPos >= max) {
      setPlaying(false);
      return;
    }
    rafId = requestAnimationFrame(tick);
  }

  function setScale(next) {
    performScale = Math.min(3, Math.max(0.5, Math.round(next * 10) / 10));
    store.set('chordpro.perform.scale', String(performScale));
    applyPerformStyle();
  }

  async function openPerform() {
    let html;
    try {
      const resp = await fetch(`/api/render?format=html&transpose=${transpose}`, {
        method: 'POST',
        body: source.value,
        headers: { 'Content-Type': 'text/plain' },
      });
      html = await resp.text();
      if (!resp.ok) {
        alert('Render failed: ' + html);
        return;
      }
    } catch (e) {
      alert('Render failed: ' + e);
      return;
    }
    performBpm.value = sourceTempo();
    await new Promise((resolve) => {
      performFrame.addEventListener('load', resolve, { once: true });
      performFrame.srcdoc = html.replace('</head>', `<style>${performCSS}</style></head>`);
    });
    // Keys pressed while the song itself has focus land in the frame.
    frameDoc().addEventListener('keydown', onPerformKey);
    // Entering fullscreen or resizing the window changes the room available.
    performFrame.contentWindow.addEventListener('resize', fitColumns);
    performView.hidden = false;
    applyPerformStyle();
    setPlaying(false);
    performView.requestFullscreen?.().catch(() => { /* stay in the overlay */ });
  }

  function closePerform() {
    if (performView.hidden) return;
    setPlaying(false);
    performView.hidden = true;
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
  }

  function onPerformKey(e) {
    if (performView.hidden) return;
    const tag = e.target.tagName;
    if (tag === 'INPUT' || tag === 'SELECT') return;
    if (e.key === ' ') {
      e.preventDefault();
      setPlaying(!playing);
    } else if (e.key === 'Escape') {
      closePerform();
    } else if (e.key === '+' || e.key === '=') {
      setScale(performScale + 0.1);
    } else if (e.key === '-') {
      setScale(performScale - 0.1);
    } else if ((e.key === 'd' || e.key === 'D') && !e.metaKey && !e.ctrlKey && !e.altKey) {
      toggleTheme();
    }
  }

  // Blur bar buttons after a click so Space goes to the play toggle rather
  // than re-activating whichever button was clicked last.
  const onClick = (btn, fn) => btn.addEventListener('click', () => { fn(); btn.blur(); });
  onClick(performBtn, openPerform);
  onClick(performPlay, () => setPlaying(!playing));
  onClick(performSmaller, () => setScale(performScale - 0.1));
  onClick(performBigger, () => setScale(performScale + 0.1));
  onClick(performTheme, toggleTheme);
  onClick(performExit, closePerform);
  performLayout.addEventListener('change', () => {
    store.set('chordpro.perform.layout', performLayout.value);
    applyPerformStyle();
    performLayout.blur();
  });
  document.addEventListener('keydown', onPerformKey);
  // Leaving fullscreen (e.g. Esc, which the browser handles itself) exits
  // perform mode too.
  document.addEventListener('fullscreenchange', () => {
    if (!document.fullscreenElement) closePerform();
  });

  // Seed the editor with a small sample so the UI is immediately useful.
  source.value = `{title: Hello World}
{artist: chordpro}
{key: C}

{start_of_chorus}
[C]Hello, [G]hello, [Am]hello [F]world
[C]Type to [G]edit and the [F]preview [C]updates
{end_of_chorus}
`;
  savedText = source.value;
  renderPreview();
})();
