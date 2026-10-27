import assert from "node:assert/strict";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./react-dom.mjs";
import { React, act, webRequire } from "./react.mjs";
import { seedPermissionQueries } from "./permission-query-fixture.mjs";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";

registerWebModuleHooks();
const { EUserProjectRoles, EUserWorkspaceRoles } = webRequire("@plane/types");
const { currentUserOptions } = await import("../../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../../apps/web/core/queries/permissions.ts");
const { projectKeys } = await import("../../../apps/web/core/queries/projects.ts");
const { workspaceKeys } = await import("../../../apps/web/core/queries/workspaces.ts");
const { WorkspaceService } = await import("../../../apps/web/core/services/workspace.service.ts");

export const user = { id: "member-1" };
export const workspace = { id: "workspace-1", slug: "acme" };

function project(id, workspaceId, sortOrder, values = {}) {
  return { id, workspace: workspaceId, sort_order: sortOrder, archived_at: null, ...values };
}

export function waitForJoinedProjectText(container, expectedText) {
  return new Promise((resolve, reject) => {
    const output = container.querySelector("[data-testid=joined-projects]");
    if (output?.textContent === expectedText) return resolve();
    const observer = new MutationObserver(() => {
      if (container.querySelector("[data-testid=joined-projects]")?.textContent !== expectedText) return;
      observer.disconnect();
      clearTimeout(timeout);
      resolve();
    });
    observer.observe(container, { childList: true, subtree: true, characterData: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(`Joined project output did not become ${JSON.stringify(expectedText)}.`));
    }, 1500);
  });
}

export async function createJoinedProjectFixture(
  t,
  content,
  { roles = { "joined-later": EUserProjectRoles.MEMBER }, includeProjectMetadata = true } = {}
) {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  const projects = [
    project("staleMetadataOnly", workspace.id, 1),
    project("joined-first", workspace.id, 4),
    project("joined-later", workspace.id, 8),
    project("archived", workspace.id, 2, { archived_at: "2026-01-01T00:00:00Z" }),
    project("other-workspace", "workspace-2", 0),
  ];
  client.setQueryData(currentUserOptions().queryKey, user);
  client.setQueryData(workspaceKeys.list(user.id), [workspace]);
  await seedPermissionQueries(client, {
    userId: user.id,
    workspaceSlug: workspace.slug,
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: {
      "joined-first": EUserProjectRoles.MEMBER,
      ...roles,
      removed: EUserProjectRoles.MEMBER,
      archived: EUserProjectRoles.MEMBER,
      "other-workspace": EUserProjectRoles.MEMBER,
    },
  });
  const roleRecord = {
    "joined-first": EUserProjectRoles.MEMBER,
    ...roles,
    removed: EUserProjectRoles.MEMBER,
    archived: EUserProjectRoles.MEMBER,
    "other-workspace": EUserProjectRoles.MEMBER,
  };
  const roleRevision = "2026-10-24T00:00:00Z";
  const scope = {
    accountId: user.id,
    sessionGeneration: 0,
    workspaceSlug: workspace.slug,
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    membershipRevision: roleRevision,
    projectRoles: Object.entries(roleRecord).toSorted(([first], [second]) => first.localeCompare(second)),
  };
  const catalogueKey = projectKeys.workspace(scope, "partial");
  const setProjectDetails = (rows) => client.setQueryData(catalogueKey, rows);
  if (includeProjectMetadata) setProjectDetails(projects);
  const render = async () => environment.render(React.createElement(QueryClientProvider, { client }, content()));
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  return { client, environment, setProjectDetails, render };
}

/** Both the joined selector and creation provider must lose stale membership. */
export async function assertJoinedMembershipRevoked(t, fixture, expected, refreshError = false) {
  if (refreshError)
    t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
      throw new Error("workspace permission read failed");
    });
  await fixture.render();
  const output = () => fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent;
  assert.equal(output(), expected);
  const key = permissionKeys.workspaceMembership(user.id, workspace.slug);
  await act(async () => {
    if (refreshError) await fixture.client.invalidateQueries({ queryKey: key, exact: true });
    else fixture.client.setQueryData(key, { is_active: false, role: null });
  });
  await act(async () => waitForJoinedProjectText(fixture.environment.container, ""));
  if (refreshError) assert.equal(fixture.client.getQueryState(key).status, "error");
  assert.equal(output(), "");
}
