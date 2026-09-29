# Level 02 — The Vertical Route (concept)

After diverting the relay, the agent cat slips through the service hatch
into the building's maintenance shaft. The AI's infrastructure plan has been
delayed, not stopped. The cat needs to reach the roof uplink without being
noticed; the people working below still think this is an ordinary night.

This level changes the rhythm of play: four fixed, full-screen side-view
rooms built around **jumping**, not camera timing or combat. The cat can run
left and right and jump from solid ground or a platform. A jump must be
readable before the player commits to it: show the landing place, keep the
first gaps forgiving, and never require a leap beyond the screen edge.
Falling returns the cat to a safe spot in the current room, with no lives
or long replay. Keep the established cat silhouette and dense industrial
Atari look, but leave clear space around ledges and landing zones.

1. **Service shaft — learn to jump.** A short gap and a broad ledge teach
   the height and distance of the jump with a safe floor below. The shaft
   gradually leads upward, not into a surprise hazard.
2. **Cable trays — choose a route.** Suspended trays at different heights
   make the cat cross the room in a series of deliberate jumps. Lower
   platforms catch missed jumps; the player can recover rather than reset
   after every mistake.
3. **Cooling plant — time the airflow.** A visible fan cycles between
   stillness and a gust that shifts a jump sideways. First let the player
   observe the cycle from safety, then cross a gap that can be cleared by
   waiting for the right moment or compensating in midair.
4. **Maintenance lift — combine what was learned.** A slow moving service
   lift carries the cat toward a high platform. It can be boarded safely
   and tried again if missed. A final jump reaches the roof access and
   leaves the AI's uplink as the next story lead, not the game's finale.

The HUD gives short contextual hints such as `JUMP` or `WAIT`, without
interrupting play with a tutorial. Moving machinery should have a clear
cycle and safe waiting area. The rooms can load from disk as the existing
level assets do, while the music keeps playing. Exact platform coordinates,
jump physics and whether the lift moves vertically or diagonally should
be settled by playtesting, not guessed in this concept.

There are **no robot vacuums in this level**. Their two-sided role belongs
to [Level 03 — The Cleaning Protocol](level-03.md), followed by
[Level 04 — The Last Command](level-04.md).
