/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useMemo } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { EIssueFilterType } from "@plane/constants";
import type { TSupportedFilterTypeForUpdate } from "@plane/constants";
import type { TIssue, TIssueKanbanFilters, TSupportedFilterForUpdate } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { isDisplayFilterChange, isDisplayPropertyChange } from "@/helpers/issue-preference-changes";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { issueListMutationOptions } from "@/queries/issue-list-mutations";
import type { IssueListMutation, IssueListMutationOwner } from "@/queries/issue-list-mutations";
import type {
  TArchivedIssueRequestInput,
  TProfileIssueRequestInput,
  TProjectIssueRequestInput,
} from "@/helpers/issue-list/request";
import type { TProjectViewIssueRequestInput } from "@/helpers/issue-list/project-view-request";
import { useOptionalParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { useOptionalProjectViewIssuePreferences } from "@/providers/project-view-issue-preferences";
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import { useLocalIssuePreferences } from "@/providers/local-issue-preferences";

export interface IssueActions {
  removeIssue: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  removeIssues: (projectId: string | undefined | null, issueIds: string[]) => Promise<void>;
  createIssue?: (projectId: string | undefined | null, data: Partial<TIssue>) => Promise<TIssue | undefined>;
  quickAddIssue?: (projectId: string | undefined | null, data: TIssue) => Promise<TIssue | undefined>;
  updateIssue?: (
    projectId: string | undefined | null,
    issueId: string,
    data: Partial<TIssue>
  ) => Promise<TIssue | undefined>;
  removeIssueFromView?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  removeIssueFromModule: (projectId: string | undefined | null, moduleId: string, issueId: string) => Promise<void>;
  archiveIssue?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  restoreIssue?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  updateIssueDates?: (
    projectId: string | undefined | null,
    updates: { id: string; start_date?: string; target_date?: string }[]
  ) => Promise<void>;
  addCycleToIssue: (projectId: string | undefined | null, cycleId: string, issueId: string) => Promise<void>;
  addIssuesToCycle: (projectId: string | undefined | null, cycleId: string, issueIds: string[]) => Promise<void>;
  removeCycleFromIssue: (projectId: string | undefined | null, cycleId: string, issueId: string) => Promise<void>;
  addIssuesToModule: (projectId: string | undefined | null, moduleId: string, issueIds: string[]) => Promise<void>;
  changeModulesInIssue: (
    projectId: string | undefined | null,
    issueId: string,
    added: string[],
    removed: string[]
  ) => Promise<void>;
  updateFilters?: (
    projectId: string,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
}

type RequestInput =
  | TProjectIssueRequestInput
  | TProjectViewIssueRequestInput
  | TProfileIssueRequestInput
  | TArchivedIssueRequestInput;

function isKanbanFilterChange(value: TSupportedFilterForUpdate): value is Partial<TIssueKanbanFilters> {
  return (
    typeof value === "object" &&
    value !== null &&
    (!("group_by" in value) || Array.isArray(value.group_by)) &&
    (!("sub_group_by" in value) || Array.isArray(value.sub_group_by))
  );
}

export const useIssuesActions = (
  storeType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>,
  requestInput?: RequestInput
): IssueActions => {
  const client = useQueryClient();
  const mutation = useMutation(issueListMutationOptions(client));
  const { workspaceSlug } = useParams({ strict: false });
  const resolveOwner = useIssueListMutationOwner(storeType, requestInput);
  const workspacePreferences = useWorkspaceIssuePreferences();
  const parentPreferences = useOptionalParentIssuePreferences();
  const parentRequestInput = parentPreferences?.requestInput;
  const projectViewPreferences = useOptionalProjectViewIssuePreferences();
  const localPreferences = useLocalIssuePreferences();

  const run = useCallback(
    async (targetProjectId: string | undefined | null, operation: IssueListMutation) => {
      if (!workspaceSlug || !targetProjectId) throw new Error("A current work item project is required.");
      const owner = await resolveOwner(workspaceSlug, targetProjectId);
      if (!owner) throw new Error("Work item access is unavailable. Retry the operation.");
      return mutation.mutateAsync({ owner, mutation: operation });
    },
    [mutation, resolveOwner, workspaceSlug]
  );

  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: Partial<TIssue>) => {
      const owner = await getOwner(resolveOwner, workspaceSlug, projectId);
      if (!owner) throw new Error("Work item access is unavailable. Retry the operation.");
      return mutation.mutateAsync({ owner, mutation: { kind: "create", data } });
    },
    [mutation, resolveOwner, workspaceSlug]
  );
  const quickAddIssue = createIssue;
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: Partial<TIssue>) =>
      run(projectId, { kind: "patch", issueId, data }),
    [run]
  );
  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      await run(projectId, { kind: "delete", issueId });
    },
    [run]
  );
  const removeIssues = useCallback(
    async (projectId: string | undefined | null, issueIds: string[]) => {
      if (issueIds.length === 0) return;
      await run(projectId, { kind: "delete-many", issueIds });
    },
    [run]
  );
  const archiveIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      await run(projectId, { kind: "archive", issueId });
    },
    [run]
  );
  const restoreIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      await run(projectId, { kind: "restore", issueId });
    },
    [run]
  );
  const updateIssueDates = useCallback(
    async (
      projectId: string | undefined | null,
      updates: { id: string; start_date?: string; target_date?: string }[]
    ) => {
      await run(projectId, { kind: "dates", updates });
    },
    [run]
  );
  const removeIssueFromView = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (storeType === EIssuesStoreType.CYCLE) {
        const parentId = parentRequestInput?.kind === "cycle" ? parentRequestInput.viewId : undefined;
        if (!parentId) throw new Error("The current cycle is unavailable. Retry the operation.");
        await run(projectId, { kind: "cycle-remove", cycleId: parentId, issueId });
      } else if (storeType === EIssuesStoreType.MODULE) {
        const parentId = parentRequestInput?.kind === "module" ? parentRequestInput.viewId : undefined;
        if (!parentId) throw new Error("The current module is unavailable. Retry the operation.");
        await run(projectId, { kind: "module-remove", moduleId: parentId, issueId });
      } else {
        throw new Error("Removing this work item from the current view is unavailable.");
      }
    },
    [parentRequestInput, run, storeType]
  );
  const addCycleToIssue = useCallback(
    async (projectId: string | undefined | null, cycleId: string, issueId: string) => {
      await run(projectId, { kind: "cycle-add", cycleId, issueIds: [issueId] });
    },
    [run]
  );
  const addIssuesToCycle = useCallback(
    async (projectId: string | undefined | null, cycleId: string, issueIds: string[]) => {
      if (issueIds.length === 0) return;
      await run(projectId, { kind: "cycle-add", cycleId, issueIds });
    },
    [run]
  );
  const removeCycleFromIssue = useCallback(
    async (projectId: string | undefined | null, cycleId: string, issueId: string) => {
      await run(projectId, { kind: "cycle-remove", cycleId, issueId });
    },
    [run]
  );
  const removeIssueFromModule = useCallback(
    async (projectId: string | undefined | null, moduleId: string, issueId: string) => {
      await run(projectId, { kind: "module-remove", moduleId, issueId });
    },
    [run]
  );
  const changeModulesInIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, added: string[], removed: string[]) => {
      await run(projectId, { kind: "module-change", issueId, added, removed });
    },
    [run]
  );
  const addIssuesToModule = useCallback(
    async (projectId: string | undefined | null, moduleId: string, issueIds: string[]) => {
      if (issueIds.length === 0) return;
      await run(projectId, { kind: "module-add", moduleId, issueIds });
    },
    [run]
  );

  const updateFilters = useCallback(
    (projectId: string, type: TSupportedFilterTypeForUpdate, values: TSupportedFilterForUpdate) =>
      updateIssueFilterPreferences({
        storeType,
        projectId,
        type,
        values,
        workspaceSlug,
        workspacePreferences,
        parentPreferences,
        projectViewPreferences,
        localPreferences,
      }),
    [localPreferences, parentPreferences, projectViewPreferences, storeType, workspacePreferences, workspaceSlug]
  );

  const actions = useMemo<IssueActions>(
    () => ({
      removeIssue,
      removeIssues,
      createIssue: storeType === EIssuesStoreType.ARCHIVED ? undefined : createIssue,
      quickAddIssue: storeType === EIssuesStoreType.ARCHIVED ? undefined : quickAddIssue,
      updateIssue,
      removeIssueFromView:
        storeType === EIssuesStoreType.CYCLE || storeType === EIssuesStoreType.MODULE ? removeIssueFromView : undefined,
      removeIssueFromModule,
      archiveIssue: storeType === EIssuesStoreType.ARCHIVED ? undefined : archiveIssue,
      restoreIssue: storeType === EIssuesStoreType.ARCHIVED ? restoreIssue : undefined,
      updateIssueDates,
      addCycleToIssue,
      addIssuesToCycle,
      addIssuesToModule,
      removeCycleFromIssue,
      changeModulesInIssue,
      updateFilters:
        storeType === EIssuesStoreType.PROJECT || storeType === EIssuesStoreType.EPIC ? undefined : updateFilters,
    }),
    [
      addCycleToIssue,
      addIssuesToCycle,
      addIssuesToModule,
      archiveIssue,
      changeModulesInIssue,
      createIssue,
      quickAddIssue,
      removeCycleFromIssue,
      removeIssue,
      removeIssues,
      removeIssueFromView,
      removeIssueFromModule,
      restoreIssue,
      storeType,
      updateFilters,
      updateIssue,
      updateIssueDates,
    ]
  );
  return actions;
};

type UpdateIssueFiltersInput = {
  storeType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
  projectId: string;
  type: TSupportedFilterTypeForUpdate;
  values: TSupportedFilterForUpdate;
  workspaceSlug: string | undefined;
  workspacePreferences: ReturnType<typeof useWorkspaceIssuePreferences>;
  parentPreferences: ReturnType<typeof useOptionalParentIssuePreferences>;
  projectViewPreferences: ReturnType<typeof useOptionalProjectViewIssuePreferences>;
  localPreferences: ReturnType<typeof useLocalIssuePreferences>;
};

async function updateIssueFilterPreferences(input: UpdateIssueFiltersInput): Promise<void> {
  switch (input.storeType) {
    case EIssuesStoreType.GLOBAL:
      updateWorkspaceIssueFilters(input.workspacePreferences, input.type, input.values);
      return;
    case EIssuesStoreType.CYCLE:
    case EIssuesStoreType.MODULE:
      await updateParentIssueFilters(input.parentPreferences, input.projectId, input.type, input.values);
      return;
    case EIssuesStoreType.PROJECT_VIEW:
      if (!input.workspaceSlug || !input.projectViewPreferences)
        throw new Error("The project view is unavailable. Retry.");
      await input.projectViewPreferences.updateFilters(input.workspaceSlug, input.projectId, input.type, input.values);
      return;
    case EIssuesStoreType.PROFILE:
      await updateLocalIssueFilters(input.localPreferences, input.type, input.values);
      return;
    case EIssuesStoreType.ARCHIVED:
      if (input.localPreferences?.entityId !== input.projectId)
        throw new Error("Issue preferences are unavailable. Retry the operation.");
      await updateLocalIssueFilters(input.localPreferences, input.type, input.values);
      return;
    default:
      throw new Error("Work item preferences are unavailable. Retry the operation.");
  }
}

function updateWorkspaceIssueFilters(
  preferences: UpdateIssueFiltersInput["workspacePreferences"],
  type: TSupportedFilterTypeForUpdate,
  values: TSupportedFilterForUpdate
) {
  if (!preferences?.requestInput?.isScopeCurrent())
    throw new Error("Workspace issue preferences are changing. Retry the operation.");
  if (type === EIssueFilterType.DISPLAY_FILTERS && isDisplayFilterChange(values)) {
    preferences.updateFilters(type, values);
    return;
  }
  if (type === EIssueFilterType.DISPLAY_PROPERTIES && isDisplayPropertyChange(values)) {
    preferences.updateFilters(type, values);
    return;
  }
  if (type === EIssueFilterType.KANBAN_FILTERS && isKanbanFilterChange(values)) {
    preferences.updateFilters(type, values);
    return;
  }
  throw new Error("Workspace issue preferences are invalid. Retry the operation.");
}

async function updateParentIssueFilters(
  preferences: UpdateIssueFiltersInput["parentPreferences"],
  projectId: string,
  type: TSupportedFilterTypeForUpdate,
  values: TSupportedFilterForUpdate
) {
  const input = preferences?.requestInput;
  if (!input || input.projectId !== projectId || !input.isScopeCurrent())
    throw new Error("Parent issue preferences are changing. Retry the operation.");
  if (type === EIssueFilterType.DISPLAY_FILTERS) {
    if (!isDisplayFilterChange(values)) throw new Error("Display grouping needs a column.");
    await preferences.setDisplay(values);
    return;
  }
  if (type === EIssueFilterType.DISPLAY_PROPERTIES) {
    if (!isDisplayPropertyChange(values)) throw new Error("Display properties need boolean values.");
    await preferences.setProperties(values);
    return;
  }
  const columns = {
    ...("group_by" in values && Array.isArray(values.group_by) ? { group_by: values.group_by } : {}),
    ...("sub_group_by" in values && Array.isArray(values.sub_group_by) ? { sub_group_by: values.sub_group_by } : {}),
  };
  if (type !== EIssueFilterType.KANBAN_FILTERS || Object.keys(columns).length === 0)
    throw new Error("Collapsed issue groups are invalid.");
  preferences.setKanban(columns);
}

async function updateLocalIssueFilters(
  preferences: UpdateIssueFiltersInput["localPreferences"],
  type: TSupportedFilterTypeForUpdate,
  values: TSupportedFilterForUpdate
) {
  if (!preferences) throw new Error("Issue preferences are unavailable. Retry the operation.");
  await preferences.updateFilters(type, values);
}

async function getOwner(
  resolveOwner: (workspaceSlug: string, projectId: string) => Promise<IssueListMutationOwner | undefined>,
  workspaceSlug: string | undefined,
  projectId: string | null | undefined
) {
  if (!workspaceSlug || !projectId) return undefined;
  return resolveOwner(workspaceSlug, projectId);
}
