import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const resendFormObserver = observeNativeForms(["packages/hooks/src/use-resend-code-form.ts"]);

await test("code resend retains the current code after failure and starts its cooldown only after success", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const { useResendCodeForm } = await import("../../packages/hooks/src/use-resend-code-form.ts");
  let finish;
  let fail;
  function Harness() {
    const [code, setCode] = React.useState("123456");
    const [error, setError] = React.useState("");
    const {
      isSubmitting: busy,
      cooldown,
      resend,
    } = useResendCodeForm({
      send: () =>
        new Promise((resolve, reject) => {
          finish = resolve;
          fail = reject;
        }),
      onSent: () => {
        setCode("");
        setError("");
      },
      onError: () => setError("Could not resend"),
    });
    return React.createElement(
      "div",
      null,
      React.createElement("input", { readOnly: true, value: code }),
      React.createElement("button", { disabled: busy || cooldown > 0, onClick: resend }, busy ? "Sending" : "Resend"),
      React.createElement("output", null, cooldown),
      React.createElement("p", { role: "alert" }, error)
    );
  }
  await env.render(React.createElement(Harness));
  const button = env.container.querySelector("button");
  await act(async () => button.click());
  assert.equal(button.disabled, true);
  await act(async () => fail(new Error("offline")));
  assert.equal(
    resendFormObserver.getForms("packages/hooks/src/use-resend-code-form.ts").at(-1)?.state.isSubmitSuccessful,
    false
  );
  assert.equal(env.container.querySelector("input").value, "123456");
  assert.equal(env.container.querySelector("output").textContent, "0");
  assert.equal(button.disabled, false);
  await act(async () => button.click());
  await act(async () => finish());
  assert.equal(
    resendFormObserver.getForms("packages/hooks/src/use-resend-code-form.ts").at(-1)?.state.isSubmitSuccessful,
    true
  );
  assert.equal(env.container.querySelector("input").value, "");
  assert.equal(env.container.querySelector("output").textContent, "5");
  assert.equal(button.disabled, true);
  const tick = () => act(async () => t.mock.timers.tick(1000));
  await tick();
  assert.equal(env.container.querySelector("output").textContent, "4");
  assert.equal(button.disabled, true);
  await tick();
  await tick();
  await tick();
  await tick();
  assert.equal(env.container.querySelector("output").textContent, "0");
  assert.equal(button.disabled, false);
});

resendFormObserver.restore();
