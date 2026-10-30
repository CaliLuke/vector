import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

async function mountPromiseToast(t, promise) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { Toast, setPromiseToast } = await import("../../packages/propel/src/toast/toast.tsx");
  await env.render(React.createElement(Toast, { theme: "light" }));
  await act(async () => {
    setPromiseToast(promise, {
      loading: "Saving test changes",
      success: { title: "Test changes saved", message: (value) => value },
      error: { title: "Test changes failed", message: () => "Try again." },
    });
  });
  assert.match(document.body.textContent, /Saving test changes/);
}

await test("promise toast shows success and preserves the request result", async (t) => {
  const pending = Promise.withResolvers();
  await mountPromiseToast(t, pending.promise);

  await act(async () => {
    pending.resolve("accepted result");
    await pending.promise;
  });

  assert.match(document.body.textContent, /Test changes saved/);
  assert.match(document.body.textContent, /accepted result/);
  assert.equal(await pending.promise, "accepted result");
});

await test("promise toast shows failure without an unhandled notification rejection", async (t) => {
  const pending = Promise.withResolvers();
  const rejection = new Error("request rejected");
  const requestResult = assert.rejects(pending.promise, (error) => error === rejection);
  await mountPromiseToast(t, pending.promise);

  await act(async () => {
    pending.reject(rejection);
    await requestResult;
    // Let node:test detect any unhandled rejection from the notification observer.
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.match(document.body.textContent, /Test changes failed/);
  assert.match(document.body.textContent, /Try again/);
});
