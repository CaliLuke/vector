/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createContext, useCallback, useContext, useMemo, useState, useRef } from "react";
import type { ReactNode } from "react";
import { hashKey, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  IIssueFilters,
  IIssueFiltersResponse,
  TIssueKanbanFilters,
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  TWorkItemFilterExpression,
} from "@plane/types";
import isEqual from "lodash-es/isEqual.js";
import { isAuthorizationError } from "@/queries/authorization-errors";
import { getComputedDisplayFilters, getComputedDisplayProperties } from "@plane/utils/work-item";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useModuleDetails } from "@/hooks/use-modules";
import { useParentCatalogueScope } from "@/hooks/use-parent-catalogue-scope";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { captureParentAccess, currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import {
  parentIssuePreferenceOptions,
  patchParentIssuePreferenceOptions,
  parentIssuePreferenceKey,
} from "@/queries/parent-issue-preferences";
import type { ParentIssuePreferencesPatch } from "@/queries/parent-issue-preferences";
import type { ParentDomain } from "@/queries/cycle-modules";
import { readParentIssueColumns, saveParentIssueColumns } from "@/helpers/parent-issue-columns";
import type { TParentIssueRequestInput } from "@/helpers/issue-list/parent-request";

type Draft = {
  richRevision?: number;
  displayRevision?: number;
  propertiesRevision?: number;
  rich?: TWorkItemFilterExpression;
  display?: Partial<IIssueDisplayFilterOptions>;
  properties?: Partial<IIssueDisplayProperties>;
};
function mergePreferences(canonical: IIssueFiltersResponse, draft: Draft, columns: TIssueKanbanFilters): IIssueFilters {
  return {
    richFilters: draft.rich ?? canonical.rich_filters ?? {},
    displayFilters: getComputedDisplayFilters({ ...canonical.display_filters, ...draft.display }),
    displayProperties: getComputedDisplayProperties({ ...canonical.display_properties, ...draft.properties }),
    kanbanFilters: columns,
  };
}
function useNativeParentPreferences(domain: ParentDomain, workspaceSlug: string, projectId: string, parentId: string) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug, projectId),
    scope = access.projectScope;
  const query = useQuery(parentIssuePreferenceOptions(domain, scope, parentId));
  const cycle = useCycleDetails(workspaceSlug, projectId, domain === "cycle" ? parentId : undefined);
  const module = useModuleDetails(workspaceSlug, projectId, domain === "module" ? parentId : undefined);
  const captureStatistics = domain === "cycle" ? cycle.captureStatistics : module.captureStatistics;
  const mutation = useMutation(patchParentIssuePreferenceOptions(client, domain, scope, parentId));
  const [draft, setDraft] = useState<Draft>({});
  const [saveErrors, setSaveErrors] = useState<Partial<Record<"rich" | "display" | "properties", unknown>>>({});
  const draftRef = useRef<Draft>(draft),
    revision = useRef(0);
  const localScope = useMemo(
    () => ({ domain, accountId: scope.accountId ?? "", workspaceSlug, parentId }),
    [domain, scope.accountId, workspaceSlug, parentId]
  );
  const [columns, setColumns] = useState(() => readParentIssueColumns(localScope));
  const mounted = useCommittedScope(JSON.stringify(parentIssuePreferenceKey(domain, scope, parentId)));
  const canonicalRich = query.data?.rich_filters,
    canonicalDisplay = query.data?.display_filters,
    canonicalProperties = query.data?.display_properties;
  const canDisplay = access.isReady && query.isSuccess;
  const requestValues = useMemo(
    () =>
      canDisplay
        ? {
            richFilters: draft.rich ?? canonicalRich ?? {},
            displayFilters: getComputedDisplayFilters({ ...canonicalDisplay, ...draft.display }),
          }
        : undefined,
    [canDisplay, canonicalRich, canonicalDisplay, draft.rich, draft.display]
  );
  const filters = useMemo(
    () =>
      requestValues
        ? {
            ...requestValues,
            displayProperties: getComputedDisplayProperties({ ...canonicalProperties, ...draft.properties }),
            kanbanFilters: columns,
          }
        : undefined,
    [requestValues, canonicalProperties, draft.properties, columns]
  );
  const requestSignature = hashKey([requestValues?.richFilters, requestValues?.displayFilters]);
  const nativeAccess = useMemo(() => captureParentAccess(client, scope), [client, scope]);
  const queryObject = client
    .getQueryCache()
    .find({ queryKey: parentIssuePreferenceKey(domain, scope, parentId), exact: true });
  const current = useCallback(() => {
    if (
      !mounted() ||
      !nativeAccess.isCurrent() ||
      !queryObject ||
      isAuthorizationError(queryObject.state.error) ||
      client.getQueryCache().find({ queryKey: parentIssuePreferenceKey(domain, scope, parentId), exact: true }) !==
        queryObject
    )
      return false;
    const canonical = client.getQueryData<IIssueFiltersResponse>(parentIssuePreferenceKey(domain, scope, parentId));
    if (!canonical) return false;
    const latest = mergePreferences(canonical, draftRef.current, { group_by: [], sub_group_by: [] });
    return hashKey([latest.richFilters, latest.displayFilters]) === requestSignature;
  }, [client, domain, mounted, nativeAccess, parentId, queryObject, scope, requestSignature]);
  const requestInput = useMemo<TParentIssueRequestInput | undefined>(
    () =>
      requestValues && scope.accountId
        ? {
            kind: domain,
            workspaceSlug,
            projectId,
            viewId: parentId,
            accountId: scope.accountId,
            richFilters: requestValues.richFilters,
            displayFilters: requestValues.displayFilters,
            parentStatistics: captureStatistics(),
            isScopeCurrent: current,
            isReadScopeCurrent: () => current() && nativeAccess.isVerified() && queryObject?.state.status === "success",
          }
        : undefined,
    [
      domain,
      workspaceSlug,
      projectId,
      parentId,
      scope.accountId,
      current,
      nativeAccess,
      queryObject,
      requestValues,
      captureStatistics,
    ]
  );

  const save = async (patch: ParentIssuePreferencesPatch, change: Draft, field: "rich" | "display" | "properties") => {
    if (!mounted()) throw new Error("The parent preference editor was replaced.");
    const expected = ++revision.current;
    const revisionKey =
      field === "rich" ? "richRevision" : field === "display" ? "displayRevision" : "propertiesRevision";
    draftRef.current = { ...draftRef.current, ...change, [revisionKey]: expected };
    setDraft(draftRef.current);
    setSaveErrors((previous) => {
      const next = { ...previous };
      delete next[field];
      return next;
    });
    let result;
    try {
      result = await mutation.mutateAsync(patch);
    } catch (error) {
      if (mounted() && draftRef.current[revisionKey] === expected)
        setSaveErrors((previous) => ({ ...previous, [field]: error }));
      throw error;
    }
    if (mounted() && result.captured.isCurrent() && draftRef.current[revisionKey] === expected) {
      const remaining = { ...draftRef.current };
      delete remaining[field];
      delete remaining[revisionKey];
      draftRef.current = remaining;
      setDraft(remaining);
    }
  };
  const setRich = (rich: TWorkItemFilterExpression) =>
    isEqual(rich, draftRef.current.rich ?? query.data?.rich_filters ?? {})
      ? Promise.resolve()
      : save({ rich_filters: rich }, { rich }, "rich");
  const setDisplay = (patch: Partial<IIssueDisplayFilterOptions>) => {
    const display = { ...query.data?.display_filters, ...draftRef.current.display, ...patch };
    if (display.group_by === null) display.sub_group_by = null;
    if (display.layout === "kanban" && display.group_by === display.sub_group_by) display.sub_group_by = null;
    if (display.layout === "kanban" && display.group_by === null) display.group_by = "state";
    return save({ display_filters: display }, { display }, "display");
  };
  const setProperties = (patch: Partial<IIssueDisplayProperties>) => {
    const properties = { ...query.data?.display_properties, ...draftRef.current.properties, ...patch };
    return save({ display_properties: properties }, { properties }, "properties");
  };
  const setKanban = (patch: Partial<TIssueKanbanFilters>) => {
    if (!current() || !scope.accountId) return;
    const next = { ...columns, ...patch };
    setColumns(next);
    saveParentIssueColumns(localScope, next);
  };
  const retrySave = async () => {
    if (saveErrors.rich && draftRef.current.rich !== undefined)
      await save({ rich_filters: draftRef.current.rich }, { rich: draftRef.current.rich }, "rich");
    if (saveErrors.display && draftRef.current.display !== undefined)
      await save(
        { display_filters: { ...query.data?.display_filters, ...draftRef.current.display } },
        { display: draftRef.current.display },
        "display"
      );
    if (saveErrors.properties && draftRef.current.properties !== undefined)
      await save(
        { display_properties: { ...query.data?.display_properties, ...draftRef.current.properties } },
        { properties: draftRef.current.properties },
        "properties"
      );
  };
  const retry = async () => {
    if (!(await access.retry())) return;
    await client.fetchQuery({
      ...parentIssuePreferenceOptions(
        domain,
        currentParentCatalogueScope(client, workspaceSlug, projectId).project,
        parentId
      ),
      staleTime: 0,
    });
  };
  return {
    domain,
    scope,
    query,
    mutation,
    filters,
    requestInput,
    isRichDirty: draft.rich !== undefined,
    setRich,
    setDisplay,
    setProperties,
    setKanban,
    retry,
    retrySave,
    isSaveError: Object.keys(saveErrors).length > 0,
    isPending: access.isPending || query.isPending,
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
  };
}
const Context = createContext<ReturnType<typeof useNativeParentPreferences> | undefined>(undefined);
export function ParentIssuePreferencesProvider({
  domain,
  workspaceSlug,
  projectId,
  parentId,
  children,
}: {
  domain: ParentDomain;
  workspaceSlug: string;
  projectId: string;
  parentId: string;
  children: ReactNode;
}) {
  const access = useParentCatalogueScope(workspaceSlug, projectId);
  const key = JSON.stringify([
    domain,
    access.projectScope.accountId,
    access.projectScope.sessionGeneration,
    workspaceSlug,
    projectId,
    parentId,
  ]);
  return (
    <ParentIssuePreferencesState
      key={key}
      domain={domain}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      parentId={parentId}
    >
      {children}
    </ParentIssuePreferencesState>
  );
}
function ParentIssuePreferencesState({
  domain,
  workspaceSlug,
  projectId,
  parentId,
  children,
}: {
  domain: ParentDomain;
  workspaceSlug: string;
  projectId: string;
  parentId: string;
  children: ReactNode;
}) {
  const value = useNativeParentPreferences(domain, workspaceSlug, projectId, parentId);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}
export function useParentIssuePreferences() {
  const value = useContext(Context);
  if (!value) throw new Error("A cycle/module issue preference scope is required.");
  return value;
}
export function useOptionalParentIssuePreferences() {
  return useContext(Context);
}
