/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useSearchParams } from "react-router";
import { Eye, EyeOff } from "lucide-react";
// plane internal packages
import type { EAdminAuthErrorCodes } from "@plane/constants";
import { API_BASE_URL } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { Input, AuthEmailInput, Spinner } from "@plane/ui";
// components
import { Banner } from "@/components/common/banner";
// local components
import { FormHeader } from "@/components/instance/form-header";
import { AuthBanner } from "./auth-banner";
import { AuthHeader } from "./auth-header";
import { authErrorHandler, submitInstanceAuthForm } from "./auth-helpers";

// service initialization

// error codes
enum EErrorCodes {
  INSTANCE_NOT_CONFIGURED = "INSTANCE_NOT_CONFIGURED",
  REQUIRED_EMAIL_PASSWORD = "REQUIRED_EMAIL_PASSWORD",
  INVALID_EMAIL = "INVALID_EMAIL",
  USER_DOES_NOT_EXIST = "USER_DOES_NOT_EXIST",
  AUTHENTICATION_FAILED = "AUTHENTICATION_FAILED",
}

type TError = {
  type: EErrorCodes | undefined;
  message: string | undefined;
};

// form data
type TFormData = {
  email: string;
  password: string;
};

const defaultFromData: TFormData = {
  email: "",
  password: "",
};

export function InstanceSignInForm() {
  // search params
  const [searchParams] = useSearchParams();
  const emailParam = searchParams.get("email") || undefined;
  const errorCode = searchParams.get("error_code") || undefined;
  const errorMessage = searchParams.get("error_message") || undefined;
  // state
  const [showPassword, setShowPassword] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const form = useForm({
    defaultValues: { ...defaultFromData, email: emailParam ?? "" },
    validators: {
      onSubmit: ({ value }) => (!value.email || !value.password ? "Complete the required fields." : undefined),
    },
    onSubmit: () => submitInstanceAuthForm(formRef.current),
  });
  const formData = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const [dismissedErrorCode, setDismissedErrorCode] = useState<string>();
  const errorInfo =
    errorCode && errorCode !== dismissedErrorCode ? authErrorHandler(errorCode as EAdminAuthErrorCodes) : undefined;

  // derived values
  const errorData: TError = useMemo(() => {
    if (errorCode && errorMessage) {
      switch (errorCode) {
        case EErrorCodes.INSTANCE_NOT_CONFIGURED:
          return { type: EErrorCodes.INSTANCE_NOT_CONFIGURED, message: errorMessage };
        case EErrorCodes.REQUIRED_EMAIL_PASSWORD:
          return { type: EErrorCodes.REQUIRED_EMAIL_PASSWORD, message: errorMessage };
        case EErrorCodes.INVALID_EMAIL:
          return { type: EErrorCodes.INVALID_EMAIL, message: errorMessage };
        case EErrorCodes.USER_DOES_NOT_EXIST:
          return { type: EErrorCodes.USER_DOES_NOT_EXIST, message: errorMessage };
        case EErrorCodes.AUTHENTICATION_FAILED:
          return { type: EErrorCodes.AUTHENTICATION_FAILED, message: errorMessage };
        default:
          return { type: undefined, message: undefined };
      }
    } else return { type: undefined, message: undefined };
  }, [errorCode, errorMessage]);

  const isButtonDisabled = useMemo(
    () => isSubmitting || !formData.email || !formData.password,
    [formData.email, formData.password, isSubmitting]
  );

  return (
    <>
      <AuthHeader />
      <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
        <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">
          <FormHeader
            heading="Manage your Vector instance"
            subHeading="Configure instance-wide settings to secure your instance"
          />
          <form
            ref={formRef}
            className="space-y-4"
            method="POST"
            action={`${API_BASE_URL}/api/instances/admins/sign-in/`}
            onSubmit={(event) => {
              event.preventDefault();
              void form.handleSubmit();
            }}
          >
            {errorData.type && errorData?.message ? (
              <Banner type="error" message={errorData?.message} />
            ) : (
              <>
                {errorInfo && (
                  <AuthBanner bannerData={errorInfo} handleBannerData={() => setDismissedErrorCode(errorCode)} />
                )}
              </>
            )}
            <input type="hidden" name="csrfmiddlewaretoken" />

            <AuthEmailInput
              label="Email *"
              readOnly={isSubmitting}
              value={formData.email}
              onChange={(event) => form.setFieldValue("email", event.target.value)}
              autoComplete="email"
            />

            <div className="w-full space-y-1">
              <label className="text-13 font-medium text-tertiary" htmlFor="password">
                Password <span className="text-danger-primary">*</span>
              </label>
              <div className="relative">
                <Input
                  readOnly={isSubmitting}
                  className="w-full border border-subtle !bg-surface-1 placeholder:text-placeholder"
                  id="password"
                  name="password"
                  type={showPassword ? "text" : "password"}
                  inputSize="md"
                  placeholder="Enter your password"
                  value={formData.password}
                  onChange={(e) => form.setFieldValue("password", e.target.value)}
                  autoComplete="off"
                />
                {showPassword ? (
                  <button
                    type="button"
                    aria-label="Hide password"
                    className="absolute top-3.5 right-3 flex items-center justify-center text-placeholder"
                    onClick={() => setShowPassword(false)}
                  >
                    <EyeOff className="h-4 w-4" />
                  </button>
                ) : (
                  <button
                    type="button"
                    aria-label="Show password"
                    className="absolute top-3.5 right-3 flex items-center justify-center text-placeholder"
                    onClick={() => setShowPassword(true)}
                  >
                    <Eye className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
            <div className="py-2">
              <Button type="submit" size="xl" className="w-full" disabled={isButtonDisabled}>
                {isSubmitting ? <Spinner height="20px" width="20px" /> : "Sign in"}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}
