import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const stubs = {
  "@/hooks/work-item-filters/use-work-item-filters-config": `export const useWorkItemFiltersConfig = () => ({
    areAllConfigsInitialized: true, configs: [], configMap: {}, isFilterEnabled: () => false,
    members: [], catalogError: false, retryCatalogs: async () => {}
  });`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });

function CaptureEditor({ editor, onCapture }) {
  React.useEffect(() => {
    onCapture(editor);
  }, [editor, onCapture]);
  return null;
}

await test("a rejected view update keeps the expression dirty and a later acceptance advances its baseline", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { WorkItemFiltersHOC } = await import("../../apps/web/core/components/work-item-filters/filters-hoc/base.tsx");
  const { WorkItemFilterUiProvider } = await import("../../apps/web/core/providers/work-item-filter-ui.tsx");
  const filterChanges = [];
  const viewUpdates = [];
  let rejectUpdate = true;
  let holdUpdate = false;
  let resolveUpdate;
  let editor;
  const captureEditor = (value) => {
    editor = value;
  };
  function Harness() {
    return React.createElement(
      WorkItemFilterUiProvider,
      { scopeKey: "alice:acme:p:project:p" },
      React.createElement(
        WorkItemFiltersHOC,
        {
          entityType: "project",
          entityId: "p",
          workspaceSlug: "acme",
          projectId: "p",
          filtersToShowByLayout: [],
          initialWorkItemFilters: {
            richFilters: { priority__exact: "high" },
            displayFilters: {},
            displayProperties: {},
          },
          updateFilters: (expression) => filterChanges.push(expression),
          updateViewOptions: {
            label: "Update",
            onViewUpdate: async (expression) => {
              viewUpdates.push(expression);
              if (rejectUpdate) throw new Error("save rejected");
              if (holdUpdate) await new Promise((resolve) => (resolveUpdate = resolve));
            },
          },
        },
        ({ editor: filterEditor }) =>
          React.createElement(CaptureEditor, { editor: filterEditor, onCapture: captureEditor })
      )
    );
  }
  await env.render(React.createElement(Harness));
  await act(async () => editor.onUpdateConditionValue(editor.conditions[0].id, "low"));
  assert.equal(editor.hasChanges, true);
  await assert.rejects(async () => {
    await act(async () => {
      await editor.onUpdate();
    });
  }, /save rejected/);
  assert.equal(editor.hasChanges, true, "the failed callback must not acknowledge its draft");
  rejectUpdate = false;
  await act(async () => editor.onUpdate());
  assert.equal(editor.hasChanges, false, "only the accepted callback advances the baseline");
  await act(async () => editor.onUpdateConditionValue(editor.conditions[0].id, "medium"));
  holdUpdate = true;
  let pendingUpdate;
  await act(async () => {
    pendingUpdate = editor.onUpdate();
  });
  assert.equal(typeof resolveUpdate, "function");
  await act(async () => editor.onUpdateConditionValue(editor.conditions[0].id, "urgent"));
  resolveUpdate();
  await act(async () => pendingUpdate);
  assert.equal(editor.hasChanges, true, "a late accepted callback cannot erase a newer edit");
  assert.deepEqual(viewUpdates, [
    { priority__exact: "low" },
    { priority__exact: "low" },
    { priority__exact: "medium" },
  ]);
  assert.deepEqual(filterChanges, [
    { priority__exact: "low" },
    { priority__exact: "medium" },
    { priority__exact: "urgent" },
  ]);
});

await test("canonical filter receipts and refreshes do not submit another preference write", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { WorkItemFiltersHOC } = await import("../../apps/web/core/components/work-item-filters/filters-hoc/base.tsx");
  const { WorkItemFilterUiProvider } = await import("../../apps/web/core/providers/work-item-filter-ui.tsx");
  const writes = [];
  let editor;
  const captureEditor = (value) => {
    editor = value;
  };
  const refreshRef = { current: undefined };
  const captureCanonical = (value) => {
    refreshRef.current = value;
  };
  function Harness() {
    const [canonical, setCanonical] = React.useState({ priority__exact: "high" });
    React.useEffect(() => {
      captureCanonical(setCanonical);
    }, [setCanonical]);
    return React.createElement(
      WorkItemFilterUiProvider,
      { scopeKey: "alice:acme:project:p" },
      React.createElement(
        WorkItemFiltersHOC,
        {
          entityType: "project",
          entityId: "p",
          workspaceSlug: "acme",
          projectId: "p",
          filtersToShowByLayout: [],
          initialWorkItemFilters: { richFilters: canonical, displayFilters: {}, displayProperties: {} },
          updateFilters: async (expression) => {
            writes.push(expression);
            setCanonical(structuredClone(expression));
          },
        },
        (value) => React.createElement(CaptureEditor, { editor: value.editor, onCapture: captureEditor })
      )
    );
  }
  await env.render(React.createElement(Harness));
  assert.equal(writes.length, 0, "initial reads must not write preferences");
  await act(async () => editor.onUpdateConditionValue(editor.conditions[0].id, "low"));
  assert.deepEqual(writes, [{ priority__exact: "low" }], "the accepted receipt must not echo the user write");
  await act(async () => refreshRef.current({ priority__exact: "urgent" }));
  assert.equal(editor.conditions[0].value, "urgent", "a canonical refresh still updates the visible filter");
  assert.equal(writes.length, 1, "a remote refresh is not a user edit");
});
