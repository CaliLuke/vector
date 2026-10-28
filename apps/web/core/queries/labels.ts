/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import sortBy from "lodash-es/sortBy.js";
import { PROJECT_LABELS, WORKSPACE_LABELS } from "@plane/constants";
import { EUserPermissions } from "@plane/constants";
import type { IIssueLabel, IIssueLabelTree, IWorkspaceMemberMe, IUserProjectsRole } from "@plane/types";
import { buildTree } from "@plane/utils/array";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "@/queries/identity";
import { workspaceCatalogueScopeReady } from "@/queries/catalogue-scope";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "@/queries/catalogue-scope";
import { permissionKeys } from "@/queries/permissions";
import { IssueLabelService } from "@/services/issue/issue_label.service";

type TLabelPosition = {
  draggingLabelId: string;
  droppedParentId: string | null;
  droppedLabelId: string | undefined;
  dropAtEndOfList: boolean;
};

type TProjectLabelWrite =
  | { type: "create"; data: Partial<IIssueLabel> }
  | { type: "update"; labelId: string; data: Partial<IIssueLabel> }
  | { type: "reorder"; position: TLabelPosition }
  | { type: "delete"; labelId: string };

type TProjectLabelWriteResult = {
  write: TProjectLabelWrite;
  label?: IIssueLabel;
  deleted?: boolean;
};

const service = new IssueLabelService();
const PROJECT_LABEL_READ_ROLES = [EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST] as const;

export const labelKeys = {
  projectPrefix: (workspaceSlug: string | undefined, projectId: string | undefined) =>
    ["PROJECT_LABELS", workspaceSlug, projectId] as const,
  project: (scope: ProjectCatalogueScope) =>
    [
      ...PROJECT_LABELS(scope.workspaceSlug, scope.projectId, scope.projectRole),
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectMembershipRole,
    ] as const,
  workspacePrefix: (workspaceSlug: string | undefined) => ["WORKSPACE_LABELS", workspaceSlug] as const,
  workspace: (scope: WorkspaceCatalogueScope) =>
    [
      ...WORKSPACE_LABELS(scope.workspaceSlug),
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRoles,
    ] as const,
};

export function projectRoleSnapshot(roles: IUserProjectsRole | undefined) {
  return sortBy(Object.entries(roles ?? {}), ([projectId]) => projectId);
}

function orderedLabels(labels: readonly IIssueLabel[]) {
  return sortBy(labels, "sort_order");
}

export function projectLabelTree(labels: readonly IIssueLabel[] | undefined): IIssueLabelTree[] | undefined {
  if (!labels) return undefined;
  return buildTree(sortBy(labels, "sort_order").map((label) => Object.assign({}, label)));
}

function canMoveProjectLabel(labels: readonly IIssueLabel[], draggingLabelId: string, parentId: string | null) {
  const labelMap = new Map(labels.map((label) => [label.id, label]));
  if (parentId && !labelMap.has(parentId)) return false;
  let currentParentId = parentId;
  const visited = new Set<string>();
  while (currentParentId) {
    if (currentParentId === draggingLabelId || visited.has(currentParentId)) return false;
    visited.add(currentParentId);
    currentParentId = labelMap.get(currentParentId)?.parent ?? null;
  }
  return true;
}

function projectLabelSiblings(
  labels: readonly IIssueLabel[],
  parentId: string | null,
  draggingLabelId: string,
  currentParentId: string | null
) {
  return sortBy(labels, "sort_order").filter(
    (label) => label.parent === parentId && !(currentParentId === parentId && label.id === draggingLabelId)
  );
}

function projectLabelSortOrder(
  siblings: readonly IIssueLabel[],
  droppedLabelId: string | undefined,
  dropAtEndOfList: boolean
) {
  let droppedIndex = siblings.findIndex((label) => label.id === droppedLabelId);
  if (dropAtEndOfList || droppedIndex === -1) droppedIndex = siblings.length;
  const previousSortOrder = siblings[droppedIndex - 1]?.sort_order;
  const nextSortOrder = siblings[droppedIndex]?.sort_order;
  if (previousSortOrder !== undefined && nextSortOrder !== undefined) return (previousSortOrder + nextSortOrder) / 2;
  if (nextSortOrder !== undefined) return nextSortOrder - 10000;
  if (previousSortOrder !== undefined) return previousSortOrder + 10000;
  return 65535;
}

export function projectLabelPositionPatch(
  labels: readonly IIssueLabel[] | undefined,
  { draggingLabelId, droppedParentId, droppedLabelId, dropAtEndOfList }: TLabelPosition
): Partial<IIssueLabel> | undefined {
  if (!labels) return undefined;
  const currentLabel = labels.find((label) => label.id === draggingLabelId);
  if (!currentLabel || (currentLabel.parent === droppedParentId && !droppedLabelId)) return undefined;
  if (droppedLabelId === draggingLabelId || !canMoveProjectLabel(labels, draggingLabelId, droppedParentId))
    return undefined;

  const siblings = projectLabelSiblings(labels, droppedParentId, draggingLabelId, currentLabel.parent);
  const patch: Partial<IIssueLabel> = { parent: droppedParentId };
  if (siblings.length > 0) patch.sort_order = projectLabelSortOrder(siblings, droppedLabelId, dropAtEndOfList);
  return patch;
}

function sameProjectRoleSnapshot(
  roles: IUserProjectsRole | undefined,
  snapshot: WorkspaceCatalogueScope["projectRoles"]
) {
  return JSON.stringify(projectRoleSnapshot(roles)) === JSON.stringify(snapshot);
}

function projectScopeIsReady(scope: ProjectCatalogueScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    isWorkspaceRole(scope.workspaceRole) &&
    scope.membershipRevision &&
    scope.projectRole !== undefined &&
    scope.projectMembershipRole !== undefined &&
    PROJECT_LABEL_READ_ROLES.includes(scope.projectRole)
  );
}

function hasLabelAccessIdentity(
  scope: Pick<ProjectCatalogueScope, "accountId" | "workspaceSlug" | "workspaceRole" | "membershipRevision">
) {
  return Boolean(
    scope.accountId && scope.workspaceSlug && isWorkspaceRole(scope.workspaceRole) && scope.membershipRevision
  );
}

function isCapturedLabelIdentityCurrent(
  client: QueryClient,
  identity: ReturnType<typeof captureIdentityScope>,
  sessionGeneration: number
) {
  return Boolean(identity && identity.generation === sessionGeneration && isIdentityScopeCurrent(client, identity));
}

function isCapturedLabelQueryCurrent(client: QueryClient, key: readonly unknown[], query: Query | undefined) {
  return Boolean(query && client.getQueryCache().find({ queryKey: key, exact: true }) === query);
}

function isWorkspaceMembershipCurrent(
  client: QueryClient,
  key: readonly unknown[],
  query: Query | undefined,
  scope: Pick<ProjectCatalogueScope, "workspaceRole" | "membershipRevision">
) {
  const membership = client.getQueryData<IWorkspaceMemberMe>(key);
  return Boolean(
    query &&
    client.getQueryCache().find({ queryKey: key, exact: true }) === query &&
    query.state.status === "success" &&
    membership?.is_active === true &&
    membership.role === scope.workspaceRole &&
    membership.updated_at === scope.membershipRevision
  );
}

function isWorkspaceProjectRolesQueryCurrent(client: QueryClient, key: readonly unknown[], query: Query | undefined) {
  return Boolean(
    query && client.getQueryCache().find({ queryKey: key, exact: true }) === query && query.state.status === "success"
  );
}

function captureLabelAccessScope(
  client: QueryClient,
  key: readonly unknown[],
  scope: Pick<
    ProjectCatalogueScope,
    "accountId" | "workspaceSlug" | "workspaceRole" | "membershipRevision" | "sessionGeneration"
  >
) {
  const identity = captureIdentityScope(client, scope.accountId);
  const labelQuery = client.getQueryCache().find({ queryKey: key, exact: true });
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision
  );
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const rolesQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
  const isCurrent = () => {
    return (
      hasLabelAccessIdentity(scope) &&
      isCapturedLabelIdentityCurrent(client, identity, scope.sessionGeneration) &&
      isCapturedLabelQueryCurrent(client, key, labelQuery) &&
      isWorkspaceMembershipCurrent(client, membershipKey, membershipQuery, scope) &&
      isWorkspaceProjectRolesQueryCurrent(client, rolesKey, rolesQuery)
    );
  };
  return {
    key,
    isCurrent,
    isVerifiedCurrent: () => isCurrent() && isIdentityScopeVerified(client, identity),
    rolesKey,
  };
}

function captureProjectLabelScope(client: QueryClient, scope: ProjectCatalogueScope) {
  const key = labelKeys.project(scope);
  const access = captureLabelAccessScope(client, key, scope);
  const hasProjectAccess = () => {
    const roles = client.getQueryData<IUserProjectsRole>(access.rolesKey);
    return (
      projectScopeIsReady(scope) &&
      roles?.[scope.projectId!] === scope.projectMembershipRole &&
      effectiveProjectRole(scope.workspaceRole, roles, scope.projectId) === scope.projectRole
    );
  };
  const isCurrent = () => access.isCurrent() && hasProjectAccess();
  const isVerifiedCurrent = () => access.isVerifiedCurrent() && hasProjectAccess();
  return { key, isCurrent, isVerifiedCurrent };
}

function captureWorkspaceLabelScope(client: QueryClient, scope: WorkspaceCatalogueScope) {
  const key = labelKeys.workspace(scope);
  const access = captureLabelAccessScope(client, key, scope);
  const hasProjectRoleSnapshot = () =>
    sameProjectRoleSnapshot(client.getQueryData<IUserProjectsRole>(access.rolesKey), scope.projectRoles);
  const isCurrent = () => access.isCurrent() && hasProjectRoleSnapshot();
  const isVerifiedCurrent = () => access.isVerifiedCurrent() && hasProjectRoleSnapshot();
  return { key, isCurrent, isVerifiedCurrent };
}

export function projectLabelOptions(scope: ProjectCatalogueScope) {
  const queryKey = labelKeys.project(scope);
  return queryOptions({
    queryKey,
    select: orderedLabels,
    queryFn: projectScopeIsReady(scope)
      ? async ({ client, signal }) => {
          const captured = captureProjectLabelScope(client, scope);
          if (!captured.isVerifiedCurrent()) throw new Error("The project label scope changed or is unverified.");
          const labels = await service.getProjectLabels(scope.workspaceSlug!, scope.projectId!, signal);
          if (!captured.isVerifiedCurrent()) throw new Error("The project label scope changed or is unverified.");
          return labels;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

export function workspaceLabelOptions(scope: WorkspaceCatalogueScope) {
  const queryKey = labelKeys.workspace(scope);
  const enabled = workspaceCatalogueScopeReady(scope);
  return queryOptions({
    queryKey,
    select: orderedLabels,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const captured = captureWorkspaceLabelScope(client, scope);
          if (!captured.isVerifiedCurrent()) throw new Error("The workspace label scope changed or is unverified.");
          const labels = await service.getWorkspaceIssueLabels(scope.workspaceSlug!, signal);
          if (!captured.isVerifiedCurrent()) throw new Error("The workspace label scope changed or is unverified.");
          return labels;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

function withoutLabelBranch(labels: readonly IIssueLabel[], deletedLabelId: string) {
  const children = new Map<string, string[]>();
  for (const label of labels) {
    if (label.parent) {
      const siblings = children.get(label.parent) ?? [];
      siblings.push(label.id);
      children.set(label.parent, siblings);
    }
  }
  const removed = new Set<string>();
  const pending = [deletedLabelId];
  while (pending.length) {
    const id = pending.pop();
    if (id === undefined || removed.has(id)) continue;
    removed.add(id);
    pending.push(...(children.get(id) ?? []));
  }
  return labels.filter((label) => !removed.has(label.id));
}

async function reconcileProjectLabelWrite(
  client: QueryClient,
  scope: ProjectCatalogueScope,
  result: TProjectLabelWriteResult,
  originalQuery: Query | undefined
) {
  const queryKey = labelKeys.project(scope);
  const isOriginalQuery = () =>
    originalQuery !== undefined && client.getQueryCache().find({ queryKey, exact: true }) === originalQuery;
  if (!isOriginalQuery()) return;
  const captured = captureProjectLabelScope(client, scope);
  if (!captured.isCurrent()) return;
  await client.cancelQueries({ queryKey, exact: true });
  if (!isOriginalQuery() || !captured.isCurrent()) return;
  let hasCanonicalLabels = false;
  if (captured.isCurrent()) {
    const labels = client.getQueryData<IIssueLabel[]>(queryKey);
    if (labels !== undefined) {
      hasCanonicalLabels = !result.deleted;
      if (result.deleted && result.write.type === "delete") {
        // The 204 receipt has no descendant IDs. Remove the known branch now,
        // then refetch to reconcile concurrent parenting or newly added children.
        client.setQueryData(queryKey, withoutLabelBranch(labels, result.write.labelId));
      } else if (result.label) {
        const index = labels.findIndex((label) => label.id === result.label?.id);
        if (index < 0 && result.write.type === "create") {
          client.setQueryData(queryKey, [...labels, result.label]);
        } else if (index >= 0) {
          client.setQueryData(
            queryKey,
            labels.map((label) => (label.id === result.label?.id ? result.label : label))
          );
        }
      }
    }
  }
  await Promise.all([
    ...(hasCanonicalLabels ? [] : [client.invalidateQueries({ queryKey, exact: true }, { throwOnError: false })]),
    client.invalidateQueries({ queryKey: labelKeys.workspacePrefix(scope.workspaceSlug) }, { throwOnError: false }),
  ]);
}

export function projectLabelMutationOptions(client: QueryClient, scope: ProjectCatalogueScope) {
  return mutationOptions({
    mutationKey: [...labelKeys.projectPrefix(scope.workspaceSlug, scope.projectId), "write"],
    scope: { id: `project-labels/${scope.workspaceSlug}/${scope.projectId}` },
    retry: false,
    onMutate: () => client.getQueryCache().find({ queryKey: labelKeys.project(scope), exact: true }),
    mutationFn: async (write: TProjectLabelWrite): Promise<TProjectLabelWriteResult> => {
      if (!projectScopeIsReady(scope)) throw new Error("Project label access is unavailable.");
      const canCreate =
        scope.workspaceRole === EUserWorkspaceRoles.ADMIN || scope.workspaceRole === EUserWorkspaceRoles.MEMBER;
      if (write.type === "create" ? !canCreate : scope.projectRole !== EUserPermissions.ADMIN)
        throw new Error(
          write.type === "create"
            ? "Workspace member access is required to create labels."
            : "Project administrator access is required to change labels."
        );
      const captured = captureProjectLabelScope(client, scope);
      if (!captured.isVerifiedCurrent()) throw new Error("The project label scope changed or is unverified.");

      switch (write.type) {
        case "create":
          return { write, label: await service.createIssueLabel(scope.workspaceSlug!, scope.projectId!, write.data) };
        case "update":
          return {
            write,
            label: await service.patchIssueLabel(scope.workspaceSlug!, scope.projectId!, write.labelId, write.data),
          };
        case "delete":
          await service.deleteIssueLabel(scope.workspaceSlug!, scope.projectId!, write.labelId);
          return { write, deleted: true };
      }
      const labels = client.getQueryData<IIssueLabel[]>(captured.key);
      const patch = projectLabelPositionPatch(labels, write.position);
      if (!patch) return { write };
      return {
        write,
        label: await service.patchIssueLabel(
          scope.workspaceSlug!,
          scope.projectId!,
          write.position.draggingLabelId,
          patch
        ),
      };
    },
    onSuccess: (result, _write, originalQuery) => reconcileProjectLabelWrite(client, scope, result, originalQuery),
  });
}

export async function invalidateProjectLabelCatalogues(client: QueryClient, workspaceSlug: string, projectId: string) {
  await Promise.all([
    client.invalidateQueries({ queryKey: labelKeys.projectPrefix(workspaceSlug, projectId) }),
    client.invalidateQueries({ queryKey: labelKeys.workspacePrefix(workspaceSlug) }),
  ]);
}
