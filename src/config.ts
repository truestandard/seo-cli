import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

export const DEFAULT_BASE_URL = "http://localhost:3012";

export type StoredConfig = {
  baseUrl?: string;
  token?: string;
  project?: string;
};

export type ResolvedConfig = {
  baseUrl: string;
  token: string | undefined;
  project: string | undefined;
};

type EnvLike = Record<string, string | undefined>;

export function configPath(env: EnvLike = process.env): string {
  return env.SEO_CONFIG_PATH ?? join(homedir(), ".config", "seo", "config.json");
}

export function loadStoredConfig(path: string = configPath()): StoredConfig {
  try {
    const parsed: unknown = JSON.parse(readFileSync(path, "utf8"));
    return isRecord(parsed) ? pickStoredFields(parsed) : {};
  } catch {
    return {};
  }
}

export function saveStoredConfig(config: StoredConfig, path: string = configPath()): void {
  mkdirSync(dirname(path), { recursive: true, mode: 0o700 });
  writeFileSync(path, JSON.stringify(pickStoredFields(config), null, 2) + "\n", { mode: 0o600 });
}

export function updateStoredConfig(patch: StoredConfig, path: string = configPath()): StoredConfig {
  const merged = { ...loadStoredConfig(path), ...patch };
  saveStoredConfig(merged, path);
  return merged;
}

export function resolveConfig(env: EnvLike = process.env, path: string = configPath(env)): ResolvedConfig {
  const stored = loadStoredConfig(path);
  return {
    baseUrl: stripTrailingSlash(env.SEO_BASE_URL || stored.baseUrl || DEFAULT_BASE_URL),
    token: env.SEO_TOKEN || stored.token,
    project: env.SEO_PROJECT || stored.project,
  };
}

export function stripTrailingSlash(url: string): string {
  return url.replace(/\/+$/, "");
}

function pickStoredFields(source: Record<string, unknown>): StoredConfig {
  const config: StoredConfig = {};
  if (typeof source.baseUrl === "string") config.baseUrl = source.baseUrl;
  if (typeof source.token === "string") config.token = source.token;
  if (typeof source.project === "string") config.project = source.project;
  return config;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
