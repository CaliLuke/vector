/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { AuthPasswordInput, AuthConfirmPasswordInput } from "@plane/ui";
import { isPasswordConfirmationValid } from "@plane/utils/auth";
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
import { authErrorHandler, EAuthenticationErrorCodes, passwordErrors } from "@/helpers/authentication.helper";
import { useCurrentUser, usePasswordMutation } from "@/hooks/use-current-user";
import { authService } from "@/services/auth.service";

export function SecurityProfileSettings() {
  const { data: currentUser } = useCurrentUser();
  const passwordMutation = usePasswordMutation();
  const { t } = useTranslation();
  const oldPasswordRequired = !currentUser?.is_password_autoset;
  const form = useForm({
    defaultValues: { old_password: "", new_password: "", confirm_password: "" },
    validators: {
      onChange: ({ value }) => {
        if (oldPasswordRequired && !value.old_password.trim()) return t("common.errors.required");
        if (value.new_password && value.new_password === value.old_password)
          return t("new_password_must_be_different_from_old_password");
        if (!isPasswordConfirmationValid(value.new_password, value.confirm_password))
          return t("auth.common.password.errors.match");
        return undefined;
      },
    },
    onSubmit: async ({ value, formApi }) => {
      try {
        const { csrf_token } = await authService.requestCSRFToken();
        if (!csrf_token) throw new Error("CSRF token not found");
        await passwordMutation.mutateAsync({
          kind: "change",
          csrfToken: csrf_token,
          ...(oldPasswordRequired && { old_password: value.old_password }),
          new_password: value.new_password,
        });
        passwordMutation.reset();
        formApi.reset();
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("auth.common.password.toast.change_password.success.title"),
          message: t("auth.common.password.toast.change_password.success.message"),
        });
      } catch (error) {
        passwordMutation.reset();
        const code = error && typeof error === "object" && "error_code" in error ? String(error.error_code) : undefined;
        const info = code ? authErrorHandler(code as EAuthenticationErrorCodes) : undefined;
        const message =
          typeof info?.message === "string"
            ? info.message
            : t("auth.common.password.toast.change_password.error.message");
        setToast({
          type: TOAST_TYPE.ERROR,
          title: info?.title ?? t("auth.common.password.toast.change_password.error.title"),
          message,
        });
        if (code && passwordErrors.includes(code as EAuthenticationErrorCodes)) {
          formApi.setFieldMeta("new_password", (meta) => ({
            ...meta,
            errorMap: { ...meta.errorMap, onChange: message },
          }));
        }
        throw error;
      }
    },
  });
  const { isSubmitting, canSubmit, values } = useStore(form.store, (state) => ({
    isSubmitting: state.isSubmitting,
    canSubmit: state.canSubmit,
    values: state.values,
  }));
  return (
    <div className="size-full">
      <ProfileSettingsHeading title={t("auth.common.password.change_password.label.default")} />
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit().catch(() => {
            // The password change failure is already reported with a toast.
          });
        }}
        className="mt-7 flex flex-col gap-8"
      >
        {oldPasswordRequired && (
          <form.Field name="old_password">
            {(field) => (
              <AuthPasswordInput
                id={field.name}
                name={field.name}
                label={t("auth.common.password.current_password.label")}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                autoComplete="current-password"
                disabled={isSubmitting}
                showPasswordStrength={false}
                placeholder={t("old_password")}
              />
            )}
          </form.Field>
        )}
        <div className="grid gap-x-4 gap-y-7 sm:grid-cols-2">
          <form.Field name="new_password">
            {(field) => (
              <AuthPasswordInput
                id={field.name}
                name={field.name}
                label={t("auth.common.password.new_password.label")}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                error={field.state.meta.errors.join(" ")}
                autoComplete="new-password"
                disabled={isSubmitting}
                placeholder={t("auth.common.password.new_password.placeholder")}
              />
            )}
          </form.Field>
          <form.Field name="confirm_password">
            {(field) => (
              <AuthConfirmPasswordInput
                id={field.name}
                name={field.name}
                label={t("auth.common.password.confirm_password.label")}
                password={values.new_password}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                autoComplete="new-password"
                disabled={isSubmitting}
                placeholder={t("auth.common.password.confirm_password.placeholder")}
              />
            )}
          </form.Field>
        </div>
        {values.new_password && values.new_password === values.old_password && (
          <p role="alert" className="text-11 text-danger-primary">
            {t("new_password_must_be_different_from_old_password")}
          </p>
        )}
        <div>
          <Button
            variant="primary"
            size="xl"
            type="submit"
            loading={isSubmitting}
            disabled={!canSubmit || !values.new_password || isSubmitting}
          >
            {t(
              isSubmitting
                ? "auth.common.password.change_password.label.submitting"
                : "auth.common.password.change_password.label.default"
            )}
          </Button>
        </div>
      </form>
    </div>
  );
}
