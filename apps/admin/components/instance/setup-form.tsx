/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useSearchParams } from "react-router";
// icons
import { Eye, EyeOff } from "lucide-react";
// plane internal packages
import { API_BASE_URL, E_PASSWORD_STRENGTH } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { Checkbox, Input, AuthEmailInput, PasswordStrengthIndicator, Spinner } from "@plane/ui";
import { getPasswordStrength } from "@plane/utils/auth";
import { validatePersonName, validateCompanyName } from "@plane/utils/validation";
// components
import { submitInstanceAuthForm, getInstanceFormError } from "@/app/(all)/(home)/auth-helpers";
import { AuthHeader } from "@/app/(all)/(home)/auth-header";
import { Banner } from "../common/banner";
import { FormHeader } from "./form-header";

// service initialization

// error codes
enum EErrorCodes {
  INSTANCE_NOT_CONFIGURED = "INSTANCE_NOT_CONFIGURED",
  ADMIN_ALREADY_EXIST = "ADMIN_ALREADY_EXIST",
  REQUIRED_EMAIL_PASSWORD_FIRST_NAME = "REQUIRED_EMAIL_PASSWORD_FIRST_NAME",
  INVALID_EMAIL = "INVALID_EMAIL",
  INVALID_PASSWORD = "INVALID_PASSWORD",
  USER_ALREADY_EXISTS = "USER_ALREADY_EXISTS",
}

// form data
type TFormData = {
  first_name: string;
  last_name: string;
  email: string;
  company_name: string;
  password: string;
  confirm_password: string;
  is_telemetry_enabled: boolean;
};

const defaultFromData: TFormData = {
  first_name: "",
  last_name: "",
  email: "",
  company_name: "",
  password: "",
  confirm_password: "",
  is_telemetry_enabled: true,
};

function setupDefaults(searchParams: URLSearchParams): TFormData {
  return {
    ...defaultFromData,
    first_name: searchParams.get("first_name") ?? "",
    last_name: searchParams.get("last_name") ?? "",
    company_name: searchParams.get("company") ?? "",
    email: searchParams.get("email") ?? "",
    is_telemetry_enabled: searchParams.get("is_telemetry_enabled") !== "False",
  };
}

function setupIsValid(value: TFormData): boolean {
  return Boolean(
    value.first_name &&
    value.email &&
    value.password &&
    getPasswordStrength(value.password) === E_PASSWORD_STRENGTH.STRENGTH_VALID &&
    value.password === value.confirm_password
  );
}

function SetupPasswordFields({
  password,
  confirmPassword,
  isSubmitting,
  error,
  onPasswordChange,
  onConfirmPasswordChange,
}: {
  password: string;
  confirmPassword: string;
  isSubmitting: boolean;
  error: string | undefined;
  onPasswordChange: (value: string) => void;
  onConfirmPasswordChange: (value: string) => void;
}) {
  const [showPassword, setShowPassword] = useState({ password: false, retypePassword: false });
  const [isPasswordInputFocused, setIsPasswordInputFocused] = useState(false);
  const [isRetryPasswordInputFocused, setIsRetryPasswordInputFocused] = useState(false);
  const handleShowPassword = (key: keyof typeof showPassword) =>
    setShowPassword((previous) => ({ ...previous, [key]: !previous[key] }));
  const renderPasswordMatchError = !isRetryPasswordInputFocused || confirmPassword.length >= password.length;
  return (
    <>
      <div className="w-full space-y-1">
        <label className="text-13 font-medium text-tertiary" htmlFor="password">
          Set a password <span className="text-danger-primary">*</span>
        </label>
        <div className="relative">
          <Input
            readOnly={isSubmitting}
            className="w-full border border-subtle !bg-surface-1 placeholder:text-placeholder"
            id="password"
            name="password"
            type={showPassword.password ? "text" : "password"}
            inputSize="md"
            placeholder="New password"
            value={password}
            onChange={(e) => onPasswordChange(e.target.value)}
            hasError={Boolean(error)}
            onFocus={() => setIsPasswordInputFocused(true)}
            onBlur={() => setIsPasswordInputFocused(false)}
            autoComplete="new-password"
          />
          {showPassword.password ? (
            <button
              type="button"
              aria-label="Hide password"
              className="absolute top-3.5 right-3 flex items-center justify-center text-placeholder"
              onClick={() => handleShowPassword("password")}
            >
              <EyeOff className="h-4 w-4" />
            </button>
          ) : (
            <button
              type="button"
              aria-label="Show password"
              className="absolute top-3.5 right-3 flex items-center justify-center text-placeholder"
              onClick={() => handleShowPassword("password")}
            >
              <Eye className="h-4 w-4" />
            </button>
          )}
        </div>
        {error && <p className="px-1 text-11 text-danger-primary">{error}</p>}
        <PasswordStrengthIndicator password={password} isFocused={isPasswordInputFocused} />
      </div>

      <div className="w-full space-y-1">
        <label className="text-13 font-medium text-tertiary" htmlFor="confirm_password">
          Confirm password <span className="text-danger-primary">*</span>
        </label>
        <div className="relative">
          <Input
            readOnly={isSubmitting}
            type={showPassword.retypePassword ? "text" : "password"}
            id="confirm_password"
            name="confirm_password"
            inputSize="md"
            value={confirmPassword}
            onChange={(e) => onConfirmPasswordChange(e.target.value)}
            placeholder="Confirm password"
            className="w-full border border-subtle !bg-surface-1 pr-12 placeholder:text-placeholder"
            onFocus={() => setIsRetryPasswordInputFocused(true)}
            onBlur={() => setIsRetryPasswordInputFocused(false)}
            autoComplete="new-password"
          />
          {showPassword.retypePassword ? (
            <button
              type="button"
              aria-label="Hide password"
              className="absolute top-3.5 right-3 flex items-center justify-center text-placeholder"
              onClick={() => handleShowPassword("retypePassword")}
            >
              <EyeOff className="h-4 w-4" />
            </button>
          ) : (
            <button
              type="button"
              aria-label="Show password"
              className="absolute top-3.5 right-3 flex items-center justify-center text-placeholder"
              onClick={() => handleShowPassword("retypePassword")}
            >
              <Eye className="h-4 w-4" />
            </button>
          )}
        </div>
        {!!confirmPassword && password !== confirmPassword && renderPasswordMatchError && (
          <span className="text-13 text-danger-primary">Passwords don{"'"}t match</span>
        )}
      </div>
    </>
  );
}

export function InstanceSetupForm() {
  // search params
  const [searchParams] = useSearchParams();
  const errorCode = searchParams.get("error_code");
  const errorMessage = searchParams.get("error_message") || undefined;
  const formRef = useRef<HTMLFormElement>(null);
  const form = useForm({
    defaultValues: setupDefaults(searchParams),
    validators: {
      onSubmit: ({ value }) => (setupIsValid(value) ? undefined : "Complete the required fields."),
    },
    onSubmit: () => submitInstanceAuthForm(formRef.current),
  });
  const formData = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  // derived values
  const errorData = getInstanceFormError(Object.values(EErrorCodes), errorCode, errorMessage);

  const isButtonDisabled = isSubmitting || !setupIsValid(formData);

  return (
    <>
      <AuthHeader />
      <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
        <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">
          <FormHeader
            heading="Setup your Vector Instance"
            subHeading="Post setup you will be able to manage this Vector instance."
          />
          {errorData.type &&
            errorData?.message &&
            ![EErrorCodes.INVALID_EMAIL, EErrorCodes.INVALID_PASSWORD].includes(errorData.type) && (
              <Banner type="error" message={errorData?.message} />
            )}
          <form
            ref={formRef}
            className="space-y-4"
            method="POST"
            action={`${API_BASE_URL}/api/instances/admins/sign-up/`}
            onSubmit={(event) => {
              event.preventDefault();
              void form.handleSubmit();
            }}
          >
            <input type="hidden" name="csrfmiddlewaretoken" />
            <input type="hidden" name="is_telemetry_enabled" value={formData.is_telemetry_enabled ? "True" : "False"} />

            <div className="flex flex-col items-center gap-4 sm:flex-row">
              <div className="w-full space-y-1">
                <label className="text-13 font-medium text-tertiary" htmlFor="first_name">
                  First name <span className="text-danger-primary">*</span>
                </label>
                <Input
                  readOnly={isSubmitting}
                  className="w-full border border-subtle !bg-surface-1 placeholder:text-placeholder"
                  id="first_name"
                  name="first_name"
                  type="text"
                  inputSize="md"
                  placeholder="Wilber"
                  value={formData.first_name}
                  onChange={(e) => {
                    const validation = validatePersonName(e.target.value);
                    if (validation === true || e.target.value === "") {
                      form.setFieldValue("first_name", e.target.value);
                    }
                  }}
                  autoComplete="off"
                  maxLength={50}
                />
              </div>
              <div className="w-full space-y-1">
                <label className="text-13 font-medium text-tertiary" htmlFor="last_name">
                  Last name <span className="text-danger-primary">*</span>
                </label>
                <Input
                  readOnly={isSubmitting}
                  className="w-full border border-subtle !bg-surface-1 placeholder:text-placeholder"
                  id="last_name"
                  name="last_name"
                  type="text"
                  inputSize="md"
                  placeholder="Wright"
                  value={formData.last_name}
                  onChange={(e) => {
                    const validation = validatePersonName(e.target.value);
                    if (validation === true || e.target.value === "") {
                      form.setFieldValue("last_name", e.target.value);
                    }
                  }}
                  autoComplete="off"
                  maxLength={50}
                />
              </div>
            </div>

            <AuthEmailInput
              label="Email *"
              readOnly={isSubmitting}
              value={formData.email}
              onChange={(event) => form.setFieldValue("email", event.target.value)}
              error={errorData.type === EErrorCodes.INVALID_EMAIL ? errorData.message : undefined}
            />

            <div className="w-full space-y-1">
              <label className="text-13 font-medium text-tertiary" htmlFor="company_name">
                Company name <span className="text-danger-primary">*</span>
              </label>
              <Input
                readOnly={isSubmitting}
                className="w-full border border-subtle !bg-surface-1 placeholder:text-placeholder"
                id="company_name"
                name="company_name"
                type="text"
                inputSize="md"
                placeholder="Company name"
                value={formData.company_name}
                onChange={(e) => {
                  const validation = validateCompanyName(e.target.value, false);
                  if (validation === true || e.target.value === "") {
                    form.setFieldValue("company_name", e.target.value);
                  }
                }}
                maxLength={80}
              />
            </div>

            <SetupPasswordFields
              password={formData.password}
              confirmPassword={formData.confirm_password}
              isSubmitting={isSubmitting}
              error={errorData.type === EErrorCodes.INVALID_PASSWORD ? errorData.message : undefined}
              onPasswordChange={(value) => form.setFieldValue("password", value)}
              onConfirmPasswordChange={(value) => form.setFieldValue("confirm_password", value)}
            />

            <div className="relative flex gap-2">
              <div>
                <Checkbox
                  disabled={isSubmitting}
                  className="h-4 w-4"
                  iconClassName="w-3 h-3"
                  id="is_telemetry_enabled"
                  onChange={() => form.setFieldValue("is_telemetry_enabled", !formData.is_telemetry_enabled)}
                  checked={formData.is_telemetry_enabled}
                />
              </div>
              <label className="cursor-pointer text-13 font-medium text-tertiary" htmlFor="is_telemetry_enabled">
                Allow Vector to anonymously collect usage events.{" "}
                <a
                  href="https://github.com/CaliLuke/vector/blob/main/docs/README.md"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-500 hover:text-blue-600 flex-shrink-0 text-13 font-medium"
                >
                  See More
                </a>
              </label>
            </div>

            <div className="py-2">
              <Button type="submit" size="xl" className="w-full" disabled={isButtonDisabled}>
                {isSubmitting ? <Spinner height="20px" width="20px" /> : "Continue"}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}
