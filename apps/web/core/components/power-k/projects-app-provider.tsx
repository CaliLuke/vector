/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useMemo, useState } from "react";

// hooks
import { usePowerK } from "@/hooks/use-power-k";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";

// plane web imports
import { ProjectLevelModals } from "@/components/modals/project-level";
import { WorkItemLevelModals } from "@/components/modals/work-item-level";
import { WorkspaceLevelModals } from "@/components/modals/workspace-level";
// local imports
import { useProjectsAppPowerKCommands } from "./config/commands";
import type { TPowerKCommandConfig, TPowerKContext } from "./core/types";
import { GlobalShortcutsProvider } from "./global-shortcuts";
import { ProjectsAppPowerKCommandsList } from "./ui/modal/commands-list";
import { ProjectsAppPowerKModalWrapper } from "./ui/modal/wrapper";

/**
 * Projects App PowerK provider
 */
export const ProjectsAppPowerKProvider = function ProjectsAppPowerKProvider() {
  // router
  const router = useRouter();
  const params = useParams({ strict: false });
  const { workspaceSlug, projectId: routerProjectId, workItem: workItemIdentifier } = params;
  // states
  const [activeCommand, setActiveCommand] = useState<TPowerKCommandConfig | null>(null);
  const [shouldShowContextBasedActions, setShouldShowContextBasedActions] = useState(true);
  // store hooks
  const { activeContext, isPowerKModalOpen, togglePowerKModal, setActivePage } = usePowerK();
  const { data: currentUser } = useCurrentUser();
  // derived values
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug?.toString());
  const [projectIdentifier, sequenceId] = (workItemIdentifier?.toString() ?? "").split("-");
  const identifierProjectId = projectCatalogue.projects?.find(
    (project) => project.identifier === projectIdentifier
  )?.id;
  const workItemDetails = useWorkItemByIdentifier(
    workspaceSlug?.toString(),
    projectIdentifier,
    sequenceId,
    identifierProjectId
  ).issue;
  const projectId = routerProjectId ?? workItemDetails?.project_id ?? undefined;
  const commands = useProjectsAppPowerKCommands();
  // Build command context from props and store
  const context: TPowerKContext = useMemo(
    () => ({
      currentUserId: currentUser?.id,
      activeCommand,
      activeContext,
      shouldShowContextBasedActions,
      setShouldShowContextBasedActions,
      params: {
        ...params,
        projectId,
      },
      router,
      closePalette: () => togglePowerKModal(false),
      setActiveCommand,
      setActivePage,
    }),
    [
      currentUser?.id,
      activeCommand,
      activeContext,
      shouldShowContextBasedActions,
      params,
      projectId,
      router,
      togglePowerKModal,
      setActivePage,
    ]
  );

  return (
    <>
      <GlobalShortcutsProvider context={context} commands={commands} />
      {workspaceSlug && <WorkspaceLevelModals workspaceSlug={workspaceSlug} />}
      {workspaceSlug && projectId && <ProjectLevelModals workspaceSlug={workspaceSlug} projectId={projectId} />}
      <WorkItemLevelModals workItemIdentifier={workItemIdentifier?.toString()} />
      <ProjectsAppPowerKModalWrapper
        commandsListComponent={ProjectsAppPowerKCommandsList}
        context={context}
        isOpen={isPowerKModalOpen}
        onClose={() => togglePowerKModal(false)}
      />
    </>
  );
};
