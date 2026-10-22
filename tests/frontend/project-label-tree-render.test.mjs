import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { format } from "node:util";
import { React, act, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

globalThis.__labelTreeTestMobx = webRequire("mobx");
globalThis.__labelTreeTestMobxUtils = webRequire("mobx-utils");
const { observer } = webRequire("mobx-react");
const { runInAction } = globalThis.__labelTreeTestMobx;

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (!decodeURIComponent(context.parentURL ?? "").endsWith("/core/store/label.store.ts")) {
      return nextResolve(specifier, context);
    }
    if (specifier === "mobx") {
      const exports = ["action", "computed", "makeObservable", "observable", "runInAction"]
        .map((name) => `export const ${name} = globalThis.__labelTreeTestMobx.${name};`)
        .join("\n");
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(exports)}` };
    }
    if (specifier === "mobx-utils") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const computedFn = globalThis.__labelTreeTestMobxUtils.computedFn;")}`,
      };
    }
    if (specifier === "@/services/issue") {
      return { shortCircuit: true, url: "data:text/javascript,export class IssueLabelService {}" };
    }
    return nextResolve(specifier, context);
  },
});

const { LabelStore } = await import("../../apps/web/core/store/label.store.ts");

test("rendering a label tree leaves stored labels untouched", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const store = new LabelStore({ router: { projectId: "project", workspaceSlug: "workspace" } });
  runInAction(() => {
    store.fetchedMap.project = true;
    store.labelMap.parent = { id: "parent", project_id: "project", parent: null, name: "Group", sort_order: 1 };
    store.labelMap.child = { id: "child", project_id: "project", parent: "parent", name: "Child", sort_order: 2 };
  });

  const warnings = [];
  const originalError = console.error;
  console.error = (...args) => warnings.push(format(...args));
  t.after(() => {
    console.error = originalError;
  });

  const LabelTree = observer(function LabelTree() {
    const groups = store.projectLabelsTree ?? [];
    return React.createElement(
      "div",
      null,
      groups.map((group) => React.createElement("div", { key: group.id }, group.name, ": ", group.children?.[0]?.name))
    );
  });
  await environment.render(React.createElement(LabelTree));

  assert.equal(environment.container.textContent, "Group: Child");
  assert.equal(Object.hasOwn(store.labelMap.parent, "children"), false);
  assert.equal(Object.hasOwn(store.labelMap.child, "children"), false);

  await act(async () => {
    runInAction(() => {
      store.labelMap.child.name = "Renamed child";
    });
  });

  assert.equal(environment.container.textContent, "Group: Renamed child");
  assert.equal(Object.hasOwn(store.labelMap.parent, "children"), false);
  assert.equal(
    warnings.some((warning) => /Cannot update a component.*while rendering/.test(warning)),
    false
  );
});
