/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { showAuthenticationError } from "@/helpers/authentication.helper";

import { useLocation } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { useStore } from "@tanstack/react-form";

// icons
// ui
import { API_BASE_URL } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { NewPasswordFields, useNewPasswordForm } from "./common/new-password-fields";
// components
import { isPasswordConfirmationValid } from "@plane/utils/auth";
// helpers
import type { EAuthenticationErrorCodes } from "@/helpers/authentication.helper";
import { EErrorAlertType, authErrorHandler } from "@/helpers/authentication.helper";
// services
import { authService } from "@/services/auth.service";
// local imports
import { AuthBanner } from "./auth-banner";
import { FormContainer } from "./common/container";
import { AuthFormHeader } from "./common/header";

export const ResetPasswordForm = function ResetPasswordForm() {
  // search params
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const uidb64 = searchParams.get("uidb64");
  const token = searchParams.get("token");
  const email = searchParams.get("email");
  const error_code = searchParams.get("error_code");
  // states
  const [dismissedErrorCode, setDismissedErrorCode] = useState<string | null>();
  const errorInfo =
    error_code && error_code !== dismissedErrorCode
      ? authErrorHandler(error_code as EAuthenticationErrorCodes)
      : undefined;
  // plane hooks
  const { t } = useTranslation();

  const formRef = useRef<HTMLFormElement>(null);
  const form = useNewPasswordForm(async () => {
    try {
      await authService.submitForm(formRef.current);
    } catch (error) {
      showAuthenticationError(t);
      throw error;
    }
  });
  const resetFormData = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const isButtonDisabled =
    isSubmitting || !isPasswordConfirmationValid(resetFormData.password, resetFormData.confirm_password);

  return (
    <FormContainer>
      <AuthFormHeader title="Reset password" description="Create a new password." />

      {errorInfo && errorInfo?.type === EErrorAlertType.BANNER_ALERT && (
        <AuthBanner message={errorInfo.message} handleBannerData={() => setDismissedErrorCode(error_code)} />
      )}
      <form
        ref={formRef}
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit().catch(() => {
            // The failed password reset is already reported with an authentication error.
          });
        }}
        className="space-y-4"
        method="POST"
        action={`${API_BASE_URL}/auth/reset-password/${uidb64?.toString()}/${token?.toString()}/`}
      >
        <input type="hidden" name="csrfmiddlewaretoken" />
        <NewPasswordFields
          email={email ?? ""}
          password={resetFormData.password}
          confirmation={resetFormData.confirm_password}
          readOnly={isSubmitting}
          onPasswordChange={(password) => form.setFieldValue("password", password)}
          onConfirmationChange={(confirmation) => form.setFieldValue("confirm_password", confirmation)}
        />
        <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
          {t("auth.common.password.submit")}
        </Button>
      </form>
    </FormContainer>
  );
};
