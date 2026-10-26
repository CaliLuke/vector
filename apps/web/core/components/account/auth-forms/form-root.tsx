/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import React, { useState } from "react";

import { EAuthModes, EAuthSteps } from "@plane/constants";
import type { IEmailCheckData } from "@plane/types";
// helpers
import type { TAuthErrorInfo } from "@/helpers/authentication.helper";
import { authErrorHandler } from "@/helpers/authentication.helper";
// hooks
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";

// services
import { authService } from "@/services/auth.service";
// local components
import { EmailEntryForm } from "@plane/ui";
import { useTranslation } from "@plane/i18n";
import { AuthPasswordForm } from "./password";
import { AuthUniqueCodeForm } from "./unique-code";

type TAuthFormRoot = {
  authStep: EAuthSteps;
  authMode: EAuthModes;
  email: string;
  setEmail: (email: string) => void;
  setAuthMode: (authMode: EAuthModes) => void;
  setAuthStep: (authStep: EAuthSteps) => void;
  setErrorInfo: (errorInfo: TAuthErrorInfo | undefined) => void;
  currentAuthMode: EAuthModes;
};

export function AuthFormRoot(props: TAuthFormRoot) {
  const { t } = useTranslation();
  const { authStep, authMode, email, setEmail, setAuthMode, setAuthStep, setErrorInfo, currentAuthMode } = props;
  // router
  const router = useRouter();
  // query params
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const nextPath = searchParams.get("next_path");
  // states
  const [isExistingEmail, setIsExistingEmail] = useState(false);
  // hooks
  const { data: config } = useQuery(instanceConfigOptions);

  const isSMTPConfigured = config?.is_smtp_configured || false;

  // submit handler- email verification
  const handleEmailVerification = async (data: IEmailCheckData) => {
    setEmail(data.email);
    setErrorInfo(undefined);
    await authService
      .emailCheck(data)
      .then(async (response) => {
        if (response.existing) {
          if (currentAuthMode === EAuthModes.SIGN_UP) setAuthMode(EAuthModes.SIGN_IN);
          if (response.status === "MAGIC_CODE") {
            setAuthStep(EAuthSteps.UNIQUE_CODE);
            generateEmailUniqueCode(data.email);
          } else if (response.status === "CREDENTIAL") {
            setAuthStep(EAuthSteps.PASSWORD);
          }
        } else {
          if (currentAuthMode === EAuthModes.SIGN_IN) setAuthMode(EAuthModes.SIGN_UP);
          if (response.status === "MAGIC_CODE") {
            setAuthStep(EAuthSteps.UNIQUE_CODE);
            generateEmailUniqueCode(data.email);
          } else if (response.status === "CREDENTIAL") {
            setAuthStep(EAuthSteps.PASSWORD);
          }
        }
        setIsExistingEmail(response.existing);
      })
      .catch((error) => {
        const errorhandler = authErrorHandler(error?.error_code?.toString(), data?.email || undefined);
        if (errorhandler?.type) setErrorInfo(errorhandler);
      });
  };

  const handleEmailClear = () => {
    setAuthMode(currentAuthMode);
    setErrorInfo(undefined);
    setEmail("");
    setAuthStep(EAuthSteps.EMAIL);
    void router.navigate({ href: currentAuthMode === EAuthModes.SIGN_IN ? `/` : "/sign-up" });
  };

  // generating the unique code
  const generateEmailUniqueCode = async (email: string): Promise<void> => {
    if (!isSMTPConfigured) return;
    const payload = { email: email };
    return await authService.generateUniqueCode(payload).catch((error) => {
      const errorhandler = authErrorHandler(error?.error_code?.toString());
      if (errorhandler?.type) setErrorInfo(errorhandler);
      throw error;
    });
  };

  if (authStep === EAuthSteps.EMAIL) {
    return (
      <EmailEntryForm
        defaultEmail={email}
        onSubmit={handleEmailVerification}
        labels={{
          email: t("auth.common.email.label"),
          placeholder: t("auth.common.email.placeholder"),
          clear: t("aria_labels.auth_forms.clear_email"),
          invalid: t("auth.common.email.errors.invalid"),
          continue: t("common.continue"),
          failure: t("common.errors.default.message"),
        }}
      />
    );
  }
  if (authStep === EAuthSteps.UNIQUE_CODE) {
    return (
      <AuthUniqueCodeForm
        mode={authMode}
        email={email}
        isExistingEmail={isExistingEmail}
        handleEmailClear={handleEmailClear}
        generateEmailUniqueCode={generateEmailUniqueCode}
        nextPath={nextPath || undefined}
      />
    );
  }
  if (authStep === EAuthSteps.PASSWORD) {
    return (
      <AuthPasswordForm
        mode={authMode}
        isSMTPConfigured={isSMTPConfigured}
        email={email}
        handleEmailClear={handleEmailClear}
        handleAuthStep={(step: EAuthSteps) => {
          if (step === EAuthSteps.UNIQUE_CODE) generateEmailUniqueCode(email);
          setAuthStep(step);
        }}
        nextPath={nextPath || undefined}
      />
    );
  }

  return <></>;
}
