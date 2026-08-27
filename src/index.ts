#!/usr/bin/env node
import { ApiError } from "./client.js";
import { buildProgram } from "./program.js";

try {
  await buildProgram().parseAsync(process.argv);
} catch (error) {
  if (error instanceof ApiError) {
    process.stderr.write(`error: ${error.message} (${error.code})\n`);
  } else if (error instanceof Error) {
    process.stderr.write(`error: ${error.message}\n`);
  } else {
    process.stderr.write(`error: ${String(error)}\n`);
  }
  process.exit(1);
}
