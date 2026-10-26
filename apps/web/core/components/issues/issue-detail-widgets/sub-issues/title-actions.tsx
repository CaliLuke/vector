/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { useParams } from "@tanstack/react-router";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, SUB_WORK_ITEM_AVAILABLE_FILTERS_FOR_WORK_ITEM_PAGE } from "@plane/constants";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IIssueFilterOptions,
  TIssueServiceType,
} from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { useSubWorkItemPreferences } from "@/providers/sub-work-item-preferences";
import { useRouteProjectStates } from "@/hooks/use-project-states";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";
import { SubIssueDisplayFilters } from "./display-filters";
import { SubIssueFilters } from "./filters";
import { SubIssuesActionButton } from "./quick-action-button";

type TSubWorkItemTitleActionsProps = {
  disabled: boolean;
  issueServiceType?: TIssueServiceType;
  parentId: string;
  projectId: string;
};

export function SubWorkItemTitleActions(props: TSubWorkItemTitleActionsProps) {
  const { disabled, issueServiceType = EIssueServiceType.ISSUES, parentId, projectId } = props;

  // store hooks
  const { projectStates } = useRouteProjectStates(projectId);
  const { data: currentUser } = useCurrentUser();
  const { workspaceSlug } = useParams({ strict: false });
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId);

  // derived values
  const projectMemberIds = projectRosterMemberIds(
    projectRoster.memberships,
    workspaceRoster.membersById,
    currentUser?.id,
    false
  );
  const { preferences: subIssueFilters, updatePreferences } = useSubWorkItemPreferences({
    workspaceSlug,
    projectId,
    parentId,
    serviceType: issueServiceType,
  });
  const layoutDisplayFiltersOptions = ISSUE_DISPLAY_FILTERS_BY_PAGE["sub_work_items"].layoutOptions.list;

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      updatePreferences({ displayFilters: updatedDisplayFilter });
    },
    [updatePreferences]
  );

  const handleDisplayPropertiesUpdate = useCallback(
    (updatedDisplayProperties: Partial<IIssueDisplayProperties>) => {
      updatePreferences({ displayProperties: updatedDisplayProperties });
    },
    [updatePreferences]
  );

  const handleFiltersUpdate = useCallback(
    (key: keyof IIssueFilterOptions, value: string | string[]) => {
      const newValues = [...(subIssueFilters.filters[key] ?? [])];

      if (Array.isArray(value)) {
        // this validation is majorly for the filter start_date, target_date custom
        value.forEach((val) => {
          if (!newValues.includes(val)) newValues.push(val);
          else newValues.splice(newValues.indexOf(val), 1);
        });
      } else {
        if (subIssueFilters?.filters?.[key]?.includes(value)) newValues.splice(newValues.indexOf(value), 1);
        else newValues.push(value);
      }
      updatePreferences({ filters: { [key]: newValues } });
    },
    [subIssueFilters.filters, updatePreferences]
  );

  return (
    // prevent click everywhere
    <div
      className="flex items-center gap-2"
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
      }}
    >
      <SubIssueDisplayFilters
        isEpic={issueServiceType === EIssueServiceType.EPICS}
        layoutDisplayFiltersOptions={layoutDisplayFiltersOptions}
        displayProperties={subIssueFilters?.displayProperties ?? {}}
        displayFilters={subIssueFilters?.displayFilters ?? {}}
        handleDisplayPropertiesUpdate={handleDisplayPropertiesUpdate}
        handleDisplayFiltersUpdate={handleDisplayFilters}
      />
      <SubIssueFilters
        handleFiltersUpdate={handleFiltersUpdate}
        filters={subIssueFilters?.filters ?? {}}
        memberIds={projectMemberIds ?? undefined}
        workspaceSlug={workspaceSlug}
        states={projectStates}
        availableFilters={SUB_WORK_ITEM_AVAILABLE_FILTERS_FOR_WORK_ITEM_PAGE}
      />
      {!disabled && (
        <SubIssuesActionButton issueId={parentId} disabled={disabled} issueServiceType={issueServiceType} />
      )}
    </div>
  );
}
