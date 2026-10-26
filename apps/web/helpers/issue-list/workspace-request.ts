/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import isEmpty from "lodash-es/isEmpty.js";
import { EIssueLayoutTypes } from "@plane/types";
import type { IIssueDisplayFilterOptions, IssuePaginationOptions, TIssueParams } from "@plane/types";
import { getComputedDisplayFilters, handleIssueQueryParamsByLayout } from "@plane/utils/work-item";
import type { TWorkspaceIssueRequestInput } from "./request";
import { computedFilteredParams, getPaginationParams } from "./request";

const workspaceStaticViewParams = (viewId: string, accountId: string) => {
  switch (viewId) {
    case "assigned":
      return { assignees: accountId };
    case "created":
      return { created_by: accountId };
    case "subscribed":
      return { subscriber: accountId };
    default:
      return undefined;
  }
};

export const getWorkspaceIssueRequestParams = (
  input: TWorkspaceIssueRequestInput,
  options: IssuePaginationOptions,
  cursor: string | undefined,
  groupId?: string,
  subGroupId?: string
): Partial<Record<TIssueParams, string | boolean>> => {
  const displayFilters: IIssueDisplayFilterOptions = {
    ...getComputedDisplayFilters(input.displayFilters, {
      layout: EIssueLayoutTypes.SPREADSHEET,
      order_by: "-created_at",
    }),
  };

  // Global issue lists do not provide the project-level manual sort order.
  if (displayFilters.order_by === "sort_order") displayFilters.order_by = "-created_at";

  const acceptableParams = handleIssueQueryParamsByLayout(EIssueLayoutTypes.SPREADSHEET, "my_issues");
  const filterParams = acceptableParams
    ? computedFilteredParams(isEmpty(input.richFilters) ? {} : input.richFilters, displayFilters, acceptableParams, {
        includePropertyProjection: true,
      })
    : {};
  const staticViewParams = workspaceStaticViewParams(input.viewId, input.accountId);

  return getPaginationParams(
    staticViewParams ? { ...filterParams, ...staticViewParams } : filterParams,
    options,
    cursor,
    groupId,
    subGroupId
  );
};
