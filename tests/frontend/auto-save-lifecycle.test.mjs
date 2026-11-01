import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { default: useAutoSave } = await import("../../apps/web/core/hooks/use-auto-save.tsx");

function Harness({ save }) {
  useAutoSave(save);
  return null;
}

const shortcut = (modifier) => {
  const event = new KeyboardEvent("keydown", { key: "s", [modifier]: true, cancelable: true });
  window.dispatchEvent(event);
  assert.equal(event.defaultPrevented, true);
};

void test("autosave uses current callbacks, debounces manual saves and cancels all pending saves on unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.timers.enable({ apis: ["Date", "setTimeout", "setInterval"] });
  const saves = [];
  const tick = (duration) => React.act(async () => t.mock.timers.tick(duration));
  await environment.render(React.createElement(Harness, { save: () => saves.push("old") }));
  await tick(29999);
  assert.deepEqual(saves, []);
  await environment.render(React.createElement(Harness, { save: () => saves.push("current") }));
  await tick(1);
  assert.deepEqual(saves, ["current"]);
  shortcut("ctrlKey");
  await tick(200);
  shortcut("metaKey");
  await tick(499);
  assert.equal(saves.length, 1);
  await tick(1);
  assert.deepEqual(saves, ["current", "current"]);
  await tick(29999);
  assert.equal(saves.length, 2, "manual save resets the automatic save interval");
  await tick(1);
  assert.equal(saves.length, 3);
  shortcut("ctrlKey");
  await environment.render(null);
  await tick(60000);
  assert.equal(saves.length, 3, "neither debounced nor periodic work survives unmount");
  const event = new KeyboardEvent("keydown", { key: "s", ctrlKey: true, cancelable: true });
  window.dispatchEvent(event);
  assert.equal(event.defaultPrevented, false);
});

void test("autosave reports synchronous and asynchronous failures without stopping later saves", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.timers.enable({ apis: ["Date", "setTimeout", "setInterval"] });
  const report = t.mock.method(console, "error", () => {});
  const syncError = new Error("sync failure");
  const asyncError = new Error("async failure");
  await environment.render(
    React.createElement(Harness, {
      save() {
        throw syncError;
      },
    })
  );
  await React.act(async () => t.mock.timers.tick(30000));
  assert.deepEqual(report.mock.calls[0].arguments, ["Autosave failed:", syncError]);
  await environment.render(React.createElement(Harness, { save: () => Promise.reject(asyncError) }));
  await React.act(async () => t.mock.timers.tick(30000));
  assert.deepEqual(report.mock.calls[1].arguments, ["Autosave failed:", asyncError]);
  const successfulSave = t.mock.fn();
  await environment.render(React.createElement(Harness, { save: successfulSave }));
  await React.act(async () => t.mock.timers.tick(30000));
  assert.equal(successfulSave.mock.callCount(), 1);
});
