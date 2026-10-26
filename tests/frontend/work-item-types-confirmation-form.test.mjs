import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";

const source = "apps/web/core/components/work-item-types/controls.tsx";
const modalCoreUrl = new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href;
const modalConstantsUrl = new URL("../../packages/ui/src/modals/constants.ts", import.meta.url).href;
const stubs = {
  "@plane/propel/button": `const React=globalThis.confirmationReact;export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "@plane/ui": `export { ModalCore } from ${JSON.stringify(modalCoreUrl)};export { EModalWidth } from ${JSON.stringify(modalConstantsUrl)};`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error instanceof Error?error.message:"Could not complete this action.";`,
  "./type-icons": `export const typeIconComponents={task:()=>null};`,
};
const nativeForms = registerWorkItemTypesFormHarness({
  sourceFiles: [source],
  stubs,
  reactGlobal: "confirmationReact",
  react: React,
});
let Confirmation;

function Screen({ onConfirm, onClose }) {
  return React.createElement(
    Confirmation,
    { title: "Confirm action", token: "remove", action: "Remove", onConfirm, onClose },
    React.createElement("p", null, "This action cannot be undone.")
  );
}
const click = (node) => act(async () => node.click());
const findButton = (root, name) => [...root.querySelectorAll("button")].find((button) => button.textContent === name);

await test("confirmation token and rejected action remain in native Form for retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  ({ Confirmation } = await import("../../apps/web/core/components/work-item-types/controls.tsx"));
  let attempts = 0;
  let resolveRetry;
  const closed = [];
  await environment.render(
    React.createElement(Screen, {
      onConfirm: async () => {
        attempts += 1;
        if (attempts === 1) throw new Error("The action failed.");
        await new Promise((resolve) => (resolveRetry = resolve));
      },
      onClose: () => closed.push(true),
    })
  );
  const input = document.body.querySelector('input[autocomplete="off"]');
  await click(findButton(document.body, "Remove"));
  assert.equal(attempts, 0);
  assert.match(document.body.querySelector('[role="alert"]').textContent, /does not match/i);
  assert.equal(nativeForms.getForms(source).at(-1).state.isSubmitSuccessful, false);

  await changeFormControl(input, "remove");
  await click(findButton(document.body, "Remove"));
  assert.equal(attempts, 1);
  assert.equal(input.value, "remove");
  assert.match(document.body.querySelector('[role="alert"]').textContent, /action failed/i);
  assert.equal(nativeForms.getForms(source).at(-1).state.isSubmitSuccessful, false);

  await click(findButton(document.body, "Remove"));
  assert.equal(attempts, 2);
  assert.equal(findButton(document.body, "Remove").disabled, true);
  assert.equal(document.body.querySelector('[role="alert"]'), null);
  await act(async () => resolveRetry());
  assert.equal(nativeForms.getForms(source).at(-1).state.isSubmitSuccessful, true);
  assert.deepEqual(closed, []);
});

await test("confirmation prevents cancel and modal dismissal while its action is pending", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  ({ Confirmation } = await import("../../apps/web/core/components/work-item-types/controls.tsx"));
  let resolveAction;
  let closeCount = 0;
  await environment.render(
    React.createElement(Screen, {
      onConfirm: () => new Promise((resolve) => (resolveAction = resolve)),
      onClose: () => {
        closeCount += 1;
      },
    })
  );
  await changeFormControl(document.body.querySelector('input[autocomplete="off"]'), "remove");
  const confirm = findButton(document.body, "Remove");
  await click(confirm);
  assert.equal(confirm.disabled, true);
  const cancel = [...document.body.querySelectorAll("button")].find((button) => button.textContent === "Cancel");
  assert.equal(cancel.disabled, true);
  await click(cancel);
  assert.equal(closeCount, 0);
  await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
  assert.equal(closeCount, 0);
  await act(async () => resolveAction());
  assert.equal(nativeForms.getForms(source).at(-1).state.isSubmitSuccessful, true);
  assert.equal(closeCount, 0);
});
