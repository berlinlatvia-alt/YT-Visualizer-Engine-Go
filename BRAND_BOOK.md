# 📕 BRAND BOOK — Eva Vibe / Вне Системы
**Status:** v1 LOCKED — derived from approved Eva Vibe meditation thumbnail
**Date:** 2026-04-25
**Authority:** anything that ships to YouTube or any other channel must follow this document.

This is the visual signature listeners must recognize. **Channel-level identity = title block + composition + palette + visualizer.** Image content can change; the four signature elements above must not.

---

## §1. The Title Block (the part Anton specifically loves)

The title block is a three-line stack in the **upper third** of every cover/frame:

```
   line 1: TAGLINE          — small caps, sans-serif, letter-spaced wide
   line 2: WORDMARK         — large serif, gold, dominant element
   line 3: SUBTITLE         — small caps, sans-serif, letter-spaced wide
```

Reference (Eva Vibe approved):
```
   MEDITATION MUSIC • DEEP RELAXATION
                E V A   V I B E
                  FIND YOUR PEACE
```

### 1.1 Typography spec

| Role     | Font family             | Weight   | Case           | Tracking | Color      | Effect                       |
|----------|-------------------------|----------|----------------|----------|------------|------------------------------|
| TAGLINE  | Inter / Lato / Futura PT| Medium   | UPPER + small | +180     | `#E8D9A8`  | none (clean)                 |
| WORDMARK | Cinzel / Trajan Pro     | Regular  | Mixed (caps for "EVA VIBE") | +120 | `#E8D9A8` → `#B8893E` vertical gradient | inner soft glow ~6px @ `#FFE9B8` 35% |
| SUBTITLE | Inter / Lato / Futura PT| Medium   | UPPER + small | +200     | `#E8D9A8`  | none                         |

**Forbidden:** Drop shadows. Hard outlines. Outline-only text. Any font with serifs that aren't Trajan-style (no Times, no Garamond). All-caps for the wordmark with no tracking (looks crammed).

### 1.2 The wordmark on its own line

The wordmark uses **wide character spacing** — not just CSS letter-spacing, but extra spaces between each character group: `E V A   V I B E` (two spaces between "EVA" and "VIBE", one space between letters within each word). This is the single most identifiable element.

### 1.3 Channel variants

| Channel       | Wordmark                                | Tagline (top)                         | Subtitle (bottom)                       |
|---------------|-----------------------------------------|---------------------------------------|-----------------------------------------|
| Eva Vibe (EN) | `E V A   V I B E`                       | `MEDITATION MUSIC • DEEP RELAXATION`  | `FIND YOUR PEACE`                       |
| Eva Vibe (intl) | `E V A   V I B E`                     | `MUSIC FOR MEDITATION`                | `BREATHE • FEEL • RELEASE`              |
| Вне Системы   | `В Н Е   С И С Т Е М Ы`                 | `МЕДИТАТИВНАЯ МУЗЫКА • ГЛУБОКИЙ ТРАНС` | `НАЙДИ СЕБЯ`                            |

For Cyrillic, swap Cinzel → **PT Serif Caption** or **EB Garamond** (Cinzel doesn't have proper Cyrillic glyphs).

---

## §2. Composition (the layout grid)

```
┌─────────────────────────────────────────┐
│                                         │  ← 5% top safe zone
│    ┌─────────TITLE BLOCK──────────┐    │
│    │   tagline (line 1)            │    │  ← top 20-30% of frame
│    │   WORDMARK (line 2)           │    │
│    │   subtitle (line 3)           │    │
│    └───────────────────────────────┘    │
│                                         │
│            [HERO IMAGE]                 │  ← center 55-65%
│      (subject + warm light source)      │
│                                         │
│    ▁▂▄▆▇█ EQUALIZER ▇▆▄▂▁                │  ← bottom 10-15%
│                                         │  ← 3% bottom safe zone
└─────────────────────────────────────────┘
```

### 2.1 Hero image rules

- **Subject anchored bottom-center** (meditator, single artifact, etc.) — not centered on the frame, lower than the wordmark.
- **Warm point-light source** must be present: candles, sunset, lamp, fire. This is what the candle-glow filter latches onto and what the palette extractor samples for `EQ_LOW`.
- **Architectural framing** (window arch, doorway, columns, mountain valley V-shape) creates the gothic/sacred feel that visually distinguishes us from generic spa/lo-fi channels.
- **Aspect:** 16:9 always. Never crop the title block to fit a different aspect — re-render it.

### 2.2 Negative space rule

The wordmark must sit on a region of the image that's **at least 35% darker than the wordmark color**. Either pick an image where the upper third is dark (Eva Vibe sunset has this naturally — dark stone arch behind the title), or apply a localized linear gradient overlay (`drawbox=color=black@0.35:y=0:h=H/3` before the title is composed in).

---

## §3. Palette (already in `VISUAL_STYLE_OVERHAUL.md` §2 — repeated here for completeness)

| Slot         | Eva Vibe (warm)       | Вне Системы (cold)        |
|--------------|------------------------|----------------------------|
| Wordmark     | `#E8D9A8` → `#B8893E`  | `#C8B8E8` → `#5A4A8A`      |
| EQ_LOW       | `#C25E2D` (ember)      | `#5A2A6E` (deep violet)    |
| EQ_MID       | `#F5C89A` (cream)      | `#9A85C5` (lavender)       |
| EQ_HIGH      | `#8C4A28` (copper)     | `#3A2858` (indigo)         |
| Glow tint    | `#F5C89A` @ α0.30      | `#9A85C5` @ α0.30          |
| Background floor | `#1A0805` (near-black) | `#0A0815` (near-black blue) |

---

## §4. The Visualizer (recap — full filter graph in `VISUAL_STYLE_OVERHAUL.md`)

- **Type:** `showfreqs` bars, mirrored on horizontal axis (NOT `showwaves`).
- **Position:** bottom 10-15% of frame, full width, centered vertically on its own band.
- **Color:** sampled from image palette per §3.
- **Treatment:** `gblur=sigma=1.0` for soft glow, alpha 0.82 so it floats over the image.

### §4.1 Anti-regression rules — learned the hard way (2026-04-25)

These exact failures cost us most of a day. Future agents touching the render
filter graph MUST not undo any of them.

1. **`showfreqs` defaults to 25 fps.** Without an explicit `,fps=30` filter
   right after `showfreqs`, the equalizer's PTS clock runs ahead of the 30 fps
   cover-frame stream and the resulting overlay terminates at ~7.7 s for any
   audio length. ALWAYS append `,fps=30`.
2. **`shortest=1` on the final overlay is forbidden.** It interprets a
   transient EOF event from the showfreqs warm-up window as the end of the
   stream and truncates output to 7.7 s — verified empirically. Do not put
   it back even if the comments around it look temptingly safe.
3. **`eof_action=pass` on the final overlay is forbidden.** Combined with the
   `-loop 1` cover input it produces an infinite video stream and the encode
   never finalizes (file grows past 500 MB, no moov atom written).
4. **Length governance is OUTPUT-LEVEL.** The only reliable way to bound the
   render is the pair `-t (audioDur+2) -loop 1 -framerate 30 -i cover.png`
   BEFORE the cover input plus `-t audioDur` at the encoder. ffprobe the
   audio for the duration before spawning ffmpeg.
5. **Alpha multiply: `colorchannelmixer=aa=0.88`, never `geq`.** The geq path
   was 0.39× realtime; colorchannelmixer is 4.4×+ for the same visual.
6. **`-framerate 30` before the cover input.** With `-loop 1`, the input rate
   becomes the frame-emission clock. `-framerate 2` produced 1.5 fps output
   regardless of `-r 30` at the encoder. Match the values.
7. **Audio normalize MUST be per-input, BEFORE concat.** Each audio input
   needs `aformat=sample_rates=48000:sample_fmts=fltp:channel_layouts=stereo,
   asetpts=N/SR/TB` before reaching `concat=...:v=0:a=1`. Skipping this
   produces the 0.725× speed bug (1h30 mix lands at 2h04, "BPM sounds delayed").

A clean Broken_Church.wav render produced 2026-04-25 lives at
`/opt/render-svc/broadcast/eva/broadcast.mp4` and matches all four
acceptance criteria below. Use it as the regression baseline.

---

## §5. Brand consistency rules

1. **Title block on every frame** — including the countdown pre-roll, including the radio still frame, including any social cuts. No "naked" composition.
2. **Wordmark is sacred** — never resize below 60% of the canonical 8% frame-height target. Never re-color outside the gold gradient. Never replace the font.
3. **Visualizer style is identical across both channels** — only the palette swaps. A listener flipping between Eva Vibe and Вне Системы should immediately see "same hand made these."
4. **Hero image must follow §2.1** — subject + warm light + architectural frame. We will reject AI-generated covers that lack any of three.
5. **Text language follows channel language**, not the listener's. Eva Vibe is English globally. Вне Системы is Russian globally.

---

## §6. Source-of-truth files for image generation

The covers we love look like they're MidJourney v6 / Flux / Imagen renders. Lock the prompt template:

```
[Eva Vibe template]
A young woman with dark hair sitting in lotus position, eyes closed, wearing
flowing earth-tone clothing, in a candlelit gothic stone chamber with arched
windows opening to a warm sunset, dozens of candles arranged around her,
plants and meditation cushions, cinematic warm lighting, golden hour, 
moody atmospheric, photorealistic, shallow depth of field, 16:9
--ar 16:9 --style raw --stylize 250
```

```
[Вне Системы template]
A solitary figure in dark robes facing away from camera, standing at the
edge of a forest clearing under a starlit night sky, distant mountains,
a single lantern casting cold blue light, fog low to the ground, mystical 
runes faintly carved into nearby standing stones, cinematic, photorealistic, 
shallow depth of field, 16:9
--ar 16:9 --style raw --stylize 300
```

These templates produce the §2.1-compliant compositions automatically. Save successful generations to `/opt/render-svc/brand/covers/{channel}/` for reuse and as palette sources.

---

## §7. Render-svc integration — the title block as overlay

The title block is **not part of the source cover image**. It's overlaid by `render-svc` at render time using `drawtext`, so we can:
- Generate identical covers across both channels (just swap text)
- Re-render with new wordmarks without re-generating images
- A/B test taglines without re-running MidJourney

```
# Add this BEFORE the equalizer overlay in the filter graph
[bg_final]
  drawtext=
    fontfile=/opt/render-svc/fonts/Inter-Medium.ttf:
    text='MEDITATION MUSIC • DEEP RELAXATION':
    fontcolor=0xE8D9A8:fontsize=42:
    x=(w-text_w)/2:y=H*0.10:
    expansion=normal,
  drawtext=
    fontfile=/opt/render-svc/fonts/Cinzel-Regular.ttf:
    text='E V A   V I B E':
    fontcolor=0xE8D9A8:fontsize=160:
    x=(w-text_w)/2:y=H*0.16:
    shadowcolor=0xFFE9B8@0.35:shadowx=0:shadowy=0:
    borderw=0,
  drawtext=
    fontfile=/opt/render-svc/fonts/Inter-Medium.ttf:
    text='FIND YOUR PEACE':
    fontcolor=0xE8D9A8:fontsize=44:
    x=(w-text_w)/2:y=H*0.28:
    expansion=normal
[bg_titled];

# Then overlay equalizer on [bg_titled] instead of [bg_final]
[bg_titled][eq]overlay=x=0:y=H-h-40:format=auto:shortest=1[outv]
```

Fonts to install on VPS:
```bash
mkdir -p /opt/render-svc/fonts
# Download once, free fonts:
curl -sL "https://github.com/googlefonts/Inter/raw/main/fonts/ttf/Inter-Medium.ttf" \
  -o /opt/render-svc/fonts/Inter-Medium.ttf
curl -sL "https://github.com/googlefonts/cinzel/raw/main/fonts/ttf/Cinzel-Regular.ttf" \
  -o /opt/render-svc/fonts/Cinzel-Regular.ttf
# For Cyrillic (Вне Системы):
curl -sL "https://github.com/googlefonts/PT_Serif/raw/main/fonts/ttf/PTSerif-Regular.ttf" \
  -o /opt/render-svc/fonts/PTSerif-Regular.ttf
```

The wordmark text is per-channel config:

```js
// render-svc/channels.json
{
  "eva": {
    "wordmark": "E V A   V I B E",
    "tagline":  "MEDITATION MUSIC • DEEP RELAXATION",
    "subtitle": "FIND YOUR PEACE",
    "wordmarkFont": "Cinzel-Regular.ttf",
    "bodyFont":     "Inter-Medium.ttf"
  },
  "vne": {
    "wordmark": "В Н Е   С И С Т Е М Ы",
    "tagline":  "МЕДИТАТИВНАЯ МУЗЫКА • ГЛУБОКИЙ ТРАНС",
    "subtitle": "НАЙДИ СЕБЯ",
    "wordmarkFont": "PTSerif-Regular.ttf",
    "bodyFont":     "Inter-Medium.ttf"
  }
}
```

---

## §8. Cross-refs
- [[VISUAL_STYLE_OVERHAUL]] — the visualizer + grade + grain spec this brand book sits on top of.
- [[AUDIO_DURATION_ANOMALY]] — the 32k-stretch bug, must be fixed before any new render.
- [[PHASE3_OPTIMIZATIONS]] — multi-channel API support (`?channel=eva|vne`) reads `channels.json` from §7.
- [[instructions.md]] §17 — render-svc's libx264 args, unchanged.
