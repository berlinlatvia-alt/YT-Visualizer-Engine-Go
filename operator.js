import fs from 'fs';
import path from 'path';
import crypto from 'crypto';
import { spawn } from 'child_process';
import 'dotenv/config';

// Config
const API_URL = process.env.VITE_RENDER_API_URL || 'http://45.76.128.121:42001';
const TOKEN = process.env.RENDER_API_TOKEN;
const STREAM_KEY = process.env.YOUTUBE_STREAM_KEY;

async function run() {
  const args = process.argv.slice(2);
  let channel = 'eva'; // Default to eva vibe
  let imagePath = '';
  let audioDir = '';

  // Simple arg parser
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--channel' && args[i+1]) { channel = args[i+1]; i++; }
    else if (!imagePath) imagePath = args[i];
    else if (!audioDir) audioDir = args[i];
  }

  if (!imagePath || !audioDir) {
    console.log('Usage: node operator.js [--channel eva|vne] <image_path> <audio_dir>');
    process.exit(1);
  }

  console.log(`🚀 Starting Autonomous Deployment for Channel: [${channel}]`);
  console.log(`🔗 Target: ${API_URL}`);

  const tracks = fs.readdirSync(audioDir)
    .filter(f => f.endsWith('.mp3') || f.endsWith('.wav'))
    .map(f => path.join(audioDir, f));

  if (tracks.length === 0) { console.error('❌ No tracks found!'); process.exit(1); }
  console.log(`📦 Found ${tracks.length} tracks. Preparing upload...`);

  // Stage 1: Render
  const formData = new FormData();
  formData.append('channel', channel);
  formData.append('image', new Blob([fs.readFileSync(imagePath)]), path.basename(imagePath));
  for (const t of tracks) {
    formData.append('audio', new Blob([fs.readFileSync(t)]), path.basename(t));
  }

  console.log('📤 Uploading assets and starting render (Stage 1)...');
  const res = await fetch(`${API_URL}/render-full`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${TOKEN}` },
    body: formData
  });

  const { jobId, error } = await res.json();
  if (error) { console.error(`❌ Server Error: ${error}`); process.exit(1); }

  console.log(`✅ Job started: ${jobId}`);
  console.log('⏳ Monitoring progress...');

  const progressUrl = `${API_URL}/progress/${jobId}?token=${TOKEN}`;
  const monitor = await fetch(progressUrl);
  const reader = monitor.body.getReader();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    const chunk = new TextDecoder().decode(value);
    for (const line of chunk.split('\n')) {
      if (line.startsWith('data: ')) {
        const ev = JSON.parse(line.slice(6));
        if (ev.kind === 'progress') process.stdout.write(`\r▓ Progress: ${ev.pct}% `);
        if (ev.kind === 'end') {
          console.log(ev.ok ? '\n✨ Render Complete!' : `\n❌ Failed: ${ev.msg}`);
          if (ev.ok) break; else process.exit(1);
        }
      }
    }
  }

  // Stage 2: Broadcast
  console.log('\n🎙️ Starting Broadcast (Stage 2)...');
  const bRes = await fetch(`${API_URL}/start-broadcast`, {
    method: 'POST',
    headers: { 
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ jobId, channel, streamKey: (STREAM_KEY === 'REPLACE_WITH_YOUR_KEY' ? undefined : STREAM_KEY) })
  });

  const bStatus = await bRes.json();
  if (bStatus.ok) {
    console.log(`🔥 SUCCESS! Channel ${channel} is now LIVE (PID: ${bStatus.pid})`);
    console.log(`📡 View: ${API_URL}/broadcast/${channel}?token=${TOKEN}`);
  } else {
    console.error(`❌ Broadcast Failed: ${bStatus.error}`);
    console.log('💡 Note: If error is "key", you need to provide a real YOUTUBE_STREAM_KEY in .env');
  }
}

run().catch(console.error);
