import { mkdtempSync, readFileSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { DEFAULT_BASE_URL, configPath, loadStoredConfig, resolveConfig, saveStoredConfig, updateStoredConfig } from "../src/config.js";

let directory: string;
let path: string;

beforeEach(() => {
  directory = mkdtempSync(join(tmpdir(), "seo-cli-config-"));
  path = join(directory, "nested", "config.json");
});

afterEach(() => {
  rmSync(directory, { recursive: true, force: true });
});

describe("config", () => {
  it("returns an empty config when the file is missing", () => {
    expect(loadStoredConfig(path)).toEqual({});
  });

  it("saves and reloads baseUrl, token, and project with owner-only permissions", () => {
    saveStoredConfig({ baseUrl: "http://localhost:3012", token: "seo_abc", project: "demo" }, path);
    expect(loadStoredConfig(path)).toEqual({ baseUrl: "http://localhost:3012", token: "seo_abc", project: "demo" });
    expect(statSync(path).mode & 0o777).toBe(0o600);
    expect(JSON.parse(readFileSync(path, "utf8"))).toEqual({ baseUrl: "http://localhost:3012", token: "seo_abc", project: "demo" });
  });

  it("merges patches without dropping existing keys", () => {
    saveStoredConfig({ baseUrl: "http://localhost:3012", token: "seo_abc" }, path);
    expect(updateStoredConfig({ project: "demo" }, path)).toEqual({ baseUrl: "http://localhost:3012", token: "seo_abc", project: "demo" });
  });

  it("ignores unknown and non-string fields", () => {
    saveStoredConfig({ baseUrl: "http://x", token: 5 as unknown as string, extra: true } as never, path);
    expect(loadStoredConfig(path)).toEqual({ baseUrl: "http://x" });
  });

  it("falls back to the default base URL", () => {
    expect(resolveConfig({}, path)).toEqual({ baseUrl: DEFAULT_BASE_URL, token: undefined, project: undefined });
  });

  it("lets environment variables override the stored file and strips trailing slashes", () => {
    saveStoredConfig({ baseUrl: "http://stored:1", token: "seo_stored", project: "stored" }, path);
    const resolved = resolveConfig({ SEO_BASE_URL: "http://env:2/", SEO_TOKEN: "seo_env", SEO_PROJECT: "env" }, path);
    expect(resolved).toEqual({ baseUrl: "http://env:2", token: "seo_env", project: "env" });
  });

  it("uses only the overrides that are set", () => {
    saveStoredConfig({ baseUrl: "http://stored:1", token: "seo_stored", project: "stored" }, path);
    expect(resolveConfig({ SEO_PROJECT: "env" }, path)).toEqual({ baseUrl: "http://stored:1", token: "seo_stored", project: "env" });
  });

  it("honours SEO_CONFIG_PATH for the config location", () => {
    expect(configPath({ SEO_CONFIG_PATH: "/tmp/custom.json" })).toBe("/tmp/custom.json");
    expect(configPath({})).toMatch(/\.config\/seo\/config\.json$/);
  });
});
