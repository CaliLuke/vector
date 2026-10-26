import assert from "node:assert/strict";
import test from "node:test";
import { createRequire, registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const requireFromWeb = createRequire(path.resolve("apps/web/package.json"));
const atlaskit = [
  "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter",
  "@atlaskit/pragmatic-drag-and-drop/combine",
  "@atlaskit/pragmatic-drag-and-drop/element/pointer-outside-of-preview",
  "@atlaskit/pragmatic-drag-and-drop/element/set-custom-native-drag-preview",
  "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge",
  "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item",
];
const atlaskitResolved = new Map(
  atlaskit.map((specifier) => [specifier, pathToFileURL(requireFromWeb.resolve(specifier)).href])
);
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    return atlaskitResolved.has(specifier)
      ? nextResolve(atlaskitResolved.get(specifier), context)
      : nextResolve(specifier, context);
  },
});
const { React, act } = await fromRoot("tests/frontend/helpers/react.mjs");
const { createQueryRouterFixture } = await fromRoot("tests/frontend/helpers/query-router.mjs");
const { seedPermissionQueries } = await fromRoot("tests/frontend/helpers/permission-query-fixture.mjs");
const { useGroupColumns } = await import("../../apps/web/core/components/issues/issue-layouts/use-group-columns.ts");
const { EditorUserMention } = await import("../../apps/web/core/components/editor/embeds/mentions/user.tsx");
const { identityKeys } = await fromRoot("apps/web/core/queries/identity.ts");
const { workspaceKeys } = await fromRoot("apps/web/core/queries/workspaces.ts");
const { memberRosterKeys } = await fromRoot("apps/web/core/queries/member-rosters.ts");
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const revision = "2026-10-24T00:00:00Z";
const scope = (workspaceSlug) => ({
  accountId: "actor",
  workspaceSlug,
  membershipRole: 15,
  membershipRevision: revision,
});
const member = (id, displayName) => ({
  id: `membership-${id}`,
  member: { id, display_name: displayName, is_bot: false },
  role: 15,
  is_active: true,
});
function Probe(props) {
  const columns = useGroupColumns({ includeNone: true, ...props });
  return React.createElement(
    "ul",
    null,
    columns?.map((column) => React.createElement("li", { key: column.id, "data-member": column.id }, column.name))
  );
}
async function fixture(t) {
  const result = await createQueryRouterFixture(t);
  result.client.setQueryData(identityKeys.currentUser(), { id: "actor" });
  result.client.setQueryData(workspaceKeys.list("actor"), [
    { id: "acme-id", slug: "acme" },
    { id: "beta-id", slug: "beta" },
  ]);
  await seedPermissionQueries(result.client, {
    userId: "actor",
    workspaceSlug: "acme",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  await seedPermissionQueries(result.client, {
    userId: "actor",
    workspaceSlug: "beta",
    workspaceRole: 15,
    projectRoles: {},
  });
  return result;
}
async function render(view, props) {
  await view.render(React.createElement(Probe, props));
}
function names(view) {
  return Array.from(view.environment.container.querySelectorAll("li"), (node) => node.textContent);
}

await test("workspace group headings hide previous members while the next scoped roster loads", async (t) => {
  const pending = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", () => pending.promise);
  const f = await fixture(t);
  f.client.setQueryData(memberRosterKeys.workspace(scope("acme")), [member("member-a", "Acme member")]);
  await render(f, { workspaceSlug: "acme", isWorkspaceLevel: true, groupBy: "assignees" });
  assert.deepEqual(names(f), ["Acme member", "None"]);
  await render(f, { workspaceSlug: "beta", isWorkspaceLevel: true, groupBy: "assignees" });
  assert.deepEqual(names(f), ["None"]);
  await act(async () => {
    pending.resolve([member("member-b", "Beta member")]);
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
  assert.deepEqual(names(f), ["Beta member", "None"]);
});

await test("project assignee groups exclude guest roles while creator groups include them", async (t) => {
  const f = await fixture(t);
  f.client.setQueryData(memberRosterKeys.workspace(scope("acme")), [
    member("normal", "Normal member"),
    member("guest", "Guest member"),
  ]);
  f.client.setQueryData(memberRosterKeys.project({ ...scope("acme"), projectId: "p1", projectRole: 20 }), [
    { id: "normal-project", member: "normal", role: 15, original_role: 15 },
    { id: "guest-project", member: "guest", role: 5, original_role: 5 },
  ]);
  await render(f, { workspaceSlug: "acme", projectId: "p1", isWorkspaceLevel: false, groupBy: "assignees" });
  assert.deepEqual(names(f), ["Normal member", "None"]);
  await render(f, { workspaceSlug: "acme", projectId: "p1", isWorkspaceLevel: false, groupBy: "created_by" });
  assert.deepEqual(names(f), ["Guest member", "Normal member"]);
});

await test("project grouping without an explicit project never reuses a cached project roster", async (t) => {
  const f = await fixture(t);
  f.client.setQueryData(memberRosterKeys.workspace(scope("acme")), [member("normal", "Normal member")]);
  f.client.setQueryData(memberRosterKeys.project({ ...scope("acme"), projectId: "p1", projectRole: 20 }), [
    { id: "normal-project", member: "normal", role: 15, original_role: 15 },
  ]);
  await render(f, { workspaceSlug: "acme", projectId: "p1", isWorkspaceLevel: false, groupBy: "created_by" });
  assert.deepEqual(names(f), ["Normal member"]);

  await render(f, { workspaceSlug: "acme", isWorkspaceLevel: false, groupBy: "created_by" });
  assert.deepEqual(names(f), []);
});

await test("failed workspace roster refresh hides cached member headings", async (t) => {
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => {
    throw new Error("Roster unavailable");
  });
  const f = await fixture(t);
  const queryKey = memberRosterKeys.workspace(scope("acme"));
  f.client.setQueryData(queryKey, [member("normal", "Normal member")]);
  await render(f, { workspaceSlug: "acme", isWorkspaceLevel: true, groupBy: "assignees" });
  assert.deepEqual(names(f), ["Normal member", "None"]);

  await act(async () => {
    await f.client.invalidateQueries({ queryKey, exact: true });
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
  assert.equal(f.client.getQueryState(queryKey)?.status, "error");
  assert.deepEqual(names(f), ["None"]);
});

await test("an unavailable mention summary does not label the user as suspended", async (t) => {
  const pending = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", () => pending.promise);
  const f = await fixture(t);
  f.client.setQueryData(memberRosterKeys.project({ ...scope("acme"), projectId: "p1", projectRole: 20 }), []);
  await f.render(React.createElement(EditorUserMention, { id: "normal" }));
  assert.equal(f.environment.container.textContent, "@user");

  await act(async () => {
    pending.reject(new Error("Roster unavailable"));
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
  assert.equal(f.client.getQueryState(memberRosterKeys.workspace(scope("acme")))?.status, "error");
  assert.equal(f.environment.container.textContent, "@user");
});
