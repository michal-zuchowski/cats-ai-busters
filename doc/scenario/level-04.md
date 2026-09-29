# Level 04 — The Last Command (finale concept)

The cat reaches the AI's control suite from [Level 03](level-03.md).
The central computer is the final boss, but not a creature with a health
bar. It controls the room: shutters block paths, lights mark a dangerous
electrified floor and its cameras follow the cat. The player first uses
what the previous levels taught: wait for a safe window, jump between
service platforms and briefly cut power to reach the desk. Failure
returns the cat to a safe spot in the same screen, without lives or a
long boss restart. The camera, floor and shutter states must agree
visually with their collision rules.

The finale unfolds across a short approach and an interactive last
confrontation, not a surprise ending on entering the room:

1. **Perimeter:** cross the AI's last automated defenses and open the
   service path to the desk. The glass of water is already visible beside
   the central unit, but out of reach.
2. **Confrontation:** on reaching a safe perch, the room stops and the AI
   addresses the cat on its own monitor. Short, original lines appear
   one by one with enough time to read; the AI voice can be represented
   by simple POKEY tones or text. It knows the intruder is a cat but still
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
   exchange finishes, give control back to the player. Skipping dialogue
   must not skip the final action.
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

The PCM cry is a **separate asset to create later**, not a sound file that
exists yet. During playback the CPU may be needed for timed POKEY writes
(as with the intro meow); plan a short fixed animation and pause the
regular music, then restore the final scene's display and audio state.
Stream the sample from disk into a buffer before the confrontation if it
does not fit resident RAM; never rely on reading disk sectors in the
middle of the timing-critical PCM loop. Choose the exact spoken/noisy
syllable and length only after hearing it on the emulator.
