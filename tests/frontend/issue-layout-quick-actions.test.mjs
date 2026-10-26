import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { IssueLayoutQuickActions } =
  await import("../../apps/web/core/components/issues/issue-layouts/issue-layout-quick-actions.tsx");

await test("layout quick actions retain the rendered issue scope and propagate failed writes", async (t) => {
  const view = await createReactDomTestEnvironment();
  t.after(() => view.unmount());
  let controls;
  const capture = (props) => {
    controls = props;
    return null;
  };
  const writes = [];
  const failure = new Error("Save failed");
  const actions = {
    removeIssue: async (projectId, issueId) => {
      writes.push(["delete", projectId, issueId]);
    },
    updateIssue: async (projectId, issueId, data) => {
      writes.push(["update", projectId, issueId, data]);
      if (data.name === "rejected") throw failure;
      return { id: issueId, project_id: projectId, ...data };
    },
  };
  const parentRef = { current: null };
  const button = React.createElement("button", null, "Actions");
  const portalElement = document.createElement("div");
  const render = (issue, readOnly) =>
    view.render(
      React.createElement(IssueLayoutQuickActions, {
        QuickActions: capture,
        issue,
        readOnly,
        actions,
        parentRef,
        customActionButton: button,
        portalElement,
        placement: "bottom-end",
      })
    );
  await render({ id: "first", project_id: "project-a" }, false);
  const first = controls;
  await first.handleUpdate({ name: "saved" });
  await assert.rejects(first.handleUpdate({ name: "rejected" }), (error) => error === failure);
  await render({ id: "second", project_id: "project-b" }, true);
  assert.equal(controls.readOnly, true);
  assert.equal(controls.parentRef, parentRef);
  assert.equal(controls.customActionButton, button);
  assert.equal(controls.portalElement, portalElement);
  assert.equal(controls.placements, "bottom-end");
  await controls.handleDelete();
  await first.handleDelete();
  assert.deepEqual(writes, [
    ["update", "project-a", "first", { name: "saved" }],
    ["update", "project-a", "first", { name: "rejected" }],
    ["delete", "project-b", "second"],
    ["delete", "project-a", "first"],
  ]);
});
