import assert from "node:assert/strict";
import { spawn, execFileSync } from "node:child_process";
import { once } from "node:events";
import { watch } from "node:fs";
import { mkdtemp, mkdir, readFile, realpath, writeFile, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

function fixtureGitEnvironment(environment = process.env) {
  // Hooks export repository-specific settings even when a child changes cwd.
  // Both Git and Turbo must discover the disposable fixture independently.
  return Object.fromEntries(Object.entries(environment).filter(([name]) => !name.startsWith("GIT_")));
}

await test("watch fixtures cannot reinitialize the repository that invoked a Git hook", async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), "vector-hook-fixture-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const parent = path.join(root, "parent");
  const hook = path.join(root, "hook");
  const fixture = path.join(root, "fixture");
  await mkdir(parent);
  await mkdir(fixture);
  // Bootstrap only disposable repositories, independently of the helper under test.
  const cleanEnvironment = { PATH: process.env.PATH, HOME: root };
  const git = (cwd, args, env = cleanEnvironment) => execFileSync("git", args, { cwd, env, encoding: "utf8" }).trim();
  git(parent, ["init", "--quiet"]);
  git(parent, [
    "-c",
    "user.name=Fixture",
    "-c",
    "user.email=fixture@example.invalid",
    "-c",
    "core.hooksPath=/dev/null",
    "commit",
    "--allow-empty",
    "--quiet",
    "-m",
    "fixture",
  ]);
  git(parent, ["-c", "core.hooksPath=/dev/null", "worktree", "add", "--quiet", "--detach", hook]);
  const parentConfig = path.join(parent, ".git/config");
  const originalConfig = await readFile(parentConfig, "utf8");
  const inheritedEnvironment = { ...cleanEnvironment, GIT_DIR: git(hook, ["rev-parse", "--absolute-git-dir"]) };

  git(fixture, ["init", "--quiet"], fixtureGitEnvironment(inheritedEnvironment));

  assert.equal(await readFile(parentConfig, "utf8"), originalConfig);
  assert.equal(git(parent, ["config", "--bool", "core.bare"]), "false");
  assert.equal(git(fixture, ["rev-parse", "--show-toplevel"]), await realpath(fixture));
});

function waitFor(directory, read, expected) {
  return new Promise((resolve, reject) => {
    const watcher = watch(directory, () => {
      void check();
    });
    const deadline = setTimeout(() => finish(new Error(`Timed out waiting for ${JSON.stringify(expected)}`)), 8_000);
    function finish(error) {
      clearTimeout(deadline);
      watcher.close();
      if (error) reject(error);
      else resolve();
    }
    async function check() {
      try {
        if ((await read()) === expected) finish();
      } catch (error) {
        finish(error);
      }
    }
    watcher.on("error", finish);
    void check();
  });
}

await test("development watches dependency rebuilds without starting per-package persistent workers", async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), "vector-watch-"));
  let child;
  let output = "";
  t.after(async () => {
    if (child && child.exitCode === null && child.signalCode === null) {
      const exited = once(child, "exit");
      child.kill("SIGTERM");
      await exited;
    }
    await rm(root, { recursive: true, force: true });
  });
  const config = JSON.parse(await readFile(new URL("../../turbo.json", import.meta.url), "utf8"));
  const manifest = JSON.parse(await readFile(new URL("../../package.json", import.meta.url), "utf8"));
  await mkdir(path.join(root, "packages/shared"), { recursive: true });
  await mkdir(path.join(root, "apps/web"), { recursive: true });
  await writeFile(path.join(root, ".gitignore"), "**/dist\n**/.turbo\n.turbo\nevents.txt\n");
  const fixtureEnvironment = fixtureGitEnvironment();
  execFileSync("git", ["init", "--quiet"], { cwd: root, env: fixtureEnvironment });
  await writeFile(
    path.join(root, "package.json"),
    JSON.stringify({ name: "watch-fixture", private: true, packageManager: manifest.packageManager })
  );
  await writeFile(path.join(root, "pnpm-workspace.yaml"), "packages:\n  - 'apps/*'\n  - 'packages/*'\n");
  await writeFile(
    path.join(root, "pnpm-lock.yaml"),
    "lockfileVersion: '9.0'\nimporters:\n  .: {}\n  packages/shared: {}\n  apps/web:\n    dependencies:\n      '@fixture/shared':\n        specifier: 'workspace:*'\n        version: 'link:../../packages/shared'\n"
  );
  await writeFile(
    path.join(root, "turbo.json"),
    JSON.stringify({ tasks: { dev: config.tasks.dev, "dev:build": config.tasks["dev:build"] } })
  );
  await writeFile(
    path.join(root, "packages/shared/package.json"),
    JSON.stringify({ name: "@fixture/shared", version: "1.0.0", scripts: { "dev:build": "node build.mjs" } })
  );
  await writeFile(path.join(root, "packages/shared/input.txt"), "initial");
  await writeFile(
    path.join(root, "packages/shared/build.mjs"),
    `import {mkdirSync,readFileSync,writeFileSync,appendFileSync} from 'node:fs';
    mkdirSync('dist',{recursive:true});writeFileSync('dist/value.txt',readFileSync('input.txt'));
    appendFileSync('../../events.txt','built\\n');`
  );
  await writeFile(
    path.join(root, "apps/web/package.json"),
    JSON.stringify({
      name: "web",
      version: "1.0.0",
      dependencies: { "@fixture/shared": "workspace:*" },
      scripts: { dev: "node server.mjs" },
    })
  );
  await writeFile(
    path.join(root, "apps/web/server.mjs"),
    `import {readFileSync,appendFileSync} from 'node:fs';
    if(readFileSync('../../packages/shared/dist/value.txt','utf8')!=='initial')process.exit(2);
    appendFileSync('../../events.txt','started\\n');setInterval(()=>{},1000);`
  );
  const turbo = fileURLToPath(new URL("../../node_modules/.bin/turbo", import.meta.url));
  child = spawn(turbo, ["watch", "dev", "--filter=./apps/*", "--concurrency=5", "--ui=stream"], {
    cwd: root,
    env: { ...fixtureEnvironment, TURBO_TELEMETRY_DISABLED: "1" },
    stdio: ["ignore", "pipe", "pipe"],
  });
  child.stdout.on("data", (data) => {
    output += data;
  });
  child.stderr.on("data", (data) => {
    output += data;
  });
  const events = async () => {
    try {
      return await readFile(path.join(root, "events.txt"), "utf8");
    } catch {
      assert.equal(child.exitCode, null, output);
      return "";
    }
  };
  await waitFor(root, events, "built\nstarted\n");
  await writeFile(path.join(root, "packages/shared/input.txt"), "changed");
  await waitFor(root, events, "built\nstarted\nbuilt\n");
  assert.equal(await readFile(path.join(root, "packages/shared/dist/value.txt"), "utf8"), "changed");
  assert.equal(child.exitCode, null, output);
});
