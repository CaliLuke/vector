import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/hooks") {
      return nextResolve(new URL("../../packages/hooks/src/index.ts", import.meta.url).href, context);
    }
    return nextResolve(specifier, context);
  },
});

await test("navigation progress configures once, starts after a zero-delay timer, and completes on idle or unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  const { useNavigationProgress } = await import("@plane/hooks");
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const calls = [];
  let started = false;
  const progress = {
    configure: (options) => calls.push(["configure", options]),
    render: (fromStart) => calls.push(["render", fromStart]),
    start: () => {
      started = true;
      calls.push(["start"]);
    },
    done: () => {
      started = false;
      calls.push(["done"]);
    },
    isStarted: () => started,
  };
  function Harness({ navigationState }) {
    useNavigationProgress(navigationState, progress);
    return null;
  }

  await environment.render(React.createElement(Harness, { navigationState: "idle" }));
  assert.deepEqual(calls.slice(0, 2), [
    [
      "configure",
      {
        showSpinner: false,
        minimum: 0.1,
        speed: 400,
        trickleSpeed: 800,
        easing: "ease",
        trickle: true,
      },
    ],
    ["render", true],
  ]);

  await environment.render(React.createElement(Harness, { navigationState: "loading" }));
  await environment.render(React.createElement(Harness, { navigationState: "idle" }));
  assert.equal(
    calls.some(([name]) => name === "start"),
    false,
    "idle should cancel a pending zero-delay start"
  );

  await environment.render(React.createElement(Harness, { navigationState: "loading" }));
  await environment.render(React.createElement(Harness, { navigationState: "submitting" }));
  await act(async () => t.mock.timers.tick(0));
  assert.equal(calls.filter(([name]) => name === "start").length, 1);

  await environment.render(React.createElement(Harness, { navigationState: "idle" }));
  assert.equal(calls.filter(([name]) => name === "done").length, 1);

  await environment.render(React.createElement(Harness, { navigationState: "submitting" }));
  await act(async () => t.mock.timers.tick(0));
  await environment.unmount();
  assert.equal(calls.filter(([name]) => name === "start").length, 2);
  assert.equal(calls.filter(([name]) => name === "done").length, 2);
});
