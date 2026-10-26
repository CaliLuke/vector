/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { showAuthenticationError } from "@/helpers/authentication.helper";

import { Link } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";

// icons
import { Info, XCircle } from "lucide-react";
// plane imports
import { API_BASE_URL, E_PASSWORD_STRENGTH, AUTH_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { CloseIcon } from "@plane/propel/icons";
import { Input, AuthPasswordInput, AuthConfirmPasswordInput, Spinner } from "@plane/ui";
import { getPasswordStrength } from "@plane/utils/auth";
// components
import { ForgotPasswordPopover } from "@/components/account/auth-forms/forgot-password-popover";
// constants
// helpers
import { EAuthModes, EAuthSteps } from "@/helpers/authentication.helper";
// services
import { authService } from "@/services/auth.service";

type Props = {
  email: string;
  isSMTPConfigured: boolean;
  mode: EAuthModes;
  handleEmailClear: () => void;
  handleAuthStep: (step: EAuthSteps) => void;
  nextPath: string | undefined;
};

export const AuthPasswordForm = function AuthPasswordForm(props: Props) {
  const { email, isSMTPConfigured, handleAuthStep, handleEmailClear, mode, nextPath } = props;
  // plane imports
  const { t } = useTranslation();
  // ref
  const formRef = useRef<HTMLFormElement>(null);
  // states
  const form = useForm({
    defaultValues: { password: "", confirm_password: "" },
    validators: {
      onSubmit: ({ value }) =>
        !value.password || (mode === EAuthModes.SIGN_UP && value.password !== value.confirm_password)
          ? t("auth.common.password.errors.match")
          : mode === EAuthModes.SIGN_UP && getPasswordStrength(value.password) !== E_PASSWORD_STRENGTH.STRENGTH_VALID
            ? t("auth.sign_up.errors.password.strength")
            : undefined,
    },
    onSubmitInvalid: ({ value }) => {
      if (mode === EAuthModes.SIGN_UP && getPasswordStrength(value.password) !== E_PASSWORD_STRENGTH.STRENGTH_VALID) {
        setBannerMessage(true);
      }
    },
    onSubmit: async () => {
      try {
        await authService.submitForm(formRef.current);
      } catch (error) {
        showAuthenticationError(t);
        throw error;
      }
    },
  });
  const passwordFormData = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const [isBannerMessage, setBannerMessage] = useState(false);

  const redirectToUniqueCodeSignIn = () => {
    handleAuthStep(EAuthSteps.UNIQUE_CODE);
  };

  const passwordSupport = mode === EAuthModes.SIGN_IN && (
    <div className="w-full">
      {isSMTPConfigured ? (
        <Link
          data-ph-element={AUTH_TRACKER_ELEMENTS.FORGOT_PASSWORD_FROM_SIGNIN}
          to="/accounts/forgot-password"
          search={{ email }}
          className="text-11 font-medium text-accent-primary"
        >
          {t("auth.common.forgot_password")}
        </Link>
      ) : (
        <ForgotPasswordPopover />
      )}
    </div>
  );

  const isButtonDisabled =
    isSubmitting ||
    !passwordFormData.password ||
    (mode === EAuthModes.SIGN_UP && passwordFormData.password !== passwordFormData.confirm_password);

  return (
    <>
      {isBannerMessage && mode === EAuthModes.SIGN_UP && (
        <div className="relative flex items-center gap-2 rounded-md border border-danger-strong/50 bg-danger-subtle p-2">
          <div className="relative flex h-4 w-4 shrink-0 items-center justify-center">
            <Info size={16} className="text-danger-primary" />
          </div>
          <div className="w-full text-13 font-medium text-danger-primary">
            {t("auth.sign_up.errors.password.strength")}
          </div>
          <button
            type="button"
            className="relative ml-auto flex h-6 w-6 cursor-pointer items-center justify-center rounded-xs text-accent-primary/80 transition-all hover:bg-danger-subtle-hover"
            onClick={() => setBannerMessage(false)}
          >
            <CloseIcon className="h-4 w-4 shrink-0 text-danger-primary" />
          </button>
        </div>
      )}
      <form
        ref={formRef}
        className="space-y-4"
        method="POST"
        action={`${API_BASE_URL}/auth/${mode === EAuthModes.SIGN_IN ? "sign-in" : "sign-up"}/`}
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit().catch(() => {
            // The failed authentication request is already reported.
          });
        }}
      >
        <input type="hidden" name="csrfmiddlewaretoken" />
        <input type="hidden" value={email} name="email" />
        {nextPath && <input type="hidden" value={nextPath} name="next_path" />}
        <div className="space-y-1">
          <label htmlFor="email" className="text-13 font-medium text-tertiary">
            {t("auth.common.email.label")}
          </label>
          <div className={`relative flex items-center rounded-md border border-strong bg-surface-1`}>
            <Input
              id="email"
              name="email"
              type="email"
              value={email}
              placeholder={t("auth.common.email.placeholder")}
              className={`h-10 w-full border-0 disable-autofill-style placeholder:text-placeholder`}
              disabled
            />
            {email.length > 0 && (
              <button
                type="button"
                className="absolute right-3 size-5"
                onClick={handleEmailClear}
                aria-label={t("aria_labels.auth_forms.clear_email")}
              >
                <XCircle className="size-5 stroke-placeholder" />
              </button>
            )}
          </div>
        </div>

        <AuthPasswordInput
          id="password"
          name="password"
          label={mode === EAuthModes.SIGN_IN ? t("auth.common.password.label") : t("auth.common.password.set_password")}
          value={passwordFormData.password}
          onChange={(event) => form.setFieldValue("password", event.target.value)}
          readOnly={isSubmitting}
          showPasswordStrength={mode === EAuthModes.SIGN_UP}
          autoComplete={mode === EAuthModes.SIGN_IN ? "current-password" : "new-password"}
        />
        {passwordSupport}
        {mode === EAuthModes.SIGN_UP && (
          <AuthConfirmPasswordInput
            id="confirm_password"
            name="confirm_password"
            label={t("auth.common.password.confirm_password.label")}
            password={passwordFormData.password}
            value={passwordFormData.confirm_password}
            onChange={(event) => form.setFieldValue("confirm_password", event.target.value)}
            readOnly={isSubmitting}
            autoComplete="new-password"
          />
        )}
        <div className="space-y-2.5">
          {mode === EAuthModes.SIGN_IN ? (
            <>
              <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
                {isSubmitting ? (
                  <Spinner height="20px" width="20px" />
                ) : isSMTPConfigured ? (
                  t("common.continue")
                ) : (
                  t("common.go_to_workspace")
                )}
              </Button>
              {isSMTPConfigured && (
                <Button
                  type="button"
                  data-ph-element={AUTH_TRACKER_ELEMENTS.SIGN_IN_WITH_UNIQUE_CODE}
                  onClick={redirectToUniqueCodeSignIn}
                  variant="secondary"
                  className="w-full"
                  size="xl"
                >
                  {t("auth.common.sign_in_with_unique_code")}
                </Button>
              )}
            </>
          ) : (
            <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
              {isSubmitting ? <Spinner height="20px" width="20px" /> : "Create account"}
            </Button>
          )}
        </div>
      </form>
    </>
  );
};
