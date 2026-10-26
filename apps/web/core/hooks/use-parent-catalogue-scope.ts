/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "./use-current-user";
import { useIdentityGeneration } from "./use-identity-generation";
import { useProjectPermissions } from "./use-permissions";
import { captureParentAccess } from "@/queries/cycle-module-scope";
import { useCommittedScope } from "./use-committed-scope";
import { currentUserOptions, getCurrentUserId, getIdentitySessionGeneration } from "@/queries/identity";
import { workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "@/queries/catalogue-scope";
import { isWorkspaceRole } from "@/helpers/permission-policy";

function parentAccessReady(
  identity: ReturnType<typeof useCurrentUser>,
  permissions: ReturnType<typeof useProjectPermissions>,
  membership: ReturnType<typeof useProjectPermissions>["membershipQuery"]["data"],
  projectId: string | undefined
) {
  return (
    identity.isSuccess &&
    !permissions.isPending &&
    !permissions.isError &&
    membership?.is_active === true &&
    permissions.roles !== undefined &&
    (!projectId || permissions.projectRole !== undefined)
  );
}

async function retryParentCatalogueAccess(input: {
  client: ReturnType<typeof useQueryClient>;
  isRouteCurrent: () => boolean;
  generation: number;
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  projectScope: ProjectCatalogueScope;
  workspaceScope: WorkspaceCatalogueScope;
}) {
  const isCurrent = () =>
    input.isRouteCurrent() &&
    getIdentitySessionGeneration(input.client) === input.generation &&
    (!input.accountId || getCurrentUserId(input.client) === input.accountId);
  if (!isCurrent()) return false;
  if (captureParentAccess(input.client, input.projectId ? input.projectScope : input.workspaceScope).isVerified())
    return true;
  const user = await input.client.fetchQuery({ ...currentUserOptions(), staleTime: 0 });
  if (!isCurrent() || !user?.id || !input.workspaceSlug) return false;
  const member = await input.client.fetchQuery({
    ...workspaceMembershipOptions(user.id, input.workspaceSlug),
    staleTime: 0,
  });
  if (!isCurrent() || !member.is_active || !isWorkspaceRole(member.role) || !member.updated_at) return false;
  await input.client.fetchQuery({
    ...workspaceProjectRolesOptions(user.id, input.workspaceSlug, member.role, member.updated_at),
    staleTime: 0,
  });
  return isCurrent();
}

/** Native prerequisite state shared by cycle/module collections and their saved issue preferences. */
export function useParentCatalogueScope(workspaceSlug: string | undefined, projectId?: string) {
  const client = useQueryClient(),
    identity = useCurrentUser(),
    generation = useIdentityGeneration(client);
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const accountId = identity.data?.id;
  const projectScope = useMemo<ProjectCatalogueScope>(
    () => ({
      accountId,
      sessionGeneration: generation,
      workspaceSlug,
      projectId,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRole: permissions.projectRole,
      projectMembershipRole: permissions.roles?.[projectId ?? ""],
    }),
    [
      accountId,
      generation,
      workspaceSlug,
      projectId,
      permissions.workspaceRole,
      membership?.updated_at,
      permissions.projectRole,
      permissions.roles,
    ]
  );
  const workspaceScope = useMemo<WorkspaceCatalogueScope>(
    () => ({
      accountId,
      sessionGeneration: generation,
      workspaceSlug,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRoles: permissions.roles ? sortBy(Object.entries(permissions.roles), ([id]) => id) : undefined,
    }),
    [accountId, generation, workspaceSlug, permissions.workspaceRole, membership?.updated_at, permissions.roles]
  );
  const ready = parentAccessReady(identity, permissions, membership, projectId);
  const pending = identity.isPending || permissions.isPending;
  const failed = identity.isError || permissions.isError;
  const denied = identity.isSuccess && !pending && !failed && !ready;
  const active = useCommittedScope(JSON.stringify([accountId, generation, workspaceSlug, projectId]));
  const retry = () =>
    retryParentCatalogueAccess({
      client,
      isRouteCurrent: active,
      generation,
      accountId,
      workspaceSlug,
      projectId,
      projectScope,
      workspaceScope,
    });
  return {
    projectScope,
    workspaceScope,
    isReady: ready,
    isPending: pending,
    isError: failed,
    isAccessDenied: denied,
    retry,
  };
}
