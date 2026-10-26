/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import orderBy from "lodash-es/orderBy.js";
import isEmpty from "lodash-es/isEmpty.js";
import indexOf from "lodash-es/indexOf.js";
import groupBy from "lodash-es/groupBy.js";
import sortBy from "lodash-es/sortBy.js";
import { ALL_ISSUES, FILTER_TO_ISSUE_MAP, ISSUE_PRIORITIES } from "@plane/constants";
import type { IIssueFilterOptions, TIssue, TIssueGroupByOptions, TIssueOrderByOptions } from "@plane/types";
import { checkDateCriteria, convertToISODateString, parseDateFilter } from "@plane/utils/datetime";
import { ISSUE_GROUP_BY_KEY } from "./properties";

/**
 * returns,
 * A compound key, if both groupId & subGroupId are defined
 * groupId, only if groupId is defined
 * ALL_ISSUES, if both groupId & subGroupId are not defined
 * @param groupId
 * @param subGroupId
 * @returns
 */
export const getGroupKey = (groupId?: string, subGroupId?: string) => {
  if (groupId && subGroupId && subGroupId !== "null") return `${groupId}_${subGroupId}`;

  if (groupId) return groupId;

  return ALL_ISSUES;
};

/**
 * This Method is mainly used to filter out empty values in the beginning
 * @param key key of the value that is to be checked if empty
 * @param object any object in which the key's value is to be checked
 * @returns 1 if empty, 0 if not empty
 */
const getSortOrderToFilterEmptyValues = (key: string, object: Record<string, unknown>) => {
  const value = object?.[key];

  if (typeof value !== "number" && isEmpty(value)) return 1;

  return 0;
};

// get IssueIds from Issue data List
const getIssueIds = (issues: TIssue[]) => issues.map((issue) => issue?.id);

/**
 * Checks if an issue meets the date filter criteria
 * @param issue The issue to check
 * @param filterKey The date field to check ('start_date' or 'target_date')
 * @param dateFilters Array of date filter strings
 * @returns boolean indicating if the issue meets the date criteria
 */
const checkIssueDateFilter = (
  issue: TIssue,
  filterKey: "start_date" | "target_date",
  dateFilters: string[]
): boolean => {
  if (!dateFilters || dateFilters.length === 0) return true;

  const issueDate = issue[filterKey];
  if (!issueDate) return false;

  // Issue should match all the date filters (AND operation)
  return dateFilters.every((filterValue) => {
    const parsed = parseDateFilter(filterValue);
    if (!parsed?.date || !parsed?.type) {
      // ignore invalid filter instead of failing the whole evaluation
      console.warn(`[filters] Ignoring unparsable date filter "${filterValue}"`);
      return true;
    }
    return checkDateCriteria(new Date(issueDate), parsed.date, parsed.type);
  });
};

/**
 * Filters the given work items based on the provided filters and display filters.
 * @param work items - The array of work items to be filtered.
 * @param filters - The filters to be applied to the issues.
 * @param displayFilters - The display filters to be applied to the issues.
 * @returns The filtered array of issues.
 */
export const getFilteredWorkItems = (workItems: TIssue[], filters: IIssueFilterOptions | undefined): TIssue[] => {
  if (!filters) return workItems;
  // Get all active filters
  const activeFilters = Object.keys(filters)
    .filter(isIssueFilterKey)
    .map((key) => [key, filters[key]] as const)
    .filter(([, values]) => Boolean(values?.length));
  // If no active filters, return all issues
  if (activeFilters.length === 0) {
    return workItems;
  }

  return workItems.filter((workItem) =>
    // Check all filter conditions (AND operation between different filters)
    activeFilters.every(([filterKey, filterValues]) => {
      // Handle date filters separately
      if (filterKey === "start_date" || filterKey === "target_date") {
        return (
          Array.isArray(filterValues) &&
          filterValues.every((value) => typeof value === "string") &&
          checkIssueDateFilter(workItem, filterKey, filterValues)
        );
      }
      // Handle regular filters
      const issueKey = FILTER_TO_ISSUE_MAP[filterKey];
      if (!issueKey) return true; // Skip if no mapping exists
      const issueValue = workItem[issueKey];
      // Handle array-based properties vs single value properties
      if (Array.isArray(issueValue)) {
        return filterValues?.some((filterValue) => issueValue.some((issueItem) => issueItem === filterValue)) ?? false;
      } else {
        return filterValues?.some((filterValue) => filterValue === issueValue) ?? false;
      }
    })
  );
};

function isIssueFilterKey(key: string): key is keyof IIssueFilterOptions {
  return key in FILTER_TO_ISSUE_MAP || key === "start_date" || key === "target_date";
}

/**
 * Orders the given work items based on the provided order by key.
 * @param workItems - The array of work items to be ordered.
 * @param orderByKey - The key to order the issues by.
 * @returns The ordered array of work items.
 */
const getOrderedWorkItems = (workItems: TIssue[], orderByKey: TIssueOrderByOptions): string[] => {
  switch (orderByKey) {
    case "-updated_at":
      return getIssueIds(orderBy(workItems, (item) => convertToISODateString(item["updated_at"]), ["desc"]));

    case "-created_at":
      return getIssueIds(orderBy(workItems, (item) => convertToISODateString(item["created_at"]), ["desc"]));

    case "-start_date":
      return getIssueIds(
        orderBy(
          workItems,
          [getSortOrderToFilterEmptyValues.bind(null, "start_date"), "start_date"], //preferring sorting based on empty values to always keep the empty values below
          ["asc", "desc"]
        )
      );

    case "-priority": {
      const sortArray = ISSUE_PRIORITIES.map((i) => i.key);
      return getIssueIds(
        orderBy(workItems, (currentIssue: TIssue) => indexOf(sortArray, currentIssue?.priority), ["asc"])
      );
    }
    default:
      return getIssueIds(workItems);
  }
};

export const getGroupedWorkItemIds = (
  workItems: TIssue[],
  groupByKey?: TIssueGroupByOptions,
  orderByKey: TIssueOrderByOptions = "-created_at"
): Record<string, string[]> => {
  // If group by is not set set default as ALL ISSUES
  if (!groupByKey) {
    return {
      [ALL_ISSUES]: getOrderedWorkItems(workItems, orderByKey),
    };
  }

  // Group work items
  const groupKey = getDefaultGroupKey(groupByKey);
  const groupedWorkItems = groupBy(workItems, (workItem) => {
    const value = groupKey ? workItem[groupKey] : null;
    if (Array.isArray(value)) {
      if (value.length === 0) return "None";
      if (value.every((item): item is string => typeof item === "string")) {
        // Sort & join to build deterministic set-like key without mutating the issue.
        return sortBy(value).join(",");
      }
      return JSON.stringify(value);
    }
    return value ?? "None";
  });

  // Convert to Record type
  const groupedWorkItemsRecord: Record<string, string[]> = {};
  Object.entries(groupedWorkItems).forEach(([key, items]) => {
    groupedWorkItemsRecord[key] = getOrderedWorkItems(items, orderByKey);
  });

  return groupedWorkItemsRecord;
};

function getDefaultGroupKey(grouping: TIssueGroupByOptions) {
  switch (grouping) {
    case "state_detail.group":
      return "state__group";
    case null:
      return null;
    default:
      return ISSUE_GROUP_BY_KEY[grouping];
  }
}
