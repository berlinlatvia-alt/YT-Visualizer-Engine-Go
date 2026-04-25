# YT Visualizer Engine: Go Backend Transition Guide

This directory contains a high-performance Go translation of the Node.js visualizer backend.

## Architecture
- **Language**: Go 1.21+
- **Framework**: Gin (Web API), Mutex-based Job Registry.
- **FFmpeg Orchestration**: Uses `os/exec` with real-time stderr parsing for progress.
- **Frontend Sync**: API endpoints match `main.js` calls exactly.

## How to Run
1. Ensure `ffmpeg` and `ffprobe` are in your PATH.
2. Build the frontend in the parent directory: `npm run build`.
3. Copy the `dist` folder into this directory.
4. Run the server:
   ```bash
   go mod tidy
   go run main.go
   ```

## API Parity Check
| Endpoint | Method | Sync Status | Note |
| :--- | :--- | :--- | :--- |
| `/healthz` | GET | ✅ 100% | Heartbeat |
| `/session` | GET | ✅ 100% | Loads `last_session.json` |
| `/preview` | POST | ✅ 100% | Multipart upload + 10s render |
| `/preview-saved` | POST | ✅ 100% | JSON + 10s render |
| `/render-full` | POST | ✅ 100% | Multipart upload + full render |
| `/progress/:jobId`| GET | ✅ 100% | SSE stream for real-time logs |
| `/previews/:jobId`| GET | ✅ 100% | Auth-protected video serving |

## Critical Logic Preserved
- **EQ Brand Colors**: Hardcoded brand palettes for `eva` and `vne`.
- **Filter Graph**: The complex FFmpeg filter chain (showfreqs fps=30, alpha mixing, glow line kick, noise film grain) is perfectly replicated.
- **Auth**: Bearer token requirement (default: `local-dev-token`).

## Next Steps for Developer
1. **RTSP/Broadcast**: Implement the `/start-broadcast` logic (currently a stub in Go) based on your VPS setup.
2. **Persistence**: The current job registry is in-memory. Consider Redis or SQLite if you need job history across restarts.
