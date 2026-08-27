import type { Command } from "commander";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema, type CallToolResult, type Tool } from "@modelcontextprotocol/sdk/types.js";
import type { Json, JsonObject, SeoClient } from "../client.js";
import { buildContext } from "../context.js";
import { asObject, asRows } from "../output.js";
import { packageVersion } from "../version.js";

const PROTOCOL_VERSION = "2025-06-18";
const PROJECT_ARGUMENT_NAMES = ["project", "slug", "project_slug"];

export function registerMcp(program: Command): void {
  program
    .command("mcp")
    .description("run a stdio MCP server that proxies the backend's MCP tools")
    .action(async (_options: unknown, command: Command) => {
      const context = buildContext(command);
      const tools = await handshake(context.client);
      const server = new Server({ name: "seo", version: packageVersion() }, { capabilities: { tools: {} } });
      server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools }));
      server.setRequestHandler(CallToolRequestSchema, async (request) => {
        const tool = tools.find((candidate) => candidate.name === request.params.name);
        const args = withDefaultProject(tool, request.params.arguments ?? {}, context.projectSlug);
        try {
          const result = await context.client.mcpRequest("tools/call", { name: request.params.name, arguments: args });
          return result as CallToolResult;
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          return { content: [{ type: "text", text: message }], isError: true } satisfies CallToolResult;
        }
      });
      await server.connect(new StdioServerTransport());
      process.stderr.write(`seo mcp: proxying ${tools.length} tools from ${context.client.baseUrl}\n`);
    });
}

async function handshake(client: SeoClient): Promise<Tool[]> {
  await client.mcpRequest("initialize", {
    protocolVersion: PROTOCOL_VERSION,
    capabilities: {},
    clientInfo: { name: "seo-cli", version: packageVersion() },
  });
  await client.mcpNotify("notifications/initialized");
  const listed = await client.mcpRequest("tools/list");
  const tools = asRows(listed, "tools") as unknown as Tool[];
  if (tools.length === 0) throw new Error(`backend at ${client.baseUrl} returned no MCP tools`);
  return tools;
}

function withDefaultProject(tool: Tool | undefined, args: Record<string, unknown>, project: string | undefined): JsonObject {
  const filled: Record<string, unknown> = { ...args };
  const properties = asObject(tool?.inputSchema.properties) ?? {};
  if (project) {
    for (const name of PROJECT_ARGUMENT_NAMES) {
      if (name in properties && filled[name] === undefined) filled[name] = project;
    }
  }
  return filled as Record<string, Json>;
}
