# Adventure

Characters, quests, dungeons and bosses, and adventure maps you can save, reset and share.

## Characters and quests

Characters and quests come from [scripts](scripting.md). A script block with `npc("Name")`
stands a character on top of itself. Right-click the character to talk, and press 1–4 or click
to answer.

Scripts give quests with `quest` and finish them with `finish_quest`. Open **Quests** (Q) to see
yours: newest first, with the next step or ✔ when done. News also shows in chat:
"📜 New quest", then each step, then "🏆 Quest done". A player keeps up to 50 quests. When the log
is full, the oldest finished one makes room.

Scripts can also remember things about each player, such as "has met Granny Ivy", and any script
can read them. That's how characters in different places can know what you've done.

## Dungeons

About one in three 64 m squares of the world hides a dungeon, but none are near spawn. Look for
a stone arch with a torch. Stairs lead down at least 8 m to a cobblestone room lit by lamps,
with two chests and a boss altar.

- **Chests** give their treasure once: 1–3 gold blocks, two other things (torches, lamps, glass,
  brick, wool, teleporters or note blocks), and sometimes a gem. What doesn't fit in your
  backpack drops at your feet.
- **Boss altars** wake a boss when you use them: King Slime or the Stone Golem, always the same
  one for each altar. Only one boss can be awake within 32 m. Whoever lands the last hit gets
  2–3 gems and 3–5 gold, and everyone hears about it.

See [Survival](survival.md#animals-and-monsters) for the bosses' strength.

## Adventure maps

A map is a zone saved with its blocks, scripts, start spot and mode. Admins can reset a map
after a playthrough, or export it to a `.scabmap` file and import it on another server.

| Command            | Does                                                                                             |
| ------------------ | ------------------------------------------------------------------------------------------------ |
| `/map sample NAME` | builds the sample adventure (Captain Pip's quest) as a new zone around you                       |
| `/map save ZONE`   | saves the zone as it is now                                                                      |
| `/map start ZONE`  | marks where you stand, and the way you face, as the start                                        |
| `/map reset ZONE`  | puts the zone back as saved, sends players in it to the start and clears their checkpoints there |
| `/map export ZONE` | downloads the saved map                                                                          |
| `/map import NAME` | opens a file picker and places the map as a new zone around you                                  |
| `/map remove ZONE` | for imported and sample maps: puts back the land that was there and removes the zone             |
| `/map forget ZONE` | drops the saved copy, keeping the zone and its blocks                                            |
| `/map list`        | lists saved maps                                                                                 |

To make one:

1. Make a zone with `/zone add NAME SIZE`, standing in the middle.
2. Build it, with characters, checkpoints, chests and puzzles.
3. Save it with `/map save NAME`, then stand at the entrance and type `/map start NAME`.

Maps can be up to 256 blocks across, 4 MB as a file, and 1,024 scripts. An imported map can't
overlap another zone or cross the world's border. Blocks this server doesn't have become air,
and its scripts belong to whoever imported it.
