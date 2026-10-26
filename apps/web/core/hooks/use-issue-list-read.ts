/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useMemo } from "react";
import { hashKey, useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import type { IssuePaginationOptions } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType as IssuesStoreType } from "@plane/types";
import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";
import type { NativeIssueStoreType } from "@/hooks/use-issue-filter-preferences";
import { useParentCatalogueScope } from "@/hooks/use-parent-catalogue-scope";
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import { useOptionalParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { useOptionalProjectViewIssuePreferences } from "@/providers/project-view-issue-preferences";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { projectUserPropertiesKeys } from "@/queries/project-user-properties";
import type { IssueListRead, IssueListRequest } from "@/queries/issue-lists";
import { parentScopeKey } from "@/queries/cycle-module-scope";
import type { TProjectIssueRequestInput } from "@/helpers/issue-list/request";
import type { TParentIssueRequestInput } from "@/helpers/issue-list/parent-request";

type IssueListPreferences = ReturnType<typeof useIssueFilterPreferences>;

function projectRequest(
  storeType: NativeIssueStoreType,
  input: TProjectIssueRequestInput | undefined,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  preferences: IssueListPreferences
): IssueListRequest | undefined {
  if (!input || workspaceSlug !== input.workspaceSlug || projectId !== input.projectId) return undefined;
  const isReadCurrent = () =>
    workspaceSlug === input.workspaceSlug &&
    projectId === input.projectId &&
    !preferences.projectPreferences?.isPending &&
    !preferences.projectPreferences?.isError;
  return storeType === IssuesStoreType.EPIC
    ? { family: "epic", serviceType: EIssueServiceType.EPICS, input, isReadCurrent }
    : { family: "project", serviceType: EIssueServiceType.ISSUES, input, isReadCurrent };
}

function createIssueListRequest(input: {
  storeType: NativeIssueStoreType;
  preferences: IssueListPreferences;
  workspacePreferences: ReturnType<typeof useWorkspaceIssuePreferences>;
  parentPreferences: ReturnType<typeof useOptionalParentIssuePreferences>;
  projectViewPreferences: ReturnType<typeof useOptionalProjectViewIssuePreferences>;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  globalViewId: string | undefined;
  userId: string | undefined;
  cycleId: string | undefined;
  moduleId: string | undefined;
  viewId: string | undefined;
}): IssueListRequest | undefined {
  const candidate = input.preferences.requestInput;
  switch (input.storeType) {
    case IssuesStoreType.GLOBAL:
      return workspaceRequest(input.workspacePreferences?.requestInput);
    case IssuesStoreType.PROFILE:
      return profileRequest(candidate);
    case IssuesStoreType.ARCHIVED:
      return archivedRequest(candidate);
    case IssuesStoreType.CYCLE:
    case IssuesStoreType.MODULE:
      return parentRequest(input.parentPreferences?.requestInput, input.storeType);
    case IssuesStoreType.PROJECT_VIEW:
      return projectViewRequest(input.projectViewPreferences?.requestInput);
    case IssuesStoreType.PROJECT:
    case IssuesStoreType.EPIC:
      return projectRequest(
        input.storeType,
        projectRequestInput(candidate),
        input.workspaceSlug,
        input.projectId,
        input.preferences
      );
    default:
      return undefined;
  }
}

function workspaceRequest(
  input: NonNullable<ReturnType<typeof useWorkspaceIssuePreferences>>["requestInput"]
): IssueListRequest | undefined {
  return input ? { family: "workspace", input, isReadCurrent: input.isReadScopeCurrent } : undefined;
}

function profileRequest(input: IssueListPreferences["requestInput"]): IssueListRequest | undefined {
  return input && "kind" in input && input.kind === "profile"
    ? { family: "profile", input, isReadCurrent: input.isReadScopeCurrent }
    : undefined;
}

function archivedRequest(input: IssueListPreferences["requestInput"]): IssueListRequest | undefined {
  return input && "kind" in input && input.kind === "archived"
    ? { family: "archived", input, isReadCurrent: input.isReadScopeCurrent }
    : undefined;
}

function projectViewRequest(
  input: NonNullable<ReturnType<typeof useOptionalProjectViewIssuePreferences>>["requestInput"] | undefined
): IssueListRequest | undefined {
  return input ? { family: "project-view", input, isReadCurrent: input.isReadScopeCurrent } : undefined;
}

function projectRequestInput(input: IssueListPreferences["requestInput"]): TProjectIssueRequestInput | undefined {
  return input && !("kind" in input) && !("viewId" in input) && "projectId" in input ? input : undefined;
}

function parentRequest(
  input: TParentIssueRequestInput | undefined,
  storeType: NativeIssueStoreType
): IssueListRequest | undefined {
  if (!input) return undefined;
  if (storeType === IssuesStoreType.CYCLE && input.kind === "cycle") {
    const cycleInput = { ...input, kind: "cycle" as const };
    return { family: "cycle", input: cycleInput, isReadCurrent: input.isReadScopeCurrent };
  }
  if (storeType === IssuesStoreType.MODULE && input.kind === "module") {
    const moduleInput = { ...input, kind: "module" as const };
    return { family: "module", input: moduleInput, isReadCurrent: input.isReadScopeCurrent };
  }
  return undefined;
}

function routeReadCurrent(
  request: IssueListRequest,
  route: {
    globalViewId: string | undefined;
    userId: string | undefined;
    projectId: string | undefined;
    cycleId: string | undefined;
    moduleId: string | undefined;
    viewId: string | undefined;
  }
) {
  switch (request.family) {
    case "workspace":
      return route.globalViewId === request.input.viewId && request.input.isReadScopeCurrent();
    case "profile":
      return route.userId === request.input.userId && request.input.isReadScopeCurrent();
    case "archived":
      return route.projectId === request.input.projectId && request.input.isReadScopeCurrent();
    case "cycle":
      return route.cycleId === request.input.viewId && request.input.isReadScopeCurrent();
    case "module":
      return route.moduleId === request.input.viewId && request.input.isReadScopeCurrent();
    case "project-view":
      return route.viewId === request.input.viewId && request.input.isReadScopeCurrent();
    default:
      return undefined;
  }
}

function requestIdentity(request: IssueListRequest | undefined) {
  if (!request) return undefined;
  const input = request.input;
  const owner =
    "projectId" in input
      ? [input.workspaceSlug, input.projectId, "viewId" in input ? input.viewId : undefined]
      : "userId" in input
        ? [input.workspaceSlug, input.userId, "view" in input ? input.view : undefined]
        : [input.workspaceSlug, input.viewId];
  return hashKey([request.family, owner, input.richFilters, input.displayFilters]);
}

export function useIssueListRead(
  storeType: NativeIssueStoreType,
  pagination: IssuePaginationOptions,
  enabled = true,
  catalogueRevision?: string
): IssueListRead | undefined {
  const client = useQueryClient();
  const { workspaceSlug, projectId, cycleId, moduleId, viewId, globalViewId, userId } = useParams({ strict: false });
  const preferences = useIssueFilterPreferences(storeType);
  const workspacePreferences = useWorkspaceIssuePreferences();
  const parentPreferences = useOptionalParentIssuePreferences();
  const projectViewPreferences = useOptionalProjectViewIssuePreferences();
  const scopes = useParentCatalogueScope(workspaceSlug, projectId);
  const request = useMemo(
    () =>
      createIssueListRequest({
        storeType,
        preferences,
        workspacePreferences,
        parentPreferences,
        projectViewPreferences,
        workspaceSlug,
        projectId,
        globalViewId,
        userId,
        cycleId,
        moduleId,
        viewId,
      }),
    [
      storeType,
      preferences,
      workspacePreferences,
      parentPreferences,
      projectViewPreferences,
      workspaceSlug,
      projectId,
      globalViewId,
      userId,
      cycleId,
      moduleId,
      viewId,
    ]
  );
  const scope =
    storeType === IssuesStoreType.GLOBAL || storeType === IssuesStoreType.PROFILE
      ? scopes.workspaceScope
      : scopes.projectScope;
  const identity = request
    ? hashKey([requestIdentity(request), scope ? parentScopeKey(scope) : undefined, pagination, catalogueRevision])
    : undefined;
  const committed = useCommittedScope(identity ?? "issue-list-unavailable");
  const projectPropertiesKey = useMemo(
    () =>
      projectUserPropertiesKeys.detail({
        accountId: scopes.projectScope.accountId,
        workspaceSlug,
        projectId,
      }),
    [scopes.projectScope.accountId, workspaceSlug, projectId]
  );
  const projectPropertiesQuery = client.getQueryCache().find({ queryKey: projectPropertiesKey, exact: true });
  const isReadCurrent = useCallback(() => {
    if (!request || !workspaceSlug || !committed()) return false;
    if (request.family !== "project" && request.family !== "epic")
      return routeReadCurrent(request, { globalViewId, userId, projectId, cycleId, moduleId, viewId }) ?? false;
    return (
      workspaceSlug === request.input.workspaceSlug &&
      projectId === request.input.projectId &&
      projectPropertiesQuery?.state.status === "success" &&
      client.getQueryCache().find({ queryKey: projectPropertiesKey, exact: true }) === projectPropertiesQuery &&
      !preferences.projectPreferences?.isPending &&
      !preferences.projectPreferences?.isError
    );
  }, [
    request,
    committed,
    workspaceSlug,
    globalViewId,
    userId,
    projectId,
    cycleId,
    moduleId,
    viewId,
    client,
    projectPropertiesKey,
    projectPropertiesQuery,
    preferences.projectPreferences?.isPending,
    preferences.projectPreferences?.isError,
  ]);
  return useMemo(() => {
    if (!enabled || !request || !scope || !identity) return undefined;
    const scopedRequest: IssueListRequest = { ...request, isReadCurrent };
    return { scope, request: scopedRequest, pagination, catalogueRevision };
  }, [catalogueRevision, enabled, identity, isReadCurrent, pagination, request, scope]);
}
