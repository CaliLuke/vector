/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
// components
import type { IState, TStateOperationsCallbacks } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
import { ProjectStateLoader, GroupList } from "@/components/project-states";
// hooks
import { useProjectStates } from "@/hooks/use-project-states";
import { useProjectPermissions } from "@/hooks/use-permissions";

type TProjectState = {
  workspaceSlug: string;
  projectId: string;
};

export function ProjectStateRoot(props: TProjectState) {
  const { workspaceSlug, projectId } = props;
  // hooks
  const stateQuery = useProjectStates(workspaceSlug, projectId);
  const { groupedProjectStates, createState, moveStatePosition, updateState, deleteState, markStateAsDefault } =
    stateQuery;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const isEditable = canProject([EUserProjectRoles.ADMIN]);

  // State operations callbacks
  const stateOperationsCallbacks: TStateOperationsCallbacks = useMemo(
    () => ({
      createState: (data: Partial<IState>) => createState(data),
      updateState: (stateId: string, data: Partial<IState>) => updateState(stateId, data),
      deleteState: (stateId: string) => deleteState(stateId),
      moveStatePosition: (stateId: string, data: Partial<IState>) => moveStatePosition(stateId, data),
      markStateAsDefault: (stateId: string) => markStateAsDefault(stateId),
    }),
    [createState, moveStatePosition, updateState, deleteState, markStateAsDefault]
  );

  // Loader
  if (stateQuery.isError) {
    return (
      <div role="alert" className="flex items-center gap-3">
        <p>Unable to load project states.</p>
        <button disabled={stateQuery.isFetching} onClick={() => void stateQuery.refetch()}>
          Retry
        </button>
      </div>
    );
  }
  if (!groupedProjectStates) return <ProjectStateLoader />;

  return (
    <GroupList
      groupedStates={groupedProjectStates}
      stateOperationsCallbacks={stateOperationsCallbacks}
      isEditable={isEditable}
      shouldTrackEvents
    />
  );
}
