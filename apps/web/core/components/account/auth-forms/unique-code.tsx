/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { showAuthenticationError } from "@/helpers/authentication.helper";

import { useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { XCircle } from "lucide-react";
import { API_BASE_URL } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { Input, Spinner, AuthCodeResend } from "@plane/ui";
// constants
// helpers
import { EAuthModes } from "@/helpers/authentication.helper";
// hooks
import { useResendCodeForm } from "@plane/hooks";
// services
import { authService } from "@/services/auth.service";

// services

type TAuthUniqueCodeForm = {
  mode: EAuthModes;
  email: string;
  isExistingEmail: boolean;
  handleEmailClear: () => void;
  generateEmailUniqueCode: (email: string) => Promise<void>;
  nextPath: string | undefined;
};

export function AuthUniqueCodeForm(props: TAuthUniqueCodeForm) {
  const { mode, email, handleEmailClear, generateEmailUniqueCode, nextPath } = props;
  // states
  const formRef = useRef<HTMLFormElement>(null);
  const form = useForm({
    defaultValues: { code: "" },
    validators: { onSubmit: ({ value }) => (value.code.trim() ? undefined : t("auth.common.unique_code.label")) },
    onSubmit: async () => {
      try {
        await authService.submitForm(formRef.current);
      } catch (error) {
        showAuthenticationError(t);
        throw error;
      }
    },
  });
  const code = useStore(form.store, (state) => state.values.code);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const { t } = useTranslation();
  const {
    isSubmitting: isRequestingNewCode,
    cooldown: resendTimerCode,
    resend,
  } = useResendCodeForm({
    send: () => generateEmailUniqueCode(email),
    onSent: () => form.setFieldValue("code", ""),
    onError: () => showAuthenticationError(t),
  });

  const isRequestNewCodeDisabled = isSubmitting || isRequestingNewCode || resendTimerCode > 0;
  const isButtonDisabled = isRequestingNewCode || !code || isSubmitting;

  return (
    <form
      ref={formRef}
      className="space-y-4"
      method="POST"
      action={`${API_BASE_URL}/auth/${mode === EAuthModes.SIGN_IN ? "magic-sign-in" : "magic-sign-up"}/`}
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
            className="h-10 w-full border-0 disable-autofill-style placeholder:text-placeholder"
            autoComplete="off"
            disabled
          />
          {email.length > 0 && (
            <button
              type="button"
              className="absolute right-3 grid size-5 place-items-center"
              aria-label={t("aria_labels.auth_forms.clear_email")}
              onClick={handleEmailClear}
            >
              <XCircle className="size-5 stroke-placeholder" />
            </button>
          )}
        </div>
      </div>

      <div className="space-y-1">
        <label htmlFor="unique-code" className="text-13 font-medium text-tertiary">
          {t("auth.common.unique_code.label")}
        </label>
        <Input
          name="code"
          readOnly={isSubmitting}
          id="unique-code"
          value={code}
          onChange={(e) => form.setFieldValue("code", e.target.value)}
          placeholder={t("auth.common.unique_code.placeholder")}
          className="h-10 w-full border border-strong !bg-surface-1 pr-12 disable-autofill-style placeholder:text-placeholder"
          autoComplete="off"
        />
        <AuthCodeResend
          hint={t("auth.common.unique_code.paste_code")}
          disabled={isRequestNewCodeDisabled}
          onResend={resend}
        >
          {resendTimerCode > 0
            ? t("auth.common.resend_in", { seconds: resendTimerCode })
            : isRequestingNewCode
              ? t("auth.common.unique_code.requesting_new_code")
              : t("common.resend")}
        </AuthCodeResend>
      </div>

      <div className="space-y-2.5">
        <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
          {isRequestingNewCode ? (
            t("auth.common.unique_code.sending_code")
          ) : isSubmitting ? (
            <Spinner height="20px" width="20px" />
          ) : (
            t("common.continue")
          )}
        </Button>
      </div>
    </form>
  );
}
