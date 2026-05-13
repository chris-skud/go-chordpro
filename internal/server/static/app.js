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

  let transpose = 0;
  let baseName = 'song';

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
    baseName = f.name.replace(/\.[^/.]+$/, '') || 'song';
    filenameLabel.textContent = f.name;
    source.value = await f.text();
    schedulePreview();
  });

  let previewTimer;
  source.addEventListener('input', schedulePreview);
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
  renderPreview();
})();
