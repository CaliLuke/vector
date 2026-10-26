/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useForm, useStore } from "@tanstack/react-form";
import { useState } from "react";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Input, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { checkEmailValidity } from "@plane/utils/string";
import { useSignOut } from "@/hooks/use-current-user";
import { authService } from "@/services/auth.service";
import userService from "@/services/user.service";

type Props = { isOpen: boolean; onClose: () => void };

export function ChangeEmailModal({ isOpen, onClose }: Props) {
  return isOpen ? <ChangeEmailDialog onClose={onClose} /> : null;
}

function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === "object") {
    for (const key of ["error_message", "error", "message"] as const) {
      if (key in error) {
        const value = Reflect.get(error, key);
        if (typeof value === "string") return value;
      }
    }
  }
  return fallback;
}

function ChangeEmailDialog({ onClose }: Pick<Props, "onClose">) {
  const [step, setStep] = useState<"email" | "code" | "verified">("email");
  const { mutateAsync: signOut } = useSignOut();
  const { t } = useTranslation();
  const label = (path: string) => t(`account_settings.profile.change_email_modal.${path}`);
  const form = useForm({
    defaultValues: { email: "", code: "" },
    validators: {
      onSubmit: (): string | undefined => undefined,
      onSubmitAsync: async ({ value }) => {
        if (step !== "email") return undefined;
        try {
          const { csrf_token } = await authService.requestCSRFToken();
          if (!csrf_token) return label("form.email.errors.validation_failed");
          const result = await userService.checkEmail(csrf_token, value.email);
          return result.existing ? label("form.email.errors.exists") : undefined;
        } catch (cause) {
          return errorMessage(cause, label("form.email.errors.validation_failed"));
        }
      },
    },
    onSubmit: async ({ value }) => {
      if (step === "email") {
        await userService.generateEmailCode({ email: value.email });
        setStep("code");
        return;
      }
      if (step === "code") {
        try {
          await userService.verifyEmailCode(value);
          setStep("verified");
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: label("toasts.success_title"),
            message: label("toasts.success_message"),
          });
        } catch (cause) {
          throw new Error(errorMessage(cause, label("form.code.errors.invalid")), { cause });
        }
      }
      try {
        await signOut();
        onClose();
      } catch (cause) {
        throw new Error(t("auth.sign_out.toast.error.message"), { cause });
      }
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const close = () => {
    if (!busy) onClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <div className="px-4 py-4">
        <h3 className="text-16 leading-6 font-medium text-primary">{label("title")}</h3>
        <p className="my-4 text-13 text-secondary">{label("description")}</p>
      </div>
      <form
        className="space-y-4 px-4"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch((cause: unknown) => {
            const fallback =
              step === "email"
                ? label("form.email.errors.validation_failed")
                : step === "code"
                  ? label("form.code.errors.invalid")
                  : t("auth.sign_out.toast.error.message");
            form.setErrorMap({ onSubmit: { form: errorMessage(cause, fallback), fields: {} } });
          });
        }}
      >
        <form.Field
          name="email"
          validators={{
            onChange: ({ value }) => (checkEmailValidity(value) ? undefined : label("form.email.errors.invalid")),
          }}
        >
          {(field) => (
            <label className="flex flex-col gap-1 text-13">
              {label("form.email.label")}
              <Input
                name={field.name}
                type="email"
                value={field.state.value}
                autoComplete="email"
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                disabled={busy || step !== "email"}
                hasError={!field.state.meta.isValid}
                placeholder={label("form.email.placeholder")}
              />
              {field.state.meta.errors.map((message) => (
                <span role="alert" key={message} className="text-danger-primary">
                  {message}
                </span>
              ))}
            </label>
          )}
        </form.Field>
        {step !== "email" && (
          <form.Field
            name="code"
            validators={{ onChange: ({ value }) => (value.trim() ? undefined : label("form.code.errors.required")) }}
          >
            {(field) => (
              <label className="flex flex-col gap-1 text-13">
                {label("form.code.label")}
                <Input
                  name={field.name}
                  value={field.state.value}
                  autoComplete="one-time-code"
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  disabled={busy || step === "verified"}
                  hasError={!field.state.meta.isValid}
                  placeholder={label("form.code.placeholder")}
                />
                {field.state.meta.errors.map((message) => (
                  <span role="alert" key={message} className="text-danger-primary">
                    {message}
                  </span>
                ))}
                <span className="text-success-primary">{label("form.code.helper_text")}</span>
              </label>
            )}
          </form.Field>
        )}
        {error && (
          <p role="alert" className="text-13 text-danger-primary">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-2 border-t border-subtle py-4">
          <Button type="button" variant="secondary" disabled={busy} onClick={close}>
            {label("actions.cancel")}
          </Button>
          <Button type="submit" variant="primary" loading={busy}>
            {step === "verified" ? "Retry sign out" : label(step === "email" ? "actions.continue" : "actions.confirm")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
