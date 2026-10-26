/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AuthEmailFields } from "./auth-email-fields";
import React, { useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
// plane imports
import { API_BASE_URL } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { AuthPasswordInput, AuthConfirmPasswordInput, Spinner } from "@plane/ui";
import { isPasswordConfirmationValid } from "@plane/utils/auth";
import { handleNativeAuthSubmit, submitNativeAuthForm } from "./native-auth-submit";
// types
import { EAuthModes, EAuthSteps } from "@/types/auth";

type Props = {
  email: string;
  isSMTPConfigured: boolean;
  mode: EAuthModes;
  nextPath: string | undefined;
  handleEmailClear: () => void;
  handleAuthStep: (step: EAuthSteps) => void;
};

export const AuthPasswordForm = function AuthPasswordForm(props: Props) {
  const { email, nextPath, isSMTPConfigured, handleAuthStep, handleEmailClear, mode } = props;
  // ref
  const formRef = useRef<HTMLFormElement>(null);
  // states
  const form = useForm({
    defaultValues: { password: "", confirm_password: "" },
    validators: {
      onSubmit: ({ value }) =>
        !value.password ||
        (mode === EAuthModes.SIGN_UP && !isPasswordConfirmationValid(value.password, value.confirm_password))
          ? "Enter a valid password and matching confirmation."
          : undefined,
    },
    onSubmit: () => submitNativeAuthForm(formRef.current),
  });
  const passwordFormData = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const redirectToUniqueCodeSignIn = () => {
    handleAuthStep(EAuthSteps.UNIQUE_CODE);
  };

  const isButtonDisabled =
    isSubmitting ||
    !passwordFormData.password ||
    (mode === EAuthModes.SIGN_UP &&
      !isPasswordConfirmationValid(passwordFormData.password, passwordFormData.confirm_password));

  return (
    <form
      ref={formRef}
      className="mt-5 space-y-4"
      method="POST"
      action={`${API_BASE_URL}/auth/spaces/${mode === EAuthModes.SIGN_IN ? "sign-in" : "sign-up"}/`}
      onSubmit={(event) => handleNativeAuthSubmit(event, () => form.handleSubmit())}
    >
      <input type="hidden" name="csrfmiddlewaretoken" />
      <AuthEmailFields email={email} nextPath={nextPath} onClear={handleEmailClear} />

      <AuthPasswordInput
        id="password"
        name="password"
        label={mode === EAuthModes.SIGN_IN ? "Password" : "Set a password"}
        value={passwordFormData.password}
        onChange={(event) => form.setFieldValue("password", event.target.value)}
        readOnly={isSubmitting}
        showPasswordStrength={mode === EAuthModes.SIGN_UP}
        autoComplete={mode === EAuthModes.SIGN_IN ? "current-password" : "new-password"}
      />

      {mode === EAuthModes.SIGN_UP && (
        <AuthConfirmPasswordInput
          id="confirm_password"
          name="confirm_password"
          label={"Confirm password"}
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
                "Continue"
              ) : (
                "Go to workspace"
              )}
            </Button>
            {isSMTPConfigured && (
              <Button
                type="button"
                onClick={redirectToUniqueCodeSignIn}
                variant="secondary"
                className="w-full"
                size="xl"
              >
                Sign in with unique code
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
  );
};
