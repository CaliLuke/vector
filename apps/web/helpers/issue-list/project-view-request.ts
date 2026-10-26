/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IssuePaginationOptions,
  TIssueKanbanFilters,
  TIssueParams,
  TWorkItemFilterExpression,
} from "@plane/types";
import {
  getComputedDisplayFilters,
  getComputedDisplayProperties,
  handleIssueQueryParamsByLayout,
} from "@plane/utils/work-item";
import { computedFilteredParams, getPaginationParams } from "./request";

export type TProjectViewIssueRequestInput = {
  workspaceSlug: string;
  projectId: string;
  viewId: string;
  accountId: string;
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions;
  displayProperties: IIssueDisplayProperties;
  kanbanFilters: TIssueKanbanFilters;
  isScopeCurrent: () => boolean;
  isReadScopeCurrent: () => boolean;
};

export function getProjectViewIssueRequestParams(
  input: TProjectViewIssueRequestInput,
  options: IssuePaginationOptions,
  cursor?: string,
  groupId?: string,
  subGroupId?: string
): Partial<Record<TIssueParams, string | boolean>> {
  const displayFilters = getComputedDisplayFilters(input.displayFilters);
  const allowedParams = handleIssueQueryParamsByLayout(displayFilters.layout, "issues");
  const filters = computedFilteredParams(input.richFilters, displayFilters, allowedParams ?? []);
  return getPaginationParams(filters, options, cursor, groupId, subGroupId);
}

export function getProjectViewIssueFilters(
  input: Pick<TProjectViewIssueRequestInput, "richFilters" | "displayFilters" | "displayProperties" | "kanbanFilters">
) {
  return {
    richFilters: input.richFilters,
    displayFilters: getComputedDisplayFilters(input.displayFilters),
    displayProperties: getComputedDisplayProperties(input.displayProperties),
    kanbanFilters: input.kanbanFilters,
  };
}
