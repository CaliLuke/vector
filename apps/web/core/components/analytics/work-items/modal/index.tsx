/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
// plane package imports
import { ModalPortal, EPortalWidth, EPortalPosition } from "@plane/propel/portal";
import type { ICycle, IModule, IProject } from "@plane/types";
import { AnalyticsStateProvider } from "@/providers/analytics-state-provider";
// plane web components
import { WorkItemsModalMainContent } from "./content";
import { WorkItemsModalHeader } from "./header";

type Props = {
  workspaceSlug: string | undefined;
  isOpen: boolean;
  onClose: () => void;
  projectDetails?: IProject | undefined;
  cycleDetails?: ICycle | undefined;
  moduleDetails?: IModule | undefined;
  isEpic?: boolean;
};

export const WorkItemsModal = function WorkItemsModal(props: Props) {
  const { workspaceSlug, isOpen, onClose, projectDetails, moduleDetails, cycleDetails, isEpic } = props;
  const [fullScreen, setFullScreen] = useState(false);

  const handleClose = () => {
    setFullScreen(false);
    onClose();
  };

  return (
    <ModalPortal
      isOpen={isOpen}
      onClose={handleClose}
      width={fullScreen ? EPortalWidth.FULL : EPortalWidth.THREE_QUARTER}
      position={EPortalPosition.RIGHT}
      fullScreen={fullScreen}
    >
      <div
        className={`flex h-full flex-col overflow-hidden border-subtle bg-surface-1 text-left ${
          fullScreen ? "rounded-lg border" : "border-l"
        }`}
      >
        <WorkItemsModalHeader
          fullScreen={fullScreen}
          handleClose={handleClose}
          setFullScreen={setFullScreen}
          title={projectDetails?.name ?? ""}
          cycle={cycleDetails}
          module={moduleDetails}
        />
        {isOpen && (
          <WorkItemsModalContentScope
            workspaceSlug={workspaceSlug}
            projectId={projectDetails?.id}
            cycleId={cycleDetails?.id}
            moduleId={moduleDetails?.id}
            isEpic={isEpic}
            fullScreen={fullScreen}
          />
        )}
      </div>
    </ModalPortal>
  );
};

function WorkItemsModalContentScope({
  workspaceSlug,
  projectId,
  cycleId,
  moduleId,
  isEpic,
  fullScreen,
}: {
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  cycleId: string | undefined;
  moduleId: string | undefined;
  isEpic: boolean | undefined;
  fullScreen: boolean;
}) {
  return (
    <AnalyticsStateProvider
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      cycleId={cycleId}
      moduleId={moduleId}
      isPeekView
      isEpic={isEpic}
      isActive
      instanceKey={`modal:${projectId ?? ""}:${cycleId ?? ""}:${moduleId ?? ""}:${isEpic ?? false}`}
    >
      <WorkItemsModalMainContent fullScreen={fullScreen} isEpic={isEpic} />
    </AnalyticsStateProvider>
  );
}
