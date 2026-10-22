import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { registerHooks } from "node:module";
import { after, afterEach, test } from "node:test";
import { act, createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { StoreContext } from "./helpers/work-item-store-context.mjs";

registerWebModuleHooks({ stubStoreContext: true });
const packageJson = readFileSync(new URL("../../apps/web/package.json", import.meta.url), "utf8");
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "package.json") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`export default ${packageJson}`)}`,
      };
    }
    return nextResolve(specifier, context);
  },
});

const environment = await createReactDomTestEnvironment();
const { WorkspaceEditionBadge } = await import("../../apps/web/core/components/workspace/edition-badge.tsx");
const { BillingRoot } = await import("../../apps/web/core/components/workspace/billing/root.tsx");
const { IssueBulkOperationsRoot } = await import("../../apps/web/core/components/issues/bulk-operations/root.tsx");

afterEach(async () => environment.render(null));
after(async () => environment.unmount());

test("the edition badge identifies a self-hosted build without opening a purchase dialog", async () => {
  await environment.render(React.createElement(WorkspaceEditionBadge));

  assert.match(environment.container.textContent, /Self-hosted/);
  assert.equal(environment.container.querySelector("button, a, [aria-haspopup='dialog']"), null);
  await act(async () => environment.container.querySelector("span").click());
  assert.equal(document.querySelector("[role='dialog']"), null);
});

test("the legacy billing screen offers no plans or checkout actions", async () => {
  await environment.render(React.createElement(BillingRoot));

  assert.match(environment.container.textContent, /does not use subscriptions or paid plans/);
  assert.equal(environment.container.querySelector("button, a"), null);
});

test("the bulk editing stub lets users clear selection without an upgrade prompt", async () => {
  function Harness() {
    const [selected, setSelected] = React.useState(true);
    return React.createElement(
      StoreContext.Provider,
      { value: { multipleSelect: { isSelectionActive: selected } } },
      React.createElement(IssueBulkOperationsRoot, {
        selectionHelpers: { isSelectionDisabled: false, handleClearSelection: () => setSelected(false) },
      })
    );
  }

  await environment.render(React.createElement(Harness));

  assert.match(environment.container.textContent, /Bulk editing is not available/);
  assert.equal(environment.container.querySelector("a"), null);
  const clearButton = environment.container.querySelector("button");
  assert.equal(clearButton.textContent, "Clear selection");
  await act(async () => clearButton.click());
  assert.equal(environment.container.textContent, "");
});

test("the bulk editing stub stays hidden when selection is disabled", async () => {
  await environment.render(
    React.createElement(
      StoreContext.Provider,
      { value: { multipleSelect: { isSelectionActive: true } } },
      React.createElement(IssueBulkOperationsRoot, {
        selectionHelpers: { isSelectionDisabled: true, handleClearSelection() {} },
      })
    )
  );

  assert.equal(environment.container.textContent, "");
});
