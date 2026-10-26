/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm } from "@tanstack/react-form";
import { isPasswordConfirmationValid } from "@plane/utils/auth";
import { useTranslation } from "@plane/i18n";
import { Input, AuthPasswordInput, AuthConfirmPasswordInput } from "@plane/ui";

type Props = {
  email: string;
  password: string;
  confirmation: string;
  readOnly: boolean;
  onPasswordChange: (value: string) => void;
  onConfirmationChange: (value: string) => void;
};

export function NewPasswordFields({
  email,
  password,
  confirmation,
  readOnly,
  onPasswordChange,
  onConfirmationChange,
}: Props) {
  const { t } = useTranslation();
  return (
    <>
      <label className="flex flex-col gap-1 text-13 font-medium text-tertiary">
        {t("auth.common.email.label")}
        <Input
          name="email"
          type="email"
          value={email}
          disabled
          className="h-10 w-full border border-strong !bg-surface-1 text-placeholder"
        />
      </label>
      <AuthPasswordInput
        id="password"
        name="password"
        label={t("auth.common.password.label")}
        value={password}
        onChange={(event) => onPasswordChange(event.target.value)}
        readOnly={readOnly}
        autoComplete="new-password"
        minLength={8}
        placeholder={t("auth.common.password.placeholder")}
      />
      <AuthConfirmPasswordInput
        id="confirm_password"
        name="confirm_password"
        label={t("auth.common.password.confirm_password.label")}
        password={password}
        value={confirmation}
        onChange={(event) => onConfirmationChange(event.target.value)}
        readOnly={readOnly}
        autoComplete="new-password"
        placeholder={t("auth.common.password.confirm_password.placeholder")}
      />
    </>
  );
}

export function useNewPasswordForm(onSubmit: (value: { password: string; confirm_password: string }) => Promise<void>) {
  const { t } = useTranslation();
  return useForm({
    defaultValues: { password: "", confirm_password: "" },
    validators: {
      onSubmit: ({ value }) =>
        isPasswordConfirmationValid(value.password, value.confirm_password)
          ? undefined
          : t("auth.common.password.errors.match"),
    },
    onSubmit: ({ value }) => onSubmit(value),
  });
}
