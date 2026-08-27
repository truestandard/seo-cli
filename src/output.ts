import type { Json } from "./client.js";

export type Row = Record<string, unknown>;

let write: (text: string) => void = (text) => {
  process.stdout.write(text);
};

export function setWriter(replacement: (text: string) => void): void {
  write = replacement;
}

export function printJson(value: unknown): void {
  write(JSON.stringify(value, null, 2) + "\n");
}

export function printLine(text: string): void {
  write(text + "\n");
}

export function printBlock(text: string): void {
  if (text === "") return;
  write(text.endsWith("\n") ? text : text + "\n");
}

export function formatCell(value: unknown): string {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "boolean") return value ? "yes" : "no";
  if (typeof value === "number") return Number.isInteger(value) ? String(value) : trimNumber(value);
  if (typeof value === "string") return value;
  if (Array.isArray(value)) return value.map(formatCell).join(",");
  return JSON.stringify(value);
}

function trimNumber(value: number): string {
  const fixed = value.toFixed(4);
  return fixed.replace(/\.?0+$/, "");
}

const NUMERIC_TEXT = /^[+-]?\d+(\.\d+)?$/;

function isNumeric(value: unknown): boolean {
  return typeof value === "number" || (typeof value === "string" && NUMERIC_TEXT.test(value));
}

export function table(rows: Row[], columns?: string[], headers?: Record<string, string>): string {
  if (rows.length === 0) return "";
  const keys = columns ?? uniqueKeys(rows);
  const cells = rows.map((row) => keys.map((key) => formatCell(row[key])));
  const titles = keys.map((key) => headers?.[key] ?? key);
  const widths = keys.map((_, column) =>
    Math.max(titles[column]?.length ?? 0, ...cells.map((line) => line[column]?.length ?? 0)),
  );
  const rightAligned = keys.map((key) => rows.some((row) => isNumeric(row[key])));
  const renderLine = (line: string[]) =>
    line
      .map((cell, column) => {
        const width = widths[column] ?? 0;
        return rightAligned[column] ? cell.padStart(width) : cell.padEnd(width);
      })
      .join("  ")
      .replace(/\s+$/, "");
  return [renderLine(titles), ...cells.map(renderLine)].join("\n");
}

function uniqueKeys(rows: Row[]): string[] {
  const keys: string[] = [];
  for (const row of rows) {
    for (const key of Object.keys(row)) {
      if (!keys.includes(key)) keys.push(key);
    }
  }
  return keys;
}

export function summaryLine(fields: Record<string, unknown>): string {
  return Object.entries(fields)
    .filter(([, value]) => value !== undefined)
    .map(([key, value]) => `${key}=${formatCell(value)}`)
    .join(" ");
}

export function keyValueBlock(fields: Record<string, unknown>): string {
  const entries = Object.entries(fields).filter(([, value]) => value !== undefined);
  const width = Math.max(0, ...entries.map(([key]) => key.length));
  return entries.map(([key, value]) => `${key.padEnd(width)}  ${formatCell(value)}`).join("\n");
}

export function money(value: unknown): string {
  const amount = typeof value === "number" ? value : Number(value);
  if (Number.isNaN(amount)) return formatCell(value);
  return `$${amount.toFixed(4).replace(/0+$/, "").replace(/\.$/, ".0")}`;
}

export function estimateLine(data: Json): string {
  const estimate = pickObject(data, "estimate") ?? asObject(data) ?? {};
  const { cost, ...rest } = estimate;
  const detail = summaryLine(rest);
  return `estimate ${money(cost)}${detail ? ` (${detail})` : ""} — dry run, nothing spent`;
}

export function asObject(value: unknown): Row | undefined {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? (value as Row) : undefined;
}

export function pickObject(value: unknown, key: string): Row | undefined {
  return asObject(asObject(value)?.[key]);
}

export function asRows(value: unknown, ...candidateKeys: string[]): Row[] {
  if (Array.isArray(value)) return value.filter((item) => asObject(item) !== undefined) as Row[];
  const object = asObject(value);
  if (!object) return [];
  for (const key of [...candidateKeys, "rows", "results", "data", "items"]) {
    const nested = object[key];
    if (Array.isArray(nested)) return nested.filter((item) => asObject(item) !== undefined) as Row[];
  }
  return [];
}

export function isDryRunEstimate(data: Json): boolean {
  return pickObject(data, "estimate") !== undefined;
}
