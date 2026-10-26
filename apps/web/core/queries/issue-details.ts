/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type { TIssue, TIssueServiceType, TIssueSubIssues, TUserPermissions } from "@plane/types";
import { EIssueServiceType, EUserPermissions } from "@plane/types";
import { captureItemReadScope } from "./work-item-read-scope";
import { captureProjectAccessScope } from "./project-access-scope";
import type { TProjectRosterScope } from "./member-rosters";
import { getIdentitySessionGeneration } from "./identity";
import { isItemAccessDenied, itemAccessOptions, readVisibleItem } from "./work-item-access";
import type { ItemAccessScope } from "./work-item-access";
import { IssueService } from "@/services/issue/issue.service";
import { isWorkspaceRole } from "@/helpers/permission-policy";

export type TIssueDetailScope = TProjectRosterScope & {
  sessionGeneration: number;
  projectMembershipRole: TProjectRosterScope["projectRole"];
  serviceType: TIssueServiceType;
};

export const issueDetailKeys = {
  detail: (scope: TIssueDetailScope, issueId: string | undefined) =>
    [
      "issue-details",
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.serviceType,
      "detail",
      issueId,
    ] as const,
  byIdentifier: (scope: TIssueDetailScope, projectIdentifier: string | undefined, sequenceId: string | undefined) =>
    [
      "issue-details",
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.serviceType,
      "identifier",
      projectIdentifier,
      sequenceId,
    ] as const,
  subItems: (scope: TIssueDetailScope, parentIssueId: string | undefined) =>
    [
      "issue-details",
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.serviceType,
      "sub-items",
      parentIssueId,
    ] as const,
};

function scopeReady(scope: TIssueDetailScope, issueId?: string) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.sessionGeneration >= 0 &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision &&
    scope.projectId &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined &&
    (issueId === undefined || issueId.length > 0)
  );
}

function detailItemAccessScope(scope: TIssueDetailScope, issueId?: string): ItemAccessScope {
  return {
    accountId: scope.accountId,
    generation: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    issueId,
    serviceType: scope.serviceType,
  };
}

function readCurrent(client: QueryClient, scope: TIssueDetailScope, queryKey: readonly unknown[], itemId?: string) {
  const access = captureProjectAccessScope(client, scope);
  const owner = captureItemReadScope(client, scope, queryKey);
  return () =>
    getIdentitySessionGeneration(client) === scope.sessionGeneration &&
    access.isVerified() &&
    owner() &&
    (!itemId || !isItemAccessDenied(client, detailItemAccessScope(scope, itemId)));
}

function assertCanonicalIssue(issue: TIssue | undefined, scope: TIssueDetailScope, issueId?: string) {
  if (
    !issue ||
    (issue.id !== issueId && issueId !== undefined) ||
    issue.project_id !== scope.projectId ||
    (scope.serviceType === EIssueServiceType.EPICS && issue.is_epic !== true)
  )
    throw new Error("The server returned a different work item.");
  return issue;
}

export function issueDetailOptions(
  client: QueryClient,
  scope: TIssueDetailScope,
  issueId: string | undefined,
  enabled = true
) {
  const queryKey = issueDetailKeys.detail(scope, issueId);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    retry: false,
    queryFn:
      enabled && issueId && scopeReady(scope, issueId)
        ? async ({ signal }) => {
            const current = readCurrent(client, scope, queryKey, issueId);
            return readVisibleItem(client, detailItemAccessScope(scope, issueId), current, async () => {
              const issue = await new IssueService(scope.serviceType).retrieve(
                scope.workspaceSlug!,
                scope.projectId!,
                issueId,
                { expand: "parent" },
                { signal }
              );
              if (!current()) throw new Error("Work item access changed while loading.");
              return assertCanonicalIssue(issue, scope, issueId);
            });
          }
        : skipToken,
  });
}

export function issueByIdentifierOptions(
  client: QueryClient,
  scope: TIssueDetailScope,
  projectIdentifier: string | undefined,
  sequenceId: string | undefined,
  enabled = true
) {
  const queryKey = issueDetailKeys.byIdentifier(scope, projectIdentifier, sequenceId);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    retry: false,
    queryFn:
      enabled && projectIdentifier && sequenceId && scopeReady(scope)
        ? async ({ signal }) => {
            const current = readCurrent(client, scope, queryKey);
            const access = captureProjectAccessScope(client, scope);
            const query = client.getQueryCache().find({ queryKey, exact: true });
            const stillCurrent = () =>
              current() && client.getQueryCache().find({ queryKey, exact: true }) === query && !signal.aborted;
            if (!stillCurrent()) throw new Error("Work item access is unavailable.");
            const issue = await new IssueService(scope.serviceType).retrieveWithIdentifier(
              scope.workspaceSlug!,
              projectIdentifier,
              sequenceId,
              { expand: "parent" },
              { signal }
            );
            if (!stillCurrent() || !access.isVerified()) throw new Error("Work item access changed while loading.");
            if (issue.project_id !== scope.projectId || String(issue.sequence_id) !== sequenceId)
              throw new Error("The server returned a different work item.");
            const canonical = assertCanonicalIssue(issue, scope);
            if (isItemAccessDenied(client, detailItemAccessScope(scope, canonical.id)))
              throw new Error("Work item access changed while loading.");
            return canonical;
          }
        : skipToken,
  });
}

export function issueSubItemsOptions(
  client: QueryClient,
  scope: TIssueDetailScope,
  parentIssueId: string | undefined,
  enabled = true
) {
  const queryKey = issueDetailKeys.subItems(scope, parentIssueId);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    retry: false,
    queryFn:
      enabled && parentIssueId && scopeReady(scope, parentIssueId)
        ? async ({ signal }) => {
            const current = readCurrent(client, scope, queryKey, parentIssueId);
            return readVisibleItem(client, detailItemAccessScope(scope, parentIssueId), current, async () => {
              const result = await new IssueService(scope.serviceType).subIssues(
                scope.workspaceSlug!,
                scope.projectId!,
                parentIssueId,
                undefined,
                { signal }
              );
              if (!current()) throw new Error("Work item access changed while loading sub-items.");
              for (const issue of allSubItems(result)) {
                if (!issue.id || (issue.parent_id && issue.parent_id !== parentIssueId))
                  throw new Error("The server returned a sub-item outside this parent.");
              }
              return result;
            });
          }
        : skipToken,
  });
}

export function allSubItems(result: TIssueSubIssues | undefined): TIssue[] {
  const rows = result?.sub_issues;
  return Array.isArray(rows) ? rows : rows ? Object.values(rows).flat() : [];
}

type TIssueDetailQueryOwner = {
  query: Query;
  queryKey: readonly unknown[];
  issueId: string;
  projectId: string;
  serviceType: TIssueServiceType;
  isCurrent: () => boolean;
};

function isProjectRole(value: unknown): value is TUserPermissions {
  return value === EUserPermissions.ADMIN || value === EUserPermissions.MEMBER || value === EUserPermissions.GUEST;
}

function isServiceType(value: unknown): value is TIssueServiceType {
  return value === EIssueServiceType.ISSUES || value === EIssueServiceType.EPICS;
}

function hasStringId(value: unknown): value is { id: string } {
  return typeof value === "object" && value !== null && "id" in value && typeof value.id === "string";
}

function queryIssueId(query: Query): string | undefined {
  const key = query.queryKey;
  if (key[10] === "detail") return typeof key[11] === "string" ? key[11] : undefined;
  return key[10] === "identifier" && hasStringId(query.state.data) ? query.state.data.id : undefined;
}

function capturedAccessScope(query: Query, scope: TIssueDetailScope) {
  const key = query.queryKey;
  const projectId = key[6];
  const workspaceRole = key[4];
  const membershipRevision = key[5];
  const projectMembershipRole = key[7];
  const projectRole = key[8];
  const serviceType = key[9];
  if (
    key[1] !== scope.accountId ||
    key[2] !== scope.sessionGeneration ||
    key[3] !== scope.workspaceSlug ||
    !isWorkspaceRole(workspaceRole) ||
    workspaceRole !== scope.membershipRole ||
    typeof membershipRevision !== "string" ||
    membershipRevision !== scope.membershipRevision ||
    typeof projectId !== "string" ||
    projectId !== scope.projectId ||
    !isProjectRole(projectMembershipRole) ||
    !isProjectRole(projectRole) ||
    !isServiceType(serviceType) ||
    serviceType !== scope.serviceType
  )
    return undefined;
  return {
    projectId,
    serviceType,
    scope: {
      ...scope,
      projectId,
      membershipRole: workspaceRole,
      projectMembershipRole,
      projectRole,
    } satisfies TIssueDetailScope,
  };
}

function capturedSubItemScope(query: Query, scope: TIssueDetailScope) {
  const access = capturedAccessScope(query, scope);
  const parentId = query.queryKey[11];
  return query.queryKey[10] === "sub-items" && typeof parentId === "string" && access
    ? { ...access, parentId }
    : undefined;
}

/** Keep reconciliation attached to the exact query instances and access captured before a write. */
function captureIssueQueryOwnership(
  client: QueryClient,
  query: Query,
  accessScope: TIssueDetailScope,
  itemScope: ItemAccessScope
) {
  const { queryKey } = query;
  const access = captureProjectAccessScope(client, accessScope);
  const itemKey = itemAccessOptions(itemScope).queryKey;
  const itemQuery = client.getQueryCache().find({ queryKey: itemKey, exact: true });
  return () =>
    client.getQueryCache().find({ queryKey, exact: true }) === query &&
    client.getQueryCache().find({ queryKey: itemKey, exact: true }) === itemQuery &&
    getIdentitySessionGeneration(client) === accessScope.sessionGeneration &&
    access.isCurrent() &&
    !isItemAccessDenied(client, itemScope);
}

/** Capture only existing exact detail/identifier queries before a write begins. */
export function captureIssueDetailReconciliation(
  client: QueryClient,
  scope: TIssueDetailScope,
  issueIds: readonly string[]
) {
  const expectedIds = new Set(issueIds);
  const queries = client.getQueryCache().findAll({
    predicate: (query) => {
      if (query.queryKey[0] !== "issue-details") return false;
      const owner = capturedAccessScope(query, scope);
      const issueId = queryIssueId(query);
      return owner !== undefined && issueId !== undefined && expectedIds.has(issueId);
    },
  });

  const owners: TIssueDetailQueryOwner[] = queries.flatMap((query) => {
    const { queryKey } = query;
    const issueId = queryIssueId(query);
    const captured = capturedAccessScope(query, scope);
    if (!issueId || !captured) return [];
    const { projectId, serviceType, scope: accessScope } = captured;
    const itemScope = {
      accountId: scope.accountId,
      generation: scope.sessionGeneration,
      workspaceSlug: scope.workspaceSlug,
      projectId,
      issueId,
      serviceType,
    };
    return [
      {
        query,
        queryKey,
        issueId,
        projectId,
        serviceType,
        isCurrent: captureIssueQueryOwnership(client, query, accessScope, itemScope),
      },
    ];
  });

  const currentOwners = () => owners.filter((owner) => owner.isCurrent());
  return {
    isCurrent: () => currentOwners().length > 0,
    reconcile: async (record: TIssue | undefined) => {
      const current = currentOwners();
      await Promise.all(
        current.map((owner) => client.cancelQueries({ queryKey: owner.queryKey, exact: true }).catch(() => undefined))
      );
      if (record === undefined) {
        await Promise.allSettled(
          currentOwners().map((owner) => client.invalidateQueries({ queryKey: owner.queryKey, exact: true }))
        );
        return 0;
      }
      let applied = 0;
      for (const owner of currentOwners()) {
        if (
          record.id !== owner.issueId ||
          record.project_id !== owner.projectId ||
          (owner.serviceType === EIssueServiceType.EPICS && record.is_epic !== true)
        )
          continue;
        if (owner.queryKey[10] === "identifier") {
          if (String(record.sequence_id) !== String(owner.queryKey[12])) {
            client.removeQueries({ queryKey: owner.queryKey, exact: true });
            continue;
          }
        }
        client.setQueryData(owner.queryKey, record);
        applied++;
      }
      return applied;
    },
    refresh: async () => {
      await Promise.allSettled(
        currentOwners().map((owner) => client.invalidateQueries({ queryKey: owner.queryKey, exact: true }))
      );
    },
    remove: () => {
      for (const owner of currentOwners()) client.removeQueries({ queryKey: owner.queryKey, exact: true });
    },
  };
}

/** Capture exact parent sub-item queries before a hierarchy relation write. */
export function captureIssueSubItemsReconciliation(
  client: QueryClient,
  scope: TIssueDetailScope,
  parentIssueIds: readonly string[]
) {
  const parentIds = new Set(parentIssueIds);
  const queries = client.getQueryCache().findAll({
    predicate: (query) => {
      if (query.queryKey[0] !== "issue-details") return false;
      const owner = capturedSubItemScope(query, scope);
      return owner !== undefined && parentIds.has(owner.parentId);
    },
  });
  const owners = queries.flatMap((query) => {
    const { queryKey } = query;
    const captured = capturedSubItemScope(query, scope);
    if (!captured) return [];
    const { projectId, parentId, scope: accessScope } = captured;
    const itemScope = {
      accountId: scope.accountId,
      generation: scope.sessionGeneration,
      workspaceSlug: scope.workspaceSlug,
      projectId,
      issueId: parentId,
      serviceType: scope.serviceType,
    };
    return [
      {
        query,
        queryKey,
        isCurrent: captureIssueQueryOwnership(client, query, accessScope, itemScope),
      },
    ];
  });
  return {
    refresh: async () => {
      await Promise.allSettled(
        owners
          .filter((owner) => owner.isCurrent())
          .map((owner) => client.invalidateQueries({ queryKey: owner.queryKey, exact: true }))
      );
    },
  };
}
