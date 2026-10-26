/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import isEqual from "lodash-es/isEqual.js";
import { workItemFiltersAdapter } from "@plane/shared-state";
import { useParams, useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IIssueFilters,
  TIssueKanbanFilters,
  TWorkItemFilterExpression,
  IWorkspaceMemberMe,
} from "@plane/types";
import { EIssueLayoutTypes, STATIC_VIEW_TYPES } from "@plane/types";
import { EIssueFilterType } from "@plane/constants";
import { getComputedDisplayFilters, getComputedDisplayProperties } from "@plane/utils/work-item";
import { sanitizeAndStabilizeExpression } from "@plane/utils/rich-filters";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useWorkspaceView } from "@/hooks/use-workspace-views";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { workspaceViewKeys } from "@/queries/global-views";
import { storage } from "@/lib/local-storage";
import type { TWorkspaceIssueRequestInput } from "@/helpers/issue-list/request";

type Draft = {
  revision: number;
  richFilters?: TWorkItemFilterExpression;
  displayFilters?: IIssueDisplayFilterOptions;
  displayProperties?: IIssueDisplayProperties;
  kanbanFilters?: TIssueKanbanFilters;
};
type StoredFilters = {
  display_filters?: IIssueDisplayFilterOptions;
  display_properties?: IIssueDisplayProperties;
  kanban_filters?: TIssueKanbanFilters;
};
type PreferenceUpdate =
  | [type: EIssueFilterType.DISPLAY_FILTERS, value: IIssueDisplayFilterOptions]
  | [type: EIssueFilterType.DISPLAY_PROPERTIES, value: IIssueDisplayProperties]
  | [type: EIssueFilterType.KANBAN_FILTERS, value: Partial<TIssueKanbanFilters>];
type LocalRecord = { accountId: string; workspaceSlug: string; viewId: string; filters: StoredFilters };
const WORKSPACE_ISSUE_PREFERENCES_KEY = "workspace_issue_preferences";

function readLocalRecords(): unknown[] {
  try {
    const value: unknown = JSON.parse(storage.get(WORKSPACE_ISSUE_PREFERENCES_KEY) ?? "[]");
    return Array.isArray(value) ? value : [];
  } catch {
    return [];
  }
}

function isLocalRecord(value: unknown, accountId: string, workspaceSlug: string, viewId: string): value is LocalRecord {
  if (typeof value !== "object" || value === null) return false;
  return (
    "accountId" in value &&
    value.accountId === accountId &&
    "workspaceSlug" in value &&
    value.workspaceSlug === workspaceSlug &&
    "viewId" in value &&
    value.viewId === viewId &&
    "filters" in value &&
    typeof value.filters === "object" &&
    value.filters !== null
  );
}

function readLocalPreferences(accountId: string, workspaceSlug: string, viewId: string): StoredFilters {
  const record = readLocalRecords().find((entry): entry is LocalRecord =>
    isLocalRecord(entry, accountId, workspaceSlug, viewId)
  );
  return record?.filters ?? {};
}

function saveLocalPreference(
  accountId: string,
  workspaceSlug: string,
  viewId: string,
  field: keyof StoredFilters,
  value: StoredFilters[keyof StoredFilters]
) {
  try {
    const records = readLocalRecords();
    const index = records.findIndex((record) => isLocalRecord(record, accountId, workspaceSlug, viewId));
    const existing = index < 0 ? undefined : records[index];
    const base = isLocalRecord(existing, accountId, workspaceSlug, viewId)
      ? existing
      : { accountId, workspaceSlug, viewId, filters: {} };
    const next = { ...base, filters: { ...base.filters, [field]: value } };
    if (index < 0) records.push(next);
    else records[index] = next;
    storage.set(WORKSPACE_ISSUE_PREFERENCES_KEY, JSON.stringify(records));
  } catch {
    // Browser appearance persistence cannot reject the in-memory preference change.
  }
}

function hasCurrentMembership(
  client: QueryClient,
  key: readonly unknown[],
  query: Query | undefined,
  expected: IWorkspaceMemberMe | undefined
) {
  const current = client.getQueryData<IWorkspaceMemberMe>(key);
  return (
    client.getQueryCache().find({ queryKey: key, exact: true }) === query &&
    query?.state.status === "success" &&
    current?.is_active === true &&
    current.role === expected?.role &&
    current.updated_at === expected?.updated_at
  );
}

function readWorkspacePreferenceAccess(
  client: QueryClient,
  metadata: ReturnType<typeof useWorkspaceView>,
  accountId: string,
  workspaceSlug: string,
  viewId: string,
  generation: number
) {
  const membership = metadata.membershipQuery.isSuccess ? metadata.membershipQuery.data : undefined;
  const membershipKey = permissionKeys.workspaceMembership(accountId, workspaceSlug);
  const membershipIdentity = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const viewIdentity = client.getQueryCache().find({
    queryKey: workspaceViewKeys.list({
      accountId,
      workspaceSlug,
      workspaceRole: membership?.role ?? undefined,
      membershipRevision: membership?.updated_at,
      sessionGeneration: generation,
    }),
    exact: true,
  });
  const isStatic = STATIC_VIEW_TYPES.includes(viewId);
  const ready =
    metadata.canRead &&
    hasCurrentMembership(client, membershipKey, membershipIdentity, membership) &&
    (isStatic || (metadata.isSuccess && metadata.view !== undefined && viewIdentity?.state.status === "success"));
  return { membership, membershipIdentity, viewIdentity, ready };
}

function stableRichExpression(expression: TWorkItemFilterExpression) {
  try {
    const tree = sanitizeAndStabilizeExpression(
      workItemFiltersAdapter.toInternal(expression),
      (condition) => condition.value !== undefined
    );
    return tree ? workItemFiltersAdapter.toExternal(tree) : {};
  } catch {
    // Unsupported saved predicates are only equivalent when their original data is equal.
    return expression;
  }
}

export function areWorkspaceRichFiltersEquivalent(left: TWorkItemFilterExpression, right: TWorkItemFilterExpression) {
  return isEqual(left, right) || isEqual(stableRichExpression(left), stableRichExpression(right));
}

function normalizeWorkspaceDisplay(filters: IIssueDisplayFilterOptions): IIssueDisplayFilterOptions {
  const next = getComputedDisplayFilters(filters, { layout: EIssueLayoutTypes.SPREADSHEET, order_by: "-created_at" });
  if (next.order_by === "sort_order") next.order_by = "-created_at";
  if (next.group_by === null) next.sub_group_by = null;
  if (next.layout === EIssueLayoutTypes.KANBAN && next.group_by === next.sub_group_by) next.sub_group_by = null;
  if (next.layout === EIssueLayoutTypes.KANBAN && next.group_by === null) next.group_by = "state";
  return next;
}

type Preferences = {
  workspaceSlug: string;
  viewId: string;
  revision: number;
  isExpressionDirty: boolean;
  metadata: ReturnType<typeof useWorkspaceView>;
  issueFilters: IIssueFilters | undefined;
  requestInput: TWorkspaceIssueRequestInput | undefined;
  updateExpression(expression: TWorkItemFilterExpression): void;
  updateFilters(...change: PreferenceUpdate): void;
  acknowledgeSave(viewId: string, revision: number, richOnly?: boolean): boolean;
};
const WorkspaceIssuePreferencesContext = createContext<Preferences | undefined>(undefined);
export const useWorkspaceIssuePreferences = () => useContext(WorkspaceIssuePreferencesContext);

export function WorkspaceIssuePreferencesProvider({ children }: { children: React.ReactNode }) {
  const { workspaceSlug, globalViewId } = useParams({ strict: false });
  const client = useQueryClient();
  const user = useCurrentUser();
  const generation = useIdentityGeneration(client);
  const accountId = user.isSuccess ? user.data?.id : undefined;
  if (!workspaceSlug || !globalViewId || !accountId)
    return (
      <WorkspaceIssuePreferencesContext.Provider value={undefined}>
        {children}
      </WorkspaceIssuePreferencesContext.Provider>
    );
  return (
    <ActiveWorkspaceIssuePreferences
      key={`${accountId}/${generation}/${workspaceSlug}/${globalViewId}`}
      workspaceSlug={workspaceSlug}
      viewId={globalViewId}
      accountId={accountId}
      generation={generation}
    >
      {children}
    </ActiveWorkspaceIssuePreferences>
  );
}

function ActiveWorkspaceIssuePreferences({
  children,
  workspaceSlug,
  viewId,
  accountId,
  generation,
}: {
  children: React.ReactNode;
  workspaceSlug: string;
  viewId: string;
  accountId: string;
  generation: number;
}) {
  const client = useQueryClient();
  const router = useRouter();
  const metadata = useWorkspaceView(workspaceSlug, viewId);
  const isStatic = STATIC_VIEW_TYPES.includes(viewId);
  const [local] = useState(() => readLocalPreferences(accountId, workspaceSlug, viewId));
  const [draft, setDraft] = useState<Draft>({ revision: 0 });
  const draftRef = useRef(draft);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useLayoutEffect(() => {
    draftRef.current = draft;
  }, [draft]);
  const canonicalRichFilters = useMemo(
    () => (isStatic ? {} : (metadata.view?.rich_filters ?? {})),
    [isStatic, metadata.view?.rich_filters]
  );
  const richFilters = draft.richFilters ?? canonicalRichFilters;
  const displayFilters = useMemo(
    () =>
      normalizeWorkspaceDisplay(
        draft.displayFilters ?? (isStatic ? local.display_filters : metadata.view?.display_filters) ?? {}
      ),
    [draft.displayFilters, isStatic, local.display_filters, metadata.view?.display_filters]
  );
  const displayProperties = useMemo(
    () =>
      getComputedDisplayProperties(
        draft.displayProperties ?? (isStatic ? local.display_properties : metadata.view?.display_properties) ?? {}
      ),
    [draft.displayProperties, isStatic, local.display_properties, metadata.view?.display_properties]
  );
  const { membership, membershipIdentity, viewIdentity, ready } = readWorkspacePreferenceAccess(
    client,
    metadata,
    accountId,
    workspaceSlug,
    viewId,
    generation
  );
  const inputToken = useMemo(() => ({ richFilters, displayFilters }), [richFilters, displayFilters]);
  const currentInput = useRef<object | undefined>(undefined);
  useLayoutEffect(() => {
    currentInput.current = ready ? inputToken : undefined;
    return () => {
      currentInput.current = undefined;
    };
  }, [inputToken, ready]);
  const requestInput = useMemo<TWorkspaceIssueRequestInput | undefined>(() => {
    if (!ready) return undefined;
    const identity = captureIdentityScope(client, accountId);
    const membershipKey = permissionKeys.workspaceMembership(accountId, workspaceSlug);
    const membershipQuery = membershipIdentity;
    const viewKey = workspaceViewKeys.list({
      accountId,
      workspaceSlug,
      workspaceRole: membership?.role ?? undefined,
      membershipRevision: membership?.updated_at,
      sessionGeneration: generation,
    });
    const viewQuery = viewIdentity;
    const permissionIsCurrent = () => hasCurrentMembership(client, membershipKey, membershipQuery, membership);
    const isScopeCurrent = () => {
      const route = router.state.matches.at(-1);
      const params = route?.params;
      return (
        params !== undefined &&
        "workspaceSlug" in params &&
        params.workspaceSlug === workspaceSlug &&
        "globalViewId" in params &&
        params.globalViewId === viewId &&
        currentInput.current === inputToken &&
        isIdentityScopeCurrent(client, identity) &&
        permissionIsCurrent() &&
        (isStatic ||
          (client.getQueryCache().find({ queryKey: viewKey, exact: true }) === viewQuery &&
            viewQuery?.state.status === "success"))
      );
    };
    return {
      kind: "workspace",
      accountId,
      workspaceSlug,
      viewId,
      ...inputToken,
      isScopeCurrent,
      isReadScopeCurrent: () => isScopeCurrent() && isIdentityScopeVerified(client, identity),
    };
  }, [
    accountId,
    client,
    generation,
    inputToken,
    isStatic,
    membership,
    membershipIdentity,
    viewIdentity,
    ready,
    router,
    viewId,
    workspaceSlug,
  ]);
  const issueFilters = useMemo<IIssueFilters | undefined>(
    () =>
      ready
        ? {
            richFilters,
            displayFilters,
            displayProperties,
            kanbanFilters: draft.kanbanFilters ?? local.kanban_filters,
          }
        : undefined,
    [ready, richFilters, displayFilters, displayProperties, draft.kanbanFilters, local.kanban_filters]
  );
  const updateExpression = useCallback(
    (expression: TWorkItemFilterExpression) => {
      setDraft((current) => {
        if (areWorkspaceRichFiltersEquivalent(current.richFilters ?? canonicalRichFilters, expression)) return current;
        return {
          ...current,
          revision: current.revision + 1,
          richFilters: areWorkspaceRichFiltersEquivalent(canonicalRichFilters, expression) ? undefined : expression,
        };
      });
    },
    [canonicalRichFilters]
  );
  const updateFilters = useCallback(
    (...[type, value]: PreferenceUpdate) => {
      setDraft((current) => {
        if (type === EIssueFilterType.DISPLAY_FILTERS)
          return {
            ...current,
            revision: current.revision + 1,
            displayFilters: normalizeWorkspaceDisplay({
              ...(current.displayFilters ?? displayFilters),
              ...value,
            }),
          };
        if (type === EIssueFilterType.DISPLAY_PROPERTIES)
          return {
            ...current,
            revision: current.revision + 1,
            displayProperties: {
              ...(current.displayProperties ?? displayProperties),
              ...value,
            },
          };
        if (type === EIssueFilterType.KANBAN_FILTERS)
          return {
            ...current,
            revision: current.revision + 1,
            kanbanFilters: {
              group_by: [],
              sub_group_by: [],
              ...(current.kanbanFilters ?? local.kanban_filters),
              ...value,
            },
          };
        return current;
      });
    },
    [displayFilters, displayProperties, local.kanban_filters]
  );
  useEffect(() => {
    if (isStatic && draft.displayFilters)
      saveLocalPreference(accountId, workspaceSlug, viewId, "display_filters", draft.displayFilters);
  }, [accountId, draft.displayFilters, isStatic, workspaceSlug, viewId]);
  useEffect(() => {
    if (isStatic && draft.displayProperties)
      saveLocalPreference(accountId, workspaceSlug, viewId, "display_properties", draft.displayProperties);
  }, [accountId, draft.displayProperties, isStatic, workspaceSlug, viewId]);
  useEffect(() => {
    if (draft.kanbanFilters)
      saveLocalPreference(accountId, workspaceSlug, viewId, "kanban_filters", draft.kanbanFilters);
  }, [accountId, draft.kanbanFilters, workspaceSlug, viewId]);

  const acknowledgeSave = useCallback(
    (savedViewId: string, revision: number, richOnly = false) => {
      if (
        !mounted.current ||
        savedViewId !== viewId ||
        draftRef.current.revision !== revision ||
        getIdentitySessionGeneration(client) !== generation ||
        !isIdentityScopeCurrent(client, captureIdentityScope(client, accountId))
      )
        return false;
      setDraft((current) =>
        current.revision !== revision
          ? current
          : richOnly
            ? { ...current, richFilters: undefined }
            : { revision: revision + 1 }
      );
      return true;
    },
    [accountId, client, generation, viewId]
  );
  const value = useMemo<Preferences>(
    () => ({
      workspaceSlug,
      viewId,
      revision: draft.revision,
      isExpressionDirty: draft.richFilters !== undefined,
      metadata,
      issueFilters,
      requestInput,
      updateExpression,
      updateFilters,
      acknowledgeSave,
    }),
    [
      workspaceSlug,
      viewId,
      draft.revision,
      draft.richFilters,
      metadata,
      issueFilters,
      requestInput,
      updateExpression,
      updateFilters,
      acknowledgeSave,
    ]
  );
  return (
    <WorkspaceIssuePreferencesContext.Provider value={value}>{children}</WorkspaceIssuePreferencesContext.Provider>
  );
}
