import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerAdminFormHooks, formToasts } from "./helpers/admin-forms.mjs";

registerAdminFormHooks();
const stringSource = new URL("../../packages/utils/src/string.ts", import.meta.url).href;
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@plane/utils/string") return next(stringSource, context);
    return next(specifier, context);
  },
});
const { CopyField } = await import("../../apps/admin/components/common/copy-field.tsx");
const { copyTextToClipboard } = await import(stringSource);

async function setup(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  formToasts.length = 0;
  return env;
}
const props = { label: "Redirect URI", url: "https://app.example/auth/callback", description: "Provider callback" };

void test("admin copy feedback waits for clipboard acceptance and permits retry after denial", async (t) => {
  const env = await setup(t);
  let pending = Promise.withResolvers();
  const writes = [];
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: {
      writeText: (value) => {
        writes.push(value);
        return pending.promise;
      },
    },
  });
  await env.render(React.createElement(CopyField, props));
  const button = env.container.querySelector("button");
  await act(async () => button.click());
  assert.deepEqual(writes, [props.url]);
  assert.deepEqual(formToasts, [], "copying must be accepted before success feedback");
  await act(async () => pending.reject(new Error("clipboard denied")));
  assert.equal(formToasts.at(-1).type, "error");
  assert.equal(formToasts.length, 1);
  pending = Promise.withResolvers();
  await act(async () => button.click());
  assert.equal(formToasts.length, 1);
  await act(async () => pending.resolve());
  assert.equal(formToasts.at(-1).type, "success");
  assert.equal(formToasts.length, 2);
  assert.deepEqual(writes, [props.url, props.url]);
});

for (const outcome of ["accepted", "denied", "throws", "unavailable"]) {
  void test(`legacy clipboard ${outcome} restores focus and removes temporary input`, async (t) => {
    const env = await setup(t);
    await env.render(React.createElement("input", { defaultValue: "original draft" }));
    const input = env.container.querySelector("input");
    input.focus();
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
    const commands = [];
    document.execCommand =
      outcome === "unavailable"
        ? undefined
        : (command) => {
            commands.push([command, document.activeElement.value]);
            if (outcome === "throws") throw new Error("browser rejected copy");
            return outcome === "accepted";
          };
    const completion = copyTextToClipboard(props.url);
    if (outcome === "accepted") await completion;
    else await assert.rejects(completion);
    assert.deepEqual(commands, outcome === "unavailable" ? [] : [["copy", props.url]]);
    assert.equal(document.querySelectorAll("textarea").length, 0);
    assert.equal(document.activeElement, input);
    assert.equal(input.value, "original draft");
  });
}
