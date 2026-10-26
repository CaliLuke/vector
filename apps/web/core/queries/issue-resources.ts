/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type {
  TIssueAttachment,
  TIssue,
  TIssueLink,
  TIssueRelation,
  TIssueRelationTypes,
  TIssueServiceType,
} from "@plane/types";
import { REVERSE_RELATIONS } from "@plane/constants";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import type { TProjectRosterScope } from "@/queries/member-rosters";
import { isItemAccessDenied, readVisibleItem } from "@/queries/work-item-access";
import type { ItemAccessScope } from "@/queries/work-item-access";
import { IssueAttachmentService } from "@/services/issue/issue_attachment.service";
import { IssueService } from "@/services/issue/issue.service";
import { IssueRelationService } from "@/services/issue/issue_relation.service";
import { refreshIssueActivity } from "@/queries/issue-discussion";

export type TIssueResourceScope = TProjectRosterScope & {
  sessionGeneration: number;
  projectMembershipRole: TProjectRosterScope["projectRole"];
  serviceType: TIssueServiceType;
  issueId: string | undefined;
};

type ResourceName = "attachments" | "links" | "relations";

export const issueResourceKeys = {
  collection: (scope: TIssueResourceScope, resource: ResourceName) =>
    [
      "issue-resources",
      resource,
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.serviceType,
      scope.issueId,
    ] as const,
  prefix: ["issue-resources"] as const,
};

function isScopeReady(scope: TIssueResourceScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision &&
    scope.projectId &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined &&
    scope.issueId
  );
}

function captureResourceAccess(client: QueryClient, scope: TIssueResourceScope) {
  const access = captureProjectAccessScope(client, scope);
  const itemScope = toItemAccessScope(scope);
  const isLineageCurrent = () => isScopeReady(scope) && access.isCurrent() && !isItemAccessDenied(client, itemScope);
  return {
    isLineageCurrent,
    isVerified: () => isLineageCurrent() && access.isVerified(),
  };
}

function toItemAccessScope(scope: TIssueResourceScope): ItemAccessScope {
  return {
    accountId: scope.accountId,
    generation: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    issueId: scope.issueId,
    serviceType: scope.serviceType,
  };
}

function captureCollectionQuery(client: QueryClient, queryKey: readonly unknown[]) {
  const query = client.getQueryCache().find({ queryKey, exact: true });
  return {
    query,
    isCurrent: () => client.getQueryCache().find({ queryKey, exact: true }) === query,
  };
}

function captureMutationOwner(client: QueryClient, scope: TIssueResourceScope, resource: ResourceName) {
  const queryKey = issueResourceKeys.collection(scope, resource);
  const access = captureResourceAccess(client, scope);
  const collection = captureCollectionQuery(client, queryKey);
  return {
    queryKey,
    query: collection.query,
    isVerified: access.isVerified,
    isCurrent: access.isLineageCurrent,
    isQueryCurrent: collection.isCurrent,
  };
}

function isSameCollectionQuery(client: QueryClient, queryKey: readonly unknown[], query: Query | undefined) {
  return Boolean(query && client.getQueryCache().find({ queryKey, exact: true }) === query);
}

function collectionOptions<TData>(
  client: QueryClient,
  scope: TIssueResourceScope,
  resource: ResourceName,
  load: (signal: AbortSignal) => Promise<TData>
) {
  const queryKey = issueResourceKeys.collection(scope, resource);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    retry: false,
    queryFn: isScopeReady(scope)
      ? async ({ signal }) => {
          const owner = captureMutationOwner(client, scope, resource);
          if (!owner.isVerified() || !owner.query || !owner.isQueryCurrent())
            throw new Error("The work item access scope is unavailable.");
          const isReadCurrent = () => !signal.aborted && owner.isVerified() && owner.isQueryCurrent();
          const result = await readVisibleItem(client, toItemAccessScope(scope), isReadCurrent, () => load(signal));
          if (!isReadCurrent()) throw new Error("The work item access scope changed while loading.");
          return result;
        }
      : skipToken,
  });
}

export function issueAttachmentsOptions(client: QueryClient, scope: TIssueResourceScope) {
  const service = new IssueAttachmentService(scope.serviceType);
  return collectionOptions(client, scope, "attachments", (signal) =>
    service.getIssueAttachments(scope.workspaceSlug!, scope.projectId!, scope.issueId!, { signal })
  );
}

export function issueLinksOptions(client: QueryClient, scope: TIssueResourceScope) {
  return collectionOptions(client, scope, "links", async (signal) => {
    const parent = await fetchVisibleParentProjection<{ id: string; project_id: string; issue_link?: TIssueLink[] }>(
      scope,
      "issue_link",
      signal
    );
    if (!Array.isArray(parent.issue_link)) throw new Error("The visible work item has no canonical links collection.");
    return parent.issue_link;
  });
}

export function issueRelationsOptions(client: QueryClient, scope: TIssueResourceScope) {
  return collectionOptions(client, scope, "relations", async (signal) => {
    const parent = await fetchVisibleParentProjection<
      TIssue & {
        issue_relation?: Array<TIssue & { relation_type?: TIssueRelationTypes }>;
        issue_related?: Array<TIssue & { relation_type?: TIssueRelationTypes }>;
      }
    >(scope, "issue_relation,issue_related", signal);
    return groupVisibleIssueRelations(parent);
  });
}

function groupVisibleIssueRelations(
  parent: TIssue & {
    issue_relation?: Array<TIssue & { relation_type?: TIssueRelationTypes }>;
    issue_related?: Array<TIssue & { relation_type?: TIssueRelationTypes }>;
  }
): TIssueRelation {
  const groups: TIssueRelation = { blocking: [], blocked_by: [], duplicate: [], relates_to: [] };
  for (const relation of parent.issue_relation ?? []) {
    if (relation.relation_type) groups[relation.relation_type].push(relation);
  }
  for (const relation of parent.issue_related ?? []) {
    if (relation.relation_type) groups[REVERSE_RELATIONS[relation.relation_type]].push(relation);
  }
  return groups;
}

async function fetchVisibleParentProjection<TParent extends { id?: string; project_id?: string | null }>(
  scope: TIssueResourceScope,
  expand: "issue_link" | "issue_relation,issue_related",
  signal: AbortSignal
) {
  const service = new IssueService(scope.serviceType);
  const response = await service.get(
    `/api/workspaces/${scope.workspaceSlug}/projects/${scope.projectId}/${scope.serviceType}/${scope.issueId}/`,
    {
      params: {
        expand,
        fields: expand === "issue_link" ? "id,project_id,issue_link" : "id,project_id,issue_relation,issue_related",
      },
      signal,
    }
  );
  const parent: TParent | undefined = response.data;
  if (!parent || parent.id !== scope.issueId || parent.project_id !== scope.projectId)
    throw new Error("The server returned a different work item.");
  return parent;
}

type TIssueLinkWrite = {
  kind: "create" | "update" | "delete";
  linkId?: string;
  data?: Partial<TIssueLink>;
};

function reconcileCollection<TData>(
  client: QueryClient,
  owner: ReturnType<typeof captureMutationOwner>,
  reconcile: (previous: TData | undefined) => TData | undefined
) {
  if (
    !owner.query ||
    !owner.isCurrent() ||
    !owner.isQueryCurrent() ||
    !isSameCollectionQuery(client, owner.queryKey, owner.query)
  )
    return;
  const previous = client.getQueryData<TData>(owner.queryKey);
  const next = reconcile(previous);
  if (
    next !== undefined &&
    owner.isCurrent() &&
    owner.isQueryCurrent() &&
    isSameCollectionQuery(client, owner.queryKey, owner.query)
  )
    client.setQueryData(owner.queryKey, next);
}

async function refreshAcceptedActivity(
  client: QueryClient,
  scope: TIssueResourceScope,
  owner: ReturnType<typeof captureMutationOwner>
) {
  if (!scope.accountId || !owner.isCurrent()) return;
  await refreshIssueActivity(
    client,
    {
      workspaceSlug: scope.workspaceSlug!,
      projectId: scope.projectId!,
      issueId: scope.issueId!,
      serviceType: scope.serviceType,
    },
    {
      accountId: scope.accountId,
      generation: scope.sessionGeneration,
      isCurrent: owner.isCurrent,
    }
  ).catch(() => undefined);
}

async function cancelOwnedCollection(client: QueryClient, owner: ReturnType<typeof captureMutationOwner>) {
  if (!owner.isCurrent()) return undefined;
  const query = owner.query
    ? owner.isQueryCurrent() && isSameCollectionQuery(client, owner.queryKey, owner.query)
      ? owner.query
      : undefined
    : client.getQueryCache().find({ queryKey: owner.queryKey, exact: true });
  if (!query || !isSameCollectionQuery(client, owner.queryKey, query)) return undefined;
  await client.cancelQueries({ queryKey: owner.queryKey, exact: true });
  return owner.isCurrent() && isSameCollectionQuery(client, owner.queryKey, query) ? query : undefined;
}

async function invalidateCapturedCollection(
  client: QueryClient,
  owner: ReturnType<typeof captureMutationOwner>,
  query: Query
) {
  if (!owner.isCurrent() || !isSameCollectionQuery(client, query.queryKey, query)) return;
  await client.cancelQueries({ queryKey: query.queryKey, exact: true });
  if (owner.isCurrent() && isSameCollectionQuery(client, query.queryKey, query))
    await client.invalidateQueries({ queryKey: query.queryKey, exact: true });
}

async function refreshAcceptedCollection<TData>(
  client: QueryClient,
  owner: ReturnType<typeof captureMutationOwner>,
  reconcile?: (previous: TData | undefined) => TData | undefined
) {
  const query = await cancelOwnedCollection(client, owner);
  if (!query) return;

  if (owner.query === query && reconcile) reconcileCollection<TData>(client, owner, reconcile);

  await invalidateCapturedCollection(client, owner, query);
}

export function issueLinkMutationOptions(client: QueryClient, scope: TIssueResourceScope) {
  const service = new IssueService(scope.serviceType);
  const queryKey = issueResourceKeys.collection(scope, "links");
  return mutationOptions({
    mutationKey: [...queryKey, "write"],
    scope: { id: JSON.stringify(queryKey) },
    retry: false,
    onMutate: () => captureMutationOwner(client, scope, "links"),
    mutationFn: async (input: TIssueLinkWrite) => {
      const access = captureResourceAccess(client, scope);
      if (!access.isVerified()) throw new Error("The work item access scope is unavailable.");
      const workspaceSlug = scope.workspaceSlug!;
      const projectId = scope.projectId!;
      const issueId = scope.issueId!;
      if (input.kind === "create") {
        const saved = await service.createIssueLink(workspaceSlug, projectId, issueId, input.data ?? {});
        if (saved.issue_id !== issueId) throw new Error("The server returned a link for another work item.");
        return { kind: input.kind, saved };
      }
      if (!input.linkId) throw new Error("A link identifier is required.");
      if (input.kind === "update") {
        const saved = await service.updateIssueLink(workspaceSlug, projectId, issueId, input.linkId, input.data ?? {});
        if (saved.id !== input.linkId || saved.issue_id !== issueId)
          throw new Error("The server returned a link for another work item.");
        return { kind: input.kind, saved };
      }
      await service.deleteIssueLink(workspaceSlug, projectId, issueId, input.linkId);
      return { kind: input.kind, linkId: input.linkId };
    },
    onSuccess: async (receipt, _input, owner) => {
      if (!owner) return;
      await refreshAcceptedCollection<TIssueLink[]>(client, owner, (previous) => {
        if (!previous) return undefined;
        if (receipt.kind === "create")
          return [...previous.filter((link) => link.id !== receipt.saved.id), receipt.saved];
        if (receipt.kind === "update")
          return previous.map((link) => (link.id === receipt.saved.id ? receipt.saved : link));
        return previous.filter((link) => link.id !== receipt.linkId);
      });
      await refreshAcceptedActivity(client, scope, owner);
    },
  });
}

export function issueAttachmentMutationOptions(client: QueryClient, scope: TIssueResourceScope) {
  const service = new IssueAttachmentService(scope.serviceType);
  const queryKey = issueResourceKeys.collection(scope, "attachments");
  return mutationOptions({
    mutationKey: [...queryKey, "write"],
    scope: { id: JSON.stringify(queryKey) },
    retry: false,
    onMutate: () => captureMutationOwner(client, scope, "attachments"),
    mutationFn: async (
      input:
        | { kind: "upload"; file: File; onProgress?: (progress: number) => void }
        | { kind: "delete"; attachmentId: string }
    ) => {
      const access = captureResourceAccess(client, scope);
      if (!access.isVerified()) throw new Error("The work item access scope is unavailable.");
      const workspaceSlug = scope.workspaceSlug!;
      const projectId = scope.projectId!;
      const issueId = scope.issueId!;
      if (input.kind === "upload") {
        const saved = await service.uploadIssueAttachment(workspaceSlug, projectId, issueId, input.file, (event) =>
          input.onProgress?.(Math.round((event.progress ?? 0) * 100))
        );
        if (saved.issue_id !== issueId) throw new Error("The server returned an attachment for another work item.");
        return { kind: input.kind, saved };
      }
      const saved = await service.deleteIssueAttachment(workspaceSlug, projectId, issueId, input.attachmentId);
      return { kind: input.kind, attachmentId: input.attachmentId, saved };
    },
    onSuccess: async (receipt, _input, owner) => {
      if (!owner) return;
      await refreshAcceptedCollection<TIssueAttachment[]>(client, owner, (previous) => {
        if (!previous) return undefined;
        if (receipt.kind === "upload")
          return [...previous.filter((attachment) => attachment.id !== receipt.saved.id), receipt.saved];
        return previous.filter((attachment) => attachment.id !== receipt.attachmentId);
      });
      await refreshAcceptedActivity(client, scope, owner);
    },
  });
}

export function issueRelationMutationOptions(client: QueryClient, scope: TIssueResourceScope) {
  const service = new IssueRelationService();
  const queryKey = issueResourceKeys.collection(scope, "relations");
  return mutationOptions({
    mutationKey: [...queryKey, "write"],
    scope: { id: JSON.stringify(queryKey) },
    retry: false,
    onMutate: () => captureMutationOwner(client, scope, "relations"),
    mutationFn: async (
      input:
        | { kind: "create"; relationType: TIssueRelationTypes; issueIds: string[] }
        | { kind: "delete"; relationType: TIssueRelationTypes; relatedIssueId: string }
    ) => {
      const access = captureResourceAccess(client, scope);
      if (!access.isVerified()) throw new Error("The work item access scope is unavailable.");
      const { workspaceSlug, projectId, issueId } = scope;
      if (!workspaceSlug || !projectId || !issueId) throw new Error("The work item scope is incomplete.");
      if (input.kind === "create") {
        const saved = await service.createIssueRelations(workspaceSlug, projectId, issueId, {
          relation_type: input.relationType,
          issues: input.issueIds,
        });
        if (saved.some((relatedIssue) => !input.issueIds.includes(relatedIssue.id)))
          throw new Error("The server returned an unexpected related work item.");
        return { ...input, saved };
      }
      await service.deleteIssueRelation(workspaceSlug, projectId, issueId, {
        relation_type: input.relationType,
        related_issue: input.relatedIssueId,
      });
      return input;
    },
    onSuccess: async (_receipt, input, owner) => {
      if (!owner || !owner.isCurrent()) return;
      await refreshAcceptedCollection(client, owner);
      await refreshAcceptedActivity(client, scope, owner);
      // Invalidate both sides: the server maintains inverse relation rows.
      const changedIssueIds = input.kind === "create" ? input.issueIds : [input.relatedIssueId];
      const relevant = client
        .getQueryCache()
        .findAll({ queryKey: issueResourceKeys.prefix })
        .filter((query) => {
          const key = query.queryKey;
          return (
            key[1] === "relations" &&
            key[2] === scope.accountId &&
            key[3] === scope.sessionGeneration &&
            key[4] === scope.workspaceSlug &&
            key[10] === scope.serviceType &&
            key[11] !== scope.issueId &&
            changedIssueIds.includes(String(key[11]))
          );
        });
      await Promise.all(relevant.map((query) => invalidateCapturedCollection(client, owner, query)));
    },
  });
}
