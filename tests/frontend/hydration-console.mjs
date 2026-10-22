import { access, mkdtemp, readFile, rm } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";

const DEFAULT_CASES = [
  { app: "web", name: "web/light", colorScheme: "light", url: "http://localhost:3000/" },
  { app: "web", name: "web/dark", colorScheme: "dark", url: "http://localhost:3000/" },
  { app: "admin", name: "admin/light", colorScheme: "light", url: "http://localhost:3001/god-mode/" },
  { app: "admin", name: "admin/dark", colorScheme: "dark", url: "http://localhost:3001/god-mode/" },
];

const HYDRATION_PATTERNS = [
  /hydration failed/i,
  /hydration error/i,
  /error while hydrating/i,
  /while hydrating this suspense boundary/i,
  /server rendered html didn't match/i,
  /server-rendered html didn't match/i,
  /server rendered.*did not match/i,
  /did not match.*client/i,
  /text content does not match/i,
  /expected server html/i,
  /tree hydrated but some attributes/i,
];

const delay = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

function isHydrationMessage(message) {
  return HYDRATION_PATTERNS.some((pattern) => pattern.test(message));
}

function makeConsoleMessage(source, level, text) {
  return { source, level, text: text.replaceAll(/\s+/g, " ").trim() };
}

function remoteValueText(value) {
  if (Object.hasOwn(value, "value")) {
    return typeof value.value === "string" ? value.value : JSON.stringify(value.value);
  }
  return value.unserializableValue ?? value.description ?? value.type;
}

function runClassifierSelfTest() {
  const positiveMessages = [
    "Hydration failed because the server rendered HTML didn't match the client.",
    "A tree hydrated but some attributes of the server rendered HTML didn't match the client properties.",
    "Text content does not match server-rendered HTML.",
  ];
  const negativeMessages = ["Failed to load instance settings", "The development server is ready"];

  if (!positiveMessages.every(isHydrationMessage) || negativeMessages.some(isHydrationMessage)) {
    throw new Error("hydration console classifier self-test failed");
  }
}

class CdpSession {
  constructor(webSocketUrl) {
    this.webSocketUrl = webSocketUrl;
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = new Map();
  }

  async connect() {
    this.socket = new WebSocket(this.webSocketUrl);
    await new Promise((resolve, reject) => {
      this.socket.addEventListener("open", resolve, { once: true });
      this.socket.addEventListener("error", reject, { once: true });
    });
    this.socket.addEventListener("message", (event) => this.onMessage(event));
  }

  on(method, listener) {
    const listeners = this.listeners.get(method) ?? [];
    listeners.push(listener);
    this.listeners.set(method, listeners);
  }

  async onMessage(event) {
    const data = typeof event.data === "string" ? event.data : await event.data.text();
    const message = JSON.parse(data);
    if (message.id !== undefined) {
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      if (message.error) pending.reject(new Error(`${message.error.message} (${message.error.code})`));
      else pending.resolve(message.result);
      return;
    }
    for (const listener of this.listeners.get(message.method) ?? []) listener(message.params);
  }

  send(method, params = {}) {
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.socket.send(JSON.stringify({ id, method, params }));
    });
  }

  waitFor(method, timeoutMilliseconds) {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error(`timed out waiting for ${method}`)), timeoutMilliseconds);
      this.on(method, (params) => {
        clearTimeout(timeout);
        resolve(params);
      });
    });
  }

  close() {
    this.socket?.close();
  }
}

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
    if (chromeProcess.exitCode !== null) throw new Error(`Chrome exited with status ${chromeProcess.exitCode}`);
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

async function openTarget(debugPort) {
  const response = await fetch(`http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent("about:blank")}`, {
    method: "PUT",
  });
  if (!response.ok) throw new Error(`DevTools target creation failed with status ${response.status}`);
  return response.json();
}

async function closeTarget(debugPort, targetId) {
  await fetch(`http://127.0.0.1:${debugPort}/json/close/${targetId}`);
}

async function inspectFreshLoad(debugPort, testCase) {
  const target = await openTarget(debugPort);
  const session = new CdpSession(target.webSocketDebuggerUrl);
  const messages = [];
  let documentStatus;

  try {
    await session.connect();
    session.on("Runtime.consoleAPICalled", ({ type, args }) => {
      messages.push(makeConsoleMessage("console", type, args.map(remoteValueText).join(" ")));
    });
    session.on("Runtime.exceptionThrown", ({ exceptionDetails }) => {
      const text = exceptionDetails.exception?.description ?? exceptionDetails.text;
      messages.push(makeConsoleMessage("exception", "error", text));
    });
    session.on("Log.entryAdded", ({ entry }) => {
      messages.push(makeConsoleMessage(entry.source, entry.level, entry.text));
    });
    session.on("Network.responseReceived", ({ type, response }) => {
      if (type === "Document" && response.url.startsWith(testCase.url)) documentStatus = response.status;
    });

    await Promise.all([
      session.send("Runtime.enable"),
      session.send("Log.enable"),
      session.send("Network.enable"),
      session.send("Page.enable"),
      session.send("Emulation.setEmulatedMedia", {
        media: "screen",
        features: [{ name: "prefers-color-scheme", value: testCase.colorScheme }],
      }),
    ]);

    const loaded = session.waitFor("Page.loadEventFired", 30_000);
    const navigation = await session.send("Page.navigate", { url: testCase.url });
    if (navigation.errorText) throw new Error(`${testCase.name} navigation failed: ${navigation.errorText}`);
    await loaded;
    await delay(Number(process.env.PLANE_HYDRATION_SETTLE_MS ?? 5_000));

    const { result, exceptionDetails } = await session.send("Runtime.evaluate", {
      expression: `(() => ({
        readyState: document.readyState,
        routeModuleCount: Object.keys(window.__reactRouterRouteModules || {}).length,
        hasRouterContext: Boolean(window.__reactRouterContext),
        bodyElementCount: document.body?.children.length || 0,
        bodyTextLength: document.body?.innerText.length || 0,
        href: window.location.href
      }))()`,
      returnByValue: true,
    });
    if (exceptionDetails) throw new Error(`${testCase.name} bootstrap inspection failed: ${exceptionDetails.text}`);

    const state = result.value;
    const hydrationMessages = messages.filter((message) => isHydrationMessage(message.text));
    const errorMessages = messages.filter((message) => message.level === "error");

    if (documentStatus !== 200)
      throw new Error(`${testCase.name} document returned status ${documentStatus ?? "unknown"}`);
    if (state.readyState !== "complete" || !state.hasRouterContext || state.routeModuleCount === 0) {
      throw new Error(`${testCase.name} client bootstrap did not complete: ${JSON.stringify(state)}`);
    }
    return { ...testCase, state, messageCount: messages.length, errorMessages, hydrationMessages };
  } finally {
    session.close();
    await closeTarget(debugPort, target.id);
  }
}

async function stopChrome(chromeProcess) {
  if (chromeProcess.exitCode !== null) return;
  chromeProcess.kill("SIGTERM");
  await Promise.race([
    new Promise((resolve) => chromeProcess.once("exit", resolve)),
    delay(5_000).then(() => chromeProcess.kill("SIGKILL")),
  ]);
}

async function main() {
  runClassifierSelfTest();
  const chromePath = await findChrome();
  const profileDirectory = await mkdtemp(path.join(os.tmpdir(), "plane-hydration-console-"));
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
    const testCases = DEFAULT_CASES.map(({ app, name, colorScheme, url }) => ({
      app,
      name,
      colorScheme,
      url: process.env[`PLANE_${app.toUpperCase()}_URL`] ?? url,
    }));
    const results = await testCases.reduce(async (resultsPromise, testCase) => {
      const completedResults = await resultsPromise;
      completedResults.push(await inspectFreshLoad(debugPort, testCase));
      return completedResults;
    }, Promise.resolve([]));

    for (const result of results) {
      const hydrationResult = result.hydrationMessages.length === 0 ? "clean hydration console" : "hydration failure";
      console.log(
        `${result.name}: ${hydrationResult}; status=200 routeModules=${result.state.routeModuleCount} consoleMessages=${result.messageCount} consoleErrors=${result.errorMessages.length} finalUrl=${result.state.href}`
      );
      if (process.env.PLANE_HYDRATION_VERBOSE_ERRORS === "1") {
        for (const error of result.errorMessages) console.log(`${result.name}: console error: ${error.text}`);
      }
    }

    const hydrationFailures = results.filter((result) => result.hydrationMessages.length > 0);
    if (hydrationFailures.length > 0) {
      const details = hydrationFailures
        .map((result) => {
          const excerpts = result.hydrationMessages.slice(0, 3).map((message) => `  - ${message.text}`);
          return `${result.name}: ${result.hydrationMessages.length} hydration console messages\n${excerpts.join("\n")}`;
        })
        .join("\n");
      throw new Error(`React hydration errors detected:\n${details}`);
    }
  } finally {
    await stopChrome(chromeProcess);
    await rm(profileDirectory, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.stack : error);
  process.exitCode = 1;
});
