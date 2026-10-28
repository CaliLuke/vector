/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TEstimatePointProjection } from "@/queries/estimates";
import type { ParentNameProjections } from "@/queries/parent-catalogue-projection";
import type { TStateProjection } from "@/queries/states";
import isEmpty from "lodash-es/isEmpty.js";
import orderBy from "lodash-es/orderBy.js";
import type { IIssueLabel, IWorkspaceMember, TIssue, TIssueOrderByOptions } from "@plane/types";
import { ALL_ISSUES } from "@plane/constants";
import { convertToISODateString } from "@plane/utils/datetime";
import type { IssueListGroupedIds } from "@/queries/issue-lists";

function isLabelNameOrder(orderByKey: TIssueOrderByOptions | undefined) {
  return orderByKey === "labels__name" || orderByKey === "-labels__name";
}

function isStateNameOrder(orderByKey: TIssueOrderByOptions | undefined) {
  return orderByKey === "state__name" || orderByKey === "-state__name";
}

function hasOrderValue(
  issue: TIssue,
  byEstimate: boolean,
  byLabel: boolean,
  byState: boolean,
  states: TStateProjection | undefined
) {
  if (byEstimate) return !isEmpty(issue.estimate_point);
  if (byLabel) return !isEmpty(issue.label_ids);
  if (byState) return !isEmpty(states?.statesById[issue.state_id ?? ""]?.name);
  return !isEmpty(issue.assignee_ids);
}

function estimateOrderValue(issue: TIssue, estimates: TEstimatePointProjection | undefined) {
  const key =
    issue.project_id && issue.estimate_point
      ? estimates?.pointsByProject?.[issue.project_id]?.[issue.estimate_point]?.key
      : undefined;
  return key === undefined ? [] : [key];
}

function stateNameOrderValue(issue: TIssue, states: TStateProjection | undefined) {
  const name = states?.statesById[issue.state_id ?? ""]?.name;
  return name ? [name.toLocaleLowerCase()] : [];
}

function labelNameOrderValue(issue: TIssue, labelsById: ReadonlyMap<string, IIssueLabel> | undefined) {
  return (issue.label_ids ?? []).flatMap((labelId) => {
    const label = labelsById?.get(labelId);
    return label?.name && label.project_id === issue.project_id ? [label.name.toLocaleLowerCase()] : [];
  });
}

function assigneeNameOrderValue(issue: TIssue, membersById: ReadonlyMap<string, IWorkspaceMember> | undefined) {
  const names = (issue.assignee_ids ?? []).flatMap((memberId) => {
    const firstName = membersById?.get(memberId)?.member.first_name;
    return firstName ? [firstName.toLocaleLowerCase()] : [];
  });
  return orderBy(names, undefined, ["asc"]);
}

function issueOrderValue(
  issue: TIssue,
  byEstimate: boolean,
  byLabel: boolean,
  byState: boolean,
  labelsById: ReadonlyMap<string, IIssueLabel> | undefined,
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined,
  estimates: TEstimatePointProjection | undefined,
  states: TStateProjection | undefined
) {
  if (byEstimate) return estimateOrderValue(issue, estimates);
  if (byState) return stateNameOrderValue(issue, states);
  return byLabel ? labelNameOrderValue(issue, labelsById) : assigneeNameOrderValue(issue, membersById);
}

function canOrderIssueIds(
  issueIds: string[] | undefined,
  orderByKey: TIssueOrderByOptions | undefined,
  labelsById: ReadonlyMap<string, IIssueLabel> | undefined,
  estimates: TEstimatePointProjection | undefined,
  states: TStateProjection | undefined,
  parents: ParentNameProjections | undefined
): issueIds is string[] {
  return Boolean(
    issueIds &&
    isNativeDisplayOrder(orderByKey) &&
    hasCurrentOrderingCatalogues(orderByKey, labelsById, estimates, states, parents)
  );
}

function hasCurrentOrderingCatalogues(
  orderByKey: TIssueOrderByOptions | undefined,
  labelsById: ReadonlyMap<string, IIssueLabel> | undefined,
  estimates: TEstimatePointProjection | undefined,
  states: TStateProjection | undefined,
  parents: ParentNameProjections | undefined
): boolean {
  const parentKind = parentOrderKind(orderByKey);
  return Boolean(
    (!isEstimateNameOrder(orderByKey) || (estimates?.pointsByProject && estimates.isCurrent())) &&
    (!isLabelNameOrder(orderByKey) || labelsById !== undefined) &&
    (!isStateNameOrder(orderByKey) || states?.isCurrent()) &&
    (parentKind !== "cycle" || parents?.cycles.isCurrent()) &&
    (parentKind !== "module" || parents?.modules.isCurrent())
  );
}

function isEstimateNameOrder(orderByKey: TIssueOrderByOptions | undefined) {
  return orderByKey === "estimate_point__key" || orderByKey === "-estimate_point__key";
}

function parentOrderKind(key: TIssueOrderByOptions | undefined) {
  if (key === "issue_cycle__cycle__name" || key === "-issue_cycle__cycle__name") return "cycle";
  if (key === "issue_module__module__name" || key === "-issue_module__module__name") return "module";
  return undefined;
}
function parentOrderValue(issue: TIssue, kind: "cycle" | "module", parents: ParentNameProjections | undefined) {
  const rows = kind === "cycle" ? parents?.cycles.rows : parents?.modules.rows;
  const ids = kind === "cycle" ? (issue.cycle_id ? [issue.cycle_id] : []) : (issue.module_ids ?? []);
  return orderBy(
    ids.flatMap((id) => {
      const row = rows?.find((parent) => parent.id === id && parent.project_id === issue.project_id);
      return row?.name ? [row.name.toLocaleLowerCase()] : [];
    }),
    undefined,
    ["asc"]
  );
}
export function isNativeDisplayOrder(orderByKey: TIssueOrderByOptions | undefined) {
  return (
    parentOrderKind(orderByKey) !== undefined ||
    orderByKey === "assignees__first_name" ||
    orderByKey === "-assignees__first_name" ||
    orderByKey === "labels__name" ||
    orderByKey === "-labels__name" ||
    orderByKey === "state__name" ||
    orderByKey === "-state__name" ||
    orderByKey === "estimate_point__key" ||
    orderByKey === "-estimate_point__key"
  );
}

export function projectIssueIdsByDisplayOrder(
  issueIds: string[] | undefined,
  getIssueById: (issueId: string) => TIssue | undefined,
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined,
  orderByKey: TIssueOrderByOptions | undefined,
  labelsById?: ReadonlyMap<string, IIssueLabel>,
  estimates?: TEstimatePointProjection,
  states?: TStateProjection,
  parents?: ParentNameProjections
) {
  if (!canOrderIssueIds(issueIds, orderByKey, labelsById, estimates, states, parents)) return issueIds;
  const parentKind = parentOrderKind(orderByKey);
  const parentValue = (issue: TIssue) => (parentKind ? parentOrderValue(issue, parentKind, parents) : []);
  const byLabel = isLabelNameOrder(orderByKey);
  const byState = isStateNameOrder(orderByKey);
  const byEstimate = isEstimateNameOrder(orderByKey);

  const issues: TIssue[] = [];
  const missingIssueIds: string[] = [];
  for (const issueId of issueIds) {
    const issue = getIssueById(issueId);
    if (issue) issues.push(issue);
    else missingIssueIds.push(issueId);
  }
  const createdAtDescending = orderBy(issues, (issue) => convertToISODateString(issue.created_at), ["desc"]);
  const nameOrder: "asc" | "desc" = orderByKey?.startsWith("-") ? "desc" : "asc";

  const sortedIssueIds = orderBy(
    createdAtDescending,
    [
      (issue) =>
        (parentKind ? parentValue(issue).length > 0 : hasOrderValue(issue, byEstimate, byLabel, byState, states))
          ? 0
          : 1,
      (issue) =>
        parentKind
          ? parentValue(issue)
          : issueOrderValue(issue, byEstimate, byLabel, byState, labelsById, membersById, estimates, states),
    ],
    ["asc", nameOrder]
  ).map((issue) => issue.id);

  return [...sortedIssueIds, ...missingIssueIds];
}

export function projectAllIssueIdsByDisplayOrder(
  groupedIssueIds: IssueListGroupedIds | undefined,
  getIssueById: (issueId: string) => TIssue | undefined,
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined,
  orderByKey: TIssueOrderByOptions | undefined,
  labelsById?: ReadonlyMap<string, IIssueLabel>,
  estimates?: TEstimatePointProjection,
  states?: TStateProjection,
  parents?: ParentNameProjections
) {
  return projectIssueIdsByDisplayOrder(
    getGroupedIssueIds(groupedIssueIds, ALL_ISSUES),
    getIssueById,
    membersById,
    orderByKey,
    labelsById,
    estimates,
    states,
    parents
  );
}

export function projectGroupedIssueIdsByDisplayOrder(
  groupedIssueIds: IssueListGroupedIds | undefined,
  getIssueById: (issueId: string) => TIssue | undefined,
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined,
  orderByKey: TIssueOrderByOptions | undefined,
  labelsById?: ReadonlyMap<string, IIssueLabel>,
  estimates?: TEstimatePointProjection,
  states?: TStateProjection,
  parents?: ParentNameProjections
): IssueListGroupedIds | undefined {
  if (
    !groupedIssueIds ||
    !isNativeDisplayOrder(orderByKey) ||
    (isLabelNameOrder(orderByKey) && labelsById === undefined) ||
    (isStateNameOrder(orderByKey) && !states?.isCurrent())
  )
    return groupedIssueIds;

  const projected: IssueListGroupedIds = {};
  for (const [groupId, group] of Object.entries(groupedIssueIds)) {
    if (Array.isArray(group)) {
      projected[groupId] =
        projectIssueIdsByDisplayOrder(
          group,
          getIssueById,
          membersById,
          orderByKey,
          labelsById,
          estimates,
          states,
          parents
        ) ?? group;
      continue;
    }
    const projectedSubGroups: Record<string, string[]> = {};
    for (const [subGroupId, issueIds] of Object.entries(group)) {
      projectedSubGroups[subGroupId] =
        projectIssueIdsByDisplayOrder(
          issueIds,
          getIssueById,
          membersById,
          orderByKey,
          labelsById,
          estimates,
          states,
          parents
        ) ?? issueIds;
    }
    projected[groupId] = projectedSubGroups;
  }
  return projected;
}

export function getGroupedIssueIds(
  groupedIssueIds: IssueListGroupedIds | undefined,
  groupId?: string,
  subGroupId?: string
) {
  if (!groupedIssueIds) return undefined;
  if (!groupId) {
    const allIssues = groupedIssueIds[ALL_ISSUES];
    return Array.isArray(allIssues) ? allIssues : undefined;
  }

  const group = groupedIssueIds[groupId];
  if (!group) return undefined;
  if (Array.isArray(group)) return group;
  return subGroupId ? group[subGroupId] : undefined;
}

export function getVisibleGroupedIssueIds(
  groupedIssueIds: IssueListGroupedIds | undefined,
  getIssueById: (issueId: string) => TIssue | undefined,
  groupId?: string,
  subGroupId?: string
) {
  return getGroupedIssueIds(groupedIssueIds, groupId, subGroupId)?.filter(
    (issueId) => getIssueById(issueId) !== undefined
  );
}
