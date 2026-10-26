/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useLocation } from "@tanstack/react-router";

import { useForm, useStore } from "@tanstack/react-form";
// icons
import { CircleCheck } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Button, getButtonStyling } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Input } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { checkEmailValidity } from "@plane/utils/string";
// hooks
import useTimer from "@/hooks/use-timer";
// services
import { authService } from "@/services/auth.service";
// local components
import { FormContainer } from "./common/container";
import { AuthFormHeader } from "./common/header";

type TForgotPasswordFormValues = {
  email: string;
};

const defaultValues: TForgotPasswordFormValues = {
  email: "",
};

// services

export const ForgotPasswordForm = function ForgotPasswordForm() {
  // search params
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const email = searchParams.get("email");
  // plane hooks
  const { t } = useTranslation();
  // timer
  const { timer: resendTimerCode, setTimer: setResendCodeTimer } = useTimer(0);

  const handleForgotPassword = async (formData: TForgotPasswordFormValues) => {
    try {
      await authService.sendResetPasswordLink({ email: formData.email });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("auth.forgot_password.toast.success.title"),
        message: t("auth.forgot_password.toast.success.message"),
      });
      setResendCodeTimer(30);
    } catch (error) {
      const errorMessage = error && typeof error === "object" && "error" in error ? Reflect.get(error, "error") : null;
      const message = typeof errorMessage === "string" ? errorMessage : t("auth.forgot_password.toast.error.message");
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("auth.forgot_password.toast.error.title"),
        message,
      });
      throw error;
    }
  };

  const form = useForm({
    defaultValues: { ...defaultValues, email: email ?? "" },
    onSubmit: async ({ value }) => {
      if (resendTimerCode > 0) return;
      await handleForgotPassword(value);
    },
  });
  const canSubmit = useStore(form.store, (state) => state.canSubmit && checkEmailValidity(state.values.email));
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);

  return (
    <FormContainer>
      <AuthFormHeader title="Reset password" description="Regain access to your account." />
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit().catch(() => {
            // The failed reset request is already reported with a toast.
          });
        }}
        className="space-y-4"
      >
        <div className="space-y-1">
          <label className="text-13 font-medium text-tertiary" htmlFor="email">
            {t("auth.common.email.label")}
          </label>
          <form.Field
            name="email"
            validators={{
              onChange: ({ value }) =>
                !value
                  ? t("auth.common.email.errors.required")
                  : !checkEmailValidity(value)
                    ? t("auth.common.email.errors.invalid")
                    : undefined,
            }}
          >
            {(field) => (
              <>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  hasError={!field.state.meta.isValid}
                  placeholder={t("auth.common.email.placeholder")}
                  className="h-10 w-full border border-strong !bg-surface-1 pr-12 placeholder:text-placeholder"
                  autoComplete="off"
                  disabled={isSubmitting || resendTimerCode > 0}
                />
                {!field.state.meta.isValid && (
                  <p role="alert" className="text-11 text-danger-primary">
                    {field.state.meta.errors.join(" ")}
                  </p>
                )}
              </>
            )}
          </form.Field>
          {resendTimerCode > 0 && (
            <p className="flex w-full items-start gap-1 px-1 text-11 font-medium text-success-primary">
              <CircleCheck height={12} width={12} className="mt-0.5" />
              {t("auth.forgot_password.email_sent")}
            </p>
          )}
        </div>
        <Button
          type="submit"
          variant="primary"
          className="w-full"
          size="xl"
          disabled={!canSubmit}
          loading={isSubmitting || resendTimerCode > 0}
        >
          {resendTimerCode > 0
            ? t("auth.common.resend_in", { seconds: resendTimerCode })
            : t("auth.forgot_password.send_reset_link")}
        </Button>
        <Link to="/" className={cn("w-full", getButtonStyling("link", "lg"))}>
          {t("auth.common.back_to_sign_in")}
        </Link>
      </form>
    </FormContainer>
  );
};
