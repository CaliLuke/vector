/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";

// plane imports
import type { IWorkspaceMemberInvitation } from "@plane/types";
import { ECreateOrJoinWorkspaceViews, EOnboardingSteps } from "@plane/types";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
// local components
import { WorkspaceCreateStep, WorkspaceJoinInvitesStep } from "./";

type Props = {
  invitations: IWorkspaceMemberInvitation[];
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => Promise<void>;
};

export const WorkspaceSetupStep = function WorkspaceSetupStep({ invitations, handleStepChange }: Props) {
  // states
  const [viewState, setViewState] = useState(() => ({
    invitations,
    view:
      invitations.length > 0
        ? ECreateOrJoinWorkspaceViews.WORKSPACE_JOIN
        : ECreateOrJoinWorkspaceViews.WORKSPACE_CREATE,
  }));
  const currentView =
    viewState.invitations === invitations
      ? viewState.view
      : invitations.length > 0
        ? ECreateOrJoinWorkspaceViews.WORKSPACE_JOIN
        : ECreateOrJoinWorkspaceViews.WORKSPACE_CREATE;
  const setCurrentView = (view: ECreateOrJoinWorkspaceViews) => setViewState({ invitations, view });
  // store hooks
  const { data: user } = useCurrentUser();

  return (
    <>
      {currentView === ECreateOrJoinWorkspaceViews.WORKSPACE_JOIN ? (
        <WorkspaceJoinInvitesStep
          invitations={invitations}
          handleNextStep={async () => {
            await handleStepChange(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN, true);
          }}
          handleCurrentViewChange={() => setCurrentView(ECreateOrJoinWorkspaceViews.WORKSPACE_CREATE)}
        />
      ) : (
        <WorkspaceCreateStep
          user={user ?? undefined}
          onComplete={(skipInvites) => handleStepChange(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN, skipInvites)}
          handleCurrentViewChange={() => setCurrentView(ECreateOrJoinWorkspaceViews.WORKSPACE_JOIN)}
          hasInvitations={invitations.length > 0}
        />
      )}
    </>
  );
};
