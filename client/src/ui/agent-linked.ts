import type { AgentLinked } from "../pb/scabbage/v1/net_pb";
import { copyText } from "./admin";
import { el } from "./dom";

/** How to point an agent at this server with a token. */
export function agentSetup(origin: string, token: string) {
  const mcpUrl = `${origin}/mcp`;
  return {
    mcpUrl,
    claude: `claude mcp add --transport http scabbage ${mcpUrl} --header "Authorization: Bearer ${token}"`,
    guide: `${origin}/api/agent/v1/`,
  };
}

/** Shows a new agent token once, with copyable setup for Claude Code and other agents. */
export function agentLinkedView(linked: AgentLinked, origin: string): HTMLElement {
  const setup = agentSetup(origin, linked.token);
  return el(
    "section",
    { class: "invite" },
    el("h2", {}, `Agent linked: ${linked.label}`),
    el(
      "p",
      {},
      "This token lets an agent fly and build as you. It's shown only now, so copy it and keep it secret. ",
      `/agent revoke ${linked.label} turns it off.`,
    ),
    el("p", {}, "Claude Code:"),
    copyRow(setup.claude, "Copy command"),
    el("p", {}, "Other MCP clients: ", el("strong", {}, setup.mcpUrl), " with the header:"),
    copyRow(`Authorization: Bearer ${linked.token}`, "Copy header"),
    el(
      "p",
      { class: "muted" },
      `Without MCP, an agent can use plain HTTP: tell it to start by reading ${setup.guide}. `,
      "Keep this tab open and in front while the agent works.",
    ),
  );
}

function copyRow(text: string, label: string): HTMLElement {
  const input = el("input", { readonly: true, value: text, onfocus: () => input.select() });
  const copy = el("button", { onclick: () => void copyText(input, copy) }, label);
  return el("div", { class: "row" }, input, copy);
}
