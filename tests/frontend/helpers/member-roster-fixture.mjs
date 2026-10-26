import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";

registerWebModuleHooks();

export const { identityKeys } = await import("../../../apps/web/core/queries/identity.ts");
export const { permissionKeys } = await import("../../../apps/web/core/queries/permissions.ts");
export const { projectKeys } = await import("../../../apps/web/core/queries/projects.ts");
export const { ProjectMemberService } =
  await import("../../../apps/web/core/services/project/project-member.service.ts");
export const { WorkspaceService } = await import("../../../apps/web/core/services/workspace.service.ts");
export const { memberRosterKeys } = await import("../../../apps/web/core/queries/member-rosters.ts");
export const {
  inviteWorkspaceOptions,
  addProjectMembersOptions,
  removeProjectMemberOptions,
  removeWorkspaceMemberOptions,
  updateProjectMemberRoleOptions,
  updateWorkspaceMemberRoleOptions,
  workspaceInvitationKeys,
  workspaceInvitationListOptions,
  updateWorkspaceInvitationOptions,
} = await import("../../../apps/web/core/queries/member-roster-mutations.ts");
export const { useWorkspaceInvitations } = await import("../../../apps/web/core/hooks/use-member-roster-mutations.ts");

export function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "actor" });
  const membershipKey = permissionKeys.workspaceMembership("actor", "acme");
  client.setQueryData(membershipKey, { is_active: true, role: 20, updated_at: "revision-2" });
  return { client, membershipKey };
}

export function seedProjectMutationAccess(client, workspaceRole, projectRole, workspaceMembers) {
  client.setQueryData(permissionKeys.workspaceMembership("actor", "acme"), {
    is_active: true,
    role: workspaceRole,
    updated_at: "revision-2",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("actor", "acme", workspaceRole, "revision-2"), {
    p1: projectRole,
  });
  client.setQueryData(
    memberRosterKeys.workspace({
      accountId: "actor",
      workspaceSlug: "acme",
      membershipRole: workspaceRole,
      membershipRevision: "revision-2",
    }),
    workspaceMembers
  );
}

export function seedTargetProjectMutationAccess(client, workspaceRole, targetWorkspaceRole) {
  seedProjectMutationAccess(client, workspaceRole, 20, [
    { id: "actor-membership", member: { id: "actor" }, role: workspaceRole, is_active: true },
    { id: "target-workspace-member", member: { id: "target" }, role: targetWorkspaceRole, is_active: true },
  ]);
}

export function seedProjectMemberships(client, workspaceRole, projectRole, memberships) {
  const projectKey = memberRosterKeys.project({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: workspaceRole,
    membershipRevision: "revision-2",
    projectRole,
    projectId: "p1",
  });
  client.setQueryData(projectKey, memberships);
  return projectKey;
}

export function waitForState(client, queryKey, status) {
  if (client.getQueryState(queryKey)?.status === status) return Promise.resolve();
  return new Promise((resolve) => {
    const unsubscribe = client.getQueryCache().subscribe(() => {
      if (client.getQueryState(queryKey)?.status !== status) return;
      unsubscribe();
      resolve();
    });
  });
}

export function waitForFetch(client, queryKey) {
  if (client.getQueryState(queryKey)?.fetchStatus === "fetching") return Promise.resolve();
  return new Promise((resolve) => {
    const unsubscribe = client.getQueryCache().subscribe(() => {
      if (client.getQueryState(queryKey)?.fetchStatus !== "fetching") return;
      unsubscribe();
      resolve();
    });
  });
}

export function workspaceCascadeQueries(client, projectId = "p1") {
  const workspaceKey = memberRosterKeys.workspace({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: 20,
    membershipRevision: "revision-2",
  });
  client.setQueryData(workspaceKey, [
    { id: "actor-membership", member: { id: "actor" }, role: 20, is_active: true },
    { id: "target-membership", member: { id: "target" }, role: 15, is_active: true },
  ]);
  const projectRosterKey = memberRosterKeys.project({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: 20,
    membershipRevision: "revision-2",
    projectRole: 20,
    projectId,
  });
  const projectScope = {
    accountId: "actor",
    sessionGeneration: 0,
    workspaceSlug: "acme",
    workspaceRole: 20,
    membershipRevision: "revision-2",
    projectRoles: [[projectId, 20]],
    projectId,
    projectMembershipRole: 20,
    projectRole: 20,
  };
  client.setQueryData(permissionKeys.workspaceProjectRoles("actor", "acme", 20, "revision-2"), { [projectId]: 20 });
  const detailKey = projectKeys.detail(projectScope);
  const catalogueKey = projectKeys.workspace(projectScope, "full");
  client.setQueryData(projectRosterKey, [{ id: "project-target", member: "target", role: 15, is_active: true }]);
  client.setQueryData(detailKey, { id: projectId, members: ["target"] });
  client.setQueryData(catalogueKey, [{ id: projectId, members: ["target"] }]);
  const queries = [projectRosterKey, detailKey, catalogueKey].map(
    (queryKey) =>
      new QueryObserver(client, { queryKey, queryFn: failMetadataRefresh, staleTime: Infinity, retry: false })
  );
  const stop = queries.map((query) => query.subscribe(() => {}));
  return { workspaceKey, projectRosterKey, detailKey, catalogueKey, stop };
}

export const failMetadataRefresh = async () => {
  throw new Error("metadata refresh offline");
};
