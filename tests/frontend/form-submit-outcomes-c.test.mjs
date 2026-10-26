import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
globalThis.formOutcomeReact = React;
const nativeForms = observeNativeForms(["apps/web/core/components/common/confirm-action-modal.tsx"]);
const stubs = {
  "@plane/propel/button": `const React=globalThis.formOutcomeReact;export const Button=({children,loading,...props})=>React.createElement("button",props,children);`,
  "@plane/ui": `const React=globalThis.formOutcomeReact;export const EModalPosition={CENTER:"center"};export const EModalWidth={XXL:"xxl"};export const ModalCore=({children})=>React.createElement("div",{role:"dialog"},children);export const Input=props=>React.createElement("input",props);`,
  "lucide-react": `export const AlertTriangle=()=>null;`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

await test("confirmation validation blocks work and failed work remains retryable", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  let attempts = 0;
  let closed = 0;
  const { ConfirmActionModal } = await import("../../apps/web/core/components/common/confirm-action-modal.tsx");
  await environment.render(
    React.createElement(ConfirmActionModal, {
      isOpen: true,
      title: "Delete item",
      description: "This cannot be undone.",
      name: "Widget",
      confirmation: "DELETE",
      actionLabel: "Delete",
      pendingLabel: "Deleting",
      errorMessage: "The delete failed.",
      onConfirm: async () => {
        attempts++;
        if (attempts === 1) throw new Error("offline");
      },
      onClose: () => closed++,
    })
  );
  const inputs = environment.container.querySelectorAll("input");
  await changeFormControl(inputs[0], "Widget");
  await changeFormControl(inputs[1], "wrong");
  await submitForm(environment.container);
  const formApi = nativeForms.getForms("apps/web/core/components/common/confirm-action-modal.tsx").at(-1);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(attempts, 0);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Confirmation does not match/);

  await changeFormControl(inputs[1], "DELETE");
  await submitForm(environment.container);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(attempts, 1);
  assert.equal(closed, 0);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /delete failed/);
  assert.equal(environment.container.querySelector('button[type="submit"]').disabled, false);

  await submitForm(environment.container);
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.equal(attempts, 2);
  assert.equal(closed, 1);
});

nativeForms.restore();
