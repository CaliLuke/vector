import type { Query, QueryClient } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import type { IWorkspaceMemberMe, TUserPermissions } from "@plane/types";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { isAuthorizationError } from "./authorization-errors";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "./identity";
import { permissionKeys } from "./permissions";
import type { TProjectRosterScope } from "./member-rosters";

export type TProjectAccessScope = TProjectRosterScope & {
  sessionGeneration?: number;
  projectMembershipRole?: TUserPermissions;
  rolesRevision?: string;
};

export function projectRolesRevision(roles: Record<string, TUserPermissions> | undefined) {
  return roles ? JSON.stringify(sortBy(Object.entries(roles), ([projectId]) => projectId)) : undefined;
}

export function captureProjectRolesQuery(
  client: QueryClient,
  accountId: string | undefined,
  workspaceSlug: string,
  membershipRole: TProjectRosterScope["membershipRole"],
  membershipRevision: string | undefined
) {
  const queryKey = permissionKeys.workspaceProjectRoles(accountId, workspaceSlug, membershipRole, membershipRevision);
  const query = client.getQueryCache().find({ queryKey, exact: true });
  const roles =
    query?.state.status === "success" ? client.getQueryData<Record<string, TUserPermissions>>(queryKey) : undefined;
  return { queryKey, query, roles, revision: projectRolesRevision(roles) };
}

function hasAuthorizationError(
  identity: ReturnType<typeof captureIdentityScope>,
  queries: readonly (Query | undefined)[]
) {
  return [identity?.query.state.error, ...queries.map((query) => query?.state.error)].some(isAuthorizationError);
}

function queryInstancesCurrent(
  client: QueryClient,
  keys: ReadonlyArray<readonly unknown[]>,
  queries: readonly (Query | undefined)[]
) {
  return keys.every(
    (key, index) => queries[index] && client.getQueryCache().find({ queryKey: key, exact: true }) === queries[index]
  );
}

function permissionDataMatches(
  scope: TProjectAccessScope,
  membership: IWorkspaceMemberMe | undefined,
  roles: Record<string, TUserPermissions> | undefined
) {
  return (
    membership?.is_active === true &&
    membership.role === scope.membershipRole &&
    membership.updated_at === scope.membershipRevision &&
    (scope.rolesRevision === undefined || projectRolesRevision(roles) === scope.rolesRevision) &&
    (scope.projectMembershipRole === undefined ||
      (scope.projectId !== undefined && roles?.[scope.projectId] === scope.projectMembershipRole)) &&
    (!scope.projectId ||
      (scope.projectRole !== undefined &&
        effectiveProjectRole(scope.membershipRole ?? null, roles, scope.projectId) === scope.projectRole))
  );
}

function permissionQueriesVerified(queries: readonly (Query | undefined)[], hasProject: boolean) {
  return queries[0]?.state.status === "success" && (!hasProject || queries[1]?.state.status === "success");
}

export function captureProjectAccessScope(client: QueryClient, scope: TProjectAccessScope) {
  const identity = captureIdentityScope(client, scope.accountId);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  const needsRoles = Boolean(scope.projectId) || scope.rolesRevision !== undefined;
  const keys = needsRoles ? [membershipKey, rolesKey] : [membershipKey];
  const queries = keys.map((key) => client.getQueryCache().find({ queryKey: key, exact: true }));
  const current = (verified: boolean) => {
    const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
    const roles = client.getQueryData<Record<string, TUserPermissions>>(rolesKey);
    return (
      (verified ? isIdentityScopeVerified(client, identity) : isIdentityScopeCurrent(client, identity)) &&
      (scope.sessionGeneration === undefined || scope.sessionGeneration === getIdentitySessionGeneration(client)) &&
      !hasAuthorizationError(identity, queries) &&
      queryInstancesCurrent(client, keys, queries) &&
      (!verified || permissionQueriesVerified(queries, needsRoles)) &&
      permissionDataMatches(scope, membership, roles)
    );
  };
  return { isVerified: () => current(true), isCurrent: () => current(false) };
}
