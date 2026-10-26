/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { useForm, useStore } from "@tanstack/react-form";
import { ImageIcon } from "lucide-react";
// plane imports
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EOnboardingSteps } from "@plane/types";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
import { isPasswordConfirmationValid } from "@plane/utils/auth";
import { validatePersonName } from "@plane/utils/validation";
// components
import { UserImageUploadModal } from "@/components/core/modals/user-image-upload-modal";
// hooks
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";
import { useCurrentUser, useUpdateCurrentUser } from "@/hooks/use-current-user";
import { useUpdateUserProfile } from "@/hooks/use-current-user-profile";
// services
import { authService } from "@/services/auth.service";
// local components
import { CommonOnboardingHeader } from "../common";
import { MarketingConsent } from "./consent";
import { SetPasswordRoot } from "./set-password";

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => Promise<void>;
};

export const ProfileSetupStep = function ProfileSetupStep({ handleStepChange }: Props) {
  // states
  const [isImageUploadModalOpen, setIsImageUploadModalOpen] = useState(false);
  // store hooks
  const { data: user } = useCurrentUser();
  const { mutateAsync: updateCurrentUser } = useUpdateCurrentUser();
  const { mutateAsync: updateUserProfile } = useUpdateUserProfile();
  const { data: instanceConfig } = useQuery(instanceConfigOptions);
  const form = useForm({
    defaultValues: {
      first_name: user?.first_name ?? "",
      last_name: user?.last_name ?? "",
      avatar_url: user?.avatar_url ?? "",
      password: "",
      confirm_password: "",
      has_marketing_email_consent: true,
    },
    validators: {
      onChange: ({ value }) =>
        user?.is_password_autoset &&
        (value.password || value.confirm_password) &&
        !isPasswordConfirmationValid(value.password, value.confirm_password)
          ? "Use a strong password and matching confirmation."
          : undefined,
      onSubmit: () => (user ? undefined : "Profile information is still loading."),
    },
    onSubmit: async ({ value, formApi }) => {
      if (!user) return;
      try {
        await updateCurrentUser({
          first_name: value.first_name,
          last_name: value.last_name,
          avatar_url: value.avatar_url,
        });
        await updateUserProfile({ has_marketing_email_consent: value.has_marketing_email_consent });
        if (user.is_password_autoset && value.password) {
          const { csrf_token } = await authService.requestCSRFToken();
          if (!csrf_token) throw new Error("CSRF token not found");
          await authService.setPassword(csrf_token, { password: value.password });
          formApi.setFieldValue("password", "");
          formApi.setFieldValue("confirm_password", "");
        }
        await handleStepChange(EOnboardingSteps.PROFILE_SETUP);
      } catch (error) {
        setToast({ type: TOAST_TYPE.ERROR, title: "Error", message: "Profile setup failed. Please try again." });
        throw error;
      }
    },
  });
  const { isSubmitting, canSubmit, values, errors } = useStore(form.store, (state) => ({
    isSubmitting: state.isSubmitting,
    canSubmit: state.canSubmit,
    values: state.values,
    errors: state.errors,
  }));
  const userAvatar = values.avatar_url;
  const isPasswordAlreadySetup = !user?.is_password_autoset;
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit().catch(() => {
          // The failed profile step is already reported with a toast.
        });
      }}
      className="flex flex-col gap-10"
    >
      {/* Header */}
      <CommonOnboardingHeader title="Create your profile." description="This is how you will appear in Vector." />

      {/* Profile Picture Section */}
      <form.Field name="avatar_url">
        {(field) => (
          <UserImageUploadModal
            isOpen={isImageUploadModalOpen}
            onClose={() => setIsImageUploadModalOpen(false)}
            handleRemove={async () => field.handleChange("")}
            onSuccess={(url) => {
              field.handleChange(url);
            }}
            value={field.state.value || null}
          />
        )}
      </form.Field>
      <div className="flex items-center gap-4">
        <button
          className="flex size-12 items-center justify-center rounded-full bg-accent-primary text-18 font-semibold text-on-color"
          type="button"
          disabled={isSubmitting}
          onClick={() => setIsImageUploadModalOpen(true)}
        >
          {userAvatar ? (
            <img
              src={getFileURL(userAvatar ?? "")}
              alt={user?.display_name}
              className="h-full w-full rounded-full object-cover"
            />
          ) : (
            <>{values.first_name[0] ?? "R"}</>
          )}
        </button>
        <button
          className="flex items-center gap-1.5 px-2 py-1 text-13 text-tertiary hover:text-secondary"
          type="button"
          disabled={isSubmitting}
          onClick={() => setIsImageUploadModalOpen(true)}
        >
          <ImageIcon className="size-4" />
          <span className="text-13">{userAvatar ? "Change image" : "Upload image"}</span>
        </button>
      </div>

      <div className="flex w-full flex-col gap-6">
        {/* Name Input */}
        <div className="flex flex-col gap-2">
          <label
            className="block text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
            htmlFor="first_name"
          >
            Name
          </label>
          <form.Field
            name="first_name"
            validators={{
              onChange: ({ value }) => {
                if (!value.trim()) return "Name is required";
                if (value.length > 50) return "Name must be within 50 characters.";
                const valid = validatePersonName(value);
                return valid === true ? undefined : valid;
              },
            }}
          >
            {(field) => (
              <>
                <input
                  id={field.name}
                  name={field.name}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  disabled={isSubmitting}
                  autoFocus
                  className={cn(
                    "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary placeholder:text-placeholder focus:ring-2 focus:ring-accent-strong focus:outline-none",
                    { "border-danger-strong": field.state.meta.errors.length > 0 }
                  )}
                  placeholder="Enter your full name"
                  autoComplete="name"
                />
                {field.state.meta.errors.map((error) => (
                  <span key={String(error)} role="alert" className="text-13 text-danger-primary">
                    {error}
                  </span>
                ))}
              </>
            )}
          </form.Field>
        </div>

        {/* setting up password for the first time */}
        {!isPasswordAlreadySetup && (
          <SetPasswordRoot
            password={values.password}
            confirmPassword={values.confirm_password}
            disabled={isSubmitting}
            onPasswordChange={(password) => form.setFieldValue("password", password)}
            onConfirmPasswordChange={(confirm_password) => form.setFieldValue("confirm_password", confirm_password)}
          />
        )}
      </div>
      {errors.map((error) => (
        <p key={error} role="alert" className="text-13 text-danger-primary">
          {error}
        </p>
      ))}
      {/* Continue Button */}
      <Button
        variant="primary"
        type="submit"
        className="w-full"
        size="xl"
        loading={isSubmitting}
        disabled={!canSubmit || isSubmitting || !values.first_name.trim()}
      >
        Continue
      </Button>

      {/* Marketing Consent */}
      {!instanceConfig?.is_self_managed && (
        <MarketingConsent
          disabled={isSubmitting}
          isChecked={values.has_marketing_email_consent}
          handleChange={(has_marketing_email_consent) =>
            form.setFieldValue("has_marketing_email_consent", has_marketing_email_consent)
          }
        />
      )}
    </form>
  );
};
