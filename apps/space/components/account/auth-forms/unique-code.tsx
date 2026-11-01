/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AuthEmailFields } from "./auth-email-fields";
import React, { useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// plane imports
import { API_BASE_URL } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { Input, Spinner, AuthCodeResend } from "@plane/ui";
import { handleNativeAuthSubmit, submitNativeAuthForm } from "./native-auth-submit";
// hooks
import { useResendCodeForm } from "@plane/hooks";
// types
import { EAuthModes } from "@/types/auth";

type TAuthUniqueCodeForm = {
  mode: EAuthModes;
  email: string;
  nextPath: string | undefined;
  handleEmailClear: () => void;
  generateEmailUniqueCode: (email: string) => Promise<void>;
};

export function AuthUniqueCodeForm(props: TAuthUniqueCodeForm) {
  const { mode, email, nextPath, handleEmailClear, generateEmailUniqueCode } = props;
  // states
  const formRef = useRef<HTMLFormElement>(null);
  const form = useForm({
    defaultValues: { code: "" },
    validators: { onSubmit: ({ value }) => (value.code.trim() ? undefined : "Enter your unique code.") },
    onSubmit: () => submitNativeAuthForm(formRef.current),
  });
  const code = useStore(form.store, (state) => state.values.code);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const {
    isSubmitting: isRequestingNewCode,
    cooldown: resendTimerCode,
    resend,
  } = useResendCodeForm({
    send: () => generateEmailUniqueCode(email),
    onSent: () => form.setFieldValue("code", ""),
    onError: () => setToast({ type: TOAST_TYPE.ERROR, title: "Could not resend code", message: "Please try again." }),
  });

  const isRequestNewCodeDisabled = isSubmitting || isRequestingNewCode || resendTimerCode > 0;
  const isButtonDisabled = isRequestingNewCode || !code || isSubmitting;

  return (
    <form
      ref={formRef}
      className="mt-5 space-y-4"
      method="POST"
      action={`${API_BASE_URL}/auth/spaces/${mode === EAuthModes.SIGN_IN ? "magic-sign-in" : "magic-sign-up"}/`}
      onSubmit={(event) => handleNativeAuthSubmit(event, () => form.handleSubmit())}
    >
      <input type="hidden" name="csrfmiddlewaretoken" />
      <AuthEmailFields email={email} nextPath={nextPath} onClear={handleEmailClear} />

      <div className="space-y-1">
        <label className="text-13 font-medium text-tertiary" htmlFor="code">
          Unique code
        </label>
        <Input
          name="code"
          id="code"
          readOnly={isSubmitting}
          value={code}
          onChange={(e) => form.setFieldValue("code", e.target.value)}
          placeholder="123456"
          className="h-10 w-full border border-subtle !bg-surface-1 pr-12 disable-autofill-style placeholder:text-placeholder"
          autoComplete="off"
        />
        <AuthCodeResend hint="Paste the code sent to your email" disabled={isRequestNewCodeDisabled} onResend={resend}>
          {resendTimerCode > 0
            ? `Resend in ${resendTimerCode}s`
            : isRequestingNewCode
              ? "Requesting new code"
              : "Resend"}
        </AuthCodeResend>
      </div>

      <div className="space-y-2.5">
        <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
          {isRequestingNewCode ? "Sending code" : isSubmitting ? <Spinner height="20px" width="20px" /> : "Continue"}
        </Button>
      </div>
    </form>
  );
}
