import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { EIssuesStoreType } from "../../packages/types/dist/index.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.calendarPreferenceWriteToasts = [];
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    resolvePackageStub(
      specifier,
      {
        "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"};export const setToast=value=>globalThis.calendarPreferenceWriteToasts.push(value);`,
      },
      context,
      nextResolve
    ),
});

const { useCalendarDisplayFilterUpdate } =
  await import("../../apps/web/core/hooks/use-calendar-display-filter-update.ts");

async function renderUpdateButton(t, { storeType, projectPreferences, updateFilters }) {
  globalThis.calendarPreferenceWriteToasts = [];
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  function Probe() {
    const update = useCalendarDisplayFilterUpdate(storeType, projectPreferences, undefined, updateFilters);
    return React.createElement(
      "button",
      {
        onClick: () => void update("project-1", { calendar: { layout: "week" } }),
      },
      "Save calendar preferences"
    );
  }
  await environment.render(React.createElement(Probe));
  await act(async () => {
    environment.container.querySelector("button").dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
  });
  return globalThis.calendarPreferenceWriteToasts;
}

await test("project calendar preference rejection is surfaced to the user", async (t) => {
  const toasts = await renderUpdateButton(t, {
    storeType: EIssuesStoreType.PROJECT,
    projectPreferences: { save: async () => Promise.reject(new Error("Preference write failed")) },
    updateFilters: async () => undefined,
  });

  assert.equal(toasts.length, 1);
  assert.equal(toasts[0].type, "error");
  assert.equal(toasts[0].message, "Preference write failed");
});

await test("non-project calendar filter rejection is surfaced to the user", async (t) => {
  const toasts = await renderUpdateButton(t, {
    storeType: EIssuesStoreType.CYCLE,
    projectPreferences: undefined,
    updateFilters: async () => Promise.reject(new Error("Filter write failed")),
  });

  assert.equal(toasts.length, 1);
  assert.equal(toasts[0].type, "error");
  assert.equal(toasts[0].message, "Filter write failed");
});
