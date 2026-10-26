/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FormEvent } from "react";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EOnboardingSteps } from "@plane/types";
import { useCurrentUserProfile, useUpdateUserProfile } from "@/hooks/use-current-user-profile";

type StepChange = (step: EOnboardingSteps, skipInvites?: boolean) => Promise<void>;

export function useProfileStepAccountData() {
  const { data: profile } = useCurrentUserProfile();
  const { mutateAsync: updateUserProfile } = useUpdateUserProfile();
  return { profile, updateUserProfile };
}

export async function saveProfileSetupStep(
  save: () => Promise<unknown>,
  handleStepChange: StepChange,
  step: EOnboardingSteps
) {
  try {
    await save();
    await handleStepChange(step);
    setToast({ type: TOAST_TYPE.SUCCESS, title: "Success", message: "Profile setup completed!" });
  } catch (error) {
    setToast({ type: TOAST_TYPE.ERROR, title: "Error", message: "Profile setup failed. Please try again!" });
    throw error;
  }
}

export function handleProfileSetupSubmit(event: FormEvent<HTMLFormElement>, handleSubmit: () => Promise<void>) {
  event.preventDefault();
  void handleSubmit().catch(() => {
    // The failed profile step is already reported with a toast.
  });
}
