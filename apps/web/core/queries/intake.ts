/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { infiniteQueryOptions, mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type { TInboxIssue, TIssue, TInboxIssueWithPagination } from "@plane/types";
import type { TProjectRosterScope } from "./member-rosters";
import { captureProjectAccessScope } from "./project-access-scope";
import { captureIssueActivityRefresh } from "./issue-discussion";
import { InboxIssueService } from "@/services/inbox";
import type { IntakePreferences } from "@/helpers/intake-preferences";
import { intakeQueryParams } from "@/helpers/intake-preferences";

export type IntakeScope = TProjectRosterScope & {
  sessionGeneration: number;
  projectMembershipRole: TProjectRosterScope["projectRole"];
};

export const intakeKeys = {
  scope: (scope: IntakeScope) =>
    [
      "intake",
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
    ] as const,
  list: (scope: IntakeScope, preferences: IntakePreferences) =>
    [...intakeKeys.scope(scope), "list", intakeQueryParams(preferences)] as const,
  detail: (scope: IntakeScope, issueId: string | undefined) => [...intakeKeys.scope(scope), "detail", issueId] as const,
};

function scopeReady(scope: IntakeScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.membershipRevision &&
    scope.membershipRole !== undefined &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined
  );
}

function endpoint(scope: IntakeScope) {
  if (!scope.workspaceSlug || !scope.projectId) throw new Error("An intake project is required.");
  return { workspaceSlug: scope.workspaceSlug, projectId: scope.projectId };
}

function captureIntakeAccess(client: QueryClient, scope: IntakeScope) {
  const access = captureProjectAccessScope(client, scope);
  return {
    isCurrent: () => scopeReady(scope) && access.isCurrent(),
    isVerified: () => scopeReady(scope) && access.isVerified(),
  };
}

function captureRead(client: QueryClient, scope: IntakeScope, key: readonly unknown[], signal: AbortSignal) {
  const access = captureIntakeAccess(client, scope);
  const query = client.getQueryCache().find({ queryKey: key, exact: true });
  const isCurrent = () =>
    !signal.aborted &&
    !!query &&
    access.isVerified() &&
    client.getQueryCache().find({ queryKey: key, exact: true }) === query;
  if (!isCurrent()) throw new Error("Intake access is unavailable.");
  return isCurrent;
}

function assertIntakeIdentity(row: TInboxIssue, scope: IntakeScope, issueId?: string) {
  if (!row?.issue?.id || row.issue.project_id !== scope.projectId || (issueId && row.issue.id !== issueId))
    throw new Error("The server returned an intake item outside the requested project.");
}

export function intakeListOptions(
  client: QueryClient,
  scope: IntakeScope,
  preferences: IntakePreferences,
  enabled = true
) {
  const key = intakeKeys.list(scope, preferences);
  const params = intakeQueryParams(preferences);
  return infiniteQueryOptions({
    queryKey: key,
    initialPageParam: "10:0:0",
    staleTime: 30_000,
    retry: false,
    queryFn:
      enabled && scopeReady(scope)
        ? async ({ pageParam, signal }) => {
            const isCurrent = captureRead(client, scope, key, signal);
            const { workspaceSlug, projectId } = endpoint(scope);
            const page = await new InboxIssueService().list(
              workspaceSlug,
              projectId,
              { ...params, per_page: 10, cursor: pageParam },
              { signal }
            );
            if (!isCurrent()) throw new Error("Intake access changed while loading.");
            for (const row of page.results) assertIntakeIdentity(row, scope);
            return page;
          }
        : skipToken,
    getNextPageParam: (page: TInboxIssueWithPagination) =>
      page.next_page_results ? page.next_cursor || undefined : undefined,
  });
}

export function intakeDetailOptions(client: QueryClient, scope: IntakeScope, issueId: string | undefined) {
  const key = intakeKeys.detail(scope, issueId);
  return queryOptions({
    queryKey: key,
    staleTime: 30_000,
    retry: false,
    queryFn:
      scopeReady(scope) && issueId
        ? async ({ signal }) => {
            const isCurrent = captureRead(client, scope, key, signal);
            const { workspaceSlug, projectId } = endpoint(scope);
            const row = await new InboxIssueService().retrieve(workspaceSlug, projectId, issueId, { signal });
            if (!isCurrent()) throw new Error("Intake access changed while loading.");
            assertIntakeIdentity(row, scope, issueId);
            return row;
          }
        : skipToken,
  });
}

export type IntakeWrite =
  | { kind: "create"; data: Partial<TIssue> }
  | { kind: "delete"; issueId: string }
  | { kind: "status"; issueId: string; data: Partial<Pick<TInboxIssue, "status" | "snoozed_till" | "duplicate_to">> }
  | { kind: "issue"; issueId: string; data: Partial<TIssue> };

type AcceptedIntakeWrite = { saved?: TInboxIssue; deletedIssueId?: string; kind: IntakeWrite["kind"] };

type CapturedIntakeEffects = {
  beforeWrite: () => Promise<void>;
  accepted: (receipt: AcceptedIntakeWrite, isCurrent: () => boolean) => Promise<void>;
};

function queryStillOwned(client: QueryClient, query: Query) {
  return client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query;
}

async function reconcileAcceptedIntake(
  client: QueryClient,
  owner: ReturnType<typeof captureIntakeAccess>,
  queries: Query[],
  receipt: AcceptedIntakeWrite
) {
  await Promise.all(
    queries.map(async (query) => {
      if (!owner.isCurrent() || !queryStillOwned(client, query)) return;
      await client.cancelQueries({ queryKey: query.queryKey, exact: true });
      if (!owner.isCurrent() || !queryStillOwned(client, query)) return;
      if (query.queryKey[9] === "detail") {
        const issueId = query.queryKey[10];
        if (receipt.saved?.issue.id === issueId) client.setQueryData(query.queryKey, receipt.saved);
        if (receipt.deletedIssueId === issueId) {
          client.removeQueries({ queryKey: query.queryKey, exact: true });
        }
        // Mutation receipts are canonical. Refetching an active detail here can
        // discard an open editor when an unrelated follow-up read fails.
        return;
      }
      // Filtered/paginated lists must be recounted by the server after status changes.
      await client.invalidateQueries({ queryKey: query.queryKey, exact: true });
    })
  );
}

export function intakeMutationOptions(
  client: QueryClient,
  scope: IntakeScope,
  captureEffects: (input: IntakeWrite) => CapturedIntakeEffects | Promise<CapturedIntakeEffects>
) {
  const key = intakeKeys.scope(scope);
  return mutationOptions({
    mutationKey: [...key, "write"],
    scope: { id: JSON.stringify(key) },
    retry: false,
    onMutate: async (input: IntakeWrite) => {
      const access = captureIntakeAccess(client, scope);
      if (!access.isVerified()) throw new Error("Intake access is unavailable.");
      const { workspaceSlug, projectId } = endpoint(scope);
      const refreshActivity =
        input.kind === "create"
          ? undefined
          : captureIssueActivityRefresh(client, {
              workspaceSlug,
              projectId,
              issueId: input.issueId,
            });
      const queries = client.getQueryCache().findAll({ queryKey: key });
      const effects = await captureEffects(input);
      await effects.beforeWrite();
      if (!access.isVerified()) throw new Error("Intake access changed before saving.");
      return { access, queries, effects, refreshActivity };
    },
    mutationFn: async (input: IntakeWrite): Promise<AcceptedIntakeWrite> => {
      if (!captureIntakeAccess(client, scope).isVerified()) throw new Error("Intake access is unavailable.");
      const { workspaceSlug, projectId } = endpoint(scope);
      const service = new InboxIssueService();
      if (input.kind === "delete") {
        await service.destroy(workspaceSlug, projectId, input.issueId);
        return { kind: input.kind, deletedIssueId: input.issueId };
      }
      const saved =
        input.kind === "create"
          ? await service.create(workspaceSlug, projectId, input.data)
          : input.kind === "status"
            ? await service.update(workspaceSlug, projectId, input.issueId, input.data)
            : await service.updateIssue(workspaceSlug, projectId, input.issueId, input.data);
      assertIntakeIdentity(saved, scope, input.kind === "create" ? undefined : input.issueId);
      return { kind: input.kind, saved };
    },
    onSuccess: async (receipt, _input, owner) => {
      if (!owner?.access.isCurrent()) return;
      await Promise.allSettled([
        reconcileAcceptedIntake(client, owner.access, owner.queries, receipt),
        owner.effects.accepted(receipt, owner.access.isCurrent),
        owner.refreshActivity?.(),
      ]);
    },
  });
}
