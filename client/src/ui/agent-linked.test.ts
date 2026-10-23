import { describe, expect, it } from "vitest";
import { agentSetup } from "./agent-linked";

describe("agentSetup", () => {
  it("gives ready-to-paste MCP and JSON setups for this server", () => {
    const s = agentSetup("https://scab.example", "scab_abc");
    expect(s.claude).toBe(
      'claude mcp add --transport http scabbage https://scab.example/mcp --header "Authorization: Bearer scab_abc"',
    );
    expect(s.mcpUrl).toBe("https://scab.example/mcp");
    expect(s.guide).toBe("https://scab.example/api/agent/v1/");
  });
});
