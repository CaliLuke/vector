import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { CdpSession } from "../helpers/cdp-session.mjs";
import { closeTarget, openTarget, withChromium } from "../helpers/chromium.mjs";

const entry = fileURLToPath(new URL("../../../apps/web/tests/browser/status-filter.tsx", import.meta.url));
const bundle = await Bun.build({
  entrypoints: [entry],
  target: "browser",
  define: { "process.env": JSON.stringify({ NODE_ENV: "production" }) },
});
if (!bundle.success) throw new AggregateError(bundle.logs, "Unable to build the status filter fixture");
const browserScript = await bundle.outputs[0].text();
const requests = [];
const preferencesGate = Promise.withResolvers();
const server = Bun.serve({
  hostname: "127.0.0.1",
  port: 0,
  async fetch(request) {
    switch (new URL(request.url).pathname) {
      case "/":
        return new Response('<!doctype html><meta charset="utf-8"><script type="module" src="/probe.js"></script>', {
          headers: { "Content-Type": "text/html" },
        });
      case "/probe.js":
        return new Response(browserScript, { headers: { "Content-Type": "text/javascript" } });
      case "/preferences":
        requests.push("preferences started");
        await preferencesGate.promise;
        requests.push("preferences responded");
        return Response.json({});
      case "/states":
        requests.push("states responded");
        return new Response("Ready");
      default:
        return new Response(null, { status: 404 });
    }
  },
});

async function evaluate(session, expression) {
  const { result, exceptionDetails } = await session.send("Runtime.evaluate", {
    expression,
    returnByValue: true,
    awaitPromise: true,
  });
  if (exceptionDetails) throw new Error(exceptionDetails.exception?.description ?? exceptionDetails.text);
  return result.value;
}

async function waitFor(probe, description) {
  const deadline = performance.now() + 5000;
  async function check() {
    if (await probe()) return;
    if (performance.now() >= deadline) throw new Error(`Timed out waiting for ${description}`);
    await Bun.sleep(20);
    return check();
  }
  return check();
}

try {
  await withChromium(async (port) => {
    const target = await openTarget(port);
    const session = new CdpSession(target.webSocketDebuggerUrl);
    const exceptions = [];
    try {
      await session.connect();
      session.on("Runtime.exceptionThrown", ({ exceptionDetails }) => {
        exceptions.push(exceptionDetails.exception?.description ?? exceptionDetails.text);
      });
      await session.send("Runtime.enable");
      await session.send("Page.navigate", { url: server.url.href });
      await waitFor(
        () => evaluate(session, "document.querySelector('button')?.textContent.includes('Todo')"),
        "initial status labels"
      );
      await evaluate(session, "void window.refreshPreferences()");
      await waitFor(() => requests.includes("preferences started"), "the held preferences request");
      await evaluate(session, "window.refreshStates()");
      await waitFor(
        () => evaluate(session, "document.querySelector('button')?.textContent.includes('Ready')"),
        "the renamed state"
      );
      preferencesGate.resolve();
      await waitFor(
        () => evaluate(session, "document.querySelector('[data-stage=\"preferences response\"]') !== null"),
        "the late preferences response"
      );
      const samples = await evaluate(session, "window.samples");
      process.stdout.write(`${JSON.stringify({ requests, samples }, null, 2)}\n`);
      assert.deepEqual(requests, ["preferences started", "states responded", "preferences responded"]);
      assert.deepEqual(
        Array.from(new Set(samples.map(({ stage }) => stage))),
        ["initial", "states response", "preferences response"],
        "capture each scenario stage in order"
      );
      for (const { stage, labels, disabled } of samples) {
        const expected = stage === "initial" ? "Backlog,Todo+1 more" : "Backlog,Ready+1 more";
        assert.equal(labels, expected, `selection changed unexpectedly during ${stage}`);
        assert.equal(disabled, false, `selector was disabled during ${stage}`);
      }
    } catch (error) {
      if (exceptions.length > 0) throw new Error(exceptions.join("\n"), { cause: error });
      throw error;
    } finally {
      session.close();
      await closeTarget(port, target.id);
    }
  });
} finally {
  preferencesGate.resolve();
  await server.stop(true);
}
