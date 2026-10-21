const FACINGS = ["north", "west", "south", "east"]; // yaw 0 looks toward -z, turning left as yaw grows

/** Where the player stands, as whole block coordinates, and which way they look. */
export function coordsText(at: { x: number; y: number; z: number }, yaw: number): string {
  const quarter = Math.round(yaw / (Math.PI / 2));
  const facing = FACINGS[((quarter % 4) + 4) % 4];
  const [x, y, z] = [at.x, at.y, at.z].map((v) => Math.floor(v) + 0); // + 0 turns -0 into 0
  return `📍 ${x}, ${y}, ${z} · facing ${facing}`;
}
