/** A protected column of the world, corners included, as the server sent it. */
export interface ZoneInfo {
  name: string;
  x0: number;
  z0: number;
  x1: number;
  z1: number;
  build: boolean; // whether we may change blocks here
  pvp: boolean; // survival players may hit each other here
}

const area = (z: ZoneInfo) => (z.x1 - z.x0 + 1) * (z.z1 - z.z0 + 1);

/** Protected zones; where they overlap the smallest wins, as on the server. */
export class Zones {
  private list: readonly ZoneInfo[] = [];

  set(list: readonly ZoneInfo[]): void {
    this.list = list;
  }

  /** The zone over block column x, z. */
  at(x: number, z: number): ZoneInfo | undefined {
    let best: ZoneInfo | undefined;
    for (const zone of this.list)
      if (x >= zone.x0 && x <= zone.x1 && z >= zone.z0 && z <= zone.z1)
        if (!best || area(zone) < area(best)) best = zone;
    return best;
  }

  canBuild(x: number, z: number): boolean {
    return this.at(x, z)?.build ?? true;
  }
}

export function zoneGreeting(zone: ZoneInfo): string {
  const lines = [`Welcome to ${zone.name}!`];
  if (!zone.build) lines.push("Only its builders can change blocks here.");
  if (zone.pvp) lines.push("Players can tag each other here.");
  return lines.join(" ");
}
