/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useCallback, useMemo } from "react";

// plane imports
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@plane/constants";
import { EIssuesStoreType, EUserProjectRoles } from "@plane/types";
// components
import { CreateUpdateWorkspaceViewModal } from "@/components/workspace/views/modal";
// hooks
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import { useWorkspaceLabels } from "@/hooks/use-labels";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useWorkspaceView, useWorkspaceViewActionScope, useWorkspaceViewMutations } from "@/hooks/use-workspace-views";
import { workspaceRosterMemberIds } from "@/helpers/member-rosters";
// local imports
import { WorkItemFiltersHOC } from "./base";
import { persistViewUpdate, requireViewForUpdate } from "./view-update";
import { useViewFilterActions } from "./use-view-filter-actions";
import type { ViewFilterPayload } from "./use-view-filter-actions";
import type { TEnableSaveViewProps, TEnableUpdateViewProps, TSharedWorkItemFiltersHOCProps } from "./shared";

type TWorkspaceLevelWorkItemFiltersHOCProps = TSharedWorkItemFiltersHOCProps & {
  workspaceSlug: string;
} & TEnableSaveViewProps &
  TEnableUpdateViewProps;

function activeWorkspaceViewPreferences(
  preferences: ReturnType<typeof useWorkspaceIssuePreferences>,
  props: { entityType: EIssuesStoreType; entityId?: string; isTemporary?: boolean }
) {
  return props.entityType === EIssuesStoreType.GLOBAL && !props.isTemporary && preferences?.viewId === props.entityId
    ? preferences
    : undefined;
}

export const WorkspaceLevelWorkItemFiltersHOC = function WorkspaceLevelWorkItemFiltersHOC(
  props: TWorkspaceLevelWorkItemFiltersHOCProps
) {
  const { children, enableSaveView, enableUpdateView, entityId, workspaceSlug } = props;
  // hooks
  const preferences = activeWorkspaceViewPreferences(useWorkspaceIssuePreferences(), props);
  const { view: viewDetails } = useWorkspaceView(workspaceSlug, entityId);
  const { update } = useWorkspaceViewMutations(workspaceSlug);
  const captureActionScope = useWorkspaceViewActionScope(workspaceSlug, true, entityId);
  const { data: currentUser } = useCurrentUser();
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const { data: catalogueWorkspace } = useCurrentWorkspace();
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug);
  const { members } = useWorkspaceMemberRoster(workspaceSlug);
  const workspaceMemberIds = workspaceRosterMemberIds(members, currentUser?.id);
  const { getWorkspaceLabelIds } = useWorkspaceLabels(workspaceSlug);
  // derived values
  const hasWorkspaceMemberLevelPermissions = canWorkspace([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);
  const isDefaultView = typeof entityId === "string" && DEFAULT_GLOBAL_VIEWS_LIST.some((view) => view.key === entityId);
  const isViewLocked = viewDetails ? viewDetails?.is_locked : false;
  const isCurrentUserOwner = viewDetails ? viewDetails.owned_by === currentUser?.id : false;
  const canCreateView = useMemo(
    () => enableSaveView && !props.saveViewOptions?.isDisabled && hasWorkspaceMemberLevelPermissions,
    [enableSaveView, props.saveViewOptions?.isDisabled, hasWorkspaceMemberLevelPermissions]
  );
  const canUpdateView = useMemo(
    () =>
      enableUpdateView &&
      !isDefaultView &&
      !props.updateViewOptions?.isDisabled &&
      !isViewLocked &&
      hasWorkspaceMemberLevelPermissions &&
      isCurrentUserOwner,
    [
      enableUpdateView,
      props.updateViewOptions?.isDisabled,
      isDefaultView,
      isViewLocked,
      hasWorkspaceMemberLevelPermissions,
      isCurrentUserOwner,
    ]
  );
  const handleViewUpdate = useCallback(
    async (payload: ViewFilterPayload) => {
      const view = requireViewForUpdate(viewDetails);
      const submittedRevision = preferences?.revision;
      return persistViewUpdate(
        () => update.mutateAsync({ viewId: view.id, data: payload }),
        captureActionScope(),
        () => {
          if (submittedRevision !== undefined) preferences?.acknowledgeSave(view.id, submittedRevision);
        }
      );
    },
    [viewDetails, update, captureActionScope, preferences]
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
      <CreateUpdateWorkspaceViewModal
        preLoadedData={createViewPayload}
        isOpen={isCreateViewModalOpen}
        onClose={close}
      />
      <WorkItemFiltersHOC
        {...props}
        memberIds={workspaceMemberIds}
        labelIds={getWorkspaceLabelIds(catalogueWorkspace?.id)}
        projectIds={joinedProjectIds}
        saveViewOptions={saveViewOptions}
        updateViewOptions={updateViewOptions}
      >
        {children}
      </WorkItemFiltersHOC>
    </>
  );
};
