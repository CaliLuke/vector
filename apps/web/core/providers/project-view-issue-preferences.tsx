/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { PropsWithChildren } from "react";
import { EIssueFilterType } from "@plane/constants";
import type { TSupportedFilterTypeForUpdate } from "@plane/constants";
import type {
  IIssueDisplayFilterOptions,
  IIssueFilters,
  TIssueKanbanFilters,
  TWorkItemFilterExpression,
} from "@plane/types";
import type { TSupportedFilterForUpdate, IProjectView } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { useRouter } from "@tanstack/react-router";
import isEqual from "lodash-es/isEqual.js";
import { getComputedDisplayFilters, getComputedDisplayProperties } from "@plane/utils/work-item";
import { readLocalIssueFilters, writeLocalIssueFilters } from "@/helpers/issue-list/local-preferences";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useQueryClient } from "@tanstack/react-query";
import { getProjectViewIssueFilters } from "@/helpers/issue-list/project-view-request";
import type { TProjectViewIssueRequestInput } from "@/helpers/issue-list/project-view-request";

type Draft = {
  richFilters?: TWorkItemFilterExpression;
  displayFilters?: Partial<IIssueDisplayFilterOptions>;
  displayProperties?: Partial<IIssueFilters["displayProperties"]>;
  kanbanFilters?: Partial<TIssueKanbanFilters>;
};

type Preferences = {
  issueFilters: IIssueFilters;
  requestInput: TProjectViewIssueRequestInput;
  updateFilters: (
    workspaceSlug: string,
    projectId: string,
    type: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
  updateRichFilters: (filters: TWorkItemFilterExpression) => Promise<void>;
};

const PreferencesContext = createContext<Preferences | undefined>(undefined);
function applyDraft<T extends object>(canonical: T, draft: Partial<T> | undefined): T {
  if (!draft) return canonical;
  const canonicalEntries = new Map(Object.entries(canonical));
  const changedEntries = Object.entries(draft).filter(([key, value]) => !isEqual(canonicalEntries.get(key), value));
  return { ...canonical, ...Object.fromEntries(changedEntries) };
}

function kanbanFilterChanges(values: TSupportedFilterForUpdate): Partial<TIssueKanbanFilters> {
  return {
    ...("group_by" in values && Array.isArray(values.group_by) ? { group_by: values.group_by } : {}),
    ...("sub_group_by" in values && Array.isArray(values.sub_group_by) ? { sub_group_by: values.sub_group_by } : {}),
  };
}

function isDisplayFilterChange(values: TSupportedFilterForUpdate): values is IIssueDisplayFilterOptions {
  return (
    !("group_by" in values && Array.isArray(values.group_by)) &&
    !("sub_group_by" in values && Array.isArray(values.sub_group_by))
  );
}

function initialIssueFilters(view: IProjectView, workspaceSlug: string, accountId: string) {
  const kanban = readLocalIssueFilters(
    EIssuesStoreType.PROJECT_VIEW,
    workspaceSlug,
    view.id,
    accountId
  )?.kanban_filters;
  return getProjectViewIssueFilters({
    richFilters: view.rich_filters ?? {},
    displayFilters: view.display_filters,
    displayProperties: view.display_properties,
    kanbanFilters: { group_by: kanban?.group_by ?? [], sub_group_by: kanban?.sub_group_by ?? [] },
  });
}

type ProjectViewPreferencesProps = PropsWithChildren<{
  view: IProjectView;
  workspaceSlug: string;
  projectId: string;
  accountId: string;
  isScopeCurrent: () => boolean;
  isReadScopeCurrent: () => boolean;
}>;

export function ProjectViewIssuePreferencesProvider(props: ProjectViewPreferencesProps) {
  const { view, workspaceSlug, projectId, accountId, isScopeCurrent, isReadScopeCurrent, children } = props;
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const key = JSON.stringify([accountId, generation, workspaceSlug, projectId, view.id]);
  return (
    <ProjectViewIssuePreferencesState
      key={key}
      view={view}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      accountId={accountId}
      isScopeCurrent={isScopeCurrent}
      isReadScopeCurrent={isReadScopeCurrent}
    >
      {children}
    </ProjectViewIssuePreferencesState>
  );
}

function ProjectViewIssuePreferencesState(props: ProjectViewPreferencesProps) {
  const { view, workspaceSlug, projectId, accountId, isScopeCurrent, isReadScopeCurrent, children } = props;
  const router = useRouter();
  const [draft, setDraft] = useState<Draft>({});
  const mounted = useRef(false);
  const currentInput = useRef<object | undefined>(undefined);
  const canonical = useMemo(
    () => initialIssueFilters(view, workspaceSlug, accountId),
    [accountId, view, workspaceSlug]
  );

  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      currentInput.current = undefined;
    };
  }, []);
  const issueFilters = useMemo<IIssueFilters>(
    () => ({
      richFilters: isEqual(draft.richFilters, canonical.richFilters)
        ? canonical.richFilters
        : (draft.richFilters ?? canonical.richFilters),
      displayFilters: applyDraft(canonical.displayFilters, draft.displayFilters),
      displayProperties: applyDraft(canonical.displayProperties, draft.displayProperties),
      kanbanFilters: applyDraft(canonical.kanbanFilters, draft.kanbanFilters),
    }),
    [canonical, draft]
  );
  useEffect(() => {
    if (!draft.kanbanFilters) return;
    writeLocalIssueFilters(
      EIssuesStoreType.PROJECT_VIEW,
      EIssueFilterType.KANBAN_FILTERS,
      workspaceSlug,
      view.id,
      accountId,
      {
        kanban_filters: applyDraft(canonical.kanbanFilters, draft.kanbanFilters),
      }
    );
  }, [accountId, canonical.kanbanFilters, draft.kanbanFilters, view.id, workspaceSlug]);
  const inputToken = useMemo(
    () => ({
      richFilters: issueFilters.richFilters ?? {},
      displayFilters: getComputedDisplayFilters(issueFilters.displayFilters),
      displayProperties: getComputedDisplayProperties(issueFilters.displayProperties),
      kanbanFilters: issueFilters.kanbanFilters ?? { group_by: [], sub_group_by: [] },
    }),
    [issueFilters]
  );
  const expectedRoutePath = `/${workspaceSlug}/projects/${projectId}/views/${view.id}`;
  const routeIsCurrent = useCallback(
    () => router.state.location.pathname.replace(/\/+$/, "") === expectedRoutePath,
    [expectedRoutePath, router]
  );
  const scopeIsCurrent = useCallback(
    () => mounted.current && routeIsCurrent() && isScopeCurrent(),
    [isScopeCurrent, routeIsCurrent]
  );
  const requestIsCurrent = useCallback(
    () => scopeIsCurrent() && currentInput.current === inputToken,
    [inputToken, scopeIsCurrent]
  );
  const requestIsVerified = useCallback(
    () => requestIsCurrent() && isReadScopeCurrent(),
    [isReadScopeCurrent, requestIsCurrent]
  );
  useLayoutEffect(() => {
    currentInput.current = scopeIsCurrent() ? inputToken : undefined;
    return () => {
      if (currentInput.current === inputToken) currentInput.current = undefined;
    };
  }, [inputToken, scopeIsCurrent]);

  const requestInput = useMemo<TProjectViewIssueRequestInput>(
    () => ({
      workspaceSlug,
      projectId,
      viewId: view.id,
      accountId,
      ...inputToken,
      isScopeCurrent: requestIsCurrent,
      isReadScopeCurrent: requestIsVerified,
    }),
    [accountId, inputToken, projectId, requestIsCurrent, requestIsVerified, view.id, workspaceSlug]
  );
  const updateRichFilters = useCallback(
    async (richFilters: TWorkItemFilterExpression) => {
      if (!scopeIsCurrent()) return;
      setDraft((current) => ({ ...current, richFilters }));
    },
    [scopeIsCurrent]
  );
  const updateFilters = useCallback(
    async (
      targetWorkspace: string,
      targetProject: string,
      type: TSupportedFilterTypeForUpdate,
      values: TSupportedFilterForUpdate
    ) => {
      if (targetWorkspace !== workspaceSlug || targetProject !== projectId || !scopeIsCurrent()) return;
      if (type === EIssueFilterType.KANBAN_FILTERS) {
        setDraft((currentDraft) => ({
          ...currentDraft,
          kanbanFilters: { ...currentDraft.kanbanFilters, ...kanbanFilterChanges(values) },
        }));
        return;
      }
      if (type === EIssueFilterType.DISPLAY_FILTERS) {
        if (!isDisplayFilterChange(values))
          throw new Error("Display grouping requires a column, not collapsed group IDs.");
        setDraft((currentDraft) => {
          const changes = { ...currentDraft.displayFilters, ...values };
          const current = applyDraft(canonical.displayFilters, changes);
          const filters = { ...current, ...values };
          if (filters.group_by === null) filters.sub_group_by = null;
          if (filters.layout === "kanban" && filters.group_by === filters.sub_group_by) filters.sub_group_by = null;
          if (filters.layout === "kanban" && filters.group_by === null) filters.group_by = "state";
          if (filters.group_by !== current.group_by) changes.group_by = filters.group_by;
          if (filters.sub_group_by !== current.sub_group_by) changes.sub_group_by = filters.sub_group_by;
          if (filters.group_by === null) changes.sub_group_by = null;
          return { ...currentDraft, displayFilters: changes };
        });
        return;
      }
      if (type === EIssueFilterType.DISPLAY_PROPERTIES) {
        setDraft((currentDraft) => ({
          ...currentDraft,
          displayProperties: { ...currentDraft.displayProperties, ...values },
        }));
      }
    },
    [canonical, projectId, scopeIsCurrent, workspaceSlug]
  );

  const value = useMemo(
    () => ({ issueFilters, requestInput, updateFilters, updateRichFilters }),
    [issueFilters, requestInput, updateFilters, updateRichFilters]
  );
  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

export function useProjectViewIssuePreferences() {
  const value = useContext(PreferencesContext);
  if (!value) throw new Error("Project view issue preferences need their scoped provider.");
  return value;
}

export function useOptionalProjectViewIssuePreferences() {
  return useContext(PreferencesContext);
}
