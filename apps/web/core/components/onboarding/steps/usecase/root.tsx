/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FormErrors } from "@plane/ui";

import { useForm, useStore } from "@tanstack/react-form";
// plane imports
import { USE_CASES } from "@plane/constants";
import { CheckIcon } from "@plane/propel/icons";
import { EOnboardingSteps } from "@plane/types";
import { cn } from "@plane/utils/common";
// local imports
import { CommonOnboardingHeader, OnboardingActions } from "../common";
import { handleProfileSetupSubmit, saveProfileSetupStep, useProfileStepAccountData } from "../profile-setup-submit";

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => Promise<void>;
};

export const UseCaseSetupStep = function UseCaseSetupStep({ handleStepChange }: Props) {
  // store hooks
  const { profile, updateUserProfile } = useProfileStepAccountData();
  const form = useForm({
    defaultValues: { use_case: profile?.use_case?.split(". ") ?? [] },
    validators: { onSubmit: () => (profile ? undefined : "Profile information is still loading.") },
    onSubmit: async ({ value }) => {
      if (!profile) return;
      await saveProfileSetupStep(
        () => updateUserProfile({ use_case: value.use_case.join(". ") }),
        handleStepChange,
        EOnboardingSteps.USE_CASE_SETUP
      );
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const canSubmit = useStore(
    form.store,
    (state) => state.canSubmit && state.values.use_case.length > 0 && Boolean(profile)
  );
  const handleSkip = () => handleStepChange(EOnboardingSteps.USE_CASE_SETUP);

  return (
    <form
      onSubmit={(event) => handleProfileSetupSubmit(event, () => form.handleSubmit())}
      className="flex flex-col gap-10"
    >
      {/* Header */}
      <CommonOnboardingHeader title="What brings you to Vector?" description="Tell us your goals and team size." />

      {/* Use Case Selection */}
      <div className="flex flex-col gap-3">
        <p className="text-body-sm-semibold text-placeholder">Select one or more</p>

        <form.Field
          name="use_case"
          validators={{ onChange: ({ value }) => (value.length > 0 ? undefined : "Please select an option.") }}
        >
          {(field) => (
            <>
              <div className="flex flex-col gap-3">
                {USE_CASES.map((useCase) => {
                  const isSelected = field.state.value.includes(useCase);
                  return (
                    <button
                      key={useCase}
                      type="button"
                      disabled={isSubmitting}
                      aria-pressed={isSelected}
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        const currentValue = field.state.value;
                        if (isSelected) {
                          // Remove from array
                          field.handleChange(currentValue.filter((item) => item !== useCase));
                        } else {
                          // Add to array
                          field.handleChange([...currentValue, useCase]);
                        }
                      }}
                      className={`flex w-full items-center gap-2 rounded-lg border px-3 py-2 transition-all duration-200 ${
                        isSelected
                          ? "border-accent-strong bg-accent-subtle text-accent-primary"
                          : "border-subtle text-tertiary hover:border-strong"
                      }`}
                    >
                      <span
                        className={cn(`flex size-4 items-center justify-center rounded-sm border-2`, {
                          "border-accent-strong bg-accent-primary": isSelected,
                          "border-strong": !isSelected,
                        })}
                      >
                        <CheckIcon
                          className={cn("h-3 w-3 text-on-color", {
                            "opacity-100": isSelected,
                            "opacity-0": !isSelected,
                          })}
                        />
                      </span>

                      <span className="text-body-sm-regular">{useCase}</span>
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
