/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FormErrors } from "@plane/ui";

import { useForm, useStore } from "@tanstack/react-form";
import { Box, PenTool, Rocket, Monitor, RefreshCw } from "lucide-react";
// plane imports
import { CheckIcon, ViewsIcon } from "@plane/propel/icons";
import { EOnboardingSteps } from "@plane/types";
// local components
import { CommonOnboardingHeader, OnboardingActions } from "../common";
import { handleProfileSetupSubmit, saveProfileSetupStep, useProfileStepAccountData } from "../profile-setup-submit";

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => Promise<void>;
};

const ROLES = [
  { id: "product-manager", label: "Product Manager", icon: Box },
  { id: "engineering-manager", label: "Engineering Manager", icon: ViewsIcon },
  { id: "designer", label: "Designer", icon: PenTool },
  { id: "developer", label: "Developer", icon: Monitor },
  { id: "founder-executive", label: "Founder/Executive", icon: Rocket },
  { id: "operations-manager", label: "Operations Manager", icon: RefreshCw },
  { id: "others", label: "Others", icon: Box },
];

export const RoleSetupStep = function RoleSetupStep({ handleStepChange }: Props) {
  // store hooks
  const { profile, updateUserProfile } = useProfileStepAccountData();
  const form = useForm({
    defaultValues: { role: profile?.role ?? "" },
    validators: { onSubmit: () => (profile ? undefined : "Profile information is still loading.") },
    onSubmit: async ({ value }) => {
      if (!profile) return;
      await saveProfileSetupStep(
        () => updateUserProfile({ role: value.role }),
        handleStepChange,
        EOnboardingSteps.ROLE_SETUP
      );
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const canSubmit = useStore(
    form.store,
    (state) => state.canSubmit && state.values.role.length > 0 && Boolean(profile)
  );
  const handleSkip = () => handleStepChange(EOnboardingSteps.ROLE_SETUP);

  return (
    <form
      onSubmit={(event) => handleProfileSetupSubmit(event, () => form.handleSubmit())}
      className="flex flex-col gap-10"
    >
      {/* Header */}
      <CommonOnboardingHeader title="What's your role?" description="Let's set up Vector for how you work." />
      {/* Role Selection */}
      <div className="flex flex-col gap-3">
        <p className="text-body-sm-semibold text-placeholder">Select one</p>
        <form.Field
          name="role"
          validators={{ onChange: ({ value }) => (value.length > 0 ? undefined : "Please select an option.") }}
        >
          {(field) => (
            <>
              <div className="flex flex-col gap-3">
                {ROLES.map((role) => {
                  const Icon = role.icon;
                  const isSelected = field.state.value === role.id;

                  return (
                    <button
                      key={role.id}
                      type="button"
                      disabled={isSubmitting}
                      aria-pressed={isSelected}
                      onClick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                        field.handleChange(role.id);
                      }}
                      className={`flex w-full items-center justify-between rounded-lg border px-3 py-2 transition-all duration-200 ${
                        isSelected
                          ? "border-accent-strong bg-accent-subtle text-accent-primary"
                          : "border-subtle text-tertiary hover:border-strong"
                      }`}
                    >
                      <div className="flex items-center space-x-3">
                        <Icon className="size-3.5" />
                        <span className="text-body-sm-semibold">{role.label}</span>
                      </div>
                      {isSelected && (
                        <>
                          <span
                            className={`border-blue-500 flex size-4 items-center justify-center rounded-sm border-2 bg-accent-primary`}
                          >
                            <CheckIcon className="h-3 w-3 text-on-color" />
                          </span>
                        </>
                      )}
                    </button>
                  );
                })}
              </div>
              <FormErrors errors={field.state.meta.errors} />
            </>
          )}
        </form.Field>
      </div>
      <OnboardingActions isSubmitting={isSubmitting} canSubmit={canSubmit} onSkip={handleSkip} />
    </form>
  );
};
