/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";

// plane imports
import { EIssuesStoreType, EUserProjectRoles } from "@plane/types";
// components
import { CreateUpdateProjectViewModal } from "@/components/views/modal";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { useProjectLabels } from "@/hooks/use-labels";
import { useProjectModules } from "@/hooks/use-modules";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
import { useProjectView, useProjectViewMutations } from "@/hooks/use-project-views";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";
// local imports
import { WorkItemFiltersHOC } from "./base";
import { persistViewUpdate, requireViewForUpdate } from "./view-update";
import { useWorkspaceViewActionScope } from "@/hooks/use-workspace-views";
import { useViewFilterActions } from "./use-view-filter-actions";
import type { ViewFilterPayload } from "./use-view-filter-actions";
import type { TEnableSaveViewProps, TEnableUpdateViewProps, TSharedWorkItemFiltersHOCProps } from "./shared";

type TProjectLevelWorkItemFiltersHOCProps = TSharedWorkItemFiltersHOCProps & {
  workspaceSlug: string;
  projectId: string;
} & TEnableSaveViewProps &
  TEnableUpdateViewProps;

function canEditViewFilters(query: ReturnType<typeof useProjectView>, accountId: string | undefined) {
  const view = query.view;
  return Boolean(
    view &&
    accountId &&
    view.owned_by === accountId &&
    !view.is_locked &&
    !query.isPending &&
    !query.isError &&
    !query.isDenied &&
    query.isScopeCurrent()
  );
}

export const ProjectLevelWorkItemFiltersHOC = function ProjectLevelWorkItemFiltersHOC(
  props: TProjectLevelWorkItemFiltersHOCProps
) {
  const { children, enableSaveView, enableUpdateView, entityId, entityType, projectId, workspaceSlug } = props;
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const viewId = entityType === EIssuesStoreType.PROJECT_VIEW ? entityId : undefined;
  const viewQuery = useProjectView(workspaceSlug, projectId, viewId);
  const { update } = useProjectViewMutations(workspaceSlug, projectId);
  const updateView = update.mutateAsync;
  const { data: currentUser } = useCurrentUser();
  const captureActionScope = useWorkspaceViewActionScope(workspaceSlug, true, entityId);
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId);
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const cycleCatalogue = useProjectCycles(workspaceSlug, projectId);
  const { labelIds: projectLabelIds } = useProjectLabels(workspaceSlug, projectId);
  const moduleCatalogue = useProjectModules(workspaceSlug, projectId);
  const { getProjectStateIds } = useProjectStates(workspaceSlug, projectId);
  // derived values
  const hasProjectMemberLevelPermissions = canProject([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);
  const projectDetails = projectRows?.find((project) => project.id === projectId);
  const viewDetails = viewQuery.view ?? null;
  const canCreateView = useMemo(
    () =>
      projectDetails?.issue_views_view === true &&
      enableSaveView &&
      !props.saveViewOptions?.isDisabled &&
      hasProjectMemberLevelPermissions,
    [
      projectDetails?.issue_views_view,
      enableSaveView,
      props.saveViewOptions?.isDisabled,
      hasProjectMemberLevelPermissions,
    ]
  );
  const canUpdateView =
    enableUpdateView &&
    !props.updateViewOptions?.isDisabled &&
    hasProjectMemberLevelPermissions &&
    canEditViewFilters(viewQuery, currentUser?.id);
  const handleViewUpdate = useCallback(
    async (payload: ViewFilterPayload) => {
      const view = requireViewForUpdate(viewDetails);
      return persistViewUpdate(() => updateView({ viewId: view.id, data: payload }), captureActionScope());
    },
    [viewDetails, updateView, captureActionScope]
  );
  const { createViewPayload, isCreateViewModalOpen, saveViewOptions, updateViewOptions, close } = useViewFilterActions(
    props,
    viewDetails,
    {
      canCreate: canCreateView,
      canUpdate: canUpdateView,
      update: handleViewUpdate,
    }
  );

  return (
    <>
      <CreateUpdateProjectViewModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        preLoadedData={createViewPayload}
        isOpen={isCreateViewModalOpen}
        onClose={close}
      />
      <WorkItemFiltersHOC
        {...props}
        workspaceSlug={workspaceSlug}
        cycleIds={cycleCatalogue.cycles?.map((row) => row.id)}
        labelIds={projectLabelIds}
        memberIds={
          projectRosterMemberIds(projectRoster.memberships, workspaceRoster.membersById, currentUser?.id, false) ??
          undefined
        }
        moduleIds={moduleCatalogue.modules?.map((row) => row.id)}
        stateIds={getProjectStateIds(projectId)}
        saveViewOptions={saveViewOptions}
        updateViewOptions={updateViewOptions}
      >
        {children}
      </WorkItemFiltersHOC>
    </>
  );
};
