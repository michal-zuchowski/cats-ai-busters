# Level 03 — The Cleaning Protocol (concept)

The roof uplink from [Level 02](level-02.md) reveals a route back down to
the AI's sealed control suite. To protect that route, the AI takes control
of ordinary robot vacuums. It still has no idea cats are coordinating
against it. The machines are funny because they are familiar household
objects; the danger and the joke must both be legible on a small screen.

Four fixed screens let the cat turn the AI's own machines against it.
The rhythm is **paw-swat, shove, ride** rather than conventional combat:
no health bars, destroyed vacuums or separate driving game.

1. **Cleaning aisle:** one AI-controlled vacuum patrols between visible
   turn points. The cat can jump over it or stalk it: pause just outside
   its path, crouch with hind paws planted, dart a **front paw** at its
   rim (`pac!`), then pull back or hop away before it rolls forward.
   A hit makes the vacuum wobble and reverse briefly, opening a safe
   route. A side collision while unprepared returns the cat to this
   screen's entrance.
2. **Chargers:** two paths overlap, with a safe place to watch them.
   A short-range paw tap near the charging alcove makes one vacuum turn
   into it; its confused blinking light shows the opening. Let the cat
   step forward, tap, retreat and reassess rather than trading blows.
   The displaced machine eventually recovers, allowing another attempt.
3. **The ride:** hop onto a slow vacuum and let it carry the cat through
   a low maintenance passage. Another vacuum approaches from behind.
   While sitting on the first, the cat casually extends a **hind paw**
   and shoves the pursuer away without leaving its seat. The push is
   funny and useful: the pursuer rolls back long enough to reach a
   broad ledge. Jump off there; a miss leaves a route back to the ride.
4. **Control-suite access:** swat the one front vacuum through the
   service reader (`RELAY` turns `DONE`), ride it over the first pit, step
   off at the service balcony and jump the last gap to the floor-height
   exit. It reuses the grounded swat and the ride; there is no pursuer or
   shove in this room.

Keep the existing left/right and up-to-jump controls. On the floor,
fire makes a short front-paw swat; sitting on a vacuum, fire makes a
short backward hind-paw shove. Land on top to ride and jump to
dismount. Show `SWAT`, `RIDE` and `PUSH` only where they apply. The
two paw moves need distinct silhouettes: a low, cautious forepaw jab
and recoil while grounded versus the lazy backward hind-paw shove while
riding. The grounded jab must connect only at the paw's short reach,
not throughout the animation; a machine that has been swatted must be
visibly different
from one about to collide. A push changes the other vacuum's position
or direction, not the cat's ride. Patrols reset on retry, checkpoints
stay within the current screen, and it must never be possible to
strand the ride out of reach. For the grounded movement, see the
[short cat-and-vacuum reference](https://www.youtube.com/watch?v=LnDVv1JWSjs):
the cat circles, cautiously reaches toward the vacuum and backs off.
It is a movement reference, not an animation to copy frame for frame.

The service door leads into [Level 04 — The Last Command](level-04.md).
Exact platform positions, timing and vacuum art remain open for playtesting.

## Redesign: outsmart, dock and ride AI vacuums

- Room 1 (corridor): one patrolling vacuum; use the niches, swat it (grounded front-paw hit at real pixel contact, best during its wobble/reversal) or jump it. Podium exit.
- Room 2 (charger): lasers (row 19) guard the aisle. Swat the vacuum so it slides right into the marked charging bay; it blinks and lasers switch off for 240 frames, then it recovers. The first valid dock opens the visibly closed door for the rest of this attempt (HUD: EXIT), even when the laser timer ends. The hatch is at floor height: walk right through it without a low shelf or a final jump. A leftward hit cannot dock it; walking directly to the door cannot bypass docking. A catch or laser hit resets the objective and closes the door.
- Room 3 (ride): board the slow vacuum (RIDE), cross the pit. A separate pursuer follows; the seated hind shove (PUSH) repels it without changing the carrier. Staying at the forward roof edge is an alternative dodge, not a phantom collision. Jump to the exit ledge.
- Room 4 (final, one carrier): approach the single vacuum from the left,
  face right and swat it through the marked reader. `RELAY` becomes `DONE`
  and the visibly closed door opens. Wait near the first pit, jump onto
  the returning vacuum and ride it right. Jump to the service balcony,
  then across the last gap to the floor-height exit. The captions and
  SWAT / RIDE / JUMP / EXIT hints follow each stage. An early dismount can
  land on the lower middle island: walk toward its right edge and jump
  the remaining gap. Falling resets the room, closes the door and returns
  the reader to `RELAY`. There is no rear pursuer or shove here: the parked
  rear robot that could trap the cat on an unmoving roof has been removed.
Lamp glyphs: patrol 91-93, struck 124/125 base, chase 1/2/6, dock bay/reader strip 12, dock/reader head reuses plinth 88.

Check commands (no compiler): `python3 tests/test_l03_assets.py`, `python3 tests/test_l03_routes.py`, and `python3 tests/test_l03_runtime.py --asset-overlay` (R4 joystick/FIRE route on the user's existing XEX with only the R4 room sector overlaid in memory; the new HUD/reader/root-guard code is not compiled there). Without `--asset-overlay` the runtime test requires a fresh build (`R4Goal` symbol) and also checks the captions, reader and roof guard.
