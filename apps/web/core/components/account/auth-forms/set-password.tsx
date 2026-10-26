/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { showAuthenticationError } from "@/helpers/authentication.helper";

import { useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useStore } from "@tanstack/react-form";

// icons
// plane imports
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { NewPasswordFields, useNewPasswordForm } from "./common/new-password-fields";
// components
import { isPasswordConfirmationValid } from "@plane/utils/auth";
// hooks
import { useCurrentUser, usePasswordMutation } from "@/hooks/use-current-user";

// services
import { authService } from "@/services/auth.service";
// local components
import { FormContainer } from "./common/container";
import { AuthFormHeader } from "./common/header";

export function SetPasswordForm() {
  // router
  const router = useRouter();
  // states
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const { data: user } = useCurrentUser();
  const passwordMutation = usePasswordMutation();
  const [isCredentialSaved, setIsCredentialSaved] = useState(false);

  const form = useNewPasswordForm(async (value) => {
    try {
      if (!isCredentialSaved) {
        const { csrf_token } = await authService.requestCSRFToken();
        if (!csrf_token) throw new Error("CSRF token not found");
        await passwordMutation.mutateAsync({ kind: "set", csrfToken: csrf_token, password: value.password });
        passwordMutation.reset();
        setIsCredentialSaved(true);
      }
      await router.navigate({ href: "/" });
    } catch (error) {
      passwordMutation.reset();
      showAuthenticationError(t);
      throw error;
    }
  });
  const passwordFormData = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const isButtonDisabled =
    isSubmitting || !isPasswordConfirmationValid(passwordFormData.password, passwordFormData.confirm_password);

  return (
    <FormContainer>
      <AuthFormHeader title="Set password" description="Create a new password." />
      <form
        className="space-y-4"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit().catch(() => {
            // The failed password update is already reported with an authentication error.
          });
        }}
      >
        <NewPasswordFields
          email={user?.email ?? ""}
          password={passwordFormData.password}
          confirmation={passwordFormData.confirm_password}
          readOnly={isSubmitting || isCredentialSaved}
          onPasswordChange={(password) => form.setFieldValue("password", password)}
          onConfirmationChange={(confirmation) => form.setFieldValue("confirm_password", confirmation)}
        />
        <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
          {t("common.continue")}
        </Button>
      </form>
    </FormContainer>
  );
}
