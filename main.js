const API = import.meta.env.VITE_RENDER_API_URL;
const TOKEN = import.meta.env.VITE_RENDER_API_TOKEN;

let imageFile = null;
let audioFiles = [];
let lastRenderJobId = null;

const dropImage = document.getElementById('drop-image');
const inputImage = document.getElementById('input-image');
const dropAudio = document.getElementById('drop-audio');
const inputAudio = document.getElementById('input-audio');
const logTerminal = document.getElementById('log-terminal');

const btnSample = document.getElementById('btn-sample');
const btnStart = document.getElementById('btn-start');
const btnStop = document.getElementById('btn-stop');
const keyEl = document.getElementById('input-stream-key');
const streamKeyRow = document.getElementById('stream-key-row');
const statusEl = document.getElementById('deploy-status');
const progEl = document.getElementById('live-progress');

const samplePanel = document.getElementById('sample-panel');
const sampleVid = document.getElementById('sample-video');
const modal = document.getElementById('preview-modal');
const modalVid = document.getElementById('preview-video');

const seoSection = document.getElementById('seo-section');
const seoTitle = document.getElementById('seo-title');
const seoTags = document.getElementById('seo-tags');
const seoDesc = document.getElementById('seo-desc');

// ── Session state ─────────────────────────────────────────────────────────────
// When true, render calls go to /preview-saved and /render-full-saved —
// no file upload needed. Flipped on by "Use Last Session" button.
let useSavedSession = false;

const sessionBanner  = document.getElementById('session-banner');
const sessionLabel   = document.getElementById('session-label');
const btnReloadSess  = document.getElementById('btn-reload-session');

// On page load: ask server if there's a saved session from a previous run
(async () => {
  try {
    const r = await fetch(`${API}/session`, { headers: { Authorization: `Bearer ${TOKEN}` } });
    if (!r.ok) return;
    const s = await r.json();
    if (!s) return;
    // Show the banner with saved file info
    const trackWord = s.audioNames.length === 1 ? 'track' : 'tracks';
    sessionLabel.innerHTML =
      `💾 Last session: <strong>${s.imageName}</strong> + ${s.audioNames.length} ${trackWord}`;
    sessionBanner.style.display = 'flex';
  } catch { /* server not running yet, ignore */ }
})();

btnReloadSess.addEventListener('click', () => {
  useSavedSession = true;
  sessionBanner.style.display = 'none';
  // Populate the drop-zone labels so the user sees what's loaded
  (async () => {
    const r = await fetch(`${API}/session`, { headers: { Authorization: `Bearer ${TOKEN}` } });
    const s = await r.json();
    const trackWord = s.audioNames.length === 1 ? 'track' : 'tracks';
    dropImage.querySelector('span').textContent = `✓  ${s.imageName}`;
    dropImage.classList.add('has-file');
    dropAudio.querySelector('span').textContent = `✓  ${s.audioNames.length} ${trackWord} (${s.audioNames[0]}${s.audioNames.length > 1 ? '…' : ''})`;
    dropAudio.classList.add('has-file');
    // Set sentinels so checkReady() is satisfied
    imageFile  = '__saved__';
    audioFiles = ['__saved__'];
    btnSample.disabled = false;
    btnStart.disabled  = false;
    addLog(`Session restored: ${s.imageName} + ${s.audioNames.length} ${trackWord}`);
  })();
});

function setProgress(pct) { progEl.style.width = `${pct}%`; }
function setStatus(msg, color = '#66fcf1') { statusEl.style.color = color; statusEl.textContent = msg; }

function addLog(msg) {
  console.log(`[ENGINE] ${msg}`);
  const line = document.createElement('div');
  line.textContent = `[${new Date().toLocaleTimeString()}] ${msg}`;
  logTerminal.appendChild(line);
  logTerminal.scrollTop = logTerminal.scrollHeight;
}

// ── Drag & Drop Logic ──────────────────────────────────────────────────────────
const initDropZone = (dropZone, inputElement, isAudio = false) => {
  dropZone.addEventListener('click', () => inputElement.click());
  ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    dropZone.addEventListener(eventName, (e) => { e.preventDefault(); e.stopPropagation(); }, false);
  });
  dropZone.addEventListener('dragover', () => dropZone.classList.add('dragover'));
  dropZone.addEventListener('dragleave', () => dropZone.classList.remove('dragover'));
  dropZone.addEventListener('drop', (e) => {
    dropZone.classList.remove('dragover');
    const files = e.dataTransfer.files;
    if (files.length) {
      if (isAudio) handleAudioFiles(files);
      else { imageFile = files[0]; updateImageUI(); }
    }
  });
  inputElement.addEventListener('change', (e) => {
    const files = e.target.files;
    if (files.length) {
      if (isAudio) handleAudioFiles(files);
      else { imageFile = files[0]; updateImageUI(); }
    }
  });
};

const updateImageUI = () => {
  if (!imageFile) return;
  dropImage.querySelector('span').textContent = `Image: ${imageFile.name}`;
  dropImage.classList.add('has-file');
  checkReady();
};

const handleAudioFiles = (files) => {
  audioFiles = Array.from(files);
  audioFiles.sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' }));
  dropAudio.querySelector('span').textContent = `${audioFiles.length} Tracks Selected (Sorted)`;
  dropAudio.classList.add('has-file');
  checkReady();
};

initDropZone(dropImage, inputImage, false);
initDropZone(dropAudio, inputAudio, true);

const checkReady = () => {
  if (imageFile && audioFiles.length > 0) {
    btnSample.disabled = false;
    btnStart.disabled = false;
    generateSEO();
  }
};

const generateSEO = () => {
  seoSection.style.display = 'block';
  const titles = [
    "Self Love Frequency 432Hz | 90 Minute Meditation | Eva Vibe",
    "Let Go of Corporate Anxiety | Deep Electronic Focus Mix",
    "Escape the Matrix | Dark Ambient Meditation (Full Album)",
    "Urban Necrosis | Cyberpunk Focus Hack | 90 Min"
  ];
  const tagsList = "electronic meditation, cyberpunk ambient, dark synthwave focus, escape the matrix, 432hz self love, lofi electronic, eva vibe, cinematic tension";
  let tracklistStr = "Tracklist:\n";
  audioFiles.forEach((file, idx) => { tracklistStr += `${idx + 1}. ${file.name.replace(/\.[^/.]+$/, "")}\n`; });
  seoTitle.textContent = titles[Math.floor(Math.random() * titles.length)];
  seoTags.textContent = tagsList;
  seoDesc.value = `The grid is loud. The frequencies are dead. I made this mix to help you focus and escape it.\n\n${tracklistStr}\nListen to the full playlist on Spotify: https://open.spotify.com/artist/1NHH6cTXQ6pM4ZpzQGc5HR\n\n☕ Support the servers: https://ko-fi.com/evavibe\n🖤 Exclusive: https://www.fanvue.com/evavibe\n\n#DarkElectronic #SelfLove #FocusMusic`;
};

// ── Network Logic ─────────────────────────────────────────────────────────────

/**
 * Fetch a rendered video from the server and return a local blob: URL.
 * Using blob URLs instead of direct HTTP src fixes two issues:
 *  1. Cross-origin video (localhost:5173 → 127.0.0.1:42002) — Brave blocks direct play
 *  2. Range-request requirement — browsers need partial-content support to play MP4;
 *     blob URLs skip that entirely since the data is already local.
 * Revokes the previous blob URL to avoid memory leaks.
 */
async function loadVideoBlob(videoEl, apiUrl) {
  const r = await fetch(apiUrl, { headers: { Authorization: `Bearer ${TOKEN}` } });
  if (!r.ok) throw new Error(`Video load failed: ${r.status} ${r.statusText}`);
  const blob = await r.blob();
  if (videoEl._blobUrl) URL.revokeObjectURL(videoEl._blobUrl);
  videoEl._blobUrl = URL.createObjectURL(blob);
  videoEl.src = videoEl._blobUrl;
  videoEl.load();
}

function buildForm() {
  const fd = new FormData();
  fd.append('image', imageFile, imageFile.name);
  audioFiles.forEach(f => fd.append('audio', f, f.name));
  return fd;
}

function uploadWithProgress(endpoint, fd, onUp) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', `${API}${endpoint}`);
    xhr.setRequestHeader('Authorization', `Bearer ${TOKEN}`);
    xhr.upload.onprogress = e => e.lengthComputable && onUp(Math.round(e.loaded / e.total * 100));
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) resolve(JSON.parse(xhr.responseText));
      else reject(new Error(`${xhr.status}: ${xhr.responseText}`));
    };
    xhr.onerror = () => reject(new Error(`network error: ${API} unreachable`));
    xhr.send(fd);
  });
}

function subscribe(jobId) {
  return new Promise((resolve, reject) => {
    const evSource = new EventSource(`${API}/progress/${jobId}?token=${encodeURIComponent(TOKEN)}`);
    
    evSource.onmessage = (e) => {
      try {
        const ev = JSON.parse(e.data);
        addLog(`Event: ${ev.kind} ${ev.stage || ''} ${ev.pct || ''}%`);
        if (ev.kind === 'progress') setProgress(ev.pct);
        if (ev.kind === 'stage') setStatus(`Stage: ${ev.stage}`);
        if (ev.kind === 'preview-ready') {
          evSource.close();
          resolve({ kind: 'preview-ready', url: ev.url });
        }
        if (ev.kind === 'end') {
          evSource.close();
          ev.ok ? resolve({ kind: 'end' }) : reject(new Error(ev.msg));
        }
      } catch (err) { console.error('SSE Parse Error:', err); }
    };

    evSource.onerror = (err) => {
      evSource.close();
      reject(new Error('SSE connection failed'));
    };
  });
}

// ── Action: Generate Sample ───────────────────────────────────────────────────
btnSample.addEventListener('click', async () => {
  btnSample.disabled = true;
  samplePanel.style.display = 'none';
  try {
    setStatus('Rendering 10s Sample...');
    setProgress(0);
    // Use saved session (no upload) or fresh upload
    const { jobId } = useSavedSession
      ? await fetch(`${API}/preview-saved`, { method: 'POST', headers: { Authorization: `Bearer ${TOKEN}` } }).then(r => r.json())
      : await uploadWithProgress('/preview', buildForm(), pct => setProgress(pct));
    const ev = await subscribe(jobId);
    if (ev.kind !== 'preview-ready') throw new Error('Preview failed');
    
    setStatus('⚙ Loading video...');
    await loadVideoBlob(sampleVid, `${API}${ev.url}`);
    samplePanel.style.display = 'block';
    sampleVid.play().catch(() => {}); // autoplay may be blocked; user can click play
    setStatus('✅ Sample Ready — hit play below.', '#66fcf1');
  } catch (e) {
    setStatus(`❌ ${e.message}`, '#ff4d4d');
  } finally {
    btnSample.disabled = false;
  }
});

// ── Action: Start Radio ───────────────────────────────────────────────────────
btnStart.addEventListener('click', async () => {
  btnStart.disabled = true;
  try {
    // 1. Show modal preview first
    setStatus('Generating Final Approval Preview...');
    const { jobId: prevId } = useSavedSession
      ? await fetch(`${API}/preview-saved`, { method: 'POST', headers: { Authorization: `Bearer ${TOKEN}` } }).then(r => r.json())
      : await uploadWithProgress('/preview', buildForm(), pct => setProgress(pct));
    const ev = await subscribe(prevId);
    if (ev.kind !== 'preview-ready') throw new Error('Preview failed');

    setStatus('⚙ Loading preview...');
    await loadVideoBlob(modalVid, `${API}${ev.url}`);
    modal.showModal();
    modalVid.play().catch(() => {});

    const confirmed = await new Promise(r => {
      document.getElementById('btn-preview-confirm').onclick = () => { modal.close(); r(true); };
      document.getElementById('btn-preview-reject').onclick  = () => { modal.close(); r(false); };
    });

    if (!confirmed) {
      setStatus('Cancelled.');
      btnStart.disabled = false;
      return;
    }

    // 2. Render Full Mix
    setStatus('Rendering Full Mix (this will take time)...');
    setProgress(0);
    const { jobId } = useSavedSession
      ? await fetch(`${API}/render-full-saved`, { method: 'POST', headers: { Authorization: `Bearer ${TOKEN}` } }).then(r => r.json())
      : await uploadWithProgress('/render-full', buildForm(), pct => setProgress(pct));
    lastRenderJobId = jobId;
    await subscribe(jobId);
    
    // 3. Start Broadcast (if VPS) or instruct local
    if (API.includes('127.0.0.1')) {
      setStatus('✅ Render Complete! Push to VPS to start radio.', '#66fcf1');
      addLog('Local render finished. Broadcast is only available on VPS.');
    } else {
      setStatus('🚀 Starting YouTube Broadcast...');
      const streamKey = keyEl.value.trim();
      const r = await fetch(`${API}/start-broadcast`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${TOKEN}` },
        body: JSON.stringify({ jobId, streamKey })
      });
      if (!r.ok) throw new Error(await r.text());
      setStatus('🔥 LIVE ON YOUTUBE!', '#66fcf1');
    }
  } catch (e) {
    setStatus(`❌ ${e.message}`, '#ff4d4d');
  } finally {
    btnStart.disabled = false;
  }
});

// ── Action: Stop Radio ────────────────────────────────────────────────────────
btnStop.addEventListener('click', async () => {
  if (!confirm('Stop the 24/7 radio stream?')) return;
  try {
    const r = await fetch(`${API}/stop`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${TOKEN}` }
    });
    setStatus(r.ok ? '🛑 Stream stopped.' : `❌ stop failed`, '#ff4d4d');
  } catch (err) { setStatus(`❌ Error: ${err.message}`, '#ff4d4d'); }
});
