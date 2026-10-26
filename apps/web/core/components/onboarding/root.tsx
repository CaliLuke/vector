/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo, useState } from "react";
// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspaceMemberInvitation, TOnboardingStep, TOnboardingSteps, TUserProfile } from "@plane/types";
import { EOnboardingSteps } from "@plane/types";
// hooks
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";
import { useWorkspaces } from "@/hooks/use-workspaces";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCurrentUserProfile, useFinishUserOnboarding, useUpdateUserProfile } from "@/hooks/use-current-user-profile";
// local components
import { OnboardingHeader } from "./header";
import { OnboardingStepRoot } from "./steps";

type Props = {
  invitations?: IWorkspaceMemberInvitation[];
};
const EMPTY_INVITATIONS: IWorkspaceMemberInvitation[] = [];

export function OnboardingRoot({ invitations = EMPTY_INVITATIONS }: Props) {
  // store hooks
  const { data: user } = useCurrentUser();
  const { data: userProfile } = useCurrentUserProfile();
  const { mutateAsync: updateUserProfile } = useUpdateUserProfile();
  const { mutateAsync: finishUserOnboarding } = useFinishUserOnboarding();
  const workspacesQuery = useWorkspaces();
  const { data: instanceConfig } = useQuery(instanceConfigOptions);

  const [currentStep, setCurrentStep] = useState<TOnboardingStep>(() => {
    const step = userProfile?.onboarding_step;
    if (step?.profile_complete && step.workspace_create && !step.workspace_invite)
      return EOnboardingSteps.INVITE_MEMBERS;
    if (step?.profile_complete && !step.workspace_create && !step.workspace_join)
      return EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN;
    return EOnboardingSteps.PROFILE_SETUP;
  });
  const workspacesList = useMemo(() => workspacesQuery.data ?? [], [workspacesQuery.data]);
  const isSelfManaged = instanceConfig?.is_self_managed;

  // Calculate total steps based on whether invitations are available
  const hasInvitations = invitations.length > 0;

  // complete onboarding
  const finishOnboarding = useCallback(async () => {
    if (!user?.id) return;
    try {
      const result = await finishUserOnboarding(workspacesList[0]?.id);
      if (result.refreshError) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Onboarding saved",
          message: "Your onboarding was saved, but account data could not refresh. Reload the page to continue.",
        });
      }
    } catch (error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Failed",
        message: "Failed to finish onboarding, Please try again later.",
      });
      throw error;
    }
  }, [user?.id, finishUserOnboarding, workspacesList]);

  // handle step change
  const stepChange = useCallback(
    async (steps: Partial<TOnboardingSteps>) => {
      if (!user?.id || !userProfile) return;

      const payload: Partial<TUserProfile> = {
        onboarding_step: {
          ...userProfile.onboarding_step,
          ...steps,
        },
      };

      await updateUserProfile(payload);
    },
    [user?.id, userProfile, updateUserProfile]
  );

  const handleStepChange = useCallback(
    async (step: EOnboardingSteps, skipInvites?: boolean) => {
      switch (step) {
        case EOnboardingSteps.PROFILE_SETUP:
          if (isSelfManaged) {
            // Skip role & use case steps for self-hosted
            await stepChange({ profile_complete: true });
            if (workspacesList.length > 0) await finishOnboarding();
            else setCurrentStep(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN);
          } else {
            setCurrentStep(EOnboardingSteps.ROLE_SETUP);
          }
          break;
        case EOnboardingSteps.ROLE_SETUP:
          setCurrentStep(EOnboardingSteps.USE_CASE_SETUP);
          break;
        case EOnboardingSteps.USE_CASE_SETUP:
          await stepChange({ profile_complete: true });
          if (workspacesList.length > 0) await finishOnboarding();
          else setCurrentStep(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN);
          break;
        case EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN:
          if (skipInvites) await finishOnboarding();
          else {
            await stepChange({ workspace_create: true });
            setCurrentStep(EOnboardingSteps.INVITE_MEMBERS);
          }
          break;
        case EOnboardingSteps.INVITE_MEMBERS:
          await stepChange({ workspace_invite: true });
          await finishOnboarding();
          break;
      }
    },
    [stepChange, finishOnboarding, workspacesList, isSelfManaged]
  );

  const updateCurrentStep = (step: EOnboardingSteps) => setCurrentStep(step);

  return (
    <div className="flex h-full flex-col">
      {/* Header with progress */}
      <OnboardingHeader
        currentStep={currentStep}
        updateCurrentStep={updateCurrentStep}
        hasInvitations={hasInvitations}
      />

      {/* Main content area */}
      <OnboardingStepRoot currentStep={currentStep} invitations={invitations} handleStepChange={handleStepChange} />
    </div>
  );
}
