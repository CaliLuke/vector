import assert from "node:assert/strict";
import { setImmediate } from "node:timers/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { installLocalLogging, reportLocalQueryError } from "../../packages/utils/src/local-logging.ts";

function browser(t) {
  const requests = [];
  const window = new EventTarget();
  window.location = new URL("http://localhost:8000/default/?token=private");
  window.fetch = async (url, options) => {
    requests.push({ url, options });
    return new Response("", { status: url === "/__local_telemetry/v1/logs" ? 200 : 500 });
  };
  window.setInterval = (callback) => {
    window.tick = callback;
    return 1;
  };
  window.clearInterval = () => {};
  class XHR extends EventTarget {
    responseURL = "http://localhost:8000/api/workspaces/default/?password=private";
    status = 500;
    pending = false;
    open() {}
    send() {
      if (!this.pending) this.dispatchEvent(new Event("loadend"));
    }
    abort() {
      this.status = 0;
      this.dispatchEvent(new Event("abort"));
      this.dispatchEvent(new Event("loadend"));
    }
  }
  t.mock.method(console, "error", () => {});
  t.mock.method(console, "warn", () => {});
  globalThis.window = window;
  globalThis.XMLHttpRequest = XHR;

  return { window, requests, XHR };
}

test("browser failures survive as OTLP records without request secrets", async (t) => {
  const { window, requests, XHR } = browser(t);
  const stop = installLocalLogging("plane-web-browser");
  t.after(() => {
    stop();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });
  await window.fetch("/api/failed/?token=private", { headers: { Authorization: "private" } });
  new XHR().send("private request body");
  console.error("password=private", { response: { data: "private body" } });
  console.error("Authorization: Bearer private\nCookie: session=private; csrf=private");
  console.error('password="two words"');
  console.error("token=two words");
  console.error('secret="outer \\"nested-secret\\" tail-secret"');
  window.tick();
  await setImmediate();
  window.tick();
  await setImmediate();
  window.tick();
  await setImmediate();
  const exported = requests.find((request) => request.url === "/__local_telemetry/v1/logs");
  assert.ok(exported);
  assert.equal(exported.options.credentials, "omit");
  assert.ok(!exported.options.body.includes("private"));
  assert.ok(!exported.options.body.includes("two words"));
  assert.ok(!exported.options.body.includes("nested-secret"));
  const body = JSON.parse(exported.options.body);
  assert.equal(body.resourceLogs[0].resource.attributes[0].value.stringValue, "plane-web-browser");
  const exports = requests.filter((request) => request.url === "/__local_telemetry/v1/logs");
  assert.equal(exports.length, 1);
  assert.equal(body.resourceLogs[0].scopeLogs[0].logRecords.length, 7);
  assert.ok(exports.every((request) => !request.options.body.includes("private")));
});

test("failed exports are retried without recursively logging the collector", async (t) => {
  const { window, requests } = browser(t);
  const original = window.fetch;
  let fail = true;
  window.fetch = async (...args) => {
    const response = await original(...args);
    return fail ? new Response("", { status: 503 }) : response;
  };
  const stop = installLocalLogging("plane-web-browser");
  t.after(() => {
    stop();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });
  console.warn("test warning");
  window.tick();
  await setImmediate();
  fail = false;
  window.tick();
  await setImmediate();
  assert.equal(requests.length, 2);
  assert.equal(requests[0].options.body, requests[1].options.body);
});

test("console diagnostics include their caller stack while keeping messages searchable", async (t) => {
  const { window, requests } = browser(t);
  const stop = installLocalLogging("plane-web-browser");
  t.after(() => {
    stop();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });
  const warningMessage = "Navigation before mount";
  const errorMessage = "Component failed";
  function navigationWarning() {
    console.warn(warningMessage);
  }
  function componentError() {
    console.error(errorMessage);
  }
  navigationWarning();
  componentError();
  runInNewContext(
    'console.warn("Runtime call site")',
    { console },
    {
      filename: "http://localhost:8000/navigation.ts?token=private&secret=private",
    }
  );
  window.tick();
  await setImmediate();

  const exported = requests.find((request) => request.url === "/__local_telemetry/v1/logs");
  const records = JSON.parse(exported.options.body).resourceLogs[0].scopeLogs[0].logRecords;
  assert.deepEqual(
    records.map((record) => [record.severityText, record.body.stringValue]),
    [
      ["WARN", "Navigation before mount"],
      ["ERROR", "Component failed"],
      ["WARN", "Runtime call site"],
    ]
  );
  const stacks = records.map(
    (record) => record.attributes.find((attribute) => attribute.key === "code.stacktrace").value.stringValue
  );
  assert.match(stacks[0], /navigationWarning/);
  assert.match(stacks[1], /componentError/);
  assert.match(stacks[2], /http:\/\/localhost:8000\/navigation\.ts/);
  assert.match(stacks[2], /http:\/\/localhost:8000\/navigation\.ts:1:9/);
  assert.equal(exported.options.body.includes("private"), false, "caller stack URLs must discard query secrets");
});

test("Unicode and control-heavy records stay below the keepalive limit across batches", async (t) => {
  const { window, requests } = browser(t);
  const stop = installLocalLogging("plane-web-browser");
  t.after(() => {
    stop();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });

  const NativeError = globalThis.Error;
  globalThis.Error = class extends NativeError {
    constructor(...args) {
      super(...args);
      this.stack = `Error\n${"𐐀".repeat(8000)}`;
    }
  };
  try {
    console.warn("\u0001".repeat(8000));
    console.warn("\u0001".repeat(8000));
    console.warn("\u0001".repeat(8000));
  } finally {
    globalThis.Error = NativeError;
  }

  window.tick();
  await setImmediate();
  window.tick();
  await setImmediate();

  const exports = requests.filter((request) => request.url === "/__local_telemetry/v1/logs");
  assert.equal(exports.length, 2);
  assert.ok(exports.every((request) => new TextEncoder().encode(request.options.body).byteLength < 64 * 1024));
  const records = exports.flatMap(
    (request) => JSON.parse(request.options.body).resourceLogs[0].scopeLogs[0].logRecords
  );
  assert.equal(records.length, 3);
  assert.ok(records.every((record) => record.body.stringValue.length > 0 && record.body.stringValue.length < 8000));
  const stacks = records.map(
    (record) => record.attributes.find((attribute) => attribute.key === "code.stacktrace").value.stringValue
  );
  assert.ok(stacks.every((stack) => stack.includes("𐐀") && stack.length < 8000));
});

test("XHR aborts are ignored while status-zero network failures are recorded", async (t) => {
  const { window, requests, XHR } = browser(t);
  const stop = installLocalLogging("plane-web-browser");
  t.after(() => {
    stop();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });

  const abortedRequest = new XHR();
  abortedRequest.pending = true;
  abortedRequest.send();
  abortedRequest.abort();

  const networkFailure = new XHR();
  networkFailure.open("GET", "/api/unavailable/?token=private");
  networkFailure.responseURL = "";
  networkFailure.pending = true;
  networkFailure.send();
  networkFailure.status = 0;
  networkFailure.dispatchEvent(new Event("error"));
  networkFailure.dispatchEvent(new Event("loadend"));

  window.tick();
  await setImmediate();

  const exported = requests.find((request) => request.url === "/__local_telemetry/v1/logs");
  assert.ok(exported);
  const records = JSON.parse(exported.options.body).resourceLogs[0].scopeLogs[0].logRecords;
  assert.equal(records.length, 1);
  assert.equal(
    records[0].attributes.find((attribute) => attribute.key === "http.response.status_code").value.intValue,
    "0"
  );
  assert.equal(
    records[0].attributes.find((attribute) => attribute.key === "http.request.path").value.stringValue,
    "/api/unavailable/"
  );
  assert.equal(exported.options.body.includes("private"), false);
});

test("logging ignores other origins and restores the browser hooks", async (t) => {
  const { window, requests, XHR } = browser(t);
  const originalFetch = window.fetch;
  // oxlint-disable-next-line unbound-method -- Compare method identity without invoking it.
  const originalSend = XHR.prototype.send;
  // oxlint-disable-next-line unbound-method -- Compare method identity without invoking it.
  const originalOpen = XHR.prototype.open;
  const originalError = console.error;
  const stop = installLocalLogging("plane-web-browser");
  await window.fetch("https://example.com/failed");
  window.tick();
  await setImmediate();
  assert.equal(requests.length, 1);
  stop();
  assert.equal(window.fetch, originalFetch);
  // oxlint-disable-next-line unbound-method -- Compare method identity without invoking it.
  assert.equal(XHR.prototype.send, originalSend);
  // oxlint-disable-next-line unbound-method -- Compare method identity without invoking it.
  assert.equal(XHR.prototype.open, originalOpen);
  assert.equal(console.error, originalError);
  delete globalThis.window;
  delete globalThis.XMLHttpRequest;
});

test("final Query and mutation failures export code locations without private error data", async (t) => {
  const { window, requests } = browser(t);
  const stop = installLocalLogging("plane-web-browser");
  const client = new QueryClient({
    queryCache: new QueryCache({ onError: (error) => reportLocalQueryError(error, "query") }),
    mutationCache: new MutationCache({ onError: (error) => reportLocalQueryError(error, "mutation") }),
  });
  t.after(() => {
    stop();
    client.clear();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });
  let attempts = 0;
  await assert.rejects(
    client.fetchQuery({
      queryKey: ["private-query-key"],
      retry: 1,
      retryDelay: 0,
      queryFn: () => {
        attempts++;
        throw Object.assign(new Error("private-response-data\n    at private-multiline-data"), {
          isAxiosError: true,
          response: { status: 200, data: "private-response-data" },
          request: { status: 200 },
        });
      },
    })
  );
  let localFailure = false;
  const mutation = client.getMutationCache().build(client, {
    mutationKey: ["private-mutation-key"],
    mutationFn: () => {
      throw new Error("private-variable-data");
    },
    onError: () => {
      localFailure = true;
    },
  });
  await assert.rejects(mutation.execute({ secret: "private-variable-data" }));
  window.tick();
  await setImmediate();
  const exported = requests.find((request) => request.url === "/__local_telemetry/v1/logs");
  const records = JSON.parse(exported.options.body).resourceLogs[0].scopeLogs[0].logRecords;
  assert.equal(attempts, 2);
  assert.equal(localFailure, true, "native mutation error callbacks must remain active");
  assert.equal(records.length, 2, "Query retries report only their final failure");
  assert.deepEqual(
    records.map((record) => record.attributes.find((attr) => attr.key === "query.operation").value.stringValue),
    ["query", "mutation"]
  );
  assert.ok(records.every((record) => record.attributes.some((attr) => attr.key === "code.stacktrace")));
  assert.equal(exported.options.body.includes("private-"), false);
});

test("local Query diagnostics ignore cancellation, already reported HTTP failures, and disposal", async (t) => {
  const { window, requests } = browser(t);
  const stop = installLocalLogging("plane-web-browser");
  t.after(() => {
    stop();
    delete globalThis.window;
    delete globalThis.XMLHttpRequest;
  });
  reportLocalQueryError(new DOMException("navigation", "AbortError"), "query");
  reportLocalQueryError({ isAxiosError: true, request: { status: 0 }, message: "network failure" }, "query");
  reportLocalQueryError({ response: { status: 422, data: "private-data" } }, "mutation");
  window.tick();
  await setImmediate();
  assert.equal(requests.length, 0);
  stop();
  reportLocalQueryError(new Error("after disposal"), "query");
  window.tick();
  await setImmediate();
  assert.equal(requests.length, 0);
});
