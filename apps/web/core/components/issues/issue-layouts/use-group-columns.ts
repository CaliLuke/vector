/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useIssueParentCatalogues } from "@/hooks/use-issue-parent-catalogues";
import { useIssueLayoutLabels } from "@/hooks/use-issue-layout-labels";
import { useProjectStates, useWorkspaceStates } from "@/hooks/use-project-states";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
import { useWorkspaceMemberRoster, useProjectMemberRoster } from "@/hooks/use-member-rosters";
import { workspaceRosterMemberIds, projectRosterMemberIds } from "@/helpers/member-rosters";
import { useWorkItemGroupingCatalog } from "@/hooks/use-work-item-grouping-catalog";
import { typeGroupColumns } from "./type-group-columns";
import { getGroupByColumns } from "./utils";
import { parentGroupColumns } from "./parent-group-columns";
import { useOptionalIssueListView } from "@/providers/issue-list-view";
import { isParentTypeGrouping, isCustomGrouping } from "@/helpers/issue-list/grouping-axis";
import { useWorkItemTableCatalog } from "@/hooks/use-work-item-table-catalog";
import { propertyGroupColumns } from "./property-group-columns";

type Options = Omit<
  Parameters<typeof getGroupByColumns>[0],
  | "assigneeIds"
  | "creatorIds"
  | "membersById"
  | "joinedProjectIds"
  | "labels"
  | "typeColumns"
  | "states"
  | "projects"
  | "cycles"
  | "modules"
> & {
  workspaceSlug: string | undefined;
  isWorkspaceLevel: boolean;
  observedIds?: string[];
};

export function useGroupColumns({ workspaceSlug, isWorkspaceLevel, observedIds = [], ...options }: Options) {
  const catalogues = useGroupCatalogueData(workspaceSlug, options.projectId, isWorkspaceLevel, options.groupBy);
  const roster = useGroupRosterData(workspaceSlug, options.projectId, isWorkspaceLevel, options.groupBy);
  const custom = isCustomGrouping(options.groupBy);
  const properties = useWorkItemTableCatalog(custom && !options.isEpic);
  const list = useOptionalIssueListView();
  if (isParentTypeGrouping(options.groupBy))
    return list?.parentGroups ? parentGroupColumns(list.parentGroups) : undefined;
  if (custom) {
    const field = properties.fields.find((candidate) => candidate.key === options.groupBy && candidate.groupable);
    return field ? propertyGroupColumns(field, observedIds) : undefined;
  }
  return getGroupByColumns({
    ...options,
    ...roster,
    ...catalogues,
  });
}

type GroupBy = Options["groupBy"];

function useGroupCatalogueData(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  isWorkspaceLevel: boolean,
  groupBy: GroupBy
) {
  const { labels: workspaceLabels } = useIssueLayoutLabels(workspaceSlug, projectId, isWorkspaceLevel);
  const labels = isWorkspaceLevel
    ? workspaceLabels
    : workspaceLabels?.filter((label) => label.project_id === projectId);
  const { projects } = useWorkspaceProjects(groupBy === "project" ? workspaceSlug : undefined);
  const parentCatalogues = useIssueParentCatalogues(workspaceSlug, projectId, isWorkspaceLevel, {
    group_by: groupBy === "cycle" || groupBy === "module" ? groupBy : undefined,
  });
  const needsStates = groupBy === "state";
  const projectStates = useProjectStates(workspaceSlug, projectId, false, needsStates && !isWorkspaceLevel);
  const workspaceStates = useWorkspaceStates(workspaceSlug, needsStates && isWorkspaceLevel);
  const catalog = useWorkItemGroupingCatalog(groupBy === "type_id", { workspaceSlug, projectId });
  const typeColumns = catalog.identities?.enabled ? typeGroupColumns(catalog.identities.types) : undefined;

  return {
    labels,
    states: isWorkspaceLevel ? workspaceStates.states : projectStates.states,
    projects,
    cycles: parentCatalogues.parents.cycles.rows,
    modules: parentCatalogues.parents.modules.rows,
    typeColumns,
  };
}

function useGroupRosterData(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  isWorkspaceLevel: boolean,
  groupBy: GroupBy
) {
  const currentUserId = useCurrentUserId();
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug);
  const { members, membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const needsProjectRoster = groupBy === "assignees" || groupBy === "created_by";
  const { memberships } = useProjectMemberRoster(workspaceSlug, projectId, {
    canLoad: !isWorkspaceLevel && needsProjectRoster,
  });
  const workspaceIds = workspaceRosterMemberIds(members, currentUserId) ?? [];
  const projectAssigneeIds = projectRosterMemberIds(memberships, membersById, currentUserId, false) ?? [];
  const projectCreatorIds = projectRosterMemberIds(memberships, membersById, currentUserId, true) ?? [];

  return {
    joinedProjectIds,
    membersById,
    assigneeIds: isWorkspaceLevel ? workspaceIds : projectAssigneeIds,
    creatorIds: isWorkspaceLevel ? workspaceIds : projectCreatorIds,
  };
}
