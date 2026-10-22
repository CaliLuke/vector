import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/hooks/store/estimates") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const useProjectEstimates = () => ({ createEstimate: async () => {} });")}`,
      };
    }
    return nextResolve(specifier, context);
  },
});

test("estimate modal Escape closes from a radio and restores its trigger focus", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const previousResizeObserver = globalThis.ResizeObserver;
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
  t.after(() => {
    if (previousResizeObserver) globalThis.ResizeObserver = previousResizeObserver;
    else delete globalThis.ResizeObserver;
  });
  const storageDescriptor = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  Object.defineProperty(globalThis, "localStorage", { configurable: true, value: window.localStorage });
  t.after(() => {
    if (storageDescriptor) Object.defineProperty(globalThis, "localStorage", storageDescriptor);
    else delete globalThis.localStorage;
  });
  const { CreateEstimateModal } = await import("../../apps/web/core/components/estimates/create/modal.tsx");
  function Workflow() {
    const [open, setOpen] = React.useState(false);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("button", { onClick: () => setOpen(true) }, "Add estimate"),
      React.createElement(CreateEstimateModal, {
        workspaceSlug: "workspace",
        projectId: "project",
        isOpen: open,
        handleClose: () => setOpen(false),
      })
    );
  }
  await environment.render(React.createElement(Workflow));
  const trigger = environment.container.querySelector("button");
  await act(async () => {
    trigger.focus();
    trigger.click();
  });
  const radio = document.querySelector('input[type="radio"]');
  assert.ok(radio);
  await act(async () => {
    radio.focus();
    radio.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await new Promise((resolve) => setTimeout(resolve, 250));
  });
  assert.ok(document.querySelector('[role="dialog"]') === null, "Escape must close the estimate dialog");
  assert.ok(document.activeElement === trigger, "focus must return to the trigger");
});
