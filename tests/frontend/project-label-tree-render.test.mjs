import assert from "node:assert/strict";
import test from "node:test";
import { format } from "node:util";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";

const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
const { labelKeys } = await import("../../apps/web/core/queries/labels.ts");
const { useProjectLabels } = await import("../../apps/web/core/hooks/use-labels.ts");

const scope = {
  accountId: "alice",
  workspaceSlug: "workspace",
  projectId: "project",
  workspaceRole: EUserWorkspaceRoles.MEMBER,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectRole: 20,
  projectMembershipRole: 20,
  sessionGeneration: 0,
};
const parent = Object.freeze({ id: "parent", project_id: "project", parent: null, name: "Group", sort_order: 1 });
const child = Object.freeze({ id: "child", project_id: "project", parent: "parent", name: "Child", sort_order: 2 });

function LabelRows({ labels }) {
  return React.createElement(
    "ul",
    null,
    labels.map((label) =>
      React.createElement(
        "li",
        { key: label.id, "data-label-id": label.id },
        React.createElement("span", null, label.name),
        label.children?.length ? React.createElement(LabelRows, { labels: label.children }) : null
      )
    )
  );
}
function LabelTree() {
  const { labelTree } = useProjectLabels("workspace", "project");
  return React.createElement(LabelRows, { labels: labelTree ?? [] });
}

await test("native label tree rendering and refetch preserve canonical records and nested rows", async (t) => {
  let response = Object.freeze([parent, child]);
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => response);
  const view = await createPermissionUiFixture(t, {
    path: "/workspace/settings/projects/project/labels",
    workspaceSlug: "workspace",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { project: 20 },
  });
  const warnings = [];
  t.mock.method(console, "error", (...args) => warnings.push(format(...args)));
  await view.render(React.createElement(LabelTree));
  const nested = () =>
    view.environment.container.querySelector('[data-label-id="parent"] > ul > [data-label-id="child"]');
  assert.equal(nested()?.textContent, "Child");
  assert.equal(view.environment.container.querySelector('[data-label-id="parent"] > span')?.textContent, "Group");
  const canonical = view.client.getQueryData(labelKeys.project(scope));
  assert.equal(Object.hasOwn(canonical[0], "children"), false);
  assert.equal(Object.hasOwn(canonical[1], "children"), false);
  assert.equal(Object.hasOwn(parent, "children"), false);
  assert.equal(Object.hasOwn(child, "children"), false);

  response = Object.freeze([parent, Object.freeze({ ...child, name: "Renamed child" })]);
  await act(async () => view.client.refetchQueries({ queryKey: labelKeys.project(scope), exact: true }));
  await settleQuery();
  assert.equal(nested()?.textContent, "Renamed child");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[1].name, "Renamed child");
  assert.equal(Object.hasOwn(view.client.getQueryData(labelKeys.project(scope))[0], "children"), false);
  assert.equal(Object.hasOwn(view.client.getQueryData(labelKeys.project(scope))[1], "children"), false);
  assert.equal(canonical[1].name, "Child");
  assert.equal(
    warnings.some((warning) => /Cannot update a component.*while rendering/.test(warning)),
    false
  );
});
