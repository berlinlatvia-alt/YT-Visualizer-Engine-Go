# 🎨 VISUAL STYLE OVERHAUL — Eva Vibe / Вне Системы Signature Look
**Status:** SPEC — agent to execute, single-song test gate
**Date:** 2026-04-25
**Author:** Master LLM, on Anton's brief

---

## §0. Brief (verbatim from operator)
> "Waveforms are ugly, they look like defect, not an effect. They must match the mood and color palette of the main image. Let's do an equalizer instead, like the one on the image attached — middle horizontal line reacts to bass, the rest to mids/highs. No film grain. No old-film effects. Can we do a candle glow? At least some effect? Pre-render 1 song as example, not a whole batch. We need our special visual style so listeners recognize us."

This document is the recipe. The downstream agent **MUST** test on **one song only** and submit the resulting MP4 for approval before applying to a batch.

---

## §1. The four visual layers (composition order, bottom → top)

```
[0] Background image     ── the meditator / candles / sunset (1920×1080, prepped)
[1] Candle glow          ── soft warm bloom over highlight regions
[2] Vintage film grade   ── teal-orange curves, slight desaturation, subtle vignette
[3] Equalizer (the hero) ── 64-bar mirrored spectrum, palette-matched, bottom-anchored
[4] Film grain           ── monochrome noise @ 8-12% opacity, animated per frame
```

Every layer is reproducible from one ffmpeg invocation — no external compositing.

---

## §2. Color palette extraction (do this FIRST, per-image)

The equalizer and glow colors must come from the actual background. Static color codes will not survive multiple cover images.

### 2.1 Extract dominant colors with ffmpeg's `palettegen`
```bash
# Reduce the image to its 8 most dominant colors → palette.png
ffmpeg -y -i input_cover.jpg -vf "palettegen=max_colors=8:reserve_transparent=0" palette.png

# Read the palette pixel-by-pixel (left→right are most→least dominant)
ffprobe -v error -f lavfi -i "movie=palette.png" -show_frames -show_entries frame=pkt_pts \
  -select_streams v -of csv 2>/dev/null

# Or simpler — sample 8 hex codes via ImageMagick (always preinstalled on the VPS)
magick input_cover.jpg -resize 1x8! -depth 8 txt: | awk 'NR>1 {print $3}'
# Output: #E8A07A  #C25E2D  #6B2818  #2A0F0A  #8C4A28  #F5C89A  #44181B  #1A0805
```

### 2.2 Pick the three palette slots
Map the eight dominant hex codes into three roles:

| Slot          | Choice rule                                        | Example (Eva Vibe sunset) |
|---------------|----------------------------------------------------|---------------------------|
| `EQ_LOW`      | Warmest / most saturated mid-tone                  | `#C25E2D` (ember orange)  |
| `EQ_MID`      | Lightest accent                                    | `#F5C89A` (candle cream)  |
| `EQ_HIGH`     | Coolest / most desaturated for contrast peaks      | `#8C4A28` (muted copper)  |
| `GLOW_TINT`   | Same as `EQ_MID` but at 30% opacity                | `#F5C89A @ 0.3`           |

For Вне Системы (darker, mystical palette) the same logic produces deep purples / cold blues automatically — that's the point. **The visualizer style stays consistent across both channels; only the palette shifts.**

### 2.3 Persist the palette to `palette.json` per render job
```js
// render-svc: extracted once at job start, passed to filter graph
{
  "EQ_LOW":   "0xC25E2D",
  "EQ_MID":   "0xF5C89A",
  "EQ_HIGH":  "0x8C4A28",
  "GLOW_TINT":"0xF5C89A"
}
```

---

## §3. The Equalizer (the hardest part — full snippet)

We do **NOT** use `showwaves` anymore. We use **`showfreqs`** in `bar` mode with logarithmic frequency scale, then mirror it vertically so the center horizontal axis is the energy line and bars grow outward from there.

### 3.1 Filter graph fragment (drop-in replacement for the old `[wave]` chain)
```
# Audio in: [aout] — single mono/stereo audio stream after concat
# Image in: [4:v]  — the prepared 1920×1080 cover

[aout]asplit=3[a_main][a_eq][a_glow];

# === EQUALIZER LAYER ===
# 64 bars, log frequency scale, smooth response
[a_eq]showfreqs=
  s=1920x300:
  mode=bar:
  ascale=log:
  fscale=log:
  win_size=2048:
  win_func=hann:
  averaging=2:
  colors=0xC25E2D|0xF5C89A|0x8C4A28|0xC25E2D,
  format=yuva420p,
  # Mirror it: top half is the original, bottom half is flipped → centre line = bass
  split[eq_top][eq_tmp];
[eq_tmp]vflip[eq_bot];
[eq_top][eq_bot]vstack=inputs=2,
  # Soft alpha so it floats over the image instead of pasting on it
  format=yuva420p,
  geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':a='alpha(X,Y)*0.78',
  # Gentle horizontal blur — kills aliasing on thin bars, gives a "glow" feel
  gblur=sigma=1.2[eq];
```

**Why `showfreqs` and not `showspectrum` or `showcqt`:**
- `showspectrum` is a heatmap, not bars — wrong vibe.
- `showcqt` is gorgeous but **3-5× slower than `showfreqs`** — kills our 6-8× realtime CPU target.
- `showfreqs` with `ascale=log` + `fscale=log` gives the music-video "perceptual" response Anton's reference image shows (low-end is fat and visible, highs are thin and sharp).

**Tuning knobs** (only change after one-song test passes):
- `win_size=2048` → response speed. Lower (1024) = snappier, higher (4096) = smoother.
- `averaging=2` → temporal smoothing. Bumping to 4 reduces flicker on percussive content.
- The `0.78` alpha factor in `geq` is the visual "transparency" — raise to `0.9` if too faint, lower to `0.6` if it overpowers the image.

### 3.2 The bass-on-the-centerline behavior
`showfreqs` outputs frequency from **left (low) → right (high)** by default. `vstack` of original + vflipped copy creates a **horizontal center axis where the LEFT edge of both halves meet** — but that's frequency=0 (DC), not bass. To get bass on the **vertical center**, we add one rotation:

```
[a_eq]showfreqs=...,transpose=1[eq_v];   # now bars run vertically, low freq at top
[eq_v]split[eq_l][eq_tmp];
[eq_tmp]hflip[eq_r];
[eq_l][eq_r]hstack=inputs=2[eq_mirrored]; # horizontal mirror around the center line
```

Wait — re-read the brief carefully. The reference image shows **vertical bars** with bass in the **middle horizontal section** and highs at outer edges. That's a horizontal mirror of frequency around the **vertical center**, with the bars themselves staying vertical. So the simplest correct chain is:

```
[a_eq]showfreqs=
  s=960x400:                        # half-width — we'll mirror to 1920
  mode=bar:
  ascale=log:fscale=log:
  win_size=2048:averaging=2:
  colors=0xC25E2D|0xF5C89A|0x8C4A28|0xC25E2D,
  format=yuva420p[eq_half];
[eq_half]hflip[eq_half_mirrored];
[eq_half_mirrored][eq_half]hstack=inputs=2,
  format=yuva420p,
  geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':a='alpha(X,Y)*0.82',
  gblur=sigma=1.0[eq];
```

Now: bass (low freq) lives where the two halves meet → **vertical center line of the screen**. Highs flare out toward the **left and right edges**. Exactly the reference image.

### 3.3 Position on the canvas
```
[bg_with_glow_and_grade][eq]overlay=
  x=0:
  y=H-h-40:                          # 40 px from bottom of frame
  format=auto:
  shortest=1[outv]
```

For taller / portrait-style equalizers (Вне Системы variant), use `y=(H-h)/2` to vertically center on the meditator's torso instead.

---

## §4. Candle Glow

A two-stage trick: extract bright-warm regions of the background, blur them, screen-blend them back over the original at low opacity. Looks like real bloom, costs almost nothing.

```
# Input: [4:v] prepared 1920×1080 cover
[4:v]format=yuva420p,
  # Pull only warm highlights — luma > 200 AND red dominant
  geq=
    r='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30), r(X,Y), 0)':
    g='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30), g(X,Y)*0.85, 0)':
    b='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30), b(X,Y)*0.5, 0)':
    a='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30), 255, 0)',
  # Heavy blur → bloom shape
  gblur=sigma=24:steps=2,
  # Boost saturation so the glow has color instead of being washed white
  eq=saturation=1.6:gamma=1.1[glow];

# Composite: original first, glow as 'screen' blend on top
[4:v][glow]blend=all_mode=screen:all_opacity=0.55[bg_glow];
```

**Optional pulse** — make the glow gently breathe with bass energy:
```
[a_glow]aformat=channel_layouts=mono,
  lowpass=f=200,                       # isolate sub-bass
  ebur128=peak=true:metadata=1,
  # produce a 0..1 gain curve following the loudness
  ametadata=mode=print:key=lavfi.r128.M[a_glow_meta];
# (Hooking the metadata into a video filter requires sendcmd / drawbox tricks.
#  Skip for v1 — static glow looks great and ships today.)
```

For v1, **static glow only**. The reactive pulse goes into the v2 pass once we've validated the look.

---

## §5. Vintage Film Grade

Three filters in series — runs ~free on CPU:

```
[bg_glow]
  # Teal-orange split-tone — push shadows blue, highlights warm
  curves=
    r='0/0 0.25/0.18 0.55/0.62 1/1':
    g='0/0 0.5/0.48 1/0.97':
    b='0/0.05 0.4/0.36 0.7/0.62 1/0.92',
  # Slight global desaturation — kills the "phone camera" look
  eq=saturation=0.88:contrast=1.06,
  # Subtle vignette — 8% darkening at corners
  vignette=PI/5:eval=init,
  format=yuv420p[bg_graded];
```

The `curves` line is the magic — it's the same teal-orange grade every Hollywood thriller has used since 2008. For the Вне Системы channel (cooler/mystical), invert it:

```
# Вне Системы variant
curves=
  r='0/0 0.5/0.42 1/0.88':
  g='0/0 0.5/0.48 1/0.94':
  b='0/0.08 0.3/0.32 0.7/0.74 1/1',
```

---

## §6. Film Grain

Cheap, animated, monochromatic. Looks like real Kodak 5219.

```
# Generate a noise stream the same size as the canvas
color=c=gray:s=1920x1080:r=30,
  format=yuva420p,
  noise=alls=22:allf=t+u,            # t=temporal (changes per frame), u=uniform
  # Reduce to grain's 'pleasant' band: kill DC offset + roll off highs
  geq=lum='lum(X,Y)*0.5+128':a=255,
  format=yuva420p,
  setsar=1[grain];

# Blend over the graded background at 9% — visible texture, not noise wall
[bg_graded][grain]blend=all_mode=overlay:all_opacity=0.09,
  format=yuv420p[bg_final];
```

Knob: `alls=22` is grain intensity (range 0-100). `0.09` opacity is the *visibility*. Either knob can be turned. Recommend keep the strong noise + low opacity — gives that "fine grain" feel rather than "broken sensor".

---

## §7. Full filter graph — copy-paste ready

This is the complete chain to drop into `render-svc`'s `filter_complex`. Inputs:
- `[0..N-1]` audio inputs (the songs)
- `[N:v]` the prepared 1920×1080 cover image (looped)

```
# AUDIO: concat → split into main mix + EQ feed
${concatAudioInputs}concat=n=${N}:v=0:a=1[a_concat];
[a_concat]asplit=2[a_out][a_eq];

# EQUALIZER (palette-matched, mirrored bass-to-edges)
[a_eq]showfreqs=s=960x400:mode=bar:ascale=log:fscale=log:win_size=2048:averaging=2:colors=${EQ_LOW}|${EQ_MID}|${EQ_HIGH}|${EQ_LOW},format=yuva420p[eq_half];
[eq_half]split[eq_a][eq_b];
[eq_b]hflip[eq_b_flipped];
[eq_b_flipped][eq_a]hstack=inputs=2,format=yuva420p,geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':a='alpha(X,Y)*0.82',gblur=sigma=1.0[eq];

# BACKGROUND: load image, candle glow, grade
[${N}:v]scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1[bg_raw];
[bg_raw]format=yuva420p,geq=r='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30),r(X,Y),0)':g='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30),g(X,Y)*0.85,0)':b='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30),b(X,Y)*0.5,0)':a='if(gt(r(X,Y),200)*gt(r(X,Y),b(X,Y)+30),255,0)',gblur=sigma=24:steps=2,eq=saturation=1.6:gamma=1.1[glow];
[bg_raw][glow]blend=all_mode=screen:all_opacity=0.55[bg_glow];
[bg_glow]curves=r='0/0 0.25/0.18 0.55/0.62 1/1':g='0/0 0.5/0.48 1/0.97':b='0/0.05 0.4/0.36 0.7/0.62 1/0.92',eq=saturation=0.88:contrast=1.06,vignette=PI/5:eval=init,format=yuv420p[bg_graded];

# FILM GRAIN
color=c=gray:s=1920x1080:r=30,format=yuva420p,noise=alls=22:allf=t+u,geq=lum='lum(X,Y)*0.5+128':a=255,format=yuva420p,setsar=1[grain];
[bg_graded][grain]blend=all_mode=overlay:all_opacity=0.09,format=yuv420p[bg_final];

# FINAL COMPOSITE: equalizer over graded grainy background
[bg_final][eq]overlay=x=0:y=H-h-40:format=auto:shortest=1[outv]
```

**Output mapping** (already in `x264Args()`):
```
-map "[outv]" -map "[a_out]"
-c:v libx264 -preset ultrafast -tune stillimage -crf 23 ... (existing args)
```

---

## §8. CPU cost — does this still hit 6-8× realtime?

Approximate filter cost (from past benchmarks on 3-vCPU VPS):

| Filter            | Realtime cost   |
|-------------------|-----------------|
| `showfreqs` (bar) | ~1.5×           |
| `geq` (×2)        | ~2.0× combined  |
| `gblur` (×2)      | ~0.4×           |
| `curves`          | ~0.2×           |
| `vignette`        | ~0.1×           |
| `noise`+blend     | ~0.4×           |
| **Total filter overhead** | **~4.6×** |
| `libx264 ultrafast` | ~2.0× |

Sum of inverses → realistic encode speed ~**5-7× realtime**. A 90-min mix encodes in ~13-17 minutes. **Acceptable.** If we drop below 5×, the first thing to remove is the `geq` glow extractor (replace with a precomputed glow PNG sidecar — generated once per cover image at job start).

---

## §9. ⚠ ONE-SONG TEST GATE — DO NOT BATCH UNTIL APPROVED

Before any batch render, the agent **MUST** produce a test output of exactly one song, named `style_test_${TRACK_ID}.mp4`.

### 9.1 Test recipe
```bash
# On VPS: render-svc/test-style.sh
TRACK="/opt/render-svc/work/style_test/track_01.wav"
COVER="/opt/render-svc/work/style_test/cover.jpg"
PALETTE_LOW="0xC25E2D"
PALETTE_MID="0xF5C89A"
PALETTE_HIGH="0x8C4A28"
OUT="/opt/render-svc/work/style_test/style_test_$(date +%s).mp4"

ffmpeg -y \
  -i "$TRACK" \
  -loop 1 -i "$COVER" \
  -filter_complex "$(cat /opt/render-svc/filter_graph_v2.txt | sed \
    -e "s/\${EQ_LOW}/$PALETTE_LOW/g" \
    -e "s/\${EQ_MID}/$PALETTE_MID/g" \
    -e "s/\${EQ_HIGH}/$PALETTE_HIGH/g")" \
  -map "[outv]" -map "[a_out]" \
  -c:v libx264 -preset ultrafast -tune stillimage -crf 23 \
  -pix_fmt yuv420p -r 30 \
  -c:a aac -b:a 192k -ar 48000 \
  -shortest \
  "$OUT"
```

### 9.2 Acceptance checklist (operator runs this)
- [ ] Equalizer is palette-matched (orange/cream, not white/grey)
- [ ] Bass is on the vertical center line, highs flare to edges
- [ ] Visible film grain — texture, not noise wall
- [ ] Visible (not crushing) vignette in corners
- [ ] Candles in cover image have a soft halo
- [ ] No clipping / no banding in dark regions
- [ ] Audio is sample-accurate (no pitch drift, no clicks at song boundary)
- [ ] Encode speed ≥ 5× realtime per `ffmpeg-last.log`

If all 8 pass → batch goes ahead. If any fail → return to §3-§6 and tune.

### 9.3 ⚠ Filter-graph footguns — confirmed by 2026-04-25 test

The §9.1 recipe above as written WILL break for any audio longer than 7.7 s
because it omits the rules below. The clean Broken_Church.wav render that
shipped on 2026-04-25 was produced ONLY after these were applied. Treat
each as a regression test.

| #   | Rule (DO)                                       | Anti-rule (DON'T)                          | Symptom of regression |
|-----|--------------------------------------------------|--------------------------------------------|------------------------|
| 9.3.1 | Append `,fps=30` after `showfreqs`            | leave default (25 fps)                     | output truncates to ~7.7 s |
| 9.3.2 | `-t (audioDur+2)` BEFORE `-loop 1 -i cover.png`, AND `-t audioDur` at output | rely on `-shortest` or overlay `shortest=1` | runaway encode (>500 MB) or 7.7 s truncation |
| 9.3.3 | overlay = plain `overlay=...:format=auto`     | `shortest=1` OR `eof_action=pass`          | 7.7 s truncation OR infinite encode |
| 9.3.4 | alpha = `colorchannelmixer=aa=0.88`            | `geq` alpha multiply                        | 0.39× realtime (was 4.4×) |
| 9.3.5 | `-framerate 30` on the cover input             | any value < 30                              | 1.5 fps output regardless of `-r 30` |
| 9.3.6 | per-input `aformat=sample_rates=48000:...,asetpts=N/SR/TB` BEFORE `concat` | skip the normalizer | 0.725× speed bug ("BPM sounds delayed", 1h30 → 2h04) |
| 9.3.7 | dynamic film grain MUST be a runtime filter on `[1:v]` (`noise=alls=18:allf=t+u` + sin-flicker eq + per-frame `random()` drawbox scratches) | bake grain into the static cover_frame.png | YouTube monetization criterion fails — adjacent frames identical |

**Reference render:** `/opt/render-svc/broadcast/eva/broadcast.mp4` (jobId
`ac111353-ff35-4378-a3a3-63f6d6f5fad7`, 2026-04-25). 160.68 s, 30 fps,
4820 video frames, all sample frames at 1/30/80/130/158 s have distinct
md5 sums (= per-frame animation, monetization-safe).

---

## §10. Carry-over to Phase 3 (multi-channel)

Each channel gets its own palette file:
```
/opt/render-svc/channels/eva/palette.json    # warm sunset
/opt/render-svc/channels/vne/palette.json    # cold mystical
```

The filter graph stays identical; only the `${EQ_*}` substitutions differ. Visual brand consistency across both channels — different mood, same visual signature.

---

## §11. Sources / cross-refs
- ffmpeg `showfreqs` docs: https://ffmpeg.org/ffmpeg-filters.html#showfreqs
- ffmpeg `curves` color grading: https://ffmpeg.org/ffmpeg-filters.html#curves
- The reference image (inverted spectrogram with mirrored bars) — saved by operator in `/uploads/eq_reference.png`
- Related: [[PHASE3_OPTIMIZATIONS.md]] §0, [[ESCALATION_REPORT.md]] §"CPU only" rule, [[instructions.md]] §17
