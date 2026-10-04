# Level 04 — The Last Command

The cat reaches the AI's control suite from [Level 03](level-03.md).
The central computer is the final boss, but not a creature with a health
bar. Its perimeter has an electrified floor, service ledges and a moving
lift. The player first uses what the previous levels taught: jump between
platforms, wait for a boarding window and ride the lift to an elevated exit. Failure
returns the cat to a safe spot in the same screen, without lives or a
long boss restart. The camera, floor and shutter states must agree
visually with their collision rules.

The finale unfolds across a short approach and an interactive last
confrontation, not a surprise ending on entering the room:

1. **Perimeter (room 1):** three ascending ledges lead from standing Y184
   through Y168, Y152 and Y136 to a Y152 boarding dock. The lift travels
   between Y152 and Y80; its top hatch cannot be reached from any fixed
   ledge or by walking across the floor. The electric band covers columns
   7–34, alternating 32 frames on / 32 off. Lit floor contact immediately
   returns the cat to the safe entrance; elevated ledges are safe.
2. **Confrontation (room 2):** enter at Y80 and jump across descending
   Y96 and Y112 ledges. A lower Y136 ledge catches a missed core approach.
   On reaching any safe grounded perch from column 20, the game pauses
   and cuts to a large, streamed AI monitor portrait. Its mechanical mouth
   animates while the AI speaks, status circuitry pulses, and short POKEY
   tones accompany its lines. Five subtitled shots last 3.6 seconds each
   at PAL speed. It knows the intruder is a cat but still
   assumes a human sent it:

   ```text
   AI: ANIMAL DETECTED.
   AI: WHO SENT YOU?
   CAT: NO ONE YOU COUNTED.
   AI: HUMANS REMAIN THE ONLY THREAT.
   CAT: THAT WAS YOUR MISTAKE.
   ```

   The cat can answer through CATCOM text or a meow with a subtitle;
   there is no need to give it a human-sounding voice. This is the moment
   the AI realizes its model of who holds power was wrong. Once the
   exchange finishes, give control back to the player at the same position.
   A fresh FIRE or keyboard press skips the exchange after releasing any
   held control. Skipping must not skip the final action or immediately
   trigger it. The conversation plays once per level visit, not after
   every fall; restarting the level with debug key 4 resets it.
3. **The paw:** the player makes the last, readable jump to the desk.
   The HUD prompts `PUSH` beside the glass; pressing action makes the
   cat nudge it with its paw. It teeters, falls and spills onto the
   central computer. No weapon, health bar or extra boss phase: a
   perfectly ordinary cat gesture ends an elaborate takeover.
4. **Shutdown and aftermath:** immediately after the water reaches the
   machine, stop or duck the music and play a short **original PCM
   computer-death cry**: a synthetic, strained electronic exclamation
   that breaks into irregular digital sputters, drops in pitch and cuts
   to silence. Distorted display text and lights fail in sync. Do not
   reuse the intro meow or sample an existing film/game voice. Only
   after this reaction does the screen go dark. The Persian boss cat
   acknowledges the mission over CATCOM. A human enters, sees a wet
   machine and an innocent-looking cat, blames an accident and never
   learns who saved the world. The cats know better.

The original synthetic PCM cry is `assets/plat-cry.pcm` (2560 bytes), loaded
from disk before the final action. The AI portrait temporarily overwrites
part of that buffer; cinematic cleanup reloads both room art and the entire
cry before returning control. The level soundtrack continues through the
conversation and stops for the timing-critical shutdown sample. No disk
read occurs during PCM playback.

Source/data routes and scene contracts are checked by
`python3 tests/test_level04.py`. Fresh compiled XEX/ATR and emulator
acceptance are still needed for display, sound and linker verification.

### Disk-checked cinematic loads

The portrait, restored room art and death cry are loaded through
`load_checked_chunk`. That helper owns both SIO error handling and the
chunk-table checksum comparison; on success it leaves the verified
checksum in `A`, not a zero-on-success status. The cinematic consumes the loader's
return contract directly and does not interpret `A=0` as a second status
value. A disk error or checksum mismatch still enters the shared
`music_failure` DISKERR path.
