# Graphics decisions

- **Intro terminal:** ANTIC 4 character mode (40 columns, 24 rows) with a compact 3×5 font for white text on a dark-green background. ANTIC 2 mixes the text and background hues, so it cannot reproduce this color scheme. Enlarge the terminal bitmap once, push in through three growing monitor frames, then reveal the lines one at a time.
- **Intro cinematic shots:** ANTIC E / Graphics 15 bitmap (160 × 192, four colors) for the storm, the cat reveal, and the close-up. Use cuts between compositions rather than implementing a real camera zoom. Animate the pupils by changing only the eye region.
- **Gameplay:** ANTIC 4 / Graphics 12 character mode (40 × 24 tiles) for rooms and scenery. Use player/missile graphics for the cat and other moving objects that need smooth motion.

The intro and gameplay do not need to share a display mode. The PCM player occupies the CPU while the meow plays, so muzzle animation runs from within its playback loop.

**CATCOM video call:** After the meow, the ANTIC 4 terminal shows a
connecting status, then `dl_call` cuts to a 160 × 80 ANTIC D bitmap
(`assets/call.pic`, 4 KB-aligned, one LMS) above a single ANTIC 4 subtitle
row. The shot follows the films: Blofeld from the neck down in a high-backed
armchair, one hand stroking the white Persian on his lap (copper eyes,
diamond collar), the other on the armrest. Colors: white cat and subtitles,
gray suit and dithered leather, tan skin and eyes. Six length-prefixed
lines from `assets/call-text.pic` stay on screen for 200 frames (~4 s) each; on boss
lines `call_mouth` swaps six bitmap bytes from `assets/call-mouth.tbl`
(closed/open) four times with a POKEY blip. A 16-tile character mosaic was
tried first and was too coarse for a readable cat. The extra bitmap fits
because the project links with `-hiaddr 0x7FFF`
(bank `$2000–$7FFF`; buffers stay at `$8000+`).

**Implemented mouth sync:** `play_meow` and `mouth_tick` are both in `main.k65`; `sample.k65` holds the PCM data. Playback calls `mouth_tick` once per 256-sample page boundary — 12 times across the ~0.59s sample — toggling the mouth directly inside the sample loop without changing its per-sample `WSYNC` timing. Keeping both functions together avoids a forward reference across separately compiled K65 files.

**GR.15 bitmap layout:** ANTIC E draws one physical scanline per display-list
instruction. `python3 tools/make_intro_art.py` generates the 256×96 panorama
(`assets/room.pic`, 64 bytes per row), 160×96 close-up
(`assets/face.pic`, 40 bytes per row), and 12×32-byte reveal overlay
(`assets/cat-strip.pic`). Each image row is fetched twice via LMS for 192
scanlines. Both bitmaps start on 4 KB boundaries so no 40-byte visible DMA
fetch crosses ANTIC's 4 KB playfield boundary, even with the panorama shifted
24 bytes. The display lists are 1 KB-aligned. Pupil and muzzle animation
changes only the relevant bytes of `FaceImg`.

**Storm/datacenter shot:** `build_storm_list` creates 96 pairs of LMS entries
with a 64-byte source stride. `storm_pan` moves each LMS source address
24 times by one byte (four pixels), holding each step for 12 PAL frames:
288 frames, approximately 5.76 seconds at 50 Hz. LEDs are independently
switched in the bitmap throughout the pan. Three brief palette flashes are
paired with fading POKEY channel-4 thunder; channels 2–3 supply quiet
continuous fan noise and hum, leaving channel 1 for the PCM meow.
The room uses dark blue, sea-green highlights and amber LEDs against black.
The floor has short staggered seams rather than converging lines.
The first zoom stage doubles an 80×48 crop around the room's monitor into
spare XL RAM at `$8000`, then reuses the 160×96 display list for that image.
The following three 40-column framed shots grow and recenter the monitor
before the text appears; the zoom buffer never overwrites the cat or terminal
assets. `$8000–$8EFF` is below the XL/XE cartridge and OS ROM regions.
The terminal has its own dark-green palette with white glyphs. Its custom
ANTIC 4 font is generated into `assets/term-font.pic`; 960 bytes of character
screen cover the full height, avoiding the blank lower screen from the old
8-row shot.

The cat is not in the panorama during the opening. After the terminal's
`Y_` prompt, `show_cat` copies only the bottom-right strip into the room and
the same display list shows the far-right end of the panorama for the reveal.
The rack colors remain blue-green above the floor in the reveal; a single
display-list interrupt switches to navy, gray fur and cream highlights
at row 70, where the cat begins. No rack is recolored halfway down its
cabinet. The close-up uses the same navy/gray/cream palette, so both shots
keep the room's cooler lighting without tinting the tabby blue. The close-up
background retains the small terminal with `Y_`. Rain remains static artwork
illuminated by lightning. `cut_to_black` also stops ambient POKEY channels.
