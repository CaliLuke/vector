/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { IIssueDisplayFilterOptions, TWorkItemFilterExpression, IssuePaginationOptions } from "@plane/types";
import { getIssueFilterRequestParams, getPaginationParams } from "./request";
export type TParentIssueRequestInput = {
  kind: "cycle" | "module";
  workspaceSlug: string;
  projectId: string;
  viewId: string;
  accountId: string;
  richFilters: TWorkItemFilterExpression;
  displayFilters: IIssueDisplayFilterOptions | undefined;
  parentStatistics?: import("@/queries/parent-statistics").ParentStatisticsEffects;
  isScopeCurrent: () => boolean;
  isReadScopeCurrent: () => boolean;
};
export function parentIssueRequestParams(
  input: TParentIssueRequestInput,
  options: IssuePaginationOptions,
  cursor?: string,
  groupId?: string,
  subGroupId?: string
) {
  const params = getIssueFilterRequestParams(
    { richFilters: input.richFilters, displayFilters: input.displayFilters },
    { includePropertyProjection: true, omit: input.kind }
  );
  return getPaginationParams({ ...params, [input.kind]: input.viewId }, options, cursor, groupId, subGroupId);
}
