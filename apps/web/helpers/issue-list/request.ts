/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  IIssueFilters,
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IProjectUserPropertiesResponse,
  IssuePaginationOptions,
  TIssueParams,
  TWorkItemFilterExpression,
} from "@plane/types";
import { EIssueLayoutTypes } from "@plane/types";
import isEmpty from "lodash-es/isEmpty.js";
import {
  EIssueGroupByToServerOptions,
  EServerGroupByToFilterOptions,
  ENABLE_ISSUE_DEPENDENCIES,
} from "@plane/constants";
import {
  getComputedDisplayFilters,
  getComputedDisplayProperties,
  handleIssueQueryParamsByLayout,
} from "@plane/utils/work-item";

export interface TProjectIssueRequestInput {
  workspaceSlug: string;
  projectId: string;
  projectIdentifier?: string;
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions | undefined;
}

export interface TWorkspaceIssueRequestInput {
  kind: "workspace";
  workspaceSlug: string;
  viewId: string;
  accountId: string;
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions;
  /** Lineage-only check used for accepted mutation projection. */
  isScopeCurrent: () => boolean;
  /** Verified identity check used before workspace reads and page commits. */
  isReadScopeCurrent: () => boolean;
}

export type TProfileIssueRequestInput = {
  kind: "profile";
  workspaceSlug: string;
  userId: string;
  accountId: string;
  view: import("@plane/types").TProfileViews;
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions | undefined;
  isCurrent: () => boolean;
  isReadScopeCurrent: () => boolean;
};

export type TArchivedIssueRequestInput = {
  kind: "archived";
  workspaceSlug: string;
  projectId: string;
  accountId: string;
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions | undefined;
  isCurrent: () => boolean;
  isReadScopeCurrent: () => boolean;
};

export const projectIssueRequestInputFromProperties = (
  workspaceSlug: string,
  projectId: string,
  properties: Pick<IProjectUserPropertiesResponse, "rich_filters" | "display_filters">,
  projectIdentifier?: string
): TProjectIssueRequestInput => ({
  workspaceSlug,
  projectId,
  projectIdentifier,
  richFilters: properties.rich_filters ?? {},
  displayFilters: getComputedDisplayFilters(properties.display_filters ?? {}),
});

export const projectIssueFiltersFromProperties = (
  properties: IProjectUserPropertiesResponse
): {
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions;
  displayProperties: IIssueDisplayProperties;
} => ({
  richFilters: properties.rich_filters ?? {},
  displayFilters: getComputedDisplayFilters(properties.display_filters ?? {}),
  displayProperties: getComputedDisplayProperties(properties.display_properties ?? {}),
});

export interface TIssueListProjection {
  orderBy: IIssueDisplayFilterOptions["order_by"] | undefined;
  groupBy: IIssueDisplayFilterOptions["group_by"] | "target_date" | undefined;
  subGroupBy: IIssueDisplayFilterOptions["sub_group_by"] | undefined;
  includeSubIssues: boolean;
}

export const getIssueListProjection = (
  displayFilters: IIssueDisplayFilterOptions | undefined
): TIssueListProjection => {
  const filters = getComputedDisplayFilters(displayFilters ?? {});
  const groupBy =
    filters.layout === EIssueLayoutTypes.CALENDAR
      ? "target_date"
      : filters.layout === EIssueLayoutTypes.LIST || filters.layout === EIssueLayoutTypes.KANBAN
        ? filters.group_by
        : undefined;

  return {
    orderBy: filters.order_by,
    groupBy,
    subGroupBy:
      filters.group_by === filters.sub_group_by
        ? undefined
        : filters.layout === EIssueLayoutTypes.KANBAN
          ? filters.sub_group_by
          : undefined,
    includeSubIssues: filters.sub_issue ?? false,
  };
};

export const matchesProjectIssueRequestScope = (
  input: Pick<TProjectIssueRequestInput, "workspaceSlug" | "projectId">,
  workspaceSlug: string,
  projectId: string
) => input.workspaceSlug === workspaceSlug && input.projectId === projectId;

function appendDisplayFilterParams(
  displayFilters: IIssueDisplayFilterOptions | undefined,
  acceptableParamsByLayout: TIssueParams[],
  target: Partial<Record<TIssueParams, string | boolean>>
) {
  const computedDisplayFilters: [TIssueParams, string[] | boolean | string | undefined][] = [
    ["group_by", displayFilters?.group_by ? EIssueGroupByToServerOptions[displayFilters.group_by] : undefined],
    [
      "sub_group_by",
      displayFilters?.sub_group_by ? EIssueGroupByToServerOptions[displayFilters.sub_group_by] : undefined,
    ],
    ["order_by", displayFilters?.order_by || undefined],
    ["sub_issue", displayFilters?.sub_issue ?? true],
  ];

  for (const [filterKey, value] of computedDisplayFilters) {
    const normalizedValue = Array.isArray(value) && value.length === 0 ? undefined : value;
    if (normalizedValue === undefined || !acceptableParamsByLayout.includes(filterKey)) continue;
    target[filterKey] = Array.isArray(normalizedValue) ? normalizedValue.join(",") : normalizedValue;
  }
}

export const computedFilteredParams = (
  richFilters: TWorkItemFilterExpression,
  displayFilters: IIssueDisplayFilterOptions | undefined,
  acceptableParamsByLayout: TIssueParams[],
  options: { includePropertyProjection?: boolean } = {}
): Partial<Record<TIssueParams, string | boolean>> => {
  const issueFiltersParams: Partial<Record<TIssueParams, boolean | string>> = {};
  appendDisplayFilterParams(displayFilters, acceptableParamsByLayout, issueFiltersParams);

  if (options.includePropertyProjection && displayFilters?.layout === EIssueLayoutTypes.SPREADSHEET)
    issueFiltersParams.include_properties = true;

  if (richFilters) issueFiltersParams.filters = JSON.stringify(richFilters);
  if (displayFilters?.layout) issueFiltersParams.layout = displayFilters.layout;
  if (ENABLE_ISSUE_DEPENDENCIES && displayFilters?.layout === EIssueLayoutTypes.GANTT)
    issueFiltersParams.expand = "issue_relation,issue_related";

  return issueFiltersParams;
};

/** Shared project-list request input with explicit read and mutation projections. */
export function getIssueFilterRequestParams(
  filters: Pick<IIssueFilters, "richFilters" | "displayFilters"> | undefined,
  options: { includePropertyProjection?: boolean; omit?: TIssueParams } = {}
) {
  if (!filters) return undefined;
  const acceptable = handleIssueQueryParamsByLayout(filters.displayFilters?.layout, "issues");
  if (!acceptable) return undefined;
  return computedFilteredParams(
    filters.richFilters,
    filters.displayFilters,
    acceptable.filter((key) => key !== options.omit),
    options
  );
}

export const getPaginationParams = (
  filterParams: Partial<Record<TIssueParams, string | boolean>> | undefined,
  options: IssuePaginationOptions,
  cursor: string | undefined,
  groupId?: string,
  subGroupId?: string
): Partial<Record<TIssueParams, string | boolean>> => {
  const pageCursor = cursor ? cursor : groupId ? `${options.perPageCount}:1:0` : `${options.perPageCount}:0:0`;
  const paginationParams: Partial<Record<TIssueParams, string | boolean>> = {
    ...filterParams,
    cursor: pageCursor,
    per_page: options.perPageCount.toString(),
  };

  if (options.groupedBy) paginationParams.group_by = options.groupedBy;
  if (options.after && options.before) paginationParams.target_date = `${options.after};after,${options.before};before`;

  if (groupId) {
    const groupBy = paginationParams.group_by;
    delete paginationParams.group_by;
    if (groupBy === "type_id") {
      paginationParams.group_filter_by = groupBy;
      paginationParams.group_filter_value = groupId;
    } else if (typeof groupBy === "string" && isServerGroupBy(groupBy))
      paginationParams[EServerGroupByToFilterOptions[groupBy]] = groupId;
  }

  if (subGroupId) {
    const subGroupBy = paginationParams.sub_group_by;
    delete paginationParams.sub_group_by;
    if (subGroupBy === "type_id") {
      paginationParams.sub_group_filter_by = subGroupBy;
      paginationParams.sub_group_filter_value = subGroupId;
    } else if (typeof subGroupBy === "string" && isServerGroupBy(subGroupBy))
      paginationParams[EServerGroupByToFilterOptions[subGroupBy]] = subGroupId;
  }

  return paginationParams;
};

function isServerGroupBy(value: string): value is keyof typeof EServerGroupByToFilterOptions {
  return Object.hasOwn(EServerGroupByToFilterOptions, value);
}

/**
 * Build a project request only from the preference snapshot supplied by the caller.
 * The caller owns scope validation and must not pass data from another project.
 */
export const getProjectIssueRequestParams = (
  input: TProjectIssueRequestInput,
  options: IssuePaginationOptions,
  cursor: string | undefined,
  groupId?: string,
  subGroupId?: string
) => {
  const richFilters = isEmpty(input.richFilters) ? {} : input.richFilters;
  const displayFilters = getComputedDisplayFilters(input.displayFilters ?? undefined);
  const acceptableParams = handleIssueQueryParamsByLayout(displayFilters.layout, "issues");
  const filterParams = acceptableParams
    ? computedFilteredParams(richFilters, displayFilters, acceptableParams, { includePropertyProjection: true })
    : undefined;

  return getPaginationParams(filterParams, options, cursor, groupId, subGroupId);
};

export const getProfileIssueRequestParams = (
  input: TProfileIssueRequestInput,
  options: IssuePaginationOptions,
  cursor: string | undefined,
  groupId?: string,
  subGroupId?: string
) => {
  const displayFilters = getComputedDisplayFilters(input.displayFilters ?? undefined);
  const acceptableParams = handleIssueQueryParamsByLayout(displayFilters.layout, "profile_issues");
  const filterParams = acceptableParams
    ? computedFilteredParams(input.richFilters, displayFilters, acceptableParams)
    : undefined;
  if (filterParams && input.view === "assigned") filterParams.assignees = input.userId;
  if (filterParams && input.view === "created") filterParams.created_by = input.userId;
  if (filterParams && input.view === "subscribed") filterParams.subscriber = input.userId;
  return getPaginationParams(filterParams, options, cursor, groupId, subGroupId);
};

export const getArchivedIssueRequestParams = (
  input: TArchivedIssueRequestInput,
  options: IssuePaginationOptions,
  cursor: string | undefined,
  groupId?: string,
  subGroupId?: string
) => {
  const filterParams = getIssueFilterRequestParams(
    { richFilters: input.richFilters, displayFilters: input.displayFilters },
    { includePropertyProjection: false }
  );
  return getPaginationParams(filterParams, options, cursor, groupId, subGroupId);
};
