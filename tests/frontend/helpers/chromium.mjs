import { access, mkdtemp, readFile, rm } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";

const delay = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

async function findChrome() {
  const configuredPath = process.env.PLANE_CHROME_PATH;
  const candidates = [
    configuredPath,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/google-chrome-stable",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
  ].filter(Boolean);

  const availableCandidates = await Promise.all(
    candidates.map(async (candidate) => {
      try {
        await access(candidate, fsConstants.X_OK);
        return candidate;
      } catch {
        return undefined;
      }
    })
  );
  const chromePath = availableCandidates.find(Boolean);
  if (chromePath) return chromePath;
  throw new Error("Chrome was not found. Set PLANE_CHROME_PATH to an executable Chrome or Chromium binary.");
}

async function waitForDevTools(profileDirectory, chromeProcess) {
  const activePortPath = path.join(profileDirectory, "DevToolsActivePort");
  const deadline = Date.now() + 20_000;
  const readPort = async () => {
    if (Date.now() >= deadline) throw new Error("Chrome did not publish its DevTools port within 20 seconds");
    if (chromeProcess.exitCode !== null || chromeProcess.signalCode !== null) {
      throw new Error(`Chrome exited with status ${chromeProcess.exitCode ?? chromeProcess.signalCode}`);
    }
    try {
      const [port] = (await readFile(activePortPath, "utf8")).trim().split("\n");
      if (port) return Number(port);
    } catch {
      // Chrome writes the file after its debugging server starts.
    }
    await delay(50);
    return readPort();
  };
  return readPort();
}

export async function openTarget(debugPort) {
  const response = await fetch(`http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent("about:blank")}`, {
    method: "PUT",
  });
  if (!response.ok) throw new Error(`DevTools target creation failed with status ${response.status}`);
  return response.json();
}

export async function closeTarget(debugPort, targetId) {
  await fetch(`http://127.0.0.1:${debugPort}/json/close/${targetId}`);
}

async function stopChrome(chromeProcess) {
  if (chromeProcess.exitCode !== null || chromeProcess.signalCode !== null) return;
  chromeProcess.kill("SIGTERM");
  const timeout = setTimeout(() => chromeProcess.kill("SIGKILL"), 5_000);
  try {
    await new Promise((resolve) => chromeProcess.once("exit", resolve));
  } finally {
    clearTimeout(timeout);
  }
}

export async function withChromium(run) {
  const chromePath = await findChrome();
  const profileDirectory = await mkdtemp(path.join(os.tmpdir(), "vector-browser-"));
  const chromeProcess = spawn(
    chromePath,
    [
      "--headless=new",
      "--disable-extensions",
      "--disable-component-extensions-with-background-pages",
      "--disable-default-apps",
      "--disable-sync",
      "--disable-background-networking",
      "--no-first-run",
      "--no-default-browser-check",
      "--remote-debugging-port=0",
      `--user-data-dir=${profileDirectory}`,
      "about:blank",
    ],
    { stdio: "ignore" }
  );

  try {
    const debugPort = await waitForDevTools(profileDirectory, chromeProcess);
    return await run(debugPort);
  } finally {
    await stopChrome(chromeProcess);
    await rm(profileDirectory, { recursive: true, force: true });
  }
}
