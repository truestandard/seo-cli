import { stripTrailingSlash } from "./config.js";

export type Json = null | boolean | number | string | Json[] | { [key: string]: Json };
export type JsonObject = { [key: string]: Json };

export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

let fetchImpl: FetchLike = (input, init) => fetch(input, init);

export function setFetch(replacement: FetchLike): void {
  fetchImpl = replacement;
}

export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

const SINCE_DAYS = /^(\d+)d$/;
const SINCE_DATE = /^(\d{4})-(\d{2})-(\d{2})$/;

export function since(value: string, now: Date = new Date()): { query: string; from: Date } {
  const trimmed = value.trim();
  const days = SINCE_DAYS.exec(trimmed);
  if (days) {
    const count = Number(days[1]);
    const from = new Date(now.getTime() - count * 86_400_000);
    return { query: `${count}d`, from };
  }
  const date = SINCE_DATE.exec(trimmed);
  if (date) {
    const from = new Date(Date.UTC(Number(date[1]), Number(date[2]) - 1, Number(date[3])));
    if (Number.isNaN(from.getTime()) || from.toISOString().slice(0, 10) !== trimmed) {
      throw new Error(`invalid date in --since: ${value}`);
    }
    return { query: trimmed, from };
  }
  throw new Error(`invalid --since value "${value}": use 7d, 30d, or YYYY-MM-DD`);
}

type Query = Record<string, string | number | boolean | undefined>;

export type ProjectAttributes = {
  slug?: string;
  name?: string;
  domain?: string;
  repo_path?: string;
  location_code?: number;
  language_code?: string;
};

export type ContextPatch = {
  sections?: Record<string, string>;
  add_competitors?: Array<{ domain: string; name?: string; notes?: string }>;
  add_key_pages?: Array<{ path: string; role?: string; topic?: string }>;
  research_log?: { kind: string; summary: string; inputs?: JsonObject };
};

export type KeywordInput = {
  keyword: string;
  track?: string;
  target_path?: string;
  set_name?: string;
  locked?: boolean;
  volume?: number;
  kd?: number;
};

export type PromptInput = { text: string; set_name?: string; locked?: boolean };

export type ExperimentInput = {
  change: string;
  page?: string;
  hypothesis?: string;
  keyword?: string;
  shipped_on?: string;
};

export type JsonRpcResponse = {
  jsonrpc: "2.0";
  id?: number | string | null;
  result?: Json;
  error?: { code: number; message: string; data?: Json };
};

export class SeoClient {
  readonly baseUrl: string;
  private readonly token: string | undefined;
  private mcpSessionId: string | undefined;
  private nextRpcId = 1;

  constructor(options: { baseUrl: string; token?: string | undefined }) {
    this.baseUrl = stripTrailingSlash(options.baseUrl);
    this.token = options.token;
  }

  whoami() {
    return this.get("/api/v1/whoami");
  }

  listProjects() {
    return this.get("/api/v1/projects");
  }

  createProject(attributes: ProjectAttributes) {
    return this.post("/api/v1/projects", attributes);
  }

  getProject(slug: string) {
    return this.get(`/api/v1/projects/${encode(slug)}`);
  }

  updateProject(slug: string, attributes: ProjectAttributes) {
    return this.patch(`/api/v1/projects/${encode(slug)}`, attributes);
  }

  getContext(slug: string) {
    return this.get(`/api/v1/projects/${encode(slug)}/context`);
  }

  updateContext(slug: string, patch: ContextPatch) {
    return this.patch(`/api/v1/projects/${encode(slug)}/context`, patch);
  }

  listKeywords(slug: string, query: { set?: string | undefined } = {}) {
    return this.get(`/api/v1/projects/${encode(slug)}/keywords`, query);
  }

  addKeywords(slug: string, keywords: KeywordInput[]) {
    return this.post(`/api/v1/projects/${encode(slug)}/keywords`, { keywords });
  }

  updateKeyword(slug: string, id: string | number, attributes: Partial<KeywordInput>) {
    return this.patch(`/api/v1/projects/${encode(slug)}/keywords/${encode(String(id))}`, attributes);
  }

  deleteKeyword(slug: string, id: string | number) {
    return this.delete(`/api/v1/projects/${encode(slug)}/keywords/${encode(String(id))}`);
  }

  research(
    slug: string,
    body: { keywords?: string[]; seeds?: string[]; limit?: number; max_kd?: number; dry_run?: boolean },
  ) {
    return this.post(`/api/v1/projects/${encode(slug)}/research`, body);
  }

  serp(slug: string, body: { keyword: string; depth?: number; dry_run?: boolean }) {
    return this.post(`/api/v1/projects/${encode(slug)}/serp`, body);
  }

  createRankRun(slug: string, body: { mode: "scheduled" | "live"; set_name?: string; dry_run?: boolean }) {
    return this.post(`/api/v1/projects/${encode(slug)}/rank_runs`, body);
  }

  getRankRun(slug: string, id: string | number) {
    return this.get(`/api/v1/projects/${encode(slug)}/rank_runs/${encode(String(id))}`);
  }

  ranks(slug: string, query: { since?: string; set?: string | undefined }) {
    return this.get(`/api/v1/projects/${encode(slug)}/ranks`, query);
  }

  listAiPrompts(slug: string, query: { set?: string | undefined } = {}) {
    return this.get(`/api/v1/projects/${encode(slug)}/ai_prompts`, query);
  }

  addAiPrompts(slug: string, prompts: PromptInput[]) {
    return this.post(`/api/v1/projects/${encode(slug)}/ai_prompts`, { prompts });
  }

  createAiScan(slug: string, body: { set_name?: string; runs_per_cell?: number; dry_run?: boolean }) {
    return this.post(`/api/v1/projects/${encode(slug)}/ai_scans`, body);
  }

  getAiScan(slug: string, id: string | number) {
    return this.get(`/api/v1/projects/${encode(slug)}/ai_scans/${encode(String(id))}`);
  }

  aiVisibility(slug: string, query: { since?: string; set?: string | undefined }) {
    return this.get(`/api/v1/projects/${encode(slug)}/ai_visibility`, query);
  }

  floor(slug: string, body: { keywords: string[]; dry_run?: boolean }) {
    return this.post(`/api/v1/projects/${encode(slug)}/floor`, body);
  }

  backlinks(slug: string, body: { domain?: string; dry_run?: boolean }) {
    return this.post(`/api/v1/projects/${encode(slug)}/backlinks`, body);
  }

  gscImport(slug: string, body: { dimension: "query" | "page"; range_start: string; range_end: string; csv: string }) {
    return this.post(`/api/v1/projects/${encode(slug)}/gsc_import`, body);
  }

  strikingDistance(slug: string) {
    return this.get(`/api/v1/projects/${encode(slug)}/gsc/striking_distance`);
  }

  ship(slug: string, body: { url: string; keyword?: string; track?: string; note?: string }) {
    return this.post(`/api/v1/projects/${encode(slug)}/ship`, body);
  }

  listExperiments(slug: string) {
    return this.get(`/api/v1/projects/${encode(slug)}/experiments`);
  }

  createExperiment(slug: string, body: ExperimentInput) {
    return this.post(`/api/v1/projects/${encode(slug)}/experiments`, body);
  }

  updateExperiment(slug: string, id: string | number, body: { outcome: string }) {
    return this.patch(`/api/v1/projects/${encode(slug)}/experiments/${encode(String(id))}`, body);
  }

  researchLog(slug: string, query: { kind?: string | undefined; days?: number | undefined }) {
    return this.get(`/api/v1/projects/${encode(slug)}/research_log`, query);
  }

  scoreboard(slug: string) {
    return this.get(`/api/v1/projects/${encode(slug)}/scoreboard`);
  }

  spend(query: { since?: string }) {
    return this.get("/api/v1/spend", query);
  }

  async mcpRequest(method: string, params: JsonObject = {}): Promise<Json> {
    const id = this.nextRpcId++;
    const response = await this.rawMcpPost({ jsonrpc: "2.0", id, method, params });
    const sessionId = response.headers.get("mcp-session-id");
    if (sessionId) this.mcpSessionId = sessionId;
    const message = await this.readJsonRpc(response, id);
    if (message.error) {
      throw new ApiError(message.error.message, `mcp_${message.error.code}`, response.status);
    }
    return message.result ?? null;
  }

  async mcpNotify(method: string, params: JsonObject = {}): Promise<void> {
    const response = await this.rawMcpPost({ jsonrpc: "2.0", method, params });
    await response.text().catch(() => "");
  }

  private async rawMcpPost(payload: JsonObject): Promise<Response> {
    const headers: Record<string, string> = {
      ...this.authHeaders(),
      "content-type": "application/json",
      accept: "application/json, text/event-stream",
    };
    if (this.mcpSessionId) headers["mcp-session-id"] = this.mcpSessionId;
    const response = await fetchImpl(`${this.baseUrl}/mcp`, {
      method: "POST",
      headers,
      body: JSON.stringify(payload),
    });
    if (!response.ok) await this.throwApiError(response);
    return response;
  }

  private async readJsonRpc(response: Response, id: number): Promise<JsonRpcResponse> {
    const contentType = response.headers.get("content-type") ?? "";
    const text = await response.text();
    if (contentType.includes("text/event-stream")) {
      const messages = parseSseJsonMessages(text);
      const match = messages.find((message) => message.id === id) ?? messages.at(-1);
      if (!match) throw new ApiError("empty MCP event stream", "mcp_empty_response", response.status);
      return match;
    }
    if (text.trim() === "") throw new ApiError("empty MCP response", "mcp_empty_response", response.status);
    const parsed: unknown = JSON.parse(text);
    if (Array.isArray(parsed)) {
      const match = (parsed as JsonRpcResponse[]).find((message) => message.id === id);
      if (match) return match;
    }
    return parsed as JsonRpcResponse;
  }

  private get(path: string, query: Query = {}) {
    return this.request("GET", path, { query });
  }

  private post(path: string, body: unknown) {
    return this.request("POST", path, { body });
  }

  private patch(path: string, body: unknown) {
    return this.request("PATCH", path, { body });
  }

  private delete(path: string) {
    return this.request("DELETE", path, {});
  }

  private async request(method: string, path: string, options: { query?: Query; body?: unknown }): Promise<Json> {
    const url = this.buildUrl(path, options.query);
    const headers: Record<string, string> = { ...this.authHeaders(), accept: "application/json" };
    const init: RequestInit = { method, headers };
    if (options.body !== undefined) {
      headers["content-type"] = "application/json";
      init.body = JSON.stringify(options.body);
    }
    let response: Response;
    try {
      response = await fetchImpl(url, init);
    } catch (cause) {
      const detail = cause instanceof Error ? cause.message : String(cause);
      throw new ApiError(`could not reach ${this.baseUrl}: ${detail}`, "network_error", 0);
    }
    if (!response.ok) await this.throwApiError(response);
    const text = await response.text();
    if (text.trim() === "") return null;
    return JSON.parse(text) as Json;
  }

  private buildUrl(path: string, query: Query = {}): string {
    const url = new URL(`${this.baseUrl}${path}`);
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== "") url.searchParams.set(key, String(value));
    }
    return url.toString();
  }

  private authHeaders(): Record<string, string> {
    return this.token ? { authorization: `Bearer ${this.token}` } : {};
  }

  private async throwApiError(response: Response): Promise<never> {
    const text = await response.text().catch(() => "");
    let message = text.trim() || response.statusText || `HTTP ${response.status}`;
    let code = `http_${response.status}`;
    try {
      const parsed: unknown = JSON.parse(text);
      if (typeof parsed === "object" && parsed !== null) {
        const body = parsed as { error?: unknown; code?: unknown };
        if (typeof body.error === "string") message = body.error;
        if (typeof body.code === "string") code = body.code;
      }
    } catch {
      if (response.status === 401 && !this.token) message = "no API token configured: run `seo login`";
    }
    throw new ApiError(message, code, response.status);
  }
}

export function parseSseJsonMessages(text: string): JsonRpcResponse[] {
  const messages: JsonRpcResponse[] = [];
  for (const block of text.split(/\n\n+/)) {
    const data = block
      .split("\n")
      .filter((line) => line.startsWith("data:"))
      .map((line) => line.slice(5).trimStart())
      .join("\n");
    if (data.trim() === "") continue;
    try {
      const parsed: unknown = JSON.parse(data);
      if (Array.isArray(parsed)) messages.push(...(parsed as JsonRpcResponse[]));
      else messages.push(parsed as JsonRpcResponse);
    } catch {
      continue;
    }
  }
  return messages;
}

function encode(segment: string): string {
  return encodeURIComponent(segment);
}
