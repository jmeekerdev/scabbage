# Logic and circuits

Wire carries power from **sources** (switches, buttons, sensors…) to **machines** (doors, lamps,
pistons…). In between, **directed** blocks such as gates and delays do the thinking. Everything
here is placed like any other block from the Blocks menu.

## Basics

- **Use** a device (flip a switch, press a button, change a setting) with right-click. Hold
  Shift while right-clicking to build against it instead, for example to stack doors.
- **Facing.** Pistons, diodes, gates, delays, relays, capacitors and block detectors face the
  way you look when you place them. Look more than 45° up or down to make them face up or down.
- **Settings.** Using a gate, delay, capacitor, timer, clock, note block, transmitter or receiver
  steps it to its next setting. Gates, delays, capacitors and timers also come in the Blocks menu
  already set.
- **Timing.** Circuits run on the server's tick, 20 a second by default. Waits are rounded to
  whole ticks.

### Wire

- Every touching piece of wire or cable is one network, joined on all six sides (up to 4,096
  pieces). If any source touches the network, all of it is on. Power doesn't fade with distance.
- Plain `wire` lies flat and also climbs one block up or down a slope, as long as the block
  above the lower wire isn't solid.
- **Cables** (`stone_cable`, `cobblestone_cable`, `planks_cable`, `brick_cable`, `dirt_cable`)
  are solid blocks that look like their material and conduct. Hide wiring inside walls with
  them. They don't climb slopes.
- A machine is powered by wire next to it, or by a source right next to it.

## Sources

Sources give power out of all six sides.

| Block             | On while                                                                                                                                                                                                    |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Switch            | flipped on; each use flips it                                                                                                                                                                               |
| Button            | for 1 second after a press                                                                                                                                                                                  |
| Pressure plate    | a player stands on it (not animals or monsters)                                                                                                                                                             |
| Tripwire          | any player or creature touches it                                                                                                                                                                           |
| Timer             | half of each beat: `timer` is 1 s on, 1 s off. `timer_1`, `_2`, `_8` and `_16` beat every 1, 2, 8 or 16 quarter-seconds. Using it steps to the next speed. All timers share one clock, so they stay in step |
| Toggle            | a memory cell: each time power arrives at it, it flips between on and off. Using it flips it too                                                                                                            |
| Counter           | it counts up 0→9 each time power arrives, then back down to 0. It gives power from 9 and on the way down. Using it resets it to 0                                                                           |
| Receiver          | any loaded transmitter on its channel is on. 16 channels, shown as colors; use it to change                                                                                                                 |
| Player detector   | a player is within 4 m (12 blocks)                                                                                                                                                                          |
| Creature detector | an animal or monster is within 4 m                                                                                                                                                                          |
| Clock             | the time of day matches its mode: day, night, dawn, dusk, noon or midnight. Use it to change                                                                                                                |
| Weather sensor    | it's raining or snowing                                                                                                                                                                                     |
| Block detector    | the block in front of it isn't air or water                                                                                                                                                                 |

Sensors check four times a second.

## Directed blocks

These take power in at the back and give it out the front only, one tick after their input
changes.

| Block     | Does                                                                                                                                                                                                                                 |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Diode     | passes power one way. Use it to stop power flowing back                                                                                                                                                                              |
| Gate      | **and**, **or**, **xor**, **nand**, **nor** or **not**; use it to change. Inputs are the back and the two sides. Only sides with something that can give power count, so you can leave one empty. A **not** gate reads only its back |
| Delay     | waits 1 to 8 quarter-seconds (0.25–2 s), then copies its input                                                                                                                                                                       |
| Relay     | passes power only while its back **and** at least one side are powered                                                                                                                                                               |
| Capacitor | keeps giving power for 1 to 10 seconds after its input stops                                                                                                                                                                         |

## Machines

| Block                      | When powered                                                                                                                                                                                                                                     |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Signal lamp                | lights up. The plain `lamp` is always on and isn't part of circuits                                                                                                                                                                              |
| Door, trapdoor, fence gate | opens. You can also open and close them by hand. Doors open together with doors above and below them, trapdoors with the trapdoors beside them, fence gates with any that touch (up to 256 pieces). A door won't close on someone standing in it |
| Piston                     | pushes the row of blocks in front of it one block, up to 36 blocks, if there's air or water at the end. It can't push bedrock, circuit blocks, usable blocks, or across a zone's edge                                                            |
| Sticky piston              | pushes like a piston, and pulls the block back when the power stops                                                                                                                                                                              |
| Note block                 | plays its note when the power comes on. 8 notes, C to C; use it to change and hear it                                                                                                                                                            |
| Number display             | shows the digit of a counter next to it. Without a counter, it counts its own power pulses 0–9. Use it to add one                                                                                                                                |
| Transmitter                | turns on every receiver on its channel, anywhere in the loaded world. 16 channels                                                                                                                                                                |
| Launch pad                 | throws a player standing on it: about 6 m forward from a sideways pad, about 2.8 m straight up from an up pad (just short of a fall that hurts)                                                                                                  |
| Script block               | runs its script's `on_power`. See [Scripting](scripting.md)                                                                                                                                                                                      |

## Teleporters and checkpoints

- **Teleporter** pads need no power. Join two or more with wire. Step onto one and you arrive at
  the next one on that wire, and with three or more the pads take turns.
- **Checkpoint** pads remember you when you step on them. When you faint, you come back at your
  last checkpoint if it's still there and in the same zone. Otherwise you come back at spawn.

## Ideas

- **Secret door:** a pressure plate on each side of a door made of `stone_cable` walls.
- **Night light:** clock (night) → wire → signal lamps.
- **Doorbell:** button → wire → note block.
- **Blinker:** timer → wire → signal lamp. Use `timer_1` for fast blinks.
- **Combination lock:** two switches into an **and** gate, the gate's front into wire to a door.
- **Scoreboard:** button → counter, with a number display next to the counter.
