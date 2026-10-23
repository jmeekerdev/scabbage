# Minigames

Any zone can become an **arena**. Players walk through a portal to join a round, play, and are
sent back to the portal at the end, with their health, inventory and place in the world put back
as they were. Admins set up arenas with `/arena`; anyone can play.

## Make an arena

1. Build the play area and save it as a zone: stand in the middle and type `/zone add NAME SIZE`
   (see [Running a server](../host/admin.md#zones)).
2. Pick a game: `/arena game NAME race` (or `flags`, `tag`, `spleef`, `last`, `custom`).
3. Stand where players should start and type `/arena spawn NAME`. Add a team color to make a
   team spawn: `/arena spawn NAME red`.
4. For games won at a goal, stand there and type `/arena goal NAME` (or `… goal NAME blue`).
5. Place a `portal` block outside the arena, stand on it and type `/arena portal NAME`.
6. Type `/arena list` to check it. Anything missing is listed on the arena's line.

Spawns and goals show as colored pads with banner poles. Each mark covers a 1 m (3-block)
square, and an arena can have up to 16 of each kind. Spawns and goals must be inside the zone;
portals can be anywhere.

### The games

`/arena game` also sets the win rule, timer and PvP to the game's defaults:

| Game     | How you win                                  | Timer  | PvP |
| -------- | -------------------------------------------- | ------ | --- |
| `race`   | first to reach a goal                        | 5 min  | off |
| `flags`  | capture the other team's flag 3 times        | 10 min | on  |
| `tag`    | be the last one left; one hit and you're out | 3 min  | on  |
| `spleef` | be the last one standing on the floor        | 5 min  | off |
| `last`   | be the last one standing                     | none   | on  |
| `custom` | whatever you set up                          |        |     |

You can change any of these afterwards:

| Command                      | Does                                                                         |
| ---------------------------- | ---------------------------------------------------------------------------- |
| `/arena win NAME goal`       | first to a goal wins                                                         |
| `/arena win NAME points N`   | first side to N points wins (1–100). Goals score a point, and so can scripts |
| `/arena win NAME last`       | last player or team in wins                                                  |
| `/arena timer NAME SECONDS`  | time limit, up to 3600; `0` for none                                         |
| `/arena pvp NAME on` / `off` | whether players can hit each other                                           |
| `/arena clear NAME`          | removes the arena (the zone stays)                                           |

What an arena needs before it can start: at least one spawn; a goal when it's won at a goal;
and for `flags`, spawns for two or more teams, each with a goal in its own color.

## Play a round

- **Join** by stepping on the arena's portal. Up to 16 can play, and you can't join a round
  that has already started.
- **Start.** A round needs one player per team, or two players for last-one-standing games, or
  one for the rest. Then a 10 second countdown starts. Players are dealt onto teams in turn.
- **Falling** means fainting, being tagged or leaving the zone. In last-one-standing games you're
  out and go back to the portal. Otherwise you go back to your spawn (dropping any flag).
- **Time up.** In last-one-standing games everyone still in wins. In points games the leaders win.
  In goal games nobody does.
- **Leave** any time with `/leave`.

The scoreboard shows the arena, how long is left, team totals and each player's points. A ⚑
marks who carries a flag.

### Flags

Step on another team's goal to pick up their flag, then step on your own goal to score.

### PvP

In arenas with PvP on, left-click hits another player for one heart, once every quarter-second,
from up to 5.5 m away. Both players must be in survival. Outside arenas, and for players whose
game is paused, nobody gets hurt.

## Scripts in arenas

Script blocks inside the arena hear `on_round_start(players)` and `on_round_end(winners)`, and
`add_points` gives or takes points (up to 10 at a time) in points games. See
[Scripting](scripting.md).
