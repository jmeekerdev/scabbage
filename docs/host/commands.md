# Commands

Type commands in chat (Enter, T or /). Tab completes names, blocks and arguments. Capitals don't
matter. `/help` lists the commands you can use.

## Everyone

| Command                           | Does                                                                |
| --------------------------------- | ------------------------------------------------------------------- |
| `/help`                           | lists commands                                                      |
| `/settings`, `/blocks`, `/quests` | open those menus                                                    |
| `/create`                         | paint your skin or a new block                                      |
| `/emotes`                         | the emote wheel                                                     |
| `/buttons`                        | show or hide the on-screen menu buttons                             |
| `/goto X Y Z`                     | the autopilot flies you there. `~` is where you are, `~5` is 5 more |
| `/fill X1 Y1 Z1 X2 Y2 Z2 BLOCK`   | fill a box's empty spots                                            |
| `/clear X1 Y1 Z1 X2 Y2 Z2`        | break every block in a box                                          |
| `/build NAME [W D H] [BLOCK]`     | build a cabin, tower, wall, floor or stairs in front of you         |
| `/yes`, `/no`                     | build or cancel the job waiting for you                             |
| `/stop`                           | stop the autopilot                                                  |
| `/undo`                           | take back the last job                                              |
| `/leave`                          | leave a minigame round                                              |
| `/agent link [LABEL]`             | link an AI agent (admins, or players an admin allowed)              |
| `/agent list`                     | your linked agents                                                  |
| `/agent revoke LABEL` / `all`     | turn an agent's token off                                           |

The autopilot commands work in creative only. See [Building](../play/building.md#autopilot) and
[Agents](../play/agents.md).

## Admins

| Command                                 | Does                                                           |
| --------------------------------------- | -------------------------------------------------------------- |
| `/invite`                               | make an invite link                                            |
| `/approve NAME`, `/deny NAME`           | answer a join request                                          |
| `/admin NAME`, `/unadmin NAME`          | give or take admin rights                                      |
| `/creative NAME`, `/survival NAME`      | set a player's mode                                            |
| `/tp X Z`, `/tp X Y Z`, `/tp NAME`      | teleport                                                       |
| `/agent allow NAME`, `/agent deny NAME` | let a player link agents, or stop them and revoke their tokens |
| `/agent confirm on`, `off`              | whether your own agents' jobs wait for your `/yes`             |

### `/zone`

| Command                                                 | Does                                                    |
| ------------------------------------------------------- | ------------------------------------------------------- |
| `/zone add NAME SIZE`                                   | a square zone SIZE blocks wide (1–768), centered on you |
| `/zone remove NAME`                                     | remove it                                               |
| `/zone allow NAME PLAYER`, `/zone disallow NAME PLAYER` | add or remove a builder                                 |
| `/zone mode NAME creative\|survival\|any`               | the zone's game mode                                    |
| `/zone list`                                            | list zones                                              |

See [Running a server](admin.md#zones).

### `/arena`

| Command                                                   | Does                                                          |
| --------------------------------------------------------- | ------------------------------------------------------------- |
| `/arena game ZONE race\|flags\|tag\|spleef\|last\|custom` | pick a game and its defaults                                  |
| `/arena spawn ZONE [TEAM]`                                | mark a spawn where you stand. Teams: red, blue, green, yellow |
| `/arena goal ZONE [TEAM]`                                 | mark a goal where you stand                                   |
| `/arena portal ZONE`                                      | make the portal block you stand on join this arena            |
| `/arena win ZONE goal\|last\|points N`                    | how a round is won                                            |
| `/arena timer ZONE SECONDS`                               | time limit (0–3600, 0 for none)                               |
| `/arena pvp ZONE on\|off`                                 | whether players can hit each other                            |
| `/arena clear ZONE`                                       | remove the arena                                              |
| `/arena list`                                             | list arenas and what each still needs                         |

See [Minigames](../play/minigames.md).

### `/map`

| Command            | Does                                             |
| ------------------ | ------------------------------------------------ |
| `/map save ZONE`   | save the zone as an adventure map                |
| `/map start ZONE`  | mark the start where you stand                   |
| `/map reset ZONE`  | put the zone back as saved                       |
| `/map export ZONE` | download a `.scabmap` file                       |
| `/map import NAME` | place a `.scabmap` file as a new zone around you |
| `/map sample NAME` | place the sample adventure                       |
| `/map remove ZONE` | remove an imported map and put the land back     |
| `/map forget ZONE` | drop the saved map, keeping the blocks           |
| `/map list`        | list saved maps                                  |

See [Adventure](../play/adventure.md#adventure-maps).

### `/world`

| Command               | Does                               |
| --------------------- | ---------------------------------- |
| `/world reset`        | start the world over from its seed |
| `/world new [SEED]`   | make a new world                   |
| `/world approve CODE` | approve another admin's request    |
| `/world deny`         | cancel the request                 |

See [Running a server](admin.md#starting-the-world-over).
