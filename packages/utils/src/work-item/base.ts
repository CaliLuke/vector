/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { differenceInCalendarDays } from "date-fns/differenceInCalendarDays";
import isEmpty from "lodash-es/isEmpty.js";
import { v4 as uuidv4 } from "uuid";
// plane imports
import type { TIssueFilterPriorityObject, TIssuePriorities } from "@plane/constants";
import {
  ISSUE_DISPLAY_FILTERS_BY_PAGE,
  ISSUE_DISPLAY_PROPERTIES_KEYS,
  ISSUE_PRIORITY_FILTERS,
  STATE_GROUPS,
} from "@plane/constants";
import type {
  IGanttBlock,
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  TGroupedIssues,
  TIssue,
  TIssueGroupByOptions,
  TIssueOrderByOptions,
  TIssueParams,
  TStateGroups,
  TSubGroupedIssues,
  TUnGroupedIssues,
} from "@plane/types";
import { EIssueLayoutTypes } from "@plane/types";
// local imports
import { orderArrayBy } from "../array";
import { getDate } from "../datetime";
import { isEditorEmpty } from "../editor/common";

type THandleIssuesMutation = (
  formData: Partial<TIssue>,
  oldGroupTitle: string,
  selectedGroupBy: TIssueGroupByOptions,
  issueIndex: number,
  orderBy: TIssueOrderByOptions,
  prevData?:
    | {
        [key: string]: TIssue[];
      }
    | TIssue[]
) =>
  | {
      [key: string]: TIssue[];
    }
  | TIssue[]
  | undefined;

export const handleIssuesMutation: THandleIssuesMutation = (
  formData,
  oldGroupTitle,
  selectedGroupBy,
  issueIndex,
  orderBy,
  prevData
) => {
  if (!prevData) return prevData;

  if (Array.isArray(prevData)) {
    const updatedIssue = {
      ...prevData[issueIndex],
      ...formData,
    };

    prevData.splice(issueIndex, 1, updatedIssue);

    return [...prevData];
  }
  return updateGroupedIssue(formData, oldGroupTitle, selectedGroupBy, issueIndex, orderBy, prevData);
};

export const handleIssueQueryParamsByLayout = (
  layout: EIssueLayoutTypes | undefined,
  viewType:
    | "my_issues"
    | "issues"
    | "profile_issues"
    | "archived_issues"
    | "draft_issues"
    | "team_issues"
    | "team_project_work_items"
): TIssueParams[] | null => {
  const queryParams: TIssueParams[] = ["filters"];

  if (!layout) return null;

  const currentViewLayoutOptions = ISSUE_DISPLAY_FILTERS_BY_PAGE[viewType].layoutOptions[layout];

  // add display filters query params
  Object.keys(currentViewLayoutOptions.display_filters).forEach((option) => {
    if (option === "group_by" || option === "sub_group_by" || option === "order_by" || option === "type") {
      queryParams.push(option);
    }
  });

  // add extra options query params
  if (currentViewLayoutOptions.extra_options.access) {
    currentViewLayoutOptions.extra_options.values.forEach((option) => {
      queryParams.push(option);
    });
  }

  return queryParams;
};

/**
 *
 * @description create a partial issue creation payload with some default values. This function also parse the form field
 * like assignees, labels, etc. and add them to the payload
 * @param projectId project id to be added in the issue payload
 * @param formData partial issue data from the form. This will override the default values
 * @returns issue creation payload with client defaults
 */
export const createIssuePayload = (projectId: string, formData: Partial<TIssue>) => {
  const payload = {
    id: uuidv4(),
    project_id: projectId,
    priority: "none",
    label_ids: [],
    assignee_ids: [],
    sub_issues_count: 0,
    attachment_count: 0,
    link_count: 0,
    // tempId is used for optimistic updates. It is not a part of the API response.
    tempId: uuidv4(),
    // to be overridden by the form data
    ...formData,
  } satisfies Partial<TIssue>;

  return payload;
};

/**
 * @description check if the issue due date should be highlighted
 * @param date
 * @param stateGroup
 * @returns boolean
 */
export const shouldHighlightIssueDueDate = (
  date: string | Date | null,
  stateGroup: TStateGroups | undefined
): boolean => {
  if (!date || !stateGroup) return false;
  // if the issue is completed or cancelled, don't highlight the due date
  if ([STATE_GROUPS.completed.key, STATE_GROUPS.cancelled.key].includes(stateGroup)) return false;

  const parsedDate = getDate(date);
  if (!parsedDate) return false;

  const targetDateDistance = differenceInCalendarDays(parsedDate, new Date());

  // if the issue is overdue, highlight the due date
  return targetDateDistance <= 0;
};

export const getIssueBlocksStructure = (block: TIssue): IGanttBlock => ({
  data: block,
  id: block?.id,
  name: block?.name,
  sort_order: block?.sort_order,
  start_date: block?.start_date ?? undefined,
  target_date: block?.target_date ?? undefined,
  meta: {
    project_id: block?.project_id ?? undefined,
  },
});

export const formatTextList = (TextArray: string[]): string => {
  const count = TextArray.length;
  switch (count) {
    case 0:
      return "";
    case 1:
      return TextArray[0];
    case 2:
      return `${TextArray[0]} and ${TextArray[1]}`;
    case 3:
      return `${TextArray.slice(0, 2).join(", ")}, and ${TextArray[2]}`;
    case 4:
      return `${TextArray.slice(0, 3).join(", ")}, and ${TextArray[3]}`;
    default:
      return `${TextArray.slice(0, 3).join(", ")}, and +${count - 3} more`;
  }
};

export const getDescriptionPlaceholderI18n = (isFocused: boolean, description: string | undefined): string => {
  const isDescriptionEmpty = isEditorEmpty(description);
  if (!isDescriptionEmpty || isFocused) return "common.press_for_commands";
  else return "common.click_to_add_description";
};

type CountedIssueIds = TGroupedIssues | TUnGroupedIssues | TSubGroupedIssues | null | undefined;

function countIssueGroups(issueIds: CountedIssueIds, depth: number): number {
  if (!issueIds) return 0;
  if (depth === 0) return Array.isArray(issueIds) ? issueIds.length : 0;
  if (Array.isArray(issueIds)) return 0;
  return Object.values(issueIds).reduce((count, group) => count + countIssueGroups(group, depth - 1), 0);
}

export const issueCountBasedOnFilters = (
  issueIds: CountedIssueIds,
  layout: unknown,
  groupBy: string | undefined,
  subGroupBy: string | undefined
): number => {
  switch (layout) {
    case EIssueLayoutTypes.SPREADSHEET:
    case EIssueLayoutTypes.GANTT:
      return countIssueGroups(issueIds, 0);
    case EIssueLayoutTypes.CALENDAR:
      return countIssueGroups(issueIds, 1);
    case EIssueLayoutTypes.LIST:
      return countIssueGroups(issueIds, groupBy ? 1 : 0);
    case EIssueLayoutTypes.KANBAN:
      return groupBy ? countIssueGroups(issueIds, subGroupBy ? 2 : 1) : 0;
    default:
      return 0;
  }
};

/**
 * @description This method is used to apply the display filters on the issues
 * @param {IIssueDisplayFilterOptions} displayFilters
 * @returns {IIssueDisplayFilterOptions}
 */
export function isParentTypeGroupingActive(filters: IIssueDisplayFilterOptions | undefined) {
  return Boolean(
    filters?.group_by?.startsWith("parent_type_") &&
    (!filters.layout || filters.layout === EIssueLayoutTypes.LIST || filters.layout === EIssueLayoutTypes.KANBAN)
  );
}

export const getComputedDisplayFilters = (
  displayFilters: IIssueDisplayFilterOptions = {},
  defaultValues?: IIssueDisplayFilterOptions
): IIssueDisplayFilterOptions => {
  const filters = (!isEmpty(displayFilters) ? displayFilters : defaultValues) ?? {};
  return {
    calendar: {
      show_weekends: filters.calendar?.show_weekends || false,
      layout: filters.calendar?.layout || "month",
    },
    layout: filters.layout || EIssueLayoutTypes.LIST,
    order_by: filters.order_by || "sort_order",
    group_by: filters.group_by || null,
    sub_group_by: filters.sub_group_by || null,
    sub_issue: computedSubIssueExpansion(filters),
    show_empty_groups: filters.show_empty_groups || false,
  };
};

/**
 * @description This method is used to apply the display properties on the issues
 * @param {IIssueDisplayProperties} displayProperties
 * @returns {IIssueDisplayProperties}
 */
function customDisplayProperties(properties: IIssueDisplayProperties) {
  return Object.fromEntries(
    Object.entries(properties ?? {}).filter(
      ([key, value]) => /^customproperty_[a-zA-Z0-9_-]+$/.test(key) && typeof value === "boolean"
    )
  );
}

export const getComputedDisplayProperties = (
  displayProperties: IIssueDisplayProperties = {}
): IIssueDisplayProperties => ({
  ...customDisplayProperties(displayProperties),
  ...Object.fromEntries(
    ISSUE_DISPLAY_PROPERTIES_KEYS.map((property) => [property, displayProperties?.[property] ?? true])
  ),
});

export const generateWorkItemLink = ({
  workspaceSlug,
  projectId,
  issueId,
  projectIdentifier,
  sequenceId,
  isArchived = false,
  isEpic = false,
}: {
  workspaceSlug: string | undefined | null;
  projectId: string | undefined | null;
  issueId: string | undefined | null;
  projectIdentifier: string | undefined | null;
  sequenceId: string | number | undefined | null;
  isArchived?: boolean;
  isEpic?: boolean;
}): string => {
  const archiveIssueLink = `/${workspaceSlug}/projects/${projectId}/archives/issues/${issueId}`;
  const workItemLink = `/${workspaceSlug}/browse/${projectIdentifier}-${sequenceId}/`;
  const epicLink = workItemLink;

  return isArchived ? archiveIssueLink : isEpic ? epicLink : workItemLink;
};

export const getIssuePriorityFilters = (priorityKey: TIssuePriorities): TIssueFilterPriorityObject | undefined => {
  const currentIssuePriority: TIssueFilterPriorityObject | undefined =
    ISSUE_PRIORITY_FILTERS && ISSUE_PRIORITY_FILTERS.length > 0
      ? ISSUE_PRIORITY_FILTERS.find((_priority) => _priority.key === priorityKey)
      : undefined;

  if (currentIssuePriority) return currentIssuePriority;
  return undefined;
};

function computedSubIssueExpansion(filters: IIssueDisplayFilterOptions | undefined) {
  return !isParentTypeGroupingActive(filters) && (filters?.sub_issue || false);
}

function updateGroupedIssue(
  formData: Partial<TIssue>,
  oldGroupTitle: string,
  selectedGroupBy: TIssueGroupByOptions,
  issueIndex: number,
  orderBy: TIssueOrderByOptions,
  prevData: Record<string, TIssue[]>
) {
  const oldGroup = prevData[oldGroupTitle ?? ""] ?? [];

  let newGroup: TIssue[] = [];

  if (selectedGroupBy === "priority") newGroup = prevData[formData.priority ?? ""] ?? [];
  else if (selectedGroupBy === "state") newGroup = prevData[formData.state_id ?? ""] ?? [];

  const updatedIssue = {
    ...oldGroup[issueIndex],
    ...formData,
  };

  if (selectedGroupBy !== Object.keys(formData)[0])
    return {
      ...prevData,
      [oldGroupTitle ?? ""]: orderArrayBy(
        oldGroup.map((i) => (i.id === updatedIssue.id ? updatedIssue : i)),
        orderBy
      ),
    };

  const groupThatIsUpdated = selectedGroupBy === "priority" ? formData.priority : formData.state_id;

  return {
    ...prevData,
    [oldGroupTitle ?? ""]: orderArrayBy(
      oldGroup.filter((i) => i.id !== updatedIssue.id),
      orderBy
    ),
    [groupThatIsUpdated ?? ""]: orderArrayBy([...newGroup, updatedIssue], orderBy),
  };
}
