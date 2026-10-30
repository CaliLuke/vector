import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

await test("DOM teardown awaits resource cleanup after unmount and before removing browser globals", async () => {
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const environment = await createReactDomTestEnvironment();
  const events = [];
  function Probe() {
    React.useEffect(() => () => events.push("React unmounted"), []);
    return null;
  }
  await environment.render(React.createElement(Probe));

  await environment.unmount(async () => {
    assert.deepEqual(events, ["React unmounted"]);
    await Promise.resolve();
    assert.equal(window.document, document);
    events.push("resources disposed");
  });

  assert.deepEqual(events, ["React unmounted", "resources disposed"]);
  assert.deepEqual(Object.getOwnPropertyDescriptor(globalThis, "window"), originalWindow);
});

await test("DOM teardown restores globals when resource cleanup rejects", async () => {
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const environment = await createReactDomTestEnvironment();
  const failure = new Error("resource cleanup failed");

  await assert.rejects(
    environment.unmount(async () => {
      throw failure;
    }),
    (error) => error === failure
  );

  assert.deepEqual(Object.getOwnPropertyDescriptor(globalThis, "window"), originalWindow);
});
