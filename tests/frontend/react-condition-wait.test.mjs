import assert from "node:assert/strict";
import test from "node:test";
import { setImmediate as nextTurn } from "node:timers/promises";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";

function DelayedCommit() {
  const [ready, setReady] = React.useState(false);
  React.useEffect(() => {
    const timer = setTimeout(() => setReady(true), 20);
    return () => clearTimeout(timer);
  }, []);
  return React.createElement("output", null, ready ? "Ready" : "Pending");
}

await test("React conditions resolve when a scheduled commit is ready, before the polling deadline", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let now = 0;
  t.mock.method(performance, "now", () => now);
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await env.render(React.createElement(DelayedCommit));
  let settled = false;
  const waiting = waitForReactCondition(
    () => env.container.textContent === "Ready",
    "The scheduled commit should be observed",
    1000
  ).then(() => {
    settled = true;
    return true;
  });
  try {
    now = 20;
    t.mock.timers.tick(20);
    await nextTurn();
    await nextTurn();
    assert.equal(
      settled,
      true,
      "the poll must release React work between attempts instead of waiting out the deadline"
    );
  } finally {
    now = 1001;
    t.mock.timers.tick(1001);
    await waiting;
  }
});

await test("React condition polling rejects with its message at the original deadline", async (t) => {
  let now = 0;
  t.mock.method(performance, "now", () => now);
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const result = waitForReactCondition(() => false, "Still pending", 20).then(
    () => null,
    (error) => error
  );
  now = 20;
  t.mock.timers.tick(20);
  const error = await result;
  assert.equal(error.message, "Still pending");
});

await test("an already satisfied React condition does not schedule a polling delay", async (t) => {
  t.mock.method(globalThis, "setTimeout", () => {
    throw Error("Unexpected polling timer");
  });
  await waitForReactCondition(() => true, "Ready", 0);
});
