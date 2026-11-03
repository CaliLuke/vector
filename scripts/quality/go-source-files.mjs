#!/usr/bin/env node
import fs from "node:fs";

// find owns the directory boundary. Filter its NUL-delimited inventory in one
// process instead of launching awk once per file. Keep the first-ten-line rule.
const files = fs.readFileSync(0, "utf8").split("\0").filter(Boolean);
const owned = files.filter((filename) => {
  const lines = fs.readFileSync(filename, "utf8").split("\n").slice(0, 10);
  // A strict end assertion preserves awk's treatment of CRLF marker lines.
  return !lines.some((line) => /^\/\/ Code generated .* DO NOT EDIT\.(?![\s\S])/.test(line));
});
if (owned.length) process.stdout.write(`${owned.join("\n")}\n`);
