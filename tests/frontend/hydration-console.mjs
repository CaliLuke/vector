import { CdpSession } from "./helpers/cdp-session.mjs";
import { closeTarget, openTarget, withChromium } from "./helpers/chromium.mjs";

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

async function main() {
  runClassifierSelfTest();
  await withChromium(async (debugPort) => {
    const testCases = DEFAULT_CASES.map(({ app, name, colorScheme, url }) => ({
      app,
      name,
      colorScheme,
      url: process.env[`PLANE_${app.toUpperCase()}_URL`] ?? url,
    }));
    const results = await testCases.reduce(
      /** @param {Promise<Awaited<ReturnType<typeof inspectFreshLoad>>[]>} resultsPromise */
      async (resultsPromise, testCase) => {
        const completedResults = await resultsPromise;
        completedResults.push(await inspectFreshLoad(debugPort, testCase));
        return completedResults;
      },
      Promise.resolve([])
    );

    for (const result of results) {
      const hydrationResult = result.hydrationMessages.length === 0 ? "clean hydration console" : "hydration failure";
      process.stdout.write(
        `${result.name}: ${hydrationResult}; status=200 routeModules=${result.state.routeModuleCount} consoleMessages=${result.messageCount} consoleErrors=${result.errorMessages.length} finalUrl=${result.state.href}\n`
      );
      if (process.env.PLANE_HYDRATION_VERBOSE_ERRORS === "1") {
        for (const error of result.errorMessages)
          process.stdout.write(`${result.name}: console error: ${error.text}\n`);
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
  });
}

main().catch((error) => {
  console.error(error instanceof Error ? error.stack : error);
  process.exitCode = 1;
});
