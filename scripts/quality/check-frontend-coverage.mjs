import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../../", import.meta.url));
const scope = JSON.parse(readFileSync(new URL("../../.quality/coverage-scope.json", import.meta.url), "utf8"));
const workspaces = Object.entries(scope.coveredSuites)
  .filter(([, suite]) => suite.testCommand === "pnpm test:frontend:coverage")
  .map(([workspace]) => workspace);
if (!workspaces.length) throw new Error("No frontend coverage suites are configured");
if (process.argv.slice(2).some((argument) => argument !== "--write-baseline"))
  throw new Error("Only --write-baseline is supported");
for (const workspace of workspaces) {
  const result = spawnSync(
    process.execPath,
    [
      "scripts/quality/check-coverage-ratchet.mjs",
      "--root",
      workspace,
      "--summary",
      `../../coverage/frontend/workspaces/${workspace.replaceAll("/", "-")}.json`,
      "--baseline",
      `../../.quality/coverage-${workspace.replaceAll("/", "-")}-baseline.json`,
      "--scope-file",
      ".quality/coverage-scope.json",
      "--scope-key",
      workspace,
      "--new-file-min",
      "80",
      ...process.argv.slice(2),
    ],
    { cwd: root, stdio: "inherit" }
  );
  if (result.error) throw result.error;
  if (result.status !== 0) process.exitCode = 1;
}
