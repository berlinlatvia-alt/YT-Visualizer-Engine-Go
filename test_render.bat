@echo off
REM ── Eva Vibe quick render test ──────────────────────────────────────────────
REM Drop your image and a single audio track into the "input" folder,
REM then double-click this file. Output lands in work_local\test_output.mp4
REM
REM Usage: just double-click, or run from terminal:
REM   test_render.bat  [image_file]  [audio_file]
REM ─────────────────────────────────────────────────────────────────────────────

setlocal

REM ── Defaults: auto-detect first image + first audio in .\input\ ─────────────
if "%~1"=="" (
    for %%f in (input\*.jpg input\*.jpeg input\*.png) do (
        if "%IMG%"=="" set "IMG=%%f"
    )
) else (
    set "IMG=%~1"
)

if "%~2"=="" (
    for %%f in (input\*.wav input\*.mp3 input\*.flac) do (
        if "%AUDIO%"=="" set "AUDIO=%%f"
    )
) else (
    set "AUDIO=%~2"
)

if "%IMG%"=="" (
    echo [ERROR] No image found in .\input\ - drop a JPG or PNG there first.
    pause & exit /b 1
)
if "%AUDIO%"=="" (
    echo [ERROR] No audio found in .\input\ - drop a WAV or MP3 there first.
    pause & exit /b 1
)

echo.
echo  Eva Vibe Test Render
echo  Image : %IMG%
echo  Audio : %AUDIO%
echo  Output: work_local\test_output.mp4
echo.

node render_test.js "%IMG%" "%AUDIO%"
if errorlevel 1 (
    echo.
    echo [FAILED] Check output above for ffmpeg errors.
    pause
) else (
    echo.
    echo [DONE] Opening output...
    start "" "work_local\test_output.mp4"
    pause
)
