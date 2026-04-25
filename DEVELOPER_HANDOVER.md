# 🛠️ DEVELOPER HANDOVER: YT Visualizer Engine (Go Migration)

**Status:** Backend successfully translated from Node.js to Go (Gin).  
**Repository:** [https://github.com/berlinlatvia-alt/YT-Visualizer-Engine-Go](https://github.com/berlinlatvia-alt/YT-Visualizer-Engine-Go)

---

## 1. System Architecture

The project is a high-performance video rendering suite for YouTube ambient music channels.

- **Frontend:** Vite + Vanilla JS (Serves the UI for track selection and preview).
- **Backend (Go):** Gin-based API that orchestrates FFmpeg for visual synthesis.
- **Engine:** FFmpeg (Handles all the heavy lifting for EQ, glow, and grain).

### Data Flow
1. User uploads cover + audio tracks.
2. Go backend generates a **palette** from the image (3-pixel sample).
3. Go backend triggers a **background FFmpeg process** with a complex filter chain.
4. **SSE (Server-Sent Events):** Go streams progress (0-100%) back to the frontend in real-time.

---

## 2. Setting Up (Local Dev)

### Prerequisites
- **Go 1.21+** (Must be installed and on PATH).
- **FFmpeg** (Must be on PATH).
- **Node.js** (Optional, only for the Vite frontend).

### Start the Engine
```bash
cd backend-go
go mod tidy
go run main.go
```
*The engine will listen on **Port 42002**.*

### Start the Frontend
The frontend files are included in the repo. You can serve them via the Go backend (it is configured to serve `./dist`) or run them via Vite:
```bash
# In the project root
npm install
npm run dev
```

---

## 3. The "Secret Sauce" (Visual Logic)

The Go backend replicates the **"Eva Vibe" Golden Copy** logic:

- **Palette Extraction:** `extractPalette()` uses `scale=3:1` area sampling to get `LOW/MID/HIGH` colors for the EQ bars.
- **Kick Glow:** A 4-layer `drawbox` stack at the EQ floor that simulates an amber bloom line.
- **Temporal Noise:** Using `noise=alls=22:allf=t+u` where `t` ensures the grain changes every single frame (vital for "organic" feel).
- **SSE Sync:** The Go `Job` struct uses a `map[chan string]bool` to broadcast FFmpeg stderr output (parsed for `out_time_ms`) to all connected browser clients.

---

## 4. Hardware Constraints (CRITICAL)

- **Local:** Uses CPU (`libx264 -preset ultrafast`).
- **VPS (Vultr):** The VPS GPU (NVIDIA A16) is **UNLICENSED**. You must stay on `libx264`. Do NOT attempt `h264_nvenc` as it will fail with CUDA errors.
- **Audio Speed:** Every audio track is conditioned with `aformat=sample_rates=48000:...,asetpts=N/SR/TB` to prevent the "1h30 becomes 2h04" sample-rate bug.

---

## 5. Testing Tools

I have included tools for testing without the browser:
- **`test_render.bat`**: Triggers a local Node.js test (legacy reference).
- **`operator.js`**: The headless sync script—shows how to talk to the API via Node.js.
- **`go_logic_test.mp4`**: A 10-second verification render I ran on the local machine to prove the Go filter chain works.

---

## 6. Next Task for You
- [ ] Install Go on the machine.
- [ ] Verify the SSE progress bar shows up in the UI during a render.
- [ ] Port the Go logic to the VPS (`server_v11.js` replacement).

---
**Handed over by Antigravity (AI)**  
*Date: April 25, 2026*
