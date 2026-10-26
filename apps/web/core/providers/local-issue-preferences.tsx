/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import type { PropsWithChildren } from "react";
import { useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { EIssueFilterType } from "@plane/constants";
import type { TSupportedFilterTypeForUpdate } from "@plane/constants";
import type { IIssueFilters, TProfileViews, TSupportedFilterForUpdate, TWorkItemFilterExpression } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
import { getComputedDisplayFilters, getComputedDisplayProperties } from "@plane/utils/work-item";
import { isDisplayFilterChange, isDisplayPropertyChange } from "@/helpers/issue-preference-changes";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { readLocalIssueFilters, writeLocalIssueFilterSnapshot } from "@/helpers/issue-list/local-preferences";
import type { TProfileIssueRequestInput, TArchivedIssueRequestInput } from "@/helpers/issue-list/request";

type LocalPreferenceDomain = EIssuesStoreType.PROFILE | EIssuesStoreType.ARCHIVED;

type LocalPreferences = {
  domain: LocalPreferenceDomain;
  workspaceSlug: string;
  entityId: string;
  accountId: string | undefined;
  issueFilters: IIssueFilters;
  updateRichFilters: (filters: TWorkItemFilterExpression) => Promise<void>;
  updateFilters: (type: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => Promise<void>;
  profileRequestInput: (view: TProfileViews) => TProfileIssueRequestInput | undefined;
  archivedRequestInput: () => TArchivedIssueRequestInput | undefined;
};

const LocalPreferencesContext = createContext<LocalPreferences | undefined>(undefined);

function normalizedFilters(
  domain: LocalPreferenceDomain,
  workspaceSlug: string,
  entityId: string,
  accountId: string | undefined
) {
  const local = accountId ? readLocalIssueFilters(domain, workspaceSlug, entityId, accountId) : undefined;
  const displayFilters = getComputedDisplayFilters({
    ...local?.display_filters,
    ...(domain === EIssuesStoreType.ARCHIVED ? { sub_issue: true } : {}),
  });
  const kanban = local?.kanban_filters;
  return {
    richFilters: local?.rich_filters ?? {},
    displayFilters,
    displayProperties: getComputedDisplayProperties(local?.display_properties ?? {}),
    kanbanFilters: { group_by: kanban?.group_by ?? [], sub_group_by: kanban?.sub_group_by ?? [] },
  } satisfies IIssueFilters;
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((entry) => typeof entry === "string");
}

function updateDisplayFilters(
  domain: LocalPreferenceDomain,
  current: IIssueFilters,
  values: TSupportedFilterForUpdate
) {
  if (!isDisplayFilterChange(values)) return current.displayFilters;
  const displayFilters = { ...current.displayFilters, ...values };
  if (displayFilters.group_by === null) displayFilters.sub_group_by = null;
  if (displayFilters.layout === EIssueLayoutTypes.KANBAN && displayFilters.group_by === displayFilters.sub_group_by)
    displayFilters.sub_group_by = null;
  if (displayFilters.layout === EIssueLayoutTypes.KANBAN && displayFilters.group_by === null)
    displayFilters.group_by = domain === EIssuesStoreType.PROFILE ? "priority" : "state";
  if (domain === EIssuesStoreType.ARCHIVED) displayFilters.sub_issue = true;
  return displayFilters;
}

function updateKanbanFilters(current: IIssueFilters, values: TSupportedFilterForUpdate) {
  const groupBy = "group_by" in values && isStringArray(values.group_by) ? values.group_by : undefined;
  const subGroupBy = "sub_group_by" in values && isStringArray(values.sub_group_by) ? values.sub_group_by : undefined;
  if (!groupBy && !subGroupBy) return current.kanbanFilters;
  return {
    group_by: groupBy ?? current.kanbanFilters?.group_by ?? [],
    sub_group_by: subGroupBy ?? current.kanbanFilters?.sub_group_by ?? [],
  };
}

function updatedIssueFilters(
  domain: LocalPreferenceDomain,
  current: IIssueFilters,
  type: TSupportedFilterTypeForUpdate,
  values: TSupportedFilterForUpdate
): IIssueFilters {
  if (type === EIssueFilterType.DISPLAY_FILTERS)
    return { ...current, displayFilters: updateDisplayFilters(domain, current, values) };
  if (type === EIssueFilterType.DISPLAY_PROPERTIES)
    return {
      ...current,
      displayProperties: isDisplayPropertyChange(values)
        ? { ...current.displayProperties, ...values }
        : current.displayProperties,
    };
  return { ...current, kanbanFilters: updateKanbanFilters(current, values) };
}

type ProviderProps = PropsWithChildren<{
  domain: LocalPreferenceDomain;
  workspaceSlug: string;
  entityId: string;
}>;

export function LocalIssuePreferencesProvider(props: ProviderProps) {
  const { domain, workspaceSlug, entityId, children } = props;
  const client = useQueryClient();
  const accountId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const key = JSON.stringify([domain, workspaceSlug, entityId, accountId, generation]);
  return (
    <LocalIssuePreferencesState
      key={key}
      domain={domain}
      workspaceSlug={workspaceSlug}
      entityId={entityId}
      accountId={accountId}
      generation={generation}
    >
      {children}
    </LocalIssuePreferencesState>
  );
}

function LocalIssuePreferencesState(props: ProviderProps & { accountId: string | undefined; generation: number }) {
  const { domain, workspaceSlug, entityId, accountId, generation, children } = props;
  const client = useQueryClient();
  const router = useRouter();
  const identity = useMemo(
    () => (accountId ? captureIdentityScope(client, accountId) : undefined),
    [accountId, client]
  );
  const scopeKey = JSON.stringify([domain, workspaceSlug, entityId, accountId, generation]);
  const isMounted = useCommittedScope(scopeKey);
  const [issueFilters, setIssueFilters] = useState<IIssueFilters>(() =>
    normalizedFilters(domain, workspaceSlug, entityId, accountId)
  );
  const filtersRef = useRef(issueFilters);

  const routeIsCurrent = useCallback(() => {
    const pathname = router.state.location.pathname.replace(/\/+$/, "");
    const prefix =
      domain === EIssuesStoreType.PROFILE
        ? `/${workspaceSlug}/profile/${entityId}`
        : `/${workspaceSlug}/projects/${entityId}/archives/issues`;
    return pathname === prefix || pathname.startsWith(`${prefix}/`);
  }, [domain, entityId, router, workspaceSlug]);
  const scopeIsCurrent = useCallback(
    () =>
      isMounted() &&
      routeIsCurrent() &&
      Boolean(accountId) &&
      getIdentitySessionGeneration(client) === generation &&
      Boolean(identity && isIdentityScopeCurrent(client, identity)),
    [accountId, client, generation, identity, isMounted, routeIsCurrent]
  );
  const readScopeIsCurrent = useCallback(
    () => scopeIsCurrent() && (identity ? isIdentityScopeVerified(client, identity) : false),
    [client, identity, scopeIsCurrent]
  );

  const persist = useCallback(
    (next: IIssueFilters) => {
      if (!accountId || !scopeIsCurrent()) return false;
      filtersRef.current = next;
      setIssueFilters(next);
      writeLocalIssueFilterSnapshot(domain, workspaceSlug, entityId, accountId, next);
      return true;
    },
    [accountId, domain, entityId, scopeIsCurrent, workspaceSlug]
  );

  const updateRichFilters = useCallback(
    async (richFilters: TWorkItemFilterExpression) => {
      persist({ ...filtersRef.current, richFilters });
    },
    [persist]
  );
  const updateFilters = useCallback(
    async (type: TSupportedFilterTypeForUpdate, values: TSupportedFilterForUpdate) => {
      persist(updatedIssueFilters(domain, filtersRef.current, type, values));
    },
    [domain, persist]
  );

  const requestIsCurrent = useCallback(
    (filters: IIssueFilters, profileView?: TProfileViews) => () => {
      if (!scopeIsCurrent() || filtersRef.current !== filters) return false;
      if (!profileView) return true;
      const pathname = router.state.location.pathname.replace(/\/+$/, "");
      return pathname === `/${workspaceSlug}/profile/${entityId}/${profileView}`;
    },
    [entityId, router, scopeIsCurrent, workspaceSlug]
  );
  const profileRequestInput = useCallback(
    (view: TProfileViews): TProfileIssueRequestInput | undefined => {
      if (domain !== EIssuesStoreType.PROFILE || !accountId) return undefined;
      const snapshot = issueFilters;
      return {
        kind: "profile",
        workspaceSlug,
        userId: entityId,
        accountId,
        view,
        richFilters: snapshot.richFilters ?? {},
        displayFilters: snapshot.displayFilters,
        isCurrent: requestIsCurrent(snapshot, view),
        isReadScopeCurrent: readScopeIsCurrent,
      };
    },
    [accountId, domain, entityId, issueFilters, readScopeIsCurrent, requestIsCurrent, workspaceSlug]
  );
  const archivedRequestInput = useCallback((): TArchivedIssueRequestInput | undefined => {
    if (domain !== EIssuesStoreType.ARCHIVED || !accountId) return undefined;
    const snapshot = issueFilters;
    return {
      kind: "archived",
      workspaceSlug,
      projectId: entityId,
      accountId,
      richFilters: snapshot.richFilters ?? {},
      displayFilters: snapshot.displayFilters,
      isCurrent: requestIsCurrent(snapshot),
      isReadScopeCurrent: readScopeIsCurrent,
    };
  }, [accountId, domain, entityId, issueFilters, readScopeIsCurrent, requestIsCurrent, workspaceSlug]);
  const value = useMemo(
    () => ({
      domain,
      workspaceSlug,
      entityId,
      accountId,
      issueFilters,
      updateRichFilters,
      updateFilters,
      profileRequestInput,
      archivedRequestInput,
    }),
    [
      accountId,
      archivedRequestInput,
      domain,
      entityId,
      issueFilters,
      profileRequestInput,
      updateFilters,
      updateRichFilters,
      workspaceSlug,
    ]
  );
  return <LocalPreferencesContext.Provider value={value}>{children}</LocalPreferencesContext.Provider>;
}

export function useLocalIssuePreferences() {
  return useContext(LocalPreferencesContext);
}
