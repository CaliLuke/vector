/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssueLayoutTypes } from "@plane/types";
import isNil from "lodash-es/isNil.js";
import orderBy from "lodash-es/orderBy.js";
// plane imports
import { SPACE_BASE_PATH, SPACE_BASE_URL } from "@plane/constants";
import type { IProjectView, TViewFilterProps, TViewFiltersSortBy, TViewFiltersSortKey } from "@plane/types";
// local imports
import { matchesFilterValue, matchesDateFilters } from "./filter";

/**
 * order views base on TViewFiltersSortKey
 * @param views
 * @param sortByKey
 * @param sortByOrder
 * @returns
 */
export const orderViews = (
  views: IProjectView[],
  sortByKey: TViewFiltersSortKey | undefined,
  sortByOrder: TViewFiltersSortBy
): IProjectView[] => {
  if (views.length === 0 || !sortByKey) return [];

  let iterableFunction;
  if (sortByKey === "name") {
    iterableFunction = (view: IProjectView) => view.name?.toLowerCase();
  }
  if (sortByKey === "created_at") {
    iterableFunction = (view: IProjectView) => view.created_at;
  }
  if (sortByKey === "updated_at") {
    iterableFunction = (view: IProjectView) => view.updated_at;
  }

  if (!iterableFunction) return [];

  return orderBy(views, [iterableFunction], [sortByOrder]);
};

/**
 * Checks if the passed down view should be filtered or not
 * @param view
 * @param filters
 * @returns
 */
export const shouldFilterView = (view: Partial<IProjectView>, filters: TViewFilterProps | undefined): boolean => {
  return (
    matchesFilterValue(view.created_by, filters?.owned_by) &&
    matchesDateFilters(view.created_at, filters?.created_at) &&
    matchesFilterValue(view.access, filters?.view_type) &&
    (!filters?.favorites || view.is_favorite === true)
  );
};

/**
 * @description returns the name of the project after checking for untitled view
 * @param {string | undefined} name
 * @returns {string}
 */
export const getViewName = (name: string | undefined) => {
  if (name === undefined) return "";
  if (!name || name.trim() === "") return "Untitled";
  return name;
};

/**
 * Adds validation for the view creation filters
 * @param data
 * @returns
 */
export const getValidatedViewFilters = (data: Partial<IProjectView>) => {
  if (
    data?.display_filters &&
    data?.display_filters?.layout === EIssueLayoutTypes.KANBAN &&
    isNil(data.display_filters.group_by)
  ) {
    data.display_filters.group_by = "state";
  }

  return data;
};

/**
 * returns published view link
 * @param anchor
 * @returns
 */
export const getPublishViewLink = (anchor: string | undefined) => {
  if (!anchor) return undefined;

  const SPACE_APP_URL = (SPACE_BASE_URL.trim() === "" ? window.location.origin : SPACE_BASE_URL) + SPACE_BASE_PATH;
  return `${SPACE_APP_URL}/views/${anchor}`;
};
