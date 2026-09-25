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
